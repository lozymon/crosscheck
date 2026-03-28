package assert

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lozymon/crosscheck/config"
	"github.com/lozymon/crosscheck/httpclient"
	"github.com/lozymon/crosscheck/interpolate"
)

// Failure describes a single assertion that did not pass.
type Failure struct {
	Field    string // e.g. "status", "header:Authorization", "body.user.id"
	Expected string
	Actual   string
}

func (f Failure) Error() string {
	return fmt.Sprintf("%s: expected %q, got %q", f.Field, f.Expected, f.Actual)
}

// Response checks every assertion in expected against resp.
// It returns a slice of Failure (empty means all passed) and an updated copy of
// vars with any captured values merged in.
func Response(expected *config.ResponseAssert, resp *httpclient.Response, vars map[string]string) ([]Failure, map[string]string) {
	if expected == nil {
		return nil, vars
	}

	var failures []Failure

	outVars := mergeVars(vars, nil)

	// Status assertion.
	if expected.Status != 0 && resp.Status != expected.Status {
		failures = append(failures, Failure{
			Field:    "status",
			Expected: fmt.Sprintf("%d", expected.Status),
			Actual:   fmt.Sprintf("%d", resp.Status),
		})
	}

	// Header assertions.
	for key, expectedVal := range expected.Headers {
		want := interpolate.Apply(expectedVal, vars)
		got := resp.Headers[key]

		if got != want {
			failures = append(failures, Failure{
				Field:    "header:" + key,
				Expected: want,
				Actual:   got,
			})
		}
	}

	// Body assertions.
	if expected.Body != nil {
		bodyFailures, captured := assertBody(expected.Body, resp, vars)
		failures = append(failures, bodyFailures...)
		outVars = mergeVars(outVars, captured)
	}

	return failures, outVars
}

// assertBody walks the expected body tree.
// Leaf values are one of:
//   - "{{ capture: varName }}" — extract via the map key as a JSONPath, store in vars
//   - "/regex/" — assert the JSON value matches the pattern
//   - anything else — exact string match against the JSON field
func assertBody(expected any, resp *httpclient.Response, vars map[string]string) ([]Failure, map[string]string) {
	captured := make(map[string]string)

	var failures []Failure

	walk("body", expected, resp, vars, captured, &failures)

	return failures, captured
}

// walk recurses into maps; leaves are string assertions or captures.
func walk(path string, node any, resp *httpclient.Response, vars map[string]string, captured map[string]string, failures *[]Failure) {
	switch v := node.(type) {
	case map[string]any:
		for k, child := range v {
			walk(path+"."+k, child, resp, vars, captured, failures)
		}

	case string:
		jsonPath := pathToJSONPath(path)
		result := resp.Get(jsonPath)
		actual := result.String()

		if captureVar, ok := parseCaptureDirective(v); ok {
			// {{ capture: varName }} — store the actual value, no failure.
			captured[captureVar] = actual

			return
		}

		if pattern, ok := parseRegexDirective(v); ok {
			re, err := regexp.Compile(pattern)
			if err != nil {
				*failures = append(*failures, Failure{
					Field:    path,
					Expected: "(valid regex) " + v,
					Actual:   "(compile error) " + err.Error(),
				})

				return
			}

			if !re.MatchString(actual) {
				*failures = append(*failures, Failure{
					Field:    path,
					Expected: v,
					Actual:   actual,
				})
			}

			return
		}

		// Exact match after interpolation.
		want := interpolate.Apply(v, vars)

		if actual != want {
			*failures = append(*failures, Failure{
				Field:    path,
				Expected: want,
				Actual:   actual,
			})
		}

	default:
		// Numeric / bool / nil — compare via formatted string.
		jsonPath := pathToJSONPath(path)
		actual := resp.Get(jsonPath).String()
		want := fmt.Sprintf("%v", v)

		if actual != want {
			*failures = append(*failures, Failure{
				Field:    path,
				Expected: want,
				Actual:   actual,
			})
		}
	}
}

// pathToJSONPath converts a dot-path like "body.user.id" into "user.id"
// (strips the leading "body." prefix so it can be passed to resp.Get).
func pathToJSONPath(path string) string {
	trimmed := strings.TrimPrefix(path, "body.")
	trimmed = strings.TrimPrefix(trimmed, "body")

	return trimmed
}

// parseCaptureDirective returns the variable name if s matches "{{ capture: varName }}".
func parseCaptureDirective(s string) (string, bool) {
	s = strings.TrimSpace(s)

	if !strings.HasPrefix(s, "{{") || !strings.HasSuffix(s, "}}") {
		return "", false
	}

	inner := strings.TrimSpace(s[2 : len(s)-2])

	if !strings.HasPrefix(inner, "capture:") {
		return "", false
	}

	varName := strings.TrimSpace(strings.TrimPrefix(inner, "capture:"))

	if varName == "" {
		return "", false
	}

	return varName, true
}

// parseRegexDirective returns the pattern if s is enclosed in forward slashes.
func parseRegexDirective(s string) (string, bool) {
	if len(s) >= 2 && s[0] == '/' && s[len(s)-1] == '/' {
		return s[1 : len(s)-1], true
	}

	return "", false
}

// mergeVars returns a new map containing all entries from base and overlay.
func mergeVars(base, overlay map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(overlay))

	for k, v := range base {
		out[k] = v
	}

	for k, v := range overlay {
		out[k] = v
	}

	return out
}
