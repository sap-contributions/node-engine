package main

import (
	"os"

	"github.com/paketo-buildpacks/libpak/v2"
	"github.com/paketo-buildpacks/libpak/v2/log"
	nodeengine "github.com/paketo-buildpacks/node-engine/v5"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/postal"
)

func main() {
	nvmrcParser := nodeengine.NewNvmrcParser()
	nodeVersionParser := nodeengine.NewNodeVersionParser()
	logEmitter := log.NewPaketoLogger(os.Stdout)
	entryResolver := draft.NewPlanner()
	dependencyManager := postal.NewService(cargo.NewTransport())
	libpak.BuildpackMain(
		nodeengine.Detect(
			nvmrcParser,
			nodeVersionParser,
		),
		nodeengine.Build(
			entryResolver,
			dependencyManager,
			logEmitter,
			chronos.DefaultClock,
		),
	)
}
