package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lozymon/crosscheck/config"
	"github.com/lozymon/crosscheck/env"
	"github.com/lozymon/crosscheck/httpclient"
	"github.com/lozymon/crosscheck/reporter"
	"github.com/lozymon/crosscheck/runner"
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

		return runTests(cmd, path)
	},
}

func init() {
	runCmd.Flags().StringVar(&runEnvFile, "env-file", ".env", "Path to .env file")
	runCmd.Flags().StringArrayVar(&runEnvVars, "env", nil, "Override env variables (KEY=VALUE)")
	runCmd.Flags().StringVar(&runFilter, "filter", "", "Run only tests matching pattern (e.g. 'order*')")
	runCmd.Flags().BoolVar(&runInsecure, "insecure", false, "Skip TLS certificate verification")
	runCmd.Flags().StringVar(&runOutputFile, "output-file", "", "Write JSON results to file")
	runCmd.Flags().StringVar(&runReporter, "reporter", "pretty", "Reporter format: pretty, json")
	runCmd.Flags().BoolVar(&runWatch, "watch", false, "Watch for file changes and re-run (Phase 2)")
}

func runTests(cmd *cobra.Command, path string) error {
	// Parse the test file.
	tf, err := config.Parse(path)

	if err != nil {
		return &ExitError{Code: ExitConfigError, Message: err.Error()}
	}

	// Build the variable namespace from all sources.
	vars := env.Load(runEnvFile, runEnvVars, tf.Env)

	// Build the HTTP client.
	client := httpclient.New(runInsecure)

	// Build the reporter — writes to stdout.
	rep, err := reporter.New(runReporter, os.Stdout)

	if err != nil {
		return &ExitError{Code: ExitConfigError, Message: err.Error()}
	}

	// Run the test file.
	result := runner.RunFile(cmd.Context(), tf, vars, client, runner.Options{})

	// Write reporter output.
	if writeErr := rep.Write(result); writeErr != nil {
		fmt.Fprintf(os.Stderr, "reporter error: %v\n", writeErr)
	}

	// Write JSON output file alongside pretty output if requested.
	if runOutputFile != "" {
		if writeErr := reporter.WriteJSONFile(runOutputFile, result); writeErr != nil {
			fmt.Fprintf(os.Stderr, "output-file error: %v\n", writeErr)
		}
	}

	// Determine exit code.
	if result.SetupErr != nil {
		return &ExitError{Code: ExitConnectError, Message: result.SetupErr.Error()}
	}

	if result.Failed > 0 {
		return &ExitError{Code: ExitTestFailure}
	}

	return nil
}
