// Package mockserver provides a local HTTP server that captures incoming
// requests so tests can assert that their application-under-test made the
// expected outbound calls.
//
// Typical usage in a *.cx.yaml file:
//
//	mock:
//	  port: 0          # 0 = auto-assign a free port
//
// crosscheck automatically injects MOCK_URL into the variable namespace so
// test requests can be pointed at the server without hard-coding a port.
//
// To assert a captured call:
//
//	services:
//	  - adapter: mock
//	    path: /webhook
//	    method: POST
//	    wait_for: { timeout: 5s, interval: 200ms }
//	    expect:
//	      event: order.created
//	      orderId: "{{ orderId }}"
package mockserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

// CapturedRequest is a single HTTP request received by the mock server.
type CapturedRequest struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    map[string]any // JSON-parsed request body; nil if body was empty or not JSON
	RawBody string         // raw body text regardless of Content-Type
}

// Server is a running mock HTTP server.
type Server struct {
	URL      string // base URL, e.g. "http://127.0.0.1:51234"
	listener net.Listener
	server   *http.Server
	mu       sync.Mutex
	requests []CapturedRequest
}

// Start launches a new mock server on the given port.
// Pass 0 to let the OS pick a free port.
func Start(port int) (*Server, error) {
	addr := fmt.Sprintf("0.0.0.0:%d", port)

	ln, err := net.Listen("tcp", addr)

	if err != nil {
		return nil, fmt.Errorf("mock server listen %s: %w", addr, err)
	}

	// Addr returns "0.0.0.0:<port>" — replace with 127.0.0.1 for the MOCK_URL
	// injected into test vars (used by the local process / curl).
	// The server itself accepts connections from all interfaces including Docker.
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	s := &Server{
		URL:      "http://127.0.0.1:" + portStr,
		listener: ln,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRequest)

	s.server = &http.Server{Handler: mux}

	go func() { _ = s.server.Serve(ln) }()

	return s, nil
}

// Close shuts down the server. Captured requests are discarded.
func (s *Server) Close() {
	_ = s.server.Close()
}

// Reset clears all captured requests.
// Call between tests when you need a clean slate.
func (s *Server) Reset() {
	s.mu.Lock()
	s.requests = nil
	s.mu.Unlock()
}

// Requests returns captured requests matching the given method and path.
// An empty method string matches any method.
// An empty path string matches any path.
func (s *Server) Requests(method, path string) []CapturedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []CapturedRequest

	for _, r := range s.requests {
		if method != "" && !strings.EqualFold(r.Method, method) {
			continue
		}

		if path != "" && r.Path != path {
			continue
		}

		out = append(out, r)
	}

	return out
}

// handleRequest records every request that arrives at the server.
// Always responds 200 OK with an empty body so the caller is not blocked.
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)

	captured := CapturedRequest{
		Method:  r.Method,
		Path:    r.URL.Path,
		Headers: flattenHeaders(r.Header),
		RawBody: string(bodyBytes),
	}

	if len(bodyBytes) > 0 {
		var parsed map[string]any

		if json.Unmarshal(bodyBytes, &parsed) == nil {
			captured.Body = parsed
		}
	}

	s.mu.Lock()
	s.requests = append(s.requests, captured)
	s.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

// flattenHeaders converts http.Header (multi-value) into a simple map using
// the first value for each key.
func flattenHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))

	for k, vs := range h {
		if len(vs) > 0 {
			out[http.CanonicalHeaderKey(k)] = vs[0]
		}
	}

	return out
}

// Assert checks that at least one of the captured requests satisfies all
// key/value pairs in expect.  Values are compared as strings after
// fmt.Sprintf("%v", ...) conversion, mirroring the behaviour of other
// crosscheck adapters.
//
// Returns a slice of errors — one per unmet expectation on the best-matching
// request (the one with the fewest mismatches).
func Assert(requests []CapturedRequest, expect map[string]any) []error {
	if len(expect) == 0 {
		if len(requests) == 0 {
			return []error{fmt.Errorf("no requests captured")}
		}

		return nil
	}

	if len(requests) == 0 {
		var errs []error

		for k, v := range expect {
			errs = append(errs, fmt.Errorf("field %q: expected %q, got no requests", k, fmt.Sprintf("%v", v)))
		}

		return errs
	}

	// Find the request with the fewest mismatches for better error messages.
	var bestErrs []error

	for _, req := range requests {
		errs := assertOne(req, expect)

		if len(errs) == 0 {
			return nil
		}

		if bestErrs == nil || len(errs) < len(bestErrs) {
			bestErrs = errs
		}
	}

	return bestErrs
}

// assertOne checks a single request against expect and returns mismatch errors.
func assertOne(req CapturedRequest, expect map[string]any) []error {
	var errs []error

	for k, v := range expect {
		expected := fmt.Sprintf("%v", v)
		actual, ok := fieldValue(req, k)

		if !ok {
			errs = append(errs, fmt.Errorf("field %q: not found in request body", k))

			continue
		}

		if actual != expected {
			errs = append(errs, fmt.Errorf("field %q: expected %q, got %q", k, expected, actual))
		}
	}

	return errs
}

// fieldValue extracts a value from the request body by key.
// Checks JSON body first, falls back to raw body under key "body".
func fieldValue(req CapturedRequest, key string) (string, bool) {
	if req.Body != nil {
		if v, ok := req.Body[key]; ok {
			return fmt.Sprintf("%v", v), true
		}
	}

	if key == "body" {
		return req.RawBody, true
	}

	return "", false
}
