package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lozymon/crosscheck/config"
)

var explainAI bool

var explainCmd = &cobra.Command{
	Use:   "explain <file>",
	Short: "Print a plain-English summary of a test file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if explainAI {
			// TODO: implement AI-powered explanation (Phase 2)
			fmt.Println("--ai mode not yet implemented")

			return nil
		}

		tf, err := config.Parse(args[0])

		if err != nil {
			return &ExitError{Code: ExitConfigError, Message: err.Error()}
		}

		printExplanation(tf)

		return nil
	},
}

func init() {
	explainCmd.Flags().BoolVar(&explainAI, "ai", false, "Use AI (Claude API) for richer explanation (requires CROSSCHECK_AI_KEY)")
}

// printExplanation writes a plain-English summary of the test file to stdout.
func printExplanation(tf *config.TestFile) {
	fmt.Printf("Suite: %s\n", tf.Name)

	if tf.Description != "" {
		fmt.Printf("       %s\n", tf.Description)
	}

	fmt.Println()

	// Auth block.
	if tf.Auth != nil {
		switch tf.Auth.Type {
		case "static":
			fmt.Printf("Auth:  static token injected as %s\n", tf.Auth.Inject.Header)
		case "login":
			url := ""

			if tf.Auth.Request != nil {
				url = tf.Auth.Request.URL
			}

			fmt.Printf("Auth:  login via %s %s\n", tf.Auth.Request.Method, url)

			if len(tf.Auth.Capture) > 0 {
				for varName, path := range tf.Auth.Capture {
					fmt.Printf("       captures %s from %s\n", varName, path)
				}
			}

			fmt.Printf("       injects %s header into every request\n", tf.Auth.Inject.Header)
		}

		fmt.Println()
	}

	// Setup / teardown hooks.
	if len(tf.Setup) > 0 {
		fmt.Printf("Setup: %d hook(s) run before all tests\n", len(tf.Setup))

		for _, h := range tf.Setup {
			fmt.Printf("       $ %s\n", h.Run)
		}

		fmt.Println()
	}

	if len(tf.Teardown) > 0 {
		fmt.Printf("Teardown: %d hook(s) run after all tests\n", len(tf.Teardown))

		for _, h := range tf.Teardown {
			fmt.Printf("          $ %s\n", h.Run)
		}

		fmt.Println()
	}

	// Tests.
	fmt.Printf("%d test(s):\n\n", len(tf.Tests))

	for i, t := range tf.Tests {
		fmt.Printf("  %d. %s\n", i+1, t.Name)

		if t.Description != "" {
			fmt.Printf("     %s\n", t.Description)
		}

		// Request.
		if t.Request != nil {
			fmt.Printf("     → %s %s\n", strings.ToUpper(t.Request.Method), t.Request.URL)
		}

		// Response assertions.
		if t.Response != nil {
			if t.Response.Status != 0 {
				fmt.Printf("     expects status %d\n", t.Response.Status)
			}

			if len(t.Response.Headers) > 0 {
				fmt.Printf("     expects %d response header(s)\n", len(t.Response.Headers))
			}

			if t.Response.Body != nil {
				captures, assertions := countBodyFields(t.Response.Body)

				if assertions > 0 {
					fmt.Printf("     asserts %d body field(s)\n", assertions)
				}

				if captures > 0 {
					fmt.Printf("     captures %d variable(s) from body\n", captures)
				}
			}
		}

		// Database assertions.
		for _, db := range t.Database {
			if db.WaitFor != nil {
				fmt.Printf("     polls %s (timeout %s): %s\n", db.Adapter, db.WaitFor.Timeout, db.Query)
			} else {
				fmt.Printf("     asserts %s: %s\n", db.Adapter, db.Query)
			}
		}

		fmt.Println()
	}
}

// countBodyFields walks the body map and counts capture directives vs plain assertions.
func countBodyFields(body any) (captures, assertions int) {
	m, ok := body.(map[string]any)

	if !ok {
		return 0, 0
	}

	for _, v := range m {
		switch val := v.(type) {
		case string:
			if strings.HasPrefix(strings.TrimSpace(val), "{{ capture:") {
				captures++
			} else {
				assertions++
			}
		case map[string]any:
			c, a := countBodyFields(val)
			captures += c
			assertions += a
		default:
			assertions++
		}
	}

	return captures, assertions
}
