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
	"time"

	"github.com/lozymon/crosscheck/adapters/dynamodb"
	"github.com/lozymon/crosscheck/adapters/lambda"
	"github.com/lozymon/crosscheck/adapters/mongodb"
	"github.com/lozymon/crosscheck/adapters/mysql"
	"github.com/lozymon/crosscheck/adapters/postgres"
	"github.com/lozymon/crosscheck/adapters/redis"
	s3adapter "github.com/lozymon/crosscheck/adapters/s3"
	"github.com/lozymon/crosscheck/adapters/sns"
	"github.com/lozymon/crosscheck/adapters/sqs"
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

	// MongoDB adapter to use for `adapter: mongodb` database assertions.
	// If nil, any test containing a mongodb assertion will fail with an error.
	MongoDB *mongodb.Adapter

	// MySQL adapter to use for `adapter: mysql` database assertions.
	// If nil, any test containing a mysql assertion will fail with an error.
	MySQL *mysql.Adapter

	// Redis adapter to use for `adapter: redis` service assertions.
	// If nil, any test containing a redis assertion will fail with an error.
	Redis *redis.Adapter

	// SQS adapter to use for `adapter: sqs` service assertions.
	SQS *sqs.Adapter

	// SNS adapter to use for `adapter: sns` service assertions (reads via SQS).
	SNS *sns.Adapter

	// S3 adapter to use for `adapter: s3` service assertions.
	S3 *s3adapter.Adapter

	// DynamoDB adapter to use for `adapter: dynamodb` service assertions.
	DynamoDB *dynamodb.Adapter

	// Lambda adapter to use for `adapter: lambda` service assertions.
	Lambda *lambda.Adapter
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

	// Service assertions (Redis, SQS, etc.).
	for i, svcAssert := range test.Services {
		step := fmt.Sprintf("services[%d]", i)
		svcFailures := runServiceAssert(ctx, svcAssert, mergedVars, opts, step)
		tr.Failures = append(tr.Failures, svcFailures...)
	}

	tr.Passed = tr.Err == nil && len(tr.Failures) == 0

	return tr
}

// runDBAssert executes a single database assertion block.
func runDBAssert(ctx context.Context, dbAssert config.DBAssert, vars map[string]string, opts Options, step string) []Failure {
	switch dbAssert.Adapter {
	case "postgres":
		return runPostgresAssert(ctx, dbAssert, vars, opts, step)
	case "mysql":
		return runMySQLAssert(ctx, dbAssert, vars, opts, step)
	case "mongodb":
		return runMongoDBAssert(ctx, dbAssert, vars, opts, step)
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

// runMySQLAssert runs a MySQL database assertion, with optional wait_for polling.
func runMySQLAssert(ctx context.Context, dbAssert config.DBAssert, vars map[string]string, opts Options, step string) []Failure {
	if opts.MySQL == nil {
		return []Failure{{
			Step:    step,
			Message: "mysql adapter not configured (set MYSQL_URL to enable)",
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
		myFailures, err := opts.MySQL.WaitFor(ctx, &assertCopy)

		if err != nil {
			return []Failure{{Step: step, Message: err.Error()}}
		}

		var out []Failure

		for _, f := range myFailures {
			out = append(out, Failure{Step: step, Message: f.Error()})
		}

		return out
	}

	rows, err := opts.MySQL.Query(ctx, assertCopy.Query, assertCopy.Params)

	if err != nil {
		return []Failure{{Step: step, Message: err.Error()}}
	}

	myFailures := mysql.Assert(rows, dbAssert.Expect)

	var out []Failure

	for _, f := range myFailures {
		out = append(out, Failure{Step: step, Message: f.Error()})
	}

	return out
}

// runMongoDBAssert runs a MongoDB database assertion, with optional wait_for polling.
// dbAssert.Query is the collection name; dbAssert.Params is the filter document.
func runMongoDBAssert(ctx context.Context, dbAssert config.DBAssert, vars map[string]string, opts Options, step string) []Failure {
	if opts.MongoDB == nil {
		return []Failure{{
			Step:    step,
			Message: "mongodb adapter not configured (set MONGODB_URL to enable)",
		}}
	}

	// Interpolate filter params.
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
		mgoFailures, err := opts.MongoDB.WaitFor(ctx, &assertCopy)

		if err != nil {
			return []Failure{{Step: step, Message: err.Error()}}
		}

		var out []Failure

		for _, f := range mgoFailures {
			out = append(out, Failure{Step: step, Message: f.Error()})
		}

		return out
	}

	docs, err := opts.MongoDB.Query(ctx, assertCopy.Query, assertCopy.Params)

	if err != nil {
		return []Failure{{Step: step, Message: err.Error()}}
	}

	mgoFailures := mongodb.Assert(docs, dbAssert.Expect)

	var out []Failure

	for _, f := range mgoFailures {
		out = append(out, Failure{Step: step, Message: f.Error()})
	}

	return out
}

// runServiceAssert dispatches a service assertion to the correct adapter handler.
func runServiceAssert(ctx context.Context, svcAssert config.ServiceAssert, vars map[string]string, opts Options, step string) []Failure {
	// Interpolate all string fields before dispatching.
	sa := interpolateServiceAssert(svcAssert, vars)

	switch sa.Adapter {
	case "redis":
		return runRedisAssert(ctx, sa, opts, step)
	case "sqs":
		return runSQSAssert(ctx, sa, opts, step)
	case "sns":
		return runSNSAssert(ctx, sa, opts, step)
	case "s3":
		return runS3Assert(ctx, sa, opts, step)
	case "dynamodb":
		return runDynamoDBAssert(ctx, sa, opts, step)
	case "lambda":
		return runLambdaAssert(ctx, sa, opts, step)
	default:
		return []Failure{{
			Step:    step,
			Message: fmt.Sprintf("adapter %q is not supported in this build", sa.Adapter),
		}}
	}
}

// runRedisAssert runs a Redis service assertion (sa fields already interpolated).
func runRedisAssert(ctx context.Context, sa config.ServiceAssert, opts Options, step string) []Failure {
	if opts.Redis == nil {
		return []Failure{{
			Step:    step,
			Message: "redis adapter not configured (set REDIS_URL to enable)",
		}}
	}

	check := func() ([]Failure, error) {
		actual, err := opts.Redis.Get(ctx, sa.Key)

		if err != nil {
			return []Failure{{Step: step, Message: err.Error()}}, nil
		}

		var out []Failure

		for _, f := range redis.Assert(actual, sa.Expect) {
			out = append(out, Failure{Step: step, Message: f.Error()})
		}

		return out, nil
	}

	if sa.WaitFor != nil {
		failures, err := pollServiceAssert(ctx, sa.WaitFor, check)

		if err != nil {
			return []Failure{{Step: step, Message: err.Error()}}
		}

		return failures
	}

	failures, _ := check()

	return failures
}

// runSQSAssert runs an SQS service assertion (sa fields already interpolated).
func runSQSAssert(ctx context.Context, sa config.ServiceAssert, opts Options, step string) []Failure {
	if opts.SQS == nil {
		return []Failure{{
			Step:    step,
			Message: "sqs adapter not configured (set AWS_REGION and credentials to enable)",
		}}
	}

	check := func() ([]Failure, error) {
		msgs, err := opts.SQS.Peek(ctx, sa.Queue)

		if err != nil {
			return []Failure{{Step: step, Message: err.Error()}}, nil
		}

		var out []Failure

		for _, f := range sqs.Assert(msgs, sa.Expect) {
			out = append(out, Failure{Step: step, Message: f.Error()})
		}

		return out, nil
	}

	if sa.WaitFor != nil {
		failures, err := pollServiceAssert(ctx, sa.WaitFor, check)

		if err != nil {
			return []Failure{{Step: step, Message: err.Error()}}
		}

		return failures
	}

	failures, _ := check()

	return failures
}

// runSNSAssert runs an SNS service assertion via SQS (sa fields already interpolated).
func runSNSAssert(ctx context.Context, sa config.ServiceAssert, opts Options, step string) []Failure {
	if opts.SNS == nil {
		return []Failure{{
			Step:    step,
			Message: "sns adapter not configured (set AWS_REGION and credentials to enable)",
		}}
	}

	check := func() ([]Failure, error) {
		msgs, err := opts.SNS.Peek(ctx, sa.Queue)

		if err != nil {
			return []Failure{{Step: step, Message: err.Error()}}, nil
		}

		var out []Failure

		for _, f := range sns.Assert(msgs, sa.Expect) {
			out = append(out, Failure{Step: step, Message: f.Error()})
		}

		return out, nil
	}

	if sa.WaitFor != nil {
		failures, err := pollServiceAssert(ctx, sa.WaitFor, check)

		if err != nil {
			return []Failure{{Step: step, Message: err.Error()}}
		}

		return failures
	}

	failures, _ := check()

	return failures
}

// runS3Assert runs an S3 service assertion (sa fields already interpolated).
func runS3Assert(ctx context.Context, sa config.ServiceAssert, opts Options, step string) []Failure {
	if opts.S3 == nil {
		return []Failure{{
			Step:    step,
			Message: "s3 adapter not configured (set AWS_REGION and credentials to enable)",
		}}
	}

	check := func() ([]Failure, error) {
		actual, err := opts.S3.GetObject(ctx, sa.Bucket, sa.Key)

		if err != nil {
			return []Failure{{Step: step, Message: err.Error()}}, nil
		}

		var out []Failure

		for _, f := range s3adapter.Assert(actual, sa.Expect) {
			out = append(out, Failure{Step: step, Message: f.Error()})
		}

		return out, nil
	}

	if sa.WaitFor != nil {
		failures, err := pollServiceAssert(ctx, sa.WaitFor, check)

		if err != nil {
			return []Failure{{Step: step, Message: err.Error()}}
		}

		return failures
	}

	failures, _ := check()

	return failures
}

// runDynamoDBAssert runs a DynamoDB service assertion (sa fields already interpolated).
func runDynamoDBAssert(ctx context.Context, sa config.ServiceAssert, opts Options, step string) []Failure {
	if opts.DynamoDB == nil {
		return []Failure{{
			Step:    step,
			Message: "dynamodb adapter not configured (set AWS_REGION and credentials to enable)",
		}}
	}

	check := func() ([]Failure, error) {
		actual, err := opts.DynamoDB.GetItem(ctx, sa.Table, sa.KeyName, sa.Key, sa.SortKeyName, sa.SortKey)

		if err != nil {
			return []Failure{{Step: step, Message: err.Error()}}, nil
		}

		var out []Failure

		for _, f := range dynamodb.Assert(actual, sa.Expect) {
			out = append(out, Failure{Step: step, Message: f.Error()})
		}

		return out, nil
	}

	if sa.WaitFor != nil {
		failures, err := pollServiceAssert(ctx, sa.WaitFor, check)

		if err != nil {
			return []Failure{{Step: step, Message: err.Error()}}
		}

		return failures
	}

	failures, _ := check()

	return failures
}

// runLambdaAssert invokes a Lambda function and asserts the response (sa fields already interpolated).
func runLambdaAssert(ctx context.Context, sa config.ServiceAssert, opts Options, step string) []Failure {
	if opts.Lambda == nil {
		return []Failure{{
			Step:    step,
			Message: "lambda adapter not configured (set AWS_REGION and credentials to enable)",
		}}
	}

	actual, err := opts.Lambda.Invoke(ctx, sa.Key, sa.Payload)

	if err != nil {
		return []Failure{{Step: step, Message: err.Error()}}
	}

	var out []Failure

	for _, f := range lambda.Assert(actual, sa.Expect) {
		out = append(out, Failure{Step: step, Message: f.Error()})
	}

	return out
}

// pollServiceAssert polls check until it returns no failures or the timeout elapses.
func pollServiceAssert(ctx context.Context, waitFor *config.WaitFor, check func() ([]Failure, error)) ([]Failure, error) {
	timeout, err := time.ParseDuration(waitFor.Timeout)

	if err != nil {
		return nil, fmt.Errorf("wait_for: invalid timeout %q: %w", waitFor.Timeout, err)
	}

	interval, err := time.ParseDuration(waitFor.Interval)

	if err != nil {
		return nil, fmt.Errorf("wait_for: invalid interval %q: %w", waitFor.Interval, err)
	}

	deadline := time.Now().Add(timeout)

	for {
		failures, checkErr := check()

		if checkErr != nil {
			return nil, checkErr
		}

		if len(failures) == 0 {
			return nil, nil
		}

		if time.Now().After(deadline) {
			return failures, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// interpolateServiceAssert returns a copy of sa with all string fields interpolated.
func interpolateServiceAssert(sa config.ServiceAssert, vars map[string]string) config.ServiceAssert {
	out := sa
	out.Key = interpolate.Apply(sa.Key, vars)
	out.KeyName = interpolate.Apply(sa.KeyName, vars)
	out.SortKey = interpolate.Apply(sa.SortKey, vars)
	out.SortKeyName = interpolate.Apply(sa.SortKeyName, vars)
	out.Queue = interpolate.Apply(sa.Queue, vars)
	out.Bucket = interpolate.Apply(sa.Bucket, vars)
	out.Table = interpolate.Apply(sa.Table, vars)

	if sa.Expect != nil {
		out.Expect = make(map[string]any, len(sa.Expect))

		for k, v := range sa.Expect {
			if s, ok := v.(string); ok {
				out.Expect[k] = interpolate.Apply(s, vars)
			} else {
				out.Expect[k] = v
			}
		}
	}

	if sa.Payload != nil {
		out.Payload = make(map[string]any, len(sa.Payload))

		for k, v := range sa.Payload {
			if s, ok := v.(string); ok {
				out.Payload[k] = interpolate.Apply(s, vars)
			} else {
				out.Payload[k] = v
			}
		}
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
