package main

import (
	"os"

	"github.com/paketo-buildpacks/libnodejs"
	"github.com/paketo-buildpacks/libpak/v2"
	"github.com/paketo-buildpacks/libpak/v2/log"
	nodeengine "github.com/paketo-buildpacks/node-engine/v5"
)

func main() {
	nvmrcParser := nodeengine.NewNvmrcParser()
	nodeVersionParser := nodeengine.NewNodeVersionParser()
	logEmitter := log.NewPaketoLogger(os.Stdout)
	entryResolver := libnodejs.NewPlanner()
	libpak.BuildpackMain(
		nodeengine.Detect(
			nvmrcParser,
			nodeVersionParser,
		),
		nodeengine.Build(
			entryResolver,
			logEmitter,
		),
	)
}
