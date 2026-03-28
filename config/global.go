package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DefaultGlobalConfigPath is the file discovered automatically at the project root.
const DefaultGlobalConfigPath = ".crosscheck.yaml"

// Global holds project-wide defaults loaded from .crosscheck.yaml.
// CLI flags always take precedence over these values.
//
// Example file:
//
//	reporter: pretty
//	timeout:  10s
//	insecure: false
//	env-file: .env
type Global struct {
	Reporter string `yaml:"reporter"`
	Timeout  string `yaml:"timeout"`
	Insecure bool   `yaml:"insecure"`
	EnvFile  string `yaml:"env-file"`
}

// LoadGlobal reads the global config from path.
//
// If the file does not exist and required is false, an empty Global is returned
// with no error — this is the normal case for the auto-discovered default path.
// If required is true (the user passed --config explicitly), a missing file is
// an error.
func LoadGlobal(path string, required bool) (*Global, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !required {
			return &Global{}, nil
		}

		return nil, fmt.Errorf("global config %s: %w", path, err)
	}

	var g Global

	if err = yaml.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("global config %s: %w", path, err)
	}

	return &g, nil
}
