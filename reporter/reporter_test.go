package reporter_test

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
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

func TestNew_unknownFormat(t *testing.T) {
	_, err := reporter.New("bogus", nil)

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

// ---- junit ----

// junitDoc is a minimal parse target for the XML produced by the JUnit reporter.
type junitDoc struct {
	XMLName  xml.Name `xml:"testsuites"`
	Tests    int      `xml:"tests,attr"`
	Failures int      `xml:"failures,attr"`
	Errors   int      `xml:"errors,attr"`
	Suites   []struct {
		Name     string `xml:"name,attr"`
		Tests    int    `xml:"tests,attr"`
		Failures int    `xml:"failures,attr"`
		Cases    []struct {
			Name    string `xml:"name,attr"`
			Failure *struct {
				Message string `xml:"message,attr"`
				Content string `xml:",chardata"`
			} `xml:"failure"`
			Error *struct {
				Message string `xml:"message,attr"`
			} `xml:"error"`
		} `xml:"testcase"`
	} `xml:"testsuite"`
}

func parseJUnit(t *testing.T, buf *bytes.Buffer) junitDoc {
	t.Helper()

	// Strip the XML declaration line before parsing the element.
	body := buf.String()
	if idx := strings.Index(body, "<testsuites"); idx >= 0 {
		body = body[idx:]
	}

	var doc junitDoc

	if err := xml.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("invalid JUnit XML: %v\noutput:\n%s", err, buf.String())
	}

	return doc
}

func TestJUnit_passingSuite(t *testing.T) {
	var buf bytes.Buffer

	r, err := reporter.New("junit", &buf)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err = r.Write(passingResult()); err != nil {
		t.Fatalf("write error: %v", err)
	}

	if err = r.Close(); err != nil {
		t.Fatalf("close error: %v", err)
	}

	doc := parseJUnit(t, &buf)

	if doc.Tests != 2 {
		t.Errorf("expected tests=2, got %d", doc.Tests)
	}

	if doc.Failures != 0 {
		t.Errorf("expected failures=0, got %d", doc.Failures)
	}

	if len(doc.Suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(doc.Suites))
	}

	if doc.Suites[0].Name != "order suite" {
		t.Errorf("expected suite name=order suite, got %q", doc.Suites[0].Name)
	}

	if len(doc.Suites[0].Cases) != 2 {
		t.Fatalf("expected 2 test cases, got %d", len(doc.Suites[0].Cases))
	}

	for _, tc := range doc.Suites[0].Cases {
		if tc.Failure != nil {
			t.Errorf("passing test %q should have no failure element", tc.Name)
		}
	}
}

func TestJUnit_failingSuite(t *testing.T) {
	var buf bytes.Buffer

	r, _ := reporter.New("junit", &buf)
	_ = r.Write(failingResult())
	_ = r.Close()

	doc := parseJUnit(t, &buf)

	if doc.Failures != 1 {
		t.Errorf("expected failures=1, got %d", doc.Failures)
	}

	failCase := doc.Suites[0].Cases[1]

	if failCase.Name != "fetch order" {
		t.Errorf("expected failing case=fetch order, got %q", failCase.Name)
	}

	if failCase.Failure == nil {
		t.Fatal("expected failure element on failing test case")
	}

	if !strings.Contains(failCase.Failure.Message, "status") {
		t.Errorf("expected failure message to contain 'status', got %q", failCase.Failure.Message)
	}

	if !strings.Contains(failCase.Failure.Content, "response") {
		t.Errorf("expected failure content to contain step 'response', got %q", failCase.Failure.Content)
	}
}

func TestJUnit_setupError(t *testing.T) {
	var buf bytes.Buffer

	r, _ := reporter.New("junit", &buf)
	_ = r.Write(&runner.FileResult{
		Name:     "suite",
		SetupErr: errors.New("seed.sql not found"),
	})
	_ = r.Close()

	doc := parseJUnit(t, &buf)

	if doc.Errors != 1 {
		t.Errorf("expected errors=1, got %d", doc.Errors)
	}

	if len(doc.Suites[0].Cases) != 1 {
		t.Fatalf("expected synthetic setup error case, got %d cases", len(doc.Suites[0].Cases))
	}

	if doc.Suites[0].Cases[0].Error == nil {
		t.Fatal("expected error element on setup error case")
	}
}

func TestJUnit_multiSuite(t *testing.T) {
	var buf bytes.Buffer

	r, _ := reporter.New("junit", &buf)
	_ = r.Write(passingResult())
	_ = r.Write(failingResult())
	_ = r.Close()

	doc := parseJUnit(t, &buf)

	if len(doc.Suites) != 2 {
		t.Fatalf("expected 2 suites, got %d", len(doc.Suites))
	}

	if doc.Tests != 4 {
		t.Errorf("expected tests=4 across both suites, got %d", doc.Tests)
	}

	if doc.Failures != 1 {
		t.Errorf("expected failures=1, got %d", doc.Failures)
	}
}

func TestJUnit_xmlDeclarationPresent(t *testing.T) {
	var buf bytes.Buffer

	r, _ := reporter.New("junit", &buf)
	_ = r.Write(passingResult())
	_ = r.Close()

	if !strings.HasPrefix(buf.String(), "<?xml") {
		t.Errorf("expected XML declaration at start of output, got:\n%s", buf.String()[:min(80, buf.Len())])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}
