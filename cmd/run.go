package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	runEnvFile    string
	runEnvVars    []string
	runFilter     string
	runInsecure   bool
	runOutputFile string
	runReporter   string
	runWatch      bool
)

var runCmd = &cobra.Command{
	Use:   "run [file or directory]",
	Short: "Run test files",
	Long: `Run crosscheck test files (*.cx.yaml).

If no path is given, recursively finds all *.cx.yaml files in the current directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."

		if len(args) == 1 {
			path = args[0]
		}

		fmt.Printf("Running tests from: %s\n", path)
		// TODO: implement runner
		return nil
	},
}

func init() {
	runCmd.Flags().StringVar(&runEnvFile, "env-file", ".env", "Path to .env file")
	runCmd.Flags().StringArrayVar(&runEnvVars, "env", nil, "Override env variables (KEY=VALUE)")
	runCmd.Flags().StringVar(&runFilter, "filter", "", "Run only tests matching pattern (e.g. 'order*')")
	runCmd.Flags().BoolVar(&runInsecure, "insecure", false, "Skip TLS certificate verification")
	runCmd.Flags().StringVar(&runOutputFile, "output-file", "", "Write JSON results to file")
	runCmd.Flags().StringVar(&runReporter, "reporter", "pretty", "Reporter format: pretty, json, junit")
	runCmd.Flags().BoolVar(&runWatch, "watch", false, "Watch for file changes and re-run")
}
