package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const initTemplate = `# yaml-language-server: $schema=https://crosscheck.dev/schema.json
version: 1
name: My API tests

# Fallback env defaults — real values go in .env (never commit secrets)
env:
  BASE_URL: http://localhost:3000

# Optional: authenticate once and inject the token into every request.
# Remove this block if your API does not require auth.
#
# auth:
#   type: login                        # or "static" for a fixed token
#   request:
#     method: POST
#     url: "{{ AUTH_SERVICE }}/auth/login"
#     body:
#       email: "{{ TEST_EMAIL }}"
#       password: "{{ TEST_PASSWORD }}"
#   capture:
#     token: "$.accessToken"           # save response field as var
#   inject:
#     header: Authorization
#     format: "Bearer {{ token }}"

# Optional: shell commands to run before all tests (seed DB, start mock, etc.)
#
# setup:
#   - run: "psql $DB_URL -f ./fixtures/seed.sql"

# Optional: shell commands to run after all tests — runs even on failure.
#
# teardown:
#   - run: "psql $DB_URL -f ./fixtures/cleanup.sql"

tests:
  - name: Health check
    request:
      method: GET
      url: "{{ BASE_URL }}/health"
    response:
      status: 200

  # Example: create a resource and capture its ID for later tests.
  #
  # - name: Create order
  #   request:
  #     method: POST
  #     url: "{{ BASE_URL }}/orders"
  #     body:
  #       productId: "abc-123"
  #   response:
  #     status: 201
  #     body:
  #       id: "{{ capture: orderId }}"   # stored as orderId for subsequent tests
  #       status: pending
  #
  # - name: Verify order in database
  #   database:
  #     - adapter: postgres
  #       query: "SELECT status FROM orders WHERE id = :orderId"
  #       params:
  #         orderId: "{{ orderId }}"
  #       wait_for:                      # poll until assertion passes or timeout
  #         timeout: 10s
  #         interval: 500ms
  #       expect:
  #         - status: pending
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a crosscheck.cx.yaml starter file",
	Long:  `Creates a heavily commented crosscheck.cx.yaml in the current directory with yaml-language-server schema hint.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		const filename = "crosscheck.cx.yaml"

		if _, err := os.Stat(filename); !errors.Is(err, os.ErrNotExist) {
			return &ExitError{
				Code:    ExitConfigError,
				Message: filename + " already exists — remove it first or edit it directly",
			}
		}

		if err := os.WriteFile(filename, []byte(initTemplate), 0o644); err != nil {
			return &ExitError{Code: ExitConfigError, Message: err.Error()}
		}

		fmt.Printf("Created %s\n", filename)
		fmt.Println("Edit it, then run:  cx run")

		return nil
	},
}
