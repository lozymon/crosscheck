package assert_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lozymon/crosscheck/assert"
	"github.com/lozymon/crosscheck/config"
	"github.com/lozymon/crosscheck/httpclient"
)

// makeResp fires a request against a handler and returns the Response.
func makeResp(t *testing.T, statusCode int, body string, headers map[string]string) *httpclient.Response {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range headers {
			w.Header().Set(k, v)
		}

		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	client := httpclient.New(false)

	resp, err := client.Do(context.Background(), &config.Request{Method: "GET", URL: srv.URL}, nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	return resp
}

func TestResponse_nilExpected(t *testing.T) {
	resp := makeResp(t, 200, `{}`, nil)
	failures, _ := assert.Response(nil, resp, nil)

	if len(failures) != 0 {
		t.Errorf("expected no failures for nil assert, got %v", failures)
	}
}

func TestResponse_statusPass(t *testing.T) {
	resp := makeResp(t, 201, `{}`, nil)
	failures, _ := assert.Response(&config.ResponseAssert{Status: 201}, resp, nil)

	if len(failures) != 0 {
		t.Errorf("unexpected failures: %v", failures)
	}
}

func TestResponse_statusFail(t *testing.T) {
	resp := makeResp(t, 404, `{}`, nil)
	failures, _ := assert.Response(&config.ResponseAssert{Status: 201}, resp, nil)

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d: %v", len(failures), failures)
	}

	if failures[0].Field != "status" {
		t.Errorf("expected field=status, got %q", failures[0].Field)
	}
}

func TestResponse_headerPass(t *testing.T) {
	resp := makeResp(t, 200, `{}`, map[string]string{"X-Request-Id": "abc-123"})
	failures, _ := assert.Response(&config.ResponseAssert{
		Headers: map[string]string{"X-Request-Id": "abc-123"},
	}, resp, nil)

	if len(failures) != 0 {
		t.Errorf("unexpected failures: %v", failures)
	}
}

func TestResponse_headerFail(t *testing.T) {
	resp := makeResp(t, 200, `{}`, map[string]string{"X-Request-Id": "wrong"})
	failures, _ := assert.Response(&config.ResponseAssert{
		Headers: map[string]string{"X-Request-Id": "abc-123"},
	}, resp, nil)

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}

	if failures[0].Field != "header:X-Request-Id" {
		t.Errorf("unexpected field: %q", failures[0].Field)
	}
}

func TestResponse_headerInterpolated(t *testing.T) {
	resp := makeResp(t, 200, `{}`, map[string]string{"Authorization": "Bearer tok_abc"})
	vars := map[string]string{"TOKEN": "tok_abc"}
	failures, _ := assert.Response(&config.ResponseAssert{
		Headers: map[string]string{"Authorization": "Bearer {{ TOKEN }}"},
	}, resp, vars)

	if len(failures) != 0 {
		t.Errorf("unexpected failures: %v", failures)
	}
}

func TestResponse_bodyExactMatch(t *testing.T) {
	resp := makeResp(t, 200, `{"status":"ok"}`, nil)
	failures, _ := assert.Response(&config.ResponseAssert{
		Body: map[string]any{"status": "ok"},
	}, resp, nil)

	if len(failures) != 0 {
		t.Errorf("unexpected failures: %v", failures)
	}
}

func TestResponse_bodyExactFail(t *testing.T) {
	resp := makeResp(t, 200, `{"status":"error"}`, nil)
	failures, _ := assert.Response(&config.ResponseAssert{
		Body: map[string]any{"status": "ok"},
	}, resp, nil)

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d: %v", len(failures), failures)
	}

	if failures[0].Field != "body.status" {
		t.Errorf("unexpected field: %q", failures[0].Field)
	}
}

func TestResponse_bodyNestedPath(t *testing.T) {
	resp := makeResp(t, 200, `{"user":{"id":"u_1","email":"a@b.com"}}`, nil)
	failures, _ := assert.Response(&config.ResponseAssert{
		Body: map[string]any{
			"user": map[string]any{
				"id":    "u_1",
				"email": "a@b.com",
			},
		},
	}, resp, nil)

	if len(failures) != 0 {
		t.Errorf("unexpected failures: %v", failures)
	}
}

func TestResponse_bodyCapture(t *testing.T) {
	resp := makeResp(t, 201, `{"id":"ord_999","status":"pending"}`, nil)
	_, outVars := assert.Response(&config.ResponseAssert{
		Body: map[string]any{"id": "{{ capture: orderId }}"},
	}, resp, nil)

	if outVars["orderId"] != "ord_999" {
		t.Errorf("expected orderId=ord_999, got %q", outVars["orderId"])
	}
}

func TestResponse_bodyRegexPass(t *testing.T) {
	resp := makeResp(t, 200, `{"id":"ord_abc123"}`, nil)
	failures, _ := assert.Response(&config.ResponseAssert{
		Body: map[string]any{"id": "/^ord_[a-z0-9]+$/"},
	}, resp, nil)

	if len(failures) != 0 {
		t.Errorf("unexpected failures: %v", failures)
	}
}

func TestResponse_bodyRegexFail(t *testing.T) {
	resp := makeResp(t, 200, `{"id":"USR-1"}`, nil)
	failures, _ := assert.Response(&config.ResponseAssert{
		Body: map[string]any{"id": "/^ord_[a-z0-9]+$/"},
	}, resp, nil)

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}
}

func TestResponse_bodyInterpolatedExpected(t *testing.T) {
	resp := makeResp(t, 200, `{"orderId":"ord_42"}`, nil)
	vars := map[string]string{"EXPECTED_ID": "ord_42"}
	failures, _ := assert.Response(&config.ResponseAssert{
		Body: map[string]any{"orderId": "{{ EXPECTED_ID }}"},
	}, resp, vars)

	if len(failures) != 0 {
		t.Errorf("unexpected failures: %v", failures)
	}
}

func TestResponse_captureDoesNotProduceFailure(t *testing.T) {
	resp := makeResp(t, 201, `{"id":"ord_1"}`, nil)
	failures, outVars := assert.Response(&config.ResponseAssert{
		Body: map[string]any{"id": "{{ capture: orderId }}"},
	}, resp, nil)

	if len(failures) != 0 {
		t.Errorf("capture should not produce failures, got %v", failures)
	}

	if outVars["orderId"] != "ord_1" {
		t.Errorf("expected captured orderId=ord_1, got %q", outVars["orderId"])
	}
}

func TestResponse_multipleFailures(t *testing.T) {
	resp := makeResp(t, 404, `{"code":"not_found"}`, nil)
	failures, _ := assert.Response(&config.ResponseAssert{
		Status: 200,
		Body:   map[string]any{"code": "ok"},
	}, resp, nil)

	if len(failures) != 2 {
		t.Errorf("expected 2 failures, got %d: %v", len(failures), failures)
	}
}
