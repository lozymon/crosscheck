// Package hooks executes shell commands for setup and teardown steps.
// Commands run via /bin/sh so they support pipes, redirects, and $VAR expansion.
// Variables from the crosscheck namespace are injected as environment variables
// so $DB_URL, $BASE_URL, etc. are available inside every command.
package hooks

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/lozymon/crosscheck/config"
)

// Run executes a single hook command.
// vars are merged into the process environment so $KEY expands inside the shell.
// stdout and stderr from the command are captured and included in any error message.
func Run(ctx context.Context, hook config.Hook, vars map[string]string) error {
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", hook.Run)

	// Start with the current process environment so system tools (psql, docker, etc.)
	// are still on PATH. Then layer in the crosscheck vars on top.
	cmd.Env = os.Environ()

	for k, v := range vars {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	var out bytes.Buffer

	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()

	if err != nil {
		return fmt.Errorf("hook %q failed: %w\n%s", hook.Run, err, out.String())
	}

	return nil
}

// RunAll executes a slice of hooks in order, stopping on the first failure.
func RunAll(ctx context.Context, hooks []config.Hook, vars map[string]string) error {
	for _, h := range hooks {
		if err := Run(ctx, h, vars); err != nil {
			return err
		}
	}

	return nil
}
