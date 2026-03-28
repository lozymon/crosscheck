package env

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Load merges variables from all sources in priority order:
// 1. CLI overrides (highest)
// 2. Shell environment
// 3. .env file
// 4. YAML env block (lowest — fallback defaults)
func Load(envFile string, cliOverrides []string, yamlDefaults map[string]string) map[string]string {
	vars := make(map[string]string)

	// 4. YAML defaults (lowest priority)
	for k, v := range yamlDefaults {
		vars[strings.ToUpper(k)] = v
	}

	// 3. .env file
	if fileVars, err := godotenv.Read(envFile); err == nil {
		for k, v := range fileVars {
			vars[k] = v
		}
	}

	// 2. Shell environment
	for _, e := range os.Environ() {
		k, v, _ := strings.Cut(e, "=")
		vars[k] = v
	}

	// 1. CLI --env KEY=VALUE overrides (highest priority)
	for _, override := range cliOverrides {
		k, v, found := strings.Cut(override, "=")
		if found {
			vars[k] = v
		}
	}

	return vars
}
