package mockserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/lozymon/crosscheck/adapters/mockserver"
)

func postJSON(t *testing.T, url string, body any) {
	t.Helper()

	data, err := json.Marshal(body)

	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(data)) //nolint:noctx

	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}

	defer func() { _ = resp.Body.Close() }()
}

func TestStart_listenOnRandomPort(t *testing.T) {
	srv, err := mockserver.Start(0)

	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer srv.Close()

	if srv.URL == "" {
		t.Error("expected non-empty URL")
	}
}

func TestRequests_capturesRequest(t *testing.T) {
	srv, err := mockserver.Start(0)

	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer srv.Close()

	postJSON(t, srv.URL+"/webhook", map[string]any{"event": "order.created"})

	reqs := srv.Requests("POST", "/webhook")

	if len(reqs) != 1 {
		t.Fatalf("expected 1 captured request, got %d", len(reqs))
	}

	if reqs[0].Method != "POST" {
		t.Errorf("expected method POST, got %q", reqs[0].Method)
	}

	if reqs[0].Path != "/webhook" {
		t.Errorf("expected path /webhook, got %q", reqs[0].Path)
	}
}

func TestRequests_methodFilter(t *testing.T) {
	srv, err := mockserver.Start(0)

	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer srv.Close()

	postJSON(t, srv.URL+"/ping", map[string]any{"x": "1"})

	// Filter by GET — should return nothing.
	if got := srv.Requests("GET", "/ping"); len(got) != 0 {
		t.Errorf("expected 0 GET requests, got %d", len(got))
	}

	// Filter by POST — should return the one request.
	if got := srv.Requests("POST", "/ping"); len(got) != 1 {
		t.Errorf("expected 1 POST request, got %d", len(got))
	}
}

func TestRequests_pathFilter(t *testing.T) {
	srv, err := mockserver.Start(0)

	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer srv.Close()

	postJSON(t, srv.URL+"/a", map[string]any{})
	postJSON(t, srv.URL+"/b", map[string]any{})

	if got := srv.Requests("", "/a"); len(got) != 1 {
		t.Errorf("expected 1 request to /a, got %d", len(got))
	}

	if got := srv.Requests("", "/b"); len(got) != 1 {
		t.Errorf("expected 1 request to /b, got %d", len(got))
	}
}

func TestReset_clearsRequests(t *testing.T) {
	srv, err := mockserver.Start(0)

	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer srv.Close()

	postJSON(t, srv.URL+"/webhook", map[string]any{})
	srv.Reset()

	if got := srv.Requests("", ""); len(got) != 0 {
		t.Errorf("expected 0 requests after Reset, got %d", len(got))
	}
}

func TestAssert_passing(t *testing.T) {
	srv, err := mockserver.Start(0)

	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer srv.Close()

	postJSON(t, srv.URL+"/webhook", map[string]any{"event": "order.created", "orderId": "abc"})

	reqs := srv.Requests("POST", "/webhook")
	errs := mockserver.Assert(reqs, map[string]any{"event": "order.created", "orderId": "abc"})

	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestAssert_failing(t *testing.T) {
	srv, err := mockserver.Start(0)

	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer srv.Close()

	postJSON(t, srv.URL+"/webhook", map[string]any{"event": "order.created"})

	reqs := srv.Requests("POST", "/webhook")
	errs := mockserver.Assert(reqs, map[string]any{"event": "order.cancelled"})

	if len(errs) == 0 {
		t.Error("expected assertion failure, got none")
	}
}

func TestAssert_noRequestsCaptured(t *testing.T) {
	errs := mockserver.Assert(nil, map[string]any{"event": "x"})

	if len(errs) == 0 {
		t.Error("expected error when no requests captured")
	}
}

func TestAssert_emptyExpect_noRequests(t *testing.T) {
	errs := mockserver.Assert(nil, map[string]any{})

	if len(errs) == 0 {
		t.Error("expected error for empty expect with no requests")
	}
}

func TestAssert_emptyExpect_withRequests(t *testing.T) {
	srv, err := mockserver.Start(0)

	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer srv.Close()

	postJSON(t, srv.URL+"/ping", map[string]any{})

	reqs := srv.Requests("", "")
	errs := mockserver.Assert(reqs, map[string]any{})

	if len(errs) != 0 {
		t.Errorf("expected no errors for empty expect with captured request, got: %v", errs)
	}
}

func TestRequests_responds200(t *testing.T) {
	srv, err := mockserver.Start(0)

	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer srv.Close()

	resp, doErr := http.Post(srv.URL+"/any", "text/plain", bytes.NewReader([]byte("hello"))) //nolint:noctx

	if doErr != nil {
		t.Fatalf("POST: %v", doErr)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
