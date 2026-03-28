package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"github.com/lozymon/crosscheck/discovery"
)

const (
	watchDebounceDelay = 300 // milliseconds — coalesces rapid saves into one run
)

// watchAndRun runs the test suite once, then re-runs it whenever any
// *.cx.yaml file in path changes. It blocks until the command context is
// cancelled (Ctrl+C).
func watchAndRun(cmd *cobra.Command, path string) error {
	// Initial run.
	_ = runTests(cmd, path)

	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		return fmt.Errorf("watch: %w", err)
	}

	defer func() { _ = watcher.Close() }()

	// Watch the root path (directory or the file's parent directory).
	watchRoot := path

	if info, statErr := os.Stat(path); statErr == nil && !info.IsDir() {
		watchRoot = filepath.Dir(path)
	}

	if err = watcher.Add(watchRoot); err != nil {
		return fmt.Errorf("watch add %s: %w", watchRoot, err)
	}

	// Also watch any subdirectories that currently contain *.cx.yaml files.
	if files, discErr := discovery.Find(path); discErr == nil {
		seen := map[string]bool{watchRoot: true}

		for _, f := range files {
			dir := filepath.Dir(f)

			if !seen[dir] {
				seen[dir] = true
				_ = watcher.Add(dir)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "\nWatching for changes... (Ctrl+C to stop)\n")

	// trigger is a 1-buffered channel; events are coalesced — if a run is
	// already queued we don't queue another one.
	trigger := make(chan string, 1)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if isCXYAML(event.Name) && isWriteOrCreate(event.Op) {
					select {
					case trigger <- event.Name:
					default: // already queued
					}
				}

			case watchErr, ok := <-watcher.Errors:
				if !ok {
					return
				}

				fmt.Fprintf(os.Stderr, "watcher error: %v\n", watchErr)
			}
		}
	}()

	for {
		select {
		case <-cmd.Context().Done():
			return nil

		case changedFile := <-trigger:
			// Simple debounce: drain any events that arrive within the window.
			drainEvents(trigger, watchDebounceDelay)

			fmt.Fprintf(os.Stderr, "\n--- %s changed ---\n", filepath.Base(changedFile))

			_ = runTests(cmd, path)

			fmt.Fprintf(os.Stderr, "\nWatching for changes... (Ctrl+C to stop)\n")
		}
	}
}

// drainEvents sleeps for windowMS milliseconds then discards any events that
// accumulated in trigger during that window, coalescing rapid saves into one run.
func drainEvents(trigger <-chan string, windowMS int) {
	time.Sleep(time.Duration(windowMS) * time.Millisecond)

	for {
		select {
		case <-trigger:
		default:
			return
		}
	}
}

// isCXYAML returns true for files that match *.cx.yaml or *.yaml
// (consistent with discovery.Find which accepts any .yaml for explicit paths).
func isCXYAML(name string) bool {
	return strings.HasSuffix(name, ".cx.yaml") || strings.HasSuffix(name, ".yaml")
}

// isWriteOrCreate returns true for write or create fsnotify operations.
func isWriteOrCreate(op fsnotify.Op) bool {
	return op.Has(fsnotify.Write) || op.Has(fsnotify.Create)
}
