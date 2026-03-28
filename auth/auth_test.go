package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lozymon/crosscheck/auth"
	"github.com/lozymon/crosscheck/config"
	"github.com/lozymon/crosscheck/httpclient"
)

func TestResolve_nil(t *testing.T) {
	result, err := auth.Resolve(context.Background(), nil, httpclient.New(false), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for nil auth, got %+v", result)
	}
}

func TestResolve_unknownType(t *testing.T) {
	_, err := auth.Resolve(context.Background(), &config.Auth{Type: "oauth2"}, httpclient.New(false), nil)
	if err == nil {
		t.Fatal("expected error for unknown auth type")
	}
}

func TestResolve_static(t *testing.T) {
	vars := map[string]string{"AUTH_TOKEN": "mysecret"}

	result, err := auth.Resolve(context.Background(), &config.Auth{
		Type: "static",
		Inject: config.AuthInject{
			Header: "Authorization",
			Format: "Bearer {{ AUTH_TOKEN }}",
		},
	}, httpclient.New(false), vars)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Header != "Authorization" {
		t.Errorf("expected header Authorization, got %q", result.Header)
	}
	if result.Value != "Bearer mysecret" {
		t.Errorf("expected value 'Bearer mysecret', got %q", result.Value)
	}
	if len(result.Vars) != 0 {
		t.Errorf("expected no captured vars for static auth, got %v", result.Vars)
	}
}

func TestResolve_login(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accessToken":"tok_abc123","userId":"u_1"}`))
	}))
	defer srv.Close()

	vars := map[string]string{"AUTH_SERVICE": srv.URL}

	result, err := auth.Resolve(context.Background(), &config.Auth{
		Type: "login",
		Request: &config.Request{
			Method: "POST",
			URL:    "{{ AUTH_SERVICE }}/auth/login",
			Body:   map[string]any{"email": "test@example.com", "password": "pass"},
		},
		Capture: config.CaptureMap{
			"token": "$.accessToken",
		},
		Inject: config.AuthInject{
			Header: "Authorization",
			Format: "Bearer {{ token }}",
		},
	}, httpclient.New(false), vars)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Vars["token"] != "tok_abc123" {
		t.Errorf("expected token=tok_abc123, got %q", result.Vars["token"])
	}
	if result.Header != "Authorization" {
		t.Errorf("expected header Authorization, got %q", result.Header)
	}
	if result.Value != "Bearer tok_abc123" {
		t.Errorf("expected value 'Bearer tok_abc123', got %q", result.Value)
	}
}

func TestResolve_login_capturePathMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid credentials"}`))
	}))
	defer srv.Close()

	_, err := auth.Resolve(context.Background(), &config.Auth{
		Type: "login",
		Request: &config.Request{
			Method: "POST",
			URL:    srv.URL + "/auth/login",
		},
		Capture: config.CaptureMap{"token": "$.accessToken"},
		Inject:  config.AuthInject{Header: "Authorization", Format: "Bearer {{ token }}"},
	}, httpclient.New(false), nil)

	if err == nil {
		t.Fatal("expected error when capture path not found")
	}
}

func TestResolve_login_missingRequestBlock(t *testing.T) {
	_, err := auth.Resolve(context.Background(), &config.Auth{Type: "login"}, httpclient.New(false), nil)
	if err == nil {
		t.Fatal("expected error when login has no request block")
	}
}
