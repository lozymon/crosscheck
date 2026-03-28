package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lozymon/crosscheck/config"
	"github.com/lozymon/crosscheck/discovery"
)

var validateCmd = &cobra.Command{
	Use:   "validate [file or directory]",
	Short: "Validate test file schema without running tests",
	Long:  `Parses and validates *.cx.yaml files against the crosscheck schema. No HTTP requests or DB connections are made.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."

		if len(args) == 1 {
			path = args[0]
		}

		files, err := discovery.Find(path)

		if err != nil {
			return &ExitError{Code: ExitConfigError, Message: err.Error()}
		}

		if len(files) == 0 {
			fmt.Fprintln(os.Stderr, "no *.cx.yaml test files found")

			return nil
		}

		invalid := 0

		for _, file := range files {
			_, parseErr := config.Parse(file)

			if parseErr != nil {
				fmt.Fprintf(os.Stderr, "✗  %s\n   %v\n", file, parseErr)
				invalid++
			} else {
				fmt.Printf("✓  %s\n", file)
			}
		}

		if invalid > 0 {
			return &ExitError{Code: ExitConfigError}
		}

		return nil
	},
}
