package reporter_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/lozymon/crosscheck/reporter"
	"github.com/lozymon/crosscheck/runner"
)

func passingResult() *runner.FileResult {
	return &runner.FileResult{
		Name:   "order suite",
		Passed: 2,
		Failed: 0,
		Tests: []runner.TestResult{
			{Name: "create order", Passed: true},
			{Name: "fetch order", Passed: true},
		},
	}
}

func failingResult() *runner.FileResult {
	return &runner.FileResult{
		Name:   "order suite",
		Passed: 1,
		Failed: 1,
		Tests: []runner.TestResult{
			{Name: "create order", Passed: true},
			{
				Name:   "fetch order",
				Passed: false,
				Failures: []runner.Failure{
					{Step: "response", Message: `status: expected "200", got "404"`},
				},
			},
		},
	}
}

// ---- pretty ----

func TestPretty_passingSuite(t *testing.T) {
	var buf bytes.Buffer

	r, err := reporter.New("pretty", &buf)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err = r.Write(passingResult()); err != nil {
		t.Fatalf("write error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "create order") {
		t.Error("expected test name in output")
	}

	if !strings.Contains(out, "2 tests") {
		t.Errorf("expected summary '2 tests', got:\n%s", out)
	}

	if !strings.Contains(out, "2 passed") {
		t.Errorf("expected '2 passed' in output, got:\n%s", out)
	}
}

func TestPretty_failingSuite(t *testing.T) {
	var buf bytes.Buffer

	r, _ := reporter.New("pretty", &buf)
	_ = r.Write(failingResult())

	out := buf.String()

	if !strings.Contains(out, "fetch order") {
		t.Error("expected failing test name in output")
	}

	if !strings.Contains(out, "1 failed") {
		t.Errorf("expected '1 failed' in output, got:\n%s", out)
	}

	if !strings.Contains(out, `status: expected "200", got "404"`) {
		t.Errorf("expected failure message in output, got:\n%s", out)
	}
}

func TestPretty_setupError(t *testing.T) {
	var buf bytes.Buffer

	r, _ := reporter.New("pretty", &buf)
	_ = r.Write(&runner.FileResult{
		Name:     "suite",
		SetupErr: errors.New("seed.sql not found"),
	})

	out := buf.String()

	if !strings.Contains(out, "setup failed") {
		t.Errorf("expected 'setup failed' in output, got:\n%s", out)
	}
}

func TestPretty_unknownFormat(t *testing.T) {
	_, err := reporter.New("junit", nil)

	if err == nil {
		t.Fatal("expected error for unknown format")
	}
}

// ---- json ----

func TestJSON_structure(t *testing.T) {
	var buf bytes.Buffer

	r, err := reporter.New("json", &buf)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err = r.Write(failingResult()); err != nil {
		t.Fatalf("write error: %v", err)
	}

	var out struct {
		Suite  string `json:"suite"`
		Passed int    `json:"passed"`
		Failed int    `json:"failed"`
		Tests  []struct {
			Name     string `json:"name"`
			Passed   bool   `json:"passed"`
			Failures []struct {
				Step    string `json:"step"`
				Message string `json:"message"`
			} `json:"failures"`
		} `json:"tests"`
	}

	if err = json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\noutput:\n%s", err, buf.String())
	}

	if out.Suite != "order suite" {
		t.Errorf("expected suite=order suite, got %q", out.Suite)
	}

	if out.Passed != 1 || out.Failed != 1 {
		t.Errorf("expected passed=1 failed=1, got passed=%d failed=%d", out.Passed, out.Failed)
	}

	if len(out.Tests) != 2 {
		t.Fatalf("expected 2 tests, got %d", len(out.Tests))
	}

	if out.Tests[1].Passed {
		t.Error("second test should not be passed")
	}

	if len(out.Tests[1].Failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(out.Tests[1].Failures))
	}

	if out.Tests[1].Failures[0].Step != "response" {
		t.Errorf("expected step=response, got %q", out.Tests[1].Failures[0].Step)
	}
}

func TestJSON_allPassed(t *testing.T) {
	var buf bytes.Buffer

	r, _ := reporter.New("json", &buf)
	_ = r.Write(passingResult())

	var out map[string]any

	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if out["failed"].(float64) != 0 {
		t.Errorf("expected failed=0, got %v", out["failed"])
	}
}
