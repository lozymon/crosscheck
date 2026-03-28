package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lozymon/crosscheck/config"
)

func TestLoadGlobal_missingFile_notRequired(t *testing.T) {
	g, err := config.LoadGlobal("/nonexistent/.crosscheck.yaml", false)

	if err != nil {
		t.Fatalf("expected no error for missing optional file, got: %v", err)
	}

	if g == nil {
		t.Fatal("expected non-nil Global")
	}

	if g.Reporter != "" || g.Timeout != "" || g.Insecure || g.EnvFile != "" {
		t.Errorf("expected zero-value Global, got %+v", g)
	}
}

func TestLoadGlobal_missingFile_required(t *testing.T) {
	_, err := config.LoadGlobal("/nonexistent/.crosscheck.yaml", true)

	if err == nil {
		t.Fatal("expected error for missing required file")
	}
}

func TestLoadGlobal_validFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".crosscheck.yaml")

	content := `reporter: junit
timeout: 15s
insecure: true
env-file: .env.staging
`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	g, err := config.LoadGlobal(path, false)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if g.Reporter != "junit" {
		t.Errorf("expected reporter=junit, got %q", g.Reporter)
	}

	if g.Timeout != "15s" {
		t.Errorf("expected timeout=15s, got %q", g.Timeout)
	}

	if !g.Insecure {
		t.Error("expected insecure=true")
	}

	if g.EnvFile != ".env.staging" {
		t.Errorf("expected env-file=.env.staging, got %q", g.EnvFile)
	}
}

func TestLoadGlobal_emptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".crosscheck.yaml")

	if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	g, err := config.LoadGlobal(path, false)

	if err != nil {
		t.Fatalf("unexpected error for empty file: %v", err)
	}

	if g.Reporter != "" || g.Timeout != "" {
		t.Errorf("expected zero-value Global for empty file, got %+v", g)
	}
}

func TestLoadGlobal_invalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".crosscheck.yaml")

	if err := os.WriteFile(path, []byte("reporter: [bad\n"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	_, err := config.LoadGlobal(path, false)

	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}
