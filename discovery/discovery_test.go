package discovery_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lozymon/crosscheck/discovery"
)

func TestFind_specificFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "orders.cx.yaml")
	_ = os.WriteFile(f, []byte(""), 0o644)

	files, err := discovery.Find(f)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 || files[0] != f {
		t.Errorf("expected [%s], got %v", f, files)
	}
}

func TestFind_nonYamlFileAccepted(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "orders.yaml")
	_ = os.WriteFile(f, []byte(""), 0o644)

	files, err := discovery.Find(f)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("explicit file path should be accepted regardless of suffix, got %v", files)
	}
}

func TestFind_directory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	_ = os.MkdirAll(sub, 0o755)

	files := []string{
		filepath.Join(dir, "orders.cx.yaml"),
		filepath.Join(sub, "auth.cx.yaml"),
		filepath.Join(dir, "ignored.yaml"), // not *.cx.yaml — should be skipped
	}

	for _, f := range files {
		_ = os.WriteFile(f, []byte(""), 0o644)
	}

	found, err := discovery.Find(dir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(found) != 2 {
		t.Errorf("expected 2 *.cx.yaml files, got %d: %v", len(found), found)
	}
}

func TestFind_emptyDirectory(t *testing.T) {
	dir := t.TempDir()

	found, err := discovery.Find(dir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(found) != 0 {
		t.Errorf("expected no files, got %v", found)
	}
}

func TestFind_pathNotExist(t *testing.T) {
	_, err := discovery.Find("/no/such/path/here")

	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
}
