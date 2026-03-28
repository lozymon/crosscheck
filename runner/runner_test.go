package runner_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lozymon/crosscheck/config"
	"github.com/lozymon/crosscheck/httpclient"
	"github.com/lozymon/crosscheck/runner"
)

// srv builds a test server that always responds with the given status and JSON body.
func srv(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(s.Close)

	return s
}

func TestRunFile_singleTestPass(t *testing.T) {
	s := srv(t, 201, `{"id":"ord_1","status":"pending"}`)

	tf := &config.TestFile{
		Name: "order suite",
		Tests: []config.Test{
			{
				Name: "create order",
				Request: &config.Request{
					Method: "POST",
					URL:    s.URL + "/orders",
				},
				Response: &config.ResponseAssert{Status: 201},
			},
		},
	}

	result := runner.RunFile(context.Background(), tf, nil, httpclient.New(false), runner.Options{})

	if result.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", result.Passed)
	}

	if result.Failed != 0 {
		t.Errorf("expected 0 failed, got %d: %+v", result.Failed, result.Tests)
	}
}

func TestRunFile_statusMismatch(t *testing.T) {
	s := srv(t, 404, `{"error":"not found"}`)

	tf := &config.TestFile{
		Name: "suite",
		Tests: []config.Test{
			{
				Name:     "expects 200",
				Request:  &config.Request{Method: "GET", URL: s.URL},
				Response: &config.ResponseAssert{Status: 200},
			},
		},
	}

	result := runner.RunFile(context.Background(), tf, nil, httpclient.New(false), runner.Options{})

	if result.Failed != 1 {
		t.Fatalf("expected 1 failed, got %d", result.Failed)
	}

	if result.Tests[0].Passed {
		t.Error("test should not have passed")
	}

	if len(result.Tests[0].Failures) == 0 {
		t.Error("expected at least one failure")
	}
}

func TestRunFile_capturedVarFlowsToNextTest(t *testing.T) {
	// First server returns an ID.
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"ord_42"}`))
	}))
	t.Cleanup(s1.Close)

	// Second server records the URL path so we can assert it contains the captured ID.
	var gotPath string

	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"ord_42","status":"pending"}`))
	}))
	t.Cleanup(s2.Close)

	tf := &config.TestFile{
		Name: "chaining suite",
		Tests: []config.Test{
			{
				Name:    "create",
				Request: &config.Request{Method: "POST", URL: s1.URL + "/orders"},
				Response: &config.ResponseAssert{
					Status: 201,
					Body:   map[string]any{"id": "{{ capture: orderId }}"},
				},
			},
			{
				Name:     "fetch by captured id",
				Request:  &config.Request{Method: "GET", URL: s2.URL + "/orders/{{ orderId }}"},
				Response: &config.ResponseAssert{Status: 200},
			},
		},
	}

	result := runner.RunFile(context.Background(), tf, map[string]string{}, httpclient.New(false), runner.Options{})

	if result.Failed != 0 {
		t.Fatalf("expected no failures, got %d: %+v", result.Failed, result.Tests)
	}

	if gotPath != "/orders/ord_42" {
		t.Errorf("expected second request path /orders/ord_42, got %q", gotPath)
	}
}

func TestRunFile_authHeaderInjected(t *testing.T) {
	var gotAuth string

	loginSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accessToken":"tok_abc"}`))
	}))
	t.Cleanup(loginSrv.Close)

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(apiSrv.Close)

	tf := &config.TestFile{
		Name: "auth suite",
		Auth: &config.Auth{
			Type: "login",
			Request: &config.Request{
				Method: "POST",
				URL:    loginSrv.URL + "/auth/login",
			},
			Capture: config.CaptureMap{"token": "$.accessToken"},
			Inject:  config.AuthInject{Header: "Authorization", Format: "Bearer {{ token }}"},
		},
		Tests: []config.Test{
			{
				Name:     "authenticated request",
				Request:  &config.Request{Method: "GET", URL: apiSrv.URL + "/me"},
				Response: &config.ResponseAssert{Status: 200},
			},
		},
	}

	result := runner.RunFile(context.Background(), tf, map[string]string{}, httpclient.New(false), runner.Options{})

	if result.Failed != 0 {
		t.Fatalf("unexpected failures: %+v", result.Tests)
	}

	if gotAuth != "Bearer tok_abc" {
		t.Errorf("expected auth header 'Bearer tok_abc', got %q", gotAuth)
	}
}

func TestRunFile_teardownRunsOnTestFailure(t *testing.T) {
	s := srv(t, 500, `{}`)

	var teardownRan bool

	// Use a side-effect file to verify teardown ran even after failure.
	// Simpler: use a channel captured in closure via a custom server.
	teardownCh := make(chan struct{}, 1)

	teardownSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		teardownCh <- struct{}{}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(teardownSrv.Close)

	tf := &config.TestFile{
		Name: "teardown suite",
		Teardown: []config.Hook{
			{Run: "curl -s " + teardownSrv.URL},
		},
		Tests: []config.Test{
			{
				Name:     "failing test",
				Request:  &config.Request{Method: "GET", URL: s.URL},
				Response: &config.ResponseAssert{Status: 200}, // will fail (got 500)
			},
		},
	}

	result := runner.RunFile(context.Background(), tf, nil, httpclient.New(false), runner.Options{})

	if result.Failed != 1 {
		t.Errorf("expected 1 failed test, got %d", result.Failed)
	}

	select {
	case <-teardownCh:
		teardownRan = true
	default:
	}

	if !teardownRan {
		t.Error("file-level teardown should have run even after test failure")
	}
}

func TestRunFile_fileLevelSetupHookRuns(t *testing.T) {
	s := srv(t, 200, `{}`)

	tf := &config.TestFile{
		Name:  "setup suite",
		Setup: []config.Hook{{Run: "true"}}, // succeeds
		Tests: []config.Test{
			{
				Name:     "after setup",
				Request:  &config.Request{Method: "GET", URL: s.URL},
				Response: &config.ResponseAssert{Status: 200},
			},
		},
	}

	result := runner.RunFile(context.Background(), tf, nil, httpclient.New(false), runner.Options{})

	if result.SetupErr != nil {
		t.Fatalf("unexpected setup error: %v", result.SetupErr)
	}

	if result.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", result.Passed)
	}
}

func TestRunFile_fileLevelSetupFailureSkipsTests(t *testing.T) {
	tf := &config.TestFile{
		Name:  "bad setup suite",
		Setup: []config.Hook{{Run: "exit 1"}},
		Tests: []config.Test{
			{
				Name:    "should not run",
				Request: &config.Request{Method: "GET", URL: "http://localhost:1"},
			},
		},
	}

	result := runner.RunFile(context.Background(), tf, nil, httpclient.New(false), runner.Options{})

	if result.SetupErr == nil {
		t.Fatal("expected setup error")
	}

	if len(result.Tests) != 0 {
		t.Errorf("no tests should have run after setup failure, got %d", len(result.Tests))
	}
}

func TestRunFile_bodyAssertion(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"name": "alice", "role": "admin"})
	}))
	t.Cleanup(s.Close)

	tf := &config.TestFile{
		Name: "body suite",
		Tests: []config.Test{
			{
				Name:    "check body fields",
				Request: &config.Request{Method: "GET", URL: s.URL},
				Response: &config.ResponseAssert{
					Status: 200,
					Body: map[string]any{
						"name": "alice",
						"role": "admin",
					},
				},
			},
		},
	}

	result := runner.RunFile(context.Background(), tf, nil, httpclient.New(false), runner.Options{})

	if result.Failed != 0 {
		t.Errorf("unexpected failures: %+v", result.Tests[0].Failures)
	}
}
