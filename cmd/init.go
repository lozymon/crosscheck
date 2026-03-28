package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a crosscheck.cx.yaml starter file",
	Long:  `Creates a heavily commented crosscheck.cx.yaml in the current directory with yaml-language-server schema hint.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Creating crosscheck.cx.yaml...")
		// TODO: implement scaffold
		return nil
	},
}
