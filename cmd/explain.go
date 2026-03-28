package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var explainAI bool

var explainCmd = &cobra.Command{
	Use:   "explain <file>",
	Short: "Print a plain-English summary of a test file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Explaining: %s\n", args[0])

		if explainAI {
			// TODO: implement AI-powered explanation (Phase 2)
			fmt.Println("--ai mode not yet implemented")
			return nil
		}
		// TODO: implement static explanation
		return nil
	},
}

func init() {
	explainCmd.Flags().BoolVar(&explainAI, "ai", false, "Use AI (Claude API) for richer explanation (requires CROSSCHECK_AI_KEY)")
}
