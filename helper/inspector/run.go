package inspector

import (
	"fmt"
	"strings"

	"github.com/paketo-buildpacks/libpak/v2/sherpa"
)

type Inspector struct{}

func (i *Inspector) Execute() (map[string]string, error) {
	variables := map[string]string{}

	sherpa.GetEnvWithDefault("BPL_DEBUG_PORT", "")

	if sherpa.ResolveBool("BPL_DEBUG_ENABLED") {
		option := "--inspect"
		if debugPort := sherpa.GetEnvWithDefault("BPL_DEBUG_PORT", ""); debugPort != "" {
			option = fmt.Sprintf("%s=127.0.0.1:%s", option, debugPort)
		}

		if nodeOpts := sherpa.GetEnvWithDefault("NODE_OPTIONS", ""); nodeOpts != "" {
			if strings.Contains(nodeOpts, "--inspect") {
				return nil, nil
			}

			option = strings.Join([]string{nodeOpts, option}, " ")
		}

		variables["NODE_OPTIONS"] = option
	}

	return variables, nil
}
