// Package reporter formats and writes FileResult output.
// Two formats are supported:
//   - "pretty" — colored human-readable terminal output (default)
//   - "json"   — structured JSON, suitable for tooling and --output-file
package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"

	"github.com/lozymon/crosscheck/runner"
)

// Reporter writes a FileResult to an output stream.
type Reporter interface {
	Write(result *runner.FileResult) error
}

// New returns the Reporter for the given format name.
// Supported formats: "pretty", "json".
func New(format string, w io.Writer) (Reporter, error) {
	switch strings.ToLower(format) {
	case "pretty", "":
		return &prettyReporter{w: w}, nil
	case "json":
		return &jsonReporter{w: w}, nil
	default:
		return nil, fmt.Errorf("unknown reporter format %q: must be \"pretty\" or \"json\"", format)
	}
}

// WriteJSONFile writes a JSON representation of result to the given file path.
// Used alongside --output-file so the pretty reporter can still write to stdout
// while a machine-readable copy is saved to disk.
func WriteJSONFile(path string, result *runner.FileResult) error {
	f, err := os.Create(path)

	if err != nil {
		return fmt.Errorf("output-file: %w", err)
	}

	defer func() { _ = f.Close() }()

	return writeJSON(f, result)
}

// ---- pretty ----

var (
	passColor    = color.New(color.FgGreen, color.Bold)
	failColor    = color.New(color.FgRed, color.Bold)
	dimColor     = color.New(color.FgHiBlack)
	messageColor = color.New(color.FgRed)
)

type prettyReporter struct {
	w io.Writer
}

func (r *prettyReporter) Write(result *runner.FileResult) error {
	w := &errWriter{w: r.w}

	// Suite header.
	w.printf("\n%s\n\n", result.Name)

	if result.SetupErr != nil {
		w.color(failColor, "  setup failed: %v\n", result.SetupErr)
		w.printf("\n")

		return w.err
	}

	// One line per test.
	for _, tr := range result.Tests {
		retryNote := ""

		if tr.Attempts > 1 {
			retryNote = fmt.Sprintf(" (%d attempts)", tr.Attempts)
		}

		if tr.Passed {
			w.color(passColor, "  ✓  ")
			w.printf("%s", tr.Name)
			w.color(dimColor, "%s\n", retryNote)
		} else {
			w.color(failColor, "  ✗  ")
			w.printf("%s", tr.Name)
			w.color(dimColor, "%s\n", retryNote)

			if tr.Err != nil {
				w.color(messageColor, "     error: %v\n", tr.Err)
			}

			for _, f := range tr.Failures {
				w.color(messageColor, "     %s: %s\n", f.Step, f.Message)
			}
		}
	}

	// Teardown error (non-fatal but worth surfacing).
	if result.TeardownErr != nil {
		w.printf("\n")
		w.color(failColor, "  teardown failed: %v\n", result.TeardownErr)
	}

	// Summary line.
	w.printf("\n")

	total := result.Passed + result.Failed
	summary := fmt.Sprintf("  %d test", total)

	if total != 1 {
		summary += "s"
	}

	w.printf("%s", summary)
	w.color(passColor, "  %d passed", result.Passed)

	if result.Failed > 0 {
		w.printf("  ")
		w.color(failColor, "%d failed", result.Failed)
	}

	w.color(dimColor, "\n")

	return w.err
}

// errWriter accumulates the first write error so callers don't have to check each call.
type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) printf(format string, args ...any) {
	if ew.err != nil {
		return
	}

	_, ew.err = fmt.Fprintf(ew.w, format, args...)
}

func (ew *errWriter) color(c *color.Color, format string, args ...any) {
	if ew.err != nil {
		return
	}

	_, ew.err = c.Fprintf(ew.w, format, args...)
}

// ---- json ----

// jsonResult is the JSON shape written to stdout or --output-file.
type jsonResult struct {
	Suite         string           `json:"suite"`
	Passed        int              `json:"passed"`
	Failed        int              `json:"failed"`
	SetupError    string           `json:"setup_error,omitempty"`
	TeardownError string           `json:"teardown_error,omitempty"`
	Tests         []jsonTestResult `json:"tests"`
}

type jsonTestResult struct {
	Name     string        `json:"name"`
	Passed   bool          `json:"passed"`
	Attempts int           `json:"attempts,omitempty"`
	Failures []jsonFailure `json:"failures,omitempty"`
	Error    string        `json:"error,omitempty"`
}

type jsonFailure struct {
	Step    string `json:"step"`
	Message string `json:"message"`
}

type jsonReporter struct {
	w io.Writer
}

func (r *jsonReporter) Write(result *runner.FileResult) error {
	return writeJSON(r.w, result)
}

func writeJSON(w io.Writer, result *runner.FileResult) error {
	out := jsonResult{
		Suite:  result.Name,
		Passed: result.Passed,
		Failed: result.Failed,
	}

	if result.SetupErr != nil {
		out.SetupError = result.SetupErr.Error()
	}

	if result.TeardownErr != nil {
		out.TeardownError = result.TeardownErr.Error()
	}

	for _, tr := range result.Tests {
		jtr := jsonTestResult{
			Name:     tr.Name,
			Passed:   tr.Passed,
			Attempts: tr.Attempts,
		}

		if tr.Err != nil {
			jtr.Error = tr.Err.Error()
		}

		for _, f := range tr.Failures {
			jtr.Failures = append(jtr.Failures, jsonFailure{
				Step:    f.Step,
				Message: f.Message,
			})
		}

		out.Tests = append(out.Tests, jtr)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("reporter json encode: %w", err)
	}

	return nil
}
