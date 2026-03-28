package reporter

import (
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/lozymon/crosscheck/runner"
)

// htmlReporter buffers FileResults and writes a single self-contained HTML
// document on Close(). The document has no external dependencies — all CSS
// is inlined so it can be saved, emailed, or uploaded as a CI artifact.
type htmlReporter struct {
	w       io.Writer
	results []*runner.FileResult
}

func (r *htmlReporter) Write(result *runner.FileResult) error {
	r.results = append(r.results, result)

	return nil
}

// Close renders and writes the complete HTML report.
func (r *htmlReporter) Close() error {
	data := buildHTMLData(r.results)

	if err := htmlTmpl.Execute(r.w, data); err != nil {
		return fmt.Errorf("html reporter: %w", err)
	}

	return nil
}

// ---- template data ----

type htmlReport struct {
	GeneratedAt string
	TotalTests  int
	TotalPassed int
	TotalFailed int
	Suites      []htmlSuite
}

type htmlSuite struct {
	Name     string
	Passed   int
	Failed   int
	SetupErr string
	Tests    []htmlTest
}

type htmlTest struct {
	Name     string
	Passed   bool
	Attempts int
	Err      string
	Failures []htmlFailure
}

type htmlFailure struct {
	Step    string
	Message string
}

func buildHTMLData(results []*runner.FileResult) htmlReport {
	data := htmlReport{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	for _, fr := range results {
		suite := htmlSuite{
			Name:   fr.Name,
			Passed: fr.Passed,
			Failed: fr.Failed,
		}

		if fr.SetupErr != nil {
			suite.SetupErr = fr.SetupErr.Error()
		}

		for _, tr := range fr.Tests {
			t := htmlTest{
				Name:     tr.Name,
				Passed:   tr.Passed,
				Attempts: tr.Attempts,
			}

			if tr.Err != nil {
				t.Err = tr.Err.Error()
			}

			for _, f := range tr.Failures {
				t.Failures = append(t.Failures, htmlFailure{
					Step:    f.Step,
					Message: f.Message,
				})
			}

			suite.Tests = append(suite.Tests, t)
		}

		data.Suites = append(data.Suites, suite)
		data.TotalTests += fr.Passed + fr.Failed
		data.TotalPassed += fr.Passed
		data.TotalFailed += fr.Failed
	}

	return data
}

// ---- template ----

var htmlTmpl = template.Must(template.New("report").Funcs(template.FuncMap{
	"plural": func(n int, word string) string {
		if n == 1 {
			return fmt.Sprintf("%d %s", n, word)
		}

		return fmt.Sprintf("%d %ss", n, word)
	},
	"joinFailures": func(failures []htmlFailure) string {
		lines := make([]string, len(failures))

		for i, f := range failures {
			lines[i] = f.Step + ": " + f.Message
		}

		return strings.Join(lines, "\n")
	},
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>crosscheck results</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      font-size: 14px;
      line-height: 1.5;
      background: #f4f4f5;
      color: #18181b;
      padding: 2rem 1rem;
    }

    a { color: inherit; }

    /* ---- layout ---- */
    .page { max-width: 860px; margin: 0 auto; }

    header {
      display: flex;
      align-items: baseline;
      gap: 1rem;
      margin-bottom: 1.5rem;
    }

    header h1 {
      font-size: 1.25rem;
      font-weight: 700;
      letter-spacing: -0.02em;
    }

    .meta { color: #71717a; font-size: 0.8rem; }

    /* ---- summary bar ---- */
    .summary {
      display: flex;
      gap: 0.5rem;
      align-items: center;
      background: #fff;
      border: 1px solid #e4e4e7;
      border-radius: 8px;
      padding: 0.875rem 1.25rem;
      margin-bottom: 1.25rem;
    }

    .summary .count { font-weight: 600; font-size: 1.1rem; }
    .summary .sep   { color: #d4d4d8; }
    .pass-text { color: #16a34a; }
    .fail-text { color: #dc2626; }

    /* ---- suite card ---- */
    .suite {
      background: #fff;
      border: 1px solid #e4e4e7;
      border-radius: 8px;
      margin-bottom: 1rem;
      overflow: hidden;
    }

    .suite-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 0.75rem 1.25rem;
      border-bottom: 1px solid #e4e4e7;
      background: #fafafa;
    }

    .suite-header h2 { font-size: 0.95rem; font-weight: 600; }

    .suite-counts { font-size: 0.8rem; color: #71717a; }

    /* ---- setup error ---- */
    .setup-error {
      padding: 0.75rem 1.25rem;
      background: #fef2f2;
      border-bottom: 1px solid #fecaca;
      color: #b91c1c;
      font-size: 0.85rem;
    }

    .setup-error strong { font-weight: 600; }

    /* ---- test row ---- */
    details.test { border-bottom: 1px solid #f4f4f5; }
    details.test:last-child { border-bottom: none; }

    details.test summary {
      display: flex;
      align-items: center;
      gap: 0.625rem;
      padding: 0.625rem 1.25rem;
      cursor: pointer;
      list-style: none;
      user-select: none;
    }

    details.test summary::-webkit-details-marker { display: none; }

    details.test summary:hover { background: #fafafa; }

    .test-icon {
      width: 1.1rem;
      height: 1.1rem;
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 0.65rem;
      font-weight: 700;
      flex-shrink: 0;
    }

    .icon-pass { background: #dcfce7; color: #15803d; }
    .icon-fail { background: #fee2e2; color: #b91c1c; }

    .test-name { flex: 1; font-size: 0.875rem; }

    .test-attempts {
      font-size: 0.75rem;
      color: #a1a1aa;
    }

    /* ---- failure details ---- */
    .failure-body {
      padding: 0.5rem 1.25rem 0.75rem 3.25rem;
      background: #fafafa;
    }

    .failure-item {
      margin-bottom: 0.375rem;
    }

    .failure-step {
      display: inline-block;
      font-size: 0.7rem;
      font-weight: 600;
      background: #fee2e2;
      color: #b91c1c;
      border-radius: 3px;
      padding: 0.1rem 0.4rem;
      margin-right: 0.375rem;
      vertical-align: middle;
    }

    .failure-message {
      font-family: "SF Mono", "Fira Code", "Fira Mono", Menlo, monospace;
      font-size: 0.8rem;
      color: #3f3f46;
      word-break: break-all;
    }

    .error-message {
      font-family: "SF Mono", "Fira Code", "Fira Mono", Menlo, monospace;
      font-size: 0.8rem;
      color: #b91c1c;
      word-break: break-all;
    }
  </style>
</head>
<body>
  <div class="page">
    <header>
      <h1>crosscheck</h1>
      <span class="meta">{{ .GeneratedAt }}</span>
    </header>

    <div class="summary">
      <span class="count">{{ plural .TotalTests "test" }}</span>
      <span class="sep">·</span>
      <span class="count pass-text">{{ plural .TotalPassed "passed" }}</span>
      {{- if gt .TotalFailed 0 }}
      <span class="sep">·</span>
      <span class="count fail-text">{{ plural .TotalFailed "failed" }}</span>
      {{- end }}
    </div>

    {{- range .Suites }}
    <div class="suite">
      <div class="suite-header">
        <h2>{{ .Name }}</h2>
        <span class="suite-counts">
          {{- plural .Passed "passed" }}
          {{- if gt .Failed 0 }} · <span class="fail-text">{{ plural .Failed "failed" }}</span>{{ end -}}
        </span>
      </div>

      {{- if .SetupErr }}
      <div class="setup-error">
        <strong>Setup failed:</strong> {{ .SetupErr }}
      </div>
      {{- end }}

      {{- range .Tests }}
      <details class="test"{{ if not .Passed }} open{{ end }}>
        <summary>
          {{- if .Passed }}
          <span class="test-icon icon-pass">✓</span>
          {{- else }}
          <span class="test-icon icon-fail">✗</span>
          {{- end }}
          <span class="test-name">{{ .Name }}</span>
          {{- if gt .Attempts 1 }}
          <span class="test-attempts">{{ .Attempts }} attempts</span>
          {{- end }}
        </summary>
        {{- if or .Err .Failures }}
        <div class="failure-body">
          {{- if .Err }}
          <div class="failure-item">
            <span class="failure-step">error</span>
            <span class="error-message">{{ .Err }}</span>
          </div>
          {{- end }}
          {{- range .Failures }}
          <div class="failure-item">
            <span class="failure-step">{{ .Step }}</span>
            <span class="failure-message">{{ .Message }}</span>
          </div>
          {{- end }}
        </div>
        {{- end }}
      </details>
      {{- end }}
    </div>
    {{- end }}
  </div>
</body>
</html>
`))
