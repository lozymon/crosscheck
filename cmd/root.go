package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "crosscheck",
	Short: "cx — E2E API test runner with DB and service assertions",
	Long: `crosscheck (cx) runs end-to-end tests against backend APIs and validates
the result across multiple layers: HTTP response, database state, and cloud services.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(explainCmd)
}
