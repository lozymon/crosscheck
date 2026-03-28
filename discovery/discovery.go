// Package discovery finds *.cx.yaml test files on the filesystem.
package discovery

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Find returns the list of test files to run for the given path.
//
//   - If path is a file, it is returned as-is (any .yaml suffix accepted).
//   - If path is a directory, all *.cx.yaml files are found recursively.
func Find(path string) ([]string, error) {
	info, err := os.Stat(path)

	if err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}

	if !info.IsDir() {
		return []string{path}, nil
	}

	return walkDir(path)
}

// walkDir recursively finds all *.cx.yaml files under root.
func walkDir(root string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(path, ".cx.yaml") {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}

	return files, nil
}
