package reporter

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/lozymon/crosscheck/runner"
)

// junitReporter buffers FileResults and writes a single JUnit XML document
// on Close(). This is necessary because JUnit's <testsuites> wrapper must
// contain all suites, but Write is called once per file.
type junitReporter struct {
	w       io.Writer
	results []*runner.FileResult
}

func (r *junitReporter) Write(result *runner.FileResult) error {
	r.results = append(r.results, result)

	return nil
}

// Close writes the complete JUnit XML document to the underlying writer.
func (r *junitReporter) Close() error {
	doc := buildJUnitDoc(r.results)

	_, err := fmt.Fprintf(r.w, "%s\n", xml.Header)

	if err != nil {
		return fmt.Errorf("junit write header: %w", err)
	}

	enc := xml.NewEncoder(r.w)
	enc.Indent("", "  ")

	if err = enc.Encode(doc); err != nil {
		return fmt.Errorf("junit encode: %w", err)
	}

	return nil
}

// ---- XML types ----

type junitTestSuites struct {
	XMLName  xml.Name         `xml:"testsuites"`
	Tests    int              `xml:"tests,attr"`
	Failures int              `xml:"failures,attr"`
	Errors   int              `xml:"errors,attr"`
	Suites   []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Errors   int             `xml:"errors,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	Error     *junitError   `xml:"error,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

type junitError struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// ---- builder ----

func buildJUnitDoc(results []*runner.FileResult) junitTestSuites {
	doc := junitTestSuites{}

	for _, fr := range results {
		suite := junitTestSuite{
			Name:     fr.Name,
			Tests:    fr.Passed + fr.Failed,
			Failures: fr.Failed,
		}

		for _, tr := range fr.Tests {
			tc := junitTestCase{
				Name:      tr.Name,
				ClassName: fr.Name,
			}

			if tr.Err != nil {
				// Unexpected error (hook failure, connection error, etc.).
				tc.Error = &junitError{
					Message: tr.Err.Error(),
					Type:    "Error",
					Content: tr.Err.Error(),
				}

				suite.Errors++
			} else if !tr.Passed && len(tr.Failures) > 0 {
				// Assertion failures.
				lines := make([]string, 0, len(tr.Failures))

				for _, f := range tr.Failures {
					lines = append(lines, fmt.Sprintf("%s: %s", f.Step, f.Message))
				}

				content := strings.Join(lines, "\n")

				tc.Failure = &junitFailure{
					Message: lines[0], // first failure as the short message
					Type:    "AssertionFailure",
					Content: content,
				}
			}

			suite.Cases = append(suite.Cases, tc)
		}

		// SetupErr surfaces as a synthetic error test case so it appears in CI output.
		if fr.SetupErr != nil {
			suite.Cases = append(suite.Cases, junitTestCase{
				Name:      "(setup)",
				ClassName: fr.Name,
				Error: &junitError{
					Message: fr.SetupErr.Error(),
					Type:    "SetupError",
					Content: fr.SetupErr.Error(),
				},
			})

			suite.Errors++
			suite.Tests++
		}

		doc.Suites = append(doc.Suites, suite)
		doc.Tests += suite.Tests
		doc.Failures += suite.Failures
		doc.Errors += suite.Errors
	}

	return doc
}
