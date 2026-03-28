// Package runner orchestrates the execution of a crosscheck test file.
// Pipeline per file:
//
//  1. File-level setup hooks
//  2. Auth resolution (once, token shared across all tests)
//  3. For each test: per-test setup → HTTP request → assertions → per-test teardown
//  4. Captured vars from each test flow into all subsequent tests
//  5. File-level teardown (always runs, even on failure)
package runner

import (
	"context"
	"fmt"

	"github.com/lozymon/crosscheck/adapters/postgres"
	"github.com/lozymon/crosscheck/assert"
	"github.com/lozymon/crosscheck/auth"
	"github.com/lozymon/crosscheck/config"
	"github.com/lozymon/crosscheck/hooks"
	"github.com/lozymon/crosscheck/httpclient"
	"github.com/lozymon/crosscheck/interpolate"
)

// Failure is a single assertion or step that did not pass.
type Failure struct {
	Step    string // e.g. "response", "database[0]", "setup"
	Message string
}

// TestResult holds the outcome of a single test.
type TestResult struct {
	Name         string
	Passed       bool
	Failures     []Failure
	Err          error             // unexpected error (hook failed, request errored, etc.)
	CapturedVars map[string]string // vars captured during this test — chained into the next
}

// FileResult holds the outcome of running a whole test file.
type FileResult struct {
	Name        string
	Tests       []TestResult
	Passed      int
	Failed      int
	SetupErr    error
	TeardownErr error
}

// Options carries optional adapter dependencies injected by the caller.
type Options struct {
	// Postgres adapter to use for `adapter: postgres` database assertions.
	// If nil, any test containing a postgres assertion will fail with an error.
	Postgres *postgres.Adapter
}

// RunFile executes all tests in a parsed test file and returns a FileResult.
// vars is the starting variable namespace (env vars, CLI overrides, YAML defaults merged).
// The map is never mutated; captured vars are accumulated into a local copy.
func RunFile(ctx context.Context, tf *config.TestFile, vars map[string]string, client *httpclient.Client, opts Options) *FileResult {
	result := &FileResult{Name: tf.Name}

	// Work on a local copy so we don't mutate the caller's map.
	// Captured vars accumulate here across tests.
	workVars := copyVars(vars)

	// File-level teardown always runs.
	defer func() {
		if err := hooks.RunAll(ctx, tf.Teardown, workVars); err != nil {
			result.TeardownErr = err
		}
	}()

	// File-level setup.
	if err := hooks.RunAll(ctx, tf.Setup, workVars); err != nil {
		result.SetupErr = err

		return result
	}

	// Resolve auth once — token is shared across all tests.
	authResult, err := auth.Resolve(ctx, tf.Auth, client, workVars)

	if err != nil {
		result.SetupErr = fmt.Errorf("auth: %w", err)

		return result
	}

	if authResult != nil {
		for k, v := range authResult.Vars {
			workVars[k] = v
		}
	}

	// Run each test, chaining captured vars forward.
	for _, test := range tf.Tests {
		tr := runTest(ctx, test, workVars, client, authResult, opts)

		// Merge this test's captured vars into the shared namespace.
		for k, v := range tr.CapturedVars {
			workVars[k] = v
		}

		if tr.Passed {
			result.Passed++
		} else {
			result.Failed++
		}

		result.Tests = append(result.Tests, tr)
	}

	return result
}

// runTest executes a single test step and returns its result.
func runTest(
	ctx context.Context,
	test config.Test,
	vars map[string]string,
	client *httpclient.Client,
	authResult *auth.Result,
	opts Options,
) TestResult {
	tr := TestResult{Name: test.Name}

	// Per-test setup.
	if err := hooks.RunAll(ctx, test.Setup, vars); err != nil {
		tr.Err = fmt.Errorf("setup: %w", err)

		return tr
	}

	// Always run per-test teardown.
	defer func() {
		if err := hooks.RunAll(ctx, test.Teardown, vars); err != nil {
			if tr.Err == nil {
				tr.Err = fmt.Errorf("teardown: %w", err)
			}
		}
	}()

	// Fire the HTTP request.
	if test.Request == nil {
		tr.Err = fmt.Errorf("test %q has no request block", test.Name)

		return tr
	}

	req := withAuthHeader(test.Request, authResult)

	resp, err := client.Do(ctx, req, vars)

	if err != nil {
		tr.Err = fmt.Errorf("request: %w", err)

		return tr
	}

	// HTTP response assertions + capture.
	assertFailures, outVars := assert.Response(test.Response, resp, vars)

	for _, f := range assertFailures {
		tr.Failures = append(tr.Failures, Failure{
			Step:    "response",
			Message: f.Error(),
		})
	}

	tr.CapturedVars = outVars

	// Merge captures immediately so DB assertions in this same test can use them.
	mergedVars := copyVars(vars)

	for k, v := range outVars {
		mergedVars[k] = v
	}

	// Database assertions.
	for i, dbAssert := range test.Database {
		step := fmt.Sprintf("database[%d]", i)
		dbFailures := runDBAssert(ctx, dbAssert, mergedVars, opts, step)
		tr.Failures = append(tr.Failures, dbFailures...)
	}

	tr.Passed = tr.Err == nil && len(tr.Failures) == 0

	return tr
}

// runDBAssert executes a single database assertion block.
func runDBAssert(ctx context.Context, dbAssert config.DBAssert, vars map[string]string, opts Options, step string) []Failure {
	switch dbAssert.Adapter {
	case "postgres":
		return runPostgresAssert(ctx, dbAssert, vars, opts, step)
	default:
		return []Failure{{
			Step:    step,
			Message: fmt.Sprintf("adapter %q is not supported in this build", dbAssert.Adapter),
		}}
	}
}

// runPostgresAssert runs a postgres database assertion, with optional wait_for polling.
func runPostgresAssert(ctx context.Context, dbAssert config.DBAssert, vars map[string]string, opts Options, step string) []Failure {
	if opts.Postgres == nil {
		return []Failure{{
			Step:    step,
			Message: "postgres adapter not configured (pass a connected *postgres.Adapter via runner.Options)",
		}}
	}

	// Interpolate query params.
	interpolatedParams := make(map[string]any, len(dbAssert.Params))

	for k, v := range dbAssert.Params {
		if s, ok := v.(string); ok {
			interpolatedParams[k] = interpolate.Apply(s, vars)
		} else {
			interpolatedParams[k] = v
		}
	}

	assertCopy := dbAssert
	assertCopy.Params = interpolatedParams

	if dbAssert.WaitFor != nil {
		pgFailures, err := opts.Postgres.WaitFor(ctx, &assertCopy)

		if err != nil {
			return []Failure{{Step: step, Message: err.Error()}}
		}

		var out []Failure

		for _, f := range pgFailures {
			out = append(out, Failure{Step: step, Message: f.Error()})
		}

		return out
	}

	rows, err := opts.Postgres.Query(ctx, assertCopy.Query, assertCopy.Params)

	if err != nil {
		return []Failure{{Step: step, Message: err.Error()}}
	}

	pgFailures := postgres.Assert(rows, dbAssert.Expect)

	var out []Failure

	for _, f := range pgFailures {
		out = append(out, Failure{Step: step, Message: f.Error()})
	}

	return out
}

// withAuthHeader returns a shallow copy of req with the auth header injected.
// The original req is never mutated.
func withAuthHeader(req *config.Request, authResult *auth.Result) *config.Request {
	if authResult == nil || authResult.Header == "" {
		return req
	}

	headers := make(map[string]string, len(req.Headers)+1)

	for k, v := range req.Headers {
		headers[k] = v
	}

	// Only inject if the test doesn't already override this header.
	if _, exists := headers[authResult.Header]; !exists {
		headers[authResult.Header] = authResult.Value
	}

	return &config.Request{
		Method:  req.Method,
		URL:     req.URL,
		Headers: headers,
		Body:    req.Body,
	}
}

// copyVars returns a shallow copy of a vars map.
func copyVars(vars map[string]string) map[string]string {
	out := make(map[string]string, len(vars))

	for k, v := range vars {
		out[k] = v
	}

	return out
}
