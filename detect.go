package nodeengine

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/buildpacks/libcnb/v2"
	"github.com/paketo-buildpacks/packit/v2/fs"
)

//go:generate faux --interface VersionParser --output fakes/version_parser.go
type VersionParser interface {
	ParseVersion(path string) (version string, err error)
}

func Detect(nvmrcParser, nodeVersionParser VersionParser) libcnb.DetectFunc {
	return func(context libcnb.DetectContext) (libcnb.DetectResult, error) {
		requirements := []libcnb.BuildPlanRequire{
			{Name: "syft"},
		}

		projectPath := context.ApplicationPath
		customProjPath := os.Getenv("BP_NODE_PROJECT_PATH")

		if customProjPath != "" {
			customProjPath = filepath.Clean(customProjPath)
			projectPath = filepath.Join(projectPath, customProjPath)
			exists, err := fs.Exists(projectPath)
			if err != nil {
				return libcnb.DetectResult{}, err
			}

			if !exists {
				return libcnb.DetectResult{},
					fmt.Errorf("expected value derived from BP_NODE_PROJECT_PATH [%s] to be an existing directory", projectPath)
			}
		}

		version, err := nvmrcParser.ParseVersion(filepath.Join(projectPath, NvmrcSource))
		if err != nil {
			return libcnb.DetectResult{}, err
		}

		if version != "" {
			requirements = append(requirements, libcnb.BuildPlanRequire{
				Name: Node,
				Metadata: map[string]interface{}{
					"version":        version,
					"version-source": NvmrcSource,
				},
			})
		}

		version = os.Getenv("BP_NODE_VERSION")
		if version != "" {
			requirements = append(requirements, libcnb.BuildPlanRequire{
				Name: Node,
				Metadata: map[string]interface{}{
					"version":        version,
					"version-source": "BP_NODE_VERSION",
				},
			})
		}

		version, err = nodeVersionParser.ParseVersion(filepath.Join(projectPath, NodeVersionSource))
		if err != nil {
			return libcnb.DetectResult{}, err
		}

		if version != "" {
			requirements = append(requirements, libcnb.BuildPlanRequire{
				Name: Node,
				Metadata: map[string]interface{}{
					"version":        version,
					"version-source": NodeVersionSource,
				},
			})
		}

		return libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: Node},
					},
					Requires: requirements,
				},
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: Node},
						{Name: Npm},
					},
					Requires: requirements,
				},
			},
		}, nil
	}
}
