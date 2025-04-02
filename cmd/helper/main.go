package main

import (
	"github.com/paketo-buildpacks/libpak/v2/sherpa"
	"github.com/paketo-buildpacks/node-engine/v5/helper/inspector"
	optimizememory "github.com/paketo-buildpacks/node-engine/v5/helper/optimize-memory"
)

func main() {
	sherpa.Execute(func() error {
		return sherpa.Helpers(map[string]sherpa.ExecD{
			"optimize-memory": &optimizememory.OptimizeMemory{},
			"inspector":       &inspector.Inspector{},
		})
	})
}
