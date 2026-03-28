package hooks_test

import (
	"context"
	"testing"

	"github.com/lozymon/crosscheck/config"
	"github.com/lozymon/crosscheck/hooks"
)

func TestRun_success(t *testing.T) {
	err := hooks.Run(context.Background(), config.Hook{Run: "echo hello"}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_failure(t *testing.T) {
	err := hooks.Run(context.Background(), config.Hook{Run: "exit 1"}, nil)

	if err == nil {
		t.Fatal("expected error for failing command")
	}
}

func TestRun_errorIncludesOutput(t *testing.T) {
	err := hooks.Run(context.Background(), config.Hook{Run: "echo 'something went wrong' && exit 1"}, nil)

	if err == nil {
		t.Fatal("expected error")
	}

	if msg := err.Error(); msg == "" {
		t.Error("expected error message to contain command output")
	}
}

func TestRun_varInjection(t *testing.T) {
	// The hook uses $GREET (shell expansion), vars injects GREET=hi.
	// If the var is missing the command will fail with a non-zero exit.
	err := hooks.Run(context.Background(), config.Hook{
		Run: `test "$GREET" = "hi"`,
	}, map[string]string{"GREET": "hi"})

	if err != nil {
		t.Fatalf("expected var to be injected into shell env: %v", err)
	}
}

func TestRun_contextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := hooks.Run(ctx, config.Hook{Run: "sleep 10"}, nil)

	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
}

func TestRunAll_stopsOnFirstFailure(t *testing.T) {
	var ranSecond bool

	// Use a temp file as a side-effect marker — if second hook runs, it creates the file.
	// We check the second hook never executes.
	hookList := []config.Hook{
		{Run: "exit 1"},
		{Run: "touch /tmp/crosscheck_hook_test_ran"},
	}

	// Clean up before and after.
	_ = hooks.Run(context.Background(), config.Hook{Run: "rm -f /tmp/crosscheck_hook_test_ran"}, nil)

	err := hooks.RunAll(context.Background(), hookList, nil)

	if err == nil {
		t.Fatal("expected error from first failing hook")
	}

	// Verify second hook never ran.
	checkErr := hooks.Run(context.Background(), config.Hook{
		Run: `test ! -f /tmp/crosscheck_hook_test_ran`,
	}, nil)

	if checkErr != nil {
		ranSecond = true
	}

	if ranSecond {
		t.Error("RunAll should have stopped after first failure, but second hook ran")
	}
}

func TestRunAll_empty(t *testing.T) {
	err := hooks.RunAll(context.Background(), nil, nil)

	if err != nil {
		t.Fatalf("empty hook list should not error: %v", err)
	}
}

func TestRunAll_allPass(t *testing.T) {
	hookList := []config.Hook{
		{Run: "echo one"},
		{Run: "echo two"},
		{Run: "echo three"},
	}

	err := hooks.RunAll(context.Background(), hookList, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
