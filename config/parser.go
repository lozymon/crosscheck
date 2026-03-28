package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const currentVersion = 1

// Parse reads and parses a *.cx.yaml file.
func Parse(path string) (*TestFile, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %w", path, err)
	}

	var tf TestFile

	if err := yaml.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("invalid YAML in %s: %w", path, err)
	}

	if tf.Version == 0 {
		fmt.Fprintf(os.Stderr, "warning: no version field in %s, assuming version %d\n", path, currentVersion)
		tf.Version = currentVersion
	}

	if tf.Version != currentVersion {
		return nil, fmt.Errorf(
			"%s: unsupported version %d (current: %d) — upgrade crosscheck",
			path, tf.Version, currentVersion,
		)
	}

	if tf.Name == "" {
		return nil, fmt.Errorf("%s: missing required field 'name'", path)
	}

	return &tf, nil
}
