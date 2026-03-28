package httpclient_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lozymon/crosscheck/config"
	"github.com/lozymon/crosscheck/httpclient"
)

func TestDo_GET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	client := httpclient.New(false)
	resp, err := client.Do(context.Background(), &config.Request{
		Method: "GET",
		URL:    srv.URL + "/ping",
	}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.Status)
	}
	if got := resp.Get("status").String(); got != "ok" {
		t.Errorf("expected status=ok, got %q", got)
	}
}

func TestDo_POST_withBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(r.Body)
		var m map[string]any
		if err := json.Unmarshal(body, &m); err != nil {
			t.Errorf("could not parse body: %v", err)
		}
		if m["productId"] != "abc-123" {
			t.Errorf("unexpected body: %v", m)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"ord_1","status":"pending"}`))
	}))
	defer srv.Close()

	client := httpclient.New(false)
	resp, err := client.Do(context.Background(), &config.Request{
		Method: "POST",
		URL:    srv.URL + "/orders",
		Body:   map[string]any{"productId": "abc-123"},
	}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.Status)
	}
}

func TestDo_interpolatesURLAndHeaders(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	vars := map[string]string{
		"BASE_URL": srv.URL,
		"TOKEN":    "secret-token",
	}

	client := httpclient.New(false)
	_, err := client.Do(context.Background(), &config.Request{
		Method:  "GET",
		URL:     "{{ BASE_URL }}/orders",
		Headers: map[string]string{"Authorization": "Bearer {{ TOKEN }}"},
	}, vars)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer secret-token" {
		t.Errorf("expected Authorization header 'Bearer secret-token', got %q", gotAuth)
	}
}

func TestResponse_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"user":{"id":"u_1","email":"test@example.com"}}`))
	}))
	defer srv.Close()

	client := httpclient.New(false)
	resp, err := client.Do(context.Background(), &config.Request{Method: "GET", URL: srv.URL}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		path string
		want string
	}{
		{"user.id", "u_1"},
		{"$.user.id", "u_1"}, // JSONPath-style with leading $.
		{"$.user.email", "test@example.com"},
	}

	for _, tt := range tests {
		if got := resp.Get(tt.path).String(); got != tt.want {
			t.Errorf("Get(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestDo_customHeaderOverridesContentType(t *testing.T) {
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := httpclient.New(false)
	_, err := client.Do(context.Background(), &config.Request{
		Method:  "POST",
		URL:     srv.URL,
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:    map[string]any{"key": "value"},
	}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Errorf("expected overridden Content-Type, got %q", gotContentType)
	}
}
