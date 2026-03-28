package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
)

// ExitError carries a specific exit code through cobra's error handling.
type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string {
	return e.Message
}

const (
	ExitTestFailure  = 1 // one or more tests failed
	ExitConfigError  = 2 // YAML / config problem
	ExitConnectError = 3 // DB or HTTP connection error
)

// Version is set at build time via -ldflags "-X github.com/lozymon/crosscheck/cmd.Version=x.y.z".
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "crosscheck",
	Short:   "cx — E2E API test runner with DB and service assertions",
	Version: Version,
	Long: `crosscheck (cx) runs end-to-end tests against backend APIs and validates
the result across multiple layers: HTTP response, database state, and cloud services.`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		var exitErr *ExitError

		if errors.As(err, &exitErr) {
			if exitErr.Message != "" {
				rootCmd.PrintErrln("Error:", exitErr.Message)
			}

			os.Exit(exitErr.Code)
		}

		rootCmd.PrintErrln("Error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(explainCmd)
}
