package optimizememory

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/paketo-buildpacks/libpak/v2/sherpa"
)

type OptimizeMemory struct{}

func (o *OptimizeMemory) Execute() (map[string]string, error) {
	root := "/"

	var memAvailable string

	if memAvailable = sherpa.GetEnvWithDefault("MEMORY_AVAILABLE", ""); memAvailable == "" {
		var path string

		_, err := os.Stat(filepath.Join(root, "sys", "fs", "cgroup", "cgroup.controllers"))
		switch {
		case err == nil:
			path = filepath.Join(root, "sys", "fs", "cgroup", "memory.max")
		case errors.Is(err, os.ErrNotExist):
			path = filepath.Join(root, "sys", "fs", "cgroup", "memory", "memory.limit_in_bytes")
		default:
			return nil, err
		}

		if path != "" {
			content, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}

			memAvailable = strings.TrimSpace(string(content))
		}
	}

	variables := map[string]string{}
	if memAvailable != "" && memAvailable != "max" {
		memory, err := strconv.Atoi(memAvailable)
		if err != nil {
			return nil, err
		}

		variables["MEMORY_AVAILABLE"] = strconv.Itoa(memory / (1024 * 1024))
	}

	if sherpa.ResolveBool("OPTIMIZE_MEMORY") {
		if _, ok := variables["MEMORY_AVAILABLE"]; ok {
			memoryMax, err := strconv.Atoi(variables["MEMORY_AVAILABLE"])
			if err != nil {
				return nil, err
			}

			options := fmt.Sprintf("--max_old_space_size=%d", memoryMax*75/100)
			if opts := sherpa.GetEnvWithDefault("NODE_OPTIONS", ""); opts != "" {
				options = strings.Join([]string{opts, options}, " ")
			}

			variables["NODE_OPTIONS"] = options
		}
	}

	return variables, nil
}
