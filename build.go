package nodeengine

import (
	"fmt"
	"os"
	"strconv"

	"github.com/buildpacks/libcnb/v2"
	"github.com/paketo-buildpacks/libnodejs"
	"github.com/paketo-buildpacks/libpak/v2"
	"github.com/paketo-buildpacks/libpak/v2/effect"
	"github.com/paketo-buildpacks/libpak/v2/log"
	"github.com/paketo-buildpacks/libpak/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
type EntryResolver interface {
	Resolve(name string, entries []libcnb.BuildpackPlanEntry, priorities []interface{}) (libcnb.BuildpackPlanEntry, []libcnb.BuildpackPlanEntry)
	MergeLayerTypes(name string, entries []libcnb.BuildpackPlanEntry) (launch bool, build bool)
}

func IsLayerReusable(nodeLayer libcnb.Layer, depChecksum string, build bool, launch bool, logger log.Logger) bool {
	logger.Debug("Checking if layer %s can be reused", nodeLayer.Path)

	metadata := nodeLayer.Metadata
	cachedChecksum, _ := metadata[DepKey].(string)
	logger.Debug("Checksum of dependency: %s", depChecksum)
	logger.Debug("Checksum of layer: %s", cachedChecksum)

	cachedBuild, found := metadata[BuildKey].(bool)
	buildOK := found && (build == cachedBuild)
	logger.Debug("Build requirements match: %v", buildOK)

	cachedLaunch, found := metadata[LaunchKey].(bool)
	launchOK := found && (launch == cachedLaunch)
	logger.Debug("Launch requirements match: %v", launchOK)

	return cargo.Checksum(depChecksum).MatchString(cachedChecksum) && buildOK && launchOK
}

func Build(entryResolver EntryResolver, logger log.Logger, clock chronos.Clock) libcnb.BuildFunc {
	return func(context libcnb.BuildContext) (libcnb.BuildResult, error) {
		logger.Title("%s %s", context.Buildpack.Info.Name, context.Buildpack.Info.Version)

		nodeLayer, err := context.Layers.Layer(Node)
		if err != nil {
			return libcnb.BuildResult{}, err
		}

		logger.Header("Resolving Node Engine version")

		entry, allEntries := libnodejs.ResolveNodeVersion(entryResolver.Resolve, context.Plan)
		if entry.Name == "" && len(allEntries) == 0 {
			logger.Header("Node no longer requested by plan, satisfied by extension")

			logger.Header("Setting up launch layer for environment variables")
			nodeLayer, err = nodeLayer.Reset()
			if err != nil {
				return libcnb.BuildResult{}, err
			}

			nodeLayer.Launch, nodeLayer.Build, nodeLayer.Cache = true, true, false
			nodeLayer.Metadata = map[string]interface{}{
				BuildKey:  true,
				LaunchKey: true,
			}

			nodeLayer.SharedEnvironment.Default("NODE_HOME", "")
		} else {
			logger.Body(allEntries)

			version, _ := entry.Metadata["version"].(string)
			buildpackMeta, err := libpak.NewBuildModuleMetadata(context.Buildpack.Metadata)
			if err != nil {
				return libcnb.BuildResult{}, fmt.Errorf("unable to create build module metadata\n%w", err)
			}
			dr, err := libpak.NewDependencyResolver(buildpackMeta, entry.Name)
			if err != nil {
				return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency resolver\n%w", err)
			}

			dc, err := libpak.NewDependencyCache(context.Buildpack.Info.ID, context.Buildpack.Info.Version, context.Buildpack.Path, context.Platform.Bindings, logger)
			if err != nil {
				return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency cache\n%w", err)
			}

			dependency, err := dr.Resolve(entry.Name, version)
			if err != nil {
				return libcnb.BuildResult{}, err
			}

			logger.Body(entry, dependency, clock.Now())

			sbomDisabled, err := checkSbomDisabled()
			if err != nil {
				return libcnb.BuildResult{}, err
			}

			launch, build := entryResolver.MergeLayerTypes("node", context.Plan.Entries)

			if IsLayerReusable(nodeLayer, dependency.SHA256, build, launch, logger) {
				logger.Header("Reusing cached layer %s", nodeLayer.Path)

				nodeLayer.Launch, nodeLayer.Build, nodeLayer.Cache = launch, build, build
				return libcnb.BuildResult{
					Layers: []libcnb.Layer{nodeLayer},
				}, nil
			}

			nodeLayer, err = nodeLayer.Reset()
			if err != nil {
				return libcnb.BuildResult{}, err
			}

			nodeLayer.Launch, nodeLayer.Build, nodeLayer.Cache = launch, build, build

			nodeLayer.Metadata = map[string]interface{}{
				DepKey:    dependency.SHA256,
				BuildKey:  build,
				LaunchKey: launch,
			}

			logger.Titlef("Installing Node Engine %s", dependency.Version)
			lc := libpak.NewDependencyLayerContributor(dependency, dc, libcnb.LayerTypes{
				Build:  build,
				Cache:  true,
				Launch: launch,
			}, logger)

			if err = lc.Contribute(&nodeLayer, nil); err != nil {
				return libcnb.BuildResult{}, err
			}

			if sbomDisabled {
				logger.Header("Skipping SBOM generation for Node Engine")
			} else {
				sbomScanner := sbom.NewSyftCLISBOMScanner(context.Layers, effect.CommandExecutor{}, logger)
				if err := sbomScanner.ScanLayer(nodeLayer, nodeLayer.Path, libcnb.SyftJSON, libcnb.CycloneDXJSON); err != nil {
					return libcnb.BuildResult{}, fmt.Errorf("unable to create Launch SBoM \n%w", err)
				}
			}
			nodeLayer.SharedEnvironment.Default("NODE_HOME", nodeLayer.Path)
		}

		var optimizedMemory bool
		if os.Getenv("BP_NODE_OPTIMIZE_MEMORY") == "true" {
			optimizedMemory = true
		}

		nodeLayer.SharedEnvironment.Default("NODE_ENV", "production")
		nodeLayer.SharedEnvironment.Default("NODE_VERBOSE", "false")
		nodeLayer.SharedEnvironment.Default("NODE_OPTIONS", "--use-openssl-ca")
		if optimizedMemory {
			nodeLayer.LaunchEnvironment.Default("OPTIMIZE_MEMORY", "true")
		}

		if err = libpak.NewHelperLayerContributor(context.Buildpack, logger, "optimize-memory", "inspector").Contribute(&nodeLayer); err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to contribute helpers: %w", nodeLayer.Path, err)
		}

		logger.Header("Writing exec.d/0-optimize-memory")
		logger.Body("Calculates available memory based on container limits at launch time.")
		logger.Body("Made available in the MEMORY_AVAILABLE environment variable.")
		if optimizedMemory {
			logger.Body("Assigns the NODE_OPTIONS environment variable with flag setting to optimize memory.")
			logger.Body("Limits the total size of all objects on the heap to 75%% of the MEMORY_AVAILABLE.")
		}
		logger.Header("Writing exec.d/1-inspector")

		return libcnb.BuildResult{
			Layers: []libcnb.Layer{nodeLayer},
		}, nil
	}
}

func checkSbomDisabled() (bool, error) {
	if disableStr, ok := os.LookupEnv("BP_DISABLE_SBOM"); ok {
		disable, err := strconv.ParseBool(disableStr)
		if err != nil {
			return false, fmt.Errorf("failed to parse BP_DISABLE_SBOM value %s: %w", disableStr, err)
		}
		return disable, nil
	}
	return false, nil
}
