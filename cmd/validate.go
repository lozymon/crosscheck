package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
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

		fmt.Printf("Validating: %s\n", path)
		// TODO: implement schema validation
		return nil
	},
}
