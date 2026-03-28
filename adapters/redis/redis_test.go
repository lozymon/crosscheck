package redis

import (
	"context"
	"os"
	"testing"
)

// Assert is a pure function — no Redis needed.

func TestAssert_pass(t *testing.T) {
	actual := map[string]any{"status": "pending", "quantity": "2"}
	expect := map[string]any{"status": "pending", "quantity": "2"}

	failures := Assert(actual, expect)

	if len(failures) != 0 {
		t.Errorf("unexpected failures: %v", failures)
	}
}

func TestAssert_fieldMismatch(t *testing.T) {
	actual := map[string]any{"status": "shipped"}
	expect := map[string]any{"status": "pending"}

	failures := Assert(actual, expect)

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}

	if failures[0].Field != "status" {
		t.Errorf("expected field=status, got %q", failures[0].Field)
	}

	if failures[0].Expected != "pending" {
		t.Errorf("expected Expected=pending, got %q", failures[0].Expected)
	}

	if failures[0].Actual != "shipped" {
		t.Errorf("expected Actual=shipped, got %q", failures[0].Actual)
	}
}

func TestAssert_missingField(t *testing.T) {
	actual := map[string]any{"status": "pending"}
	expect := map[string]any{"status": "pending", "quantity": "2"}

	failures := Assert(actual, expect)

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for missing field, got %d", len(failures))
	}

	if failures[0].Field != "quantity" {
		t.Errorf("expected field=quantity, got %q", failures[0].Field)
	}

	if failures[0].Actual != "(field not found)" {
		t.Errorf("expected Actual=(field not found), got %q", failures[0].Actual)
	}
}

func TestAssert_emptyExpect(t *testing.T) {
	actual := map[string]any{"status": "pending"}

	failures := Assert(actual, map[string]any{})

	if len(failures) != 0 {
		t.Errorf("empty expect should always pass, got %v", failures)
	}
}

func TestAssert_multipleFailures(t *testing.T) {
	actual := map[string]any{"status": "shipped", "quantity": "5"}
	expect := map[string]any{"status": "pending", "quantity": "2"}

	failures := Assert(actual, expect)

	if len(failures) != 2 {
		t.Errorf("expected 2 failures, got %d", len(failures))
	}
}

// Integration tests — only run when REDIS_URL is set.

func TestNew_ping(t *testing.T) {
	addr := os.Getenv("REDIS_URL")

	if addr == "" {
		t.Skip("REDIS_URL not set — skipping integration test")
	}

	ctx := context.Background()

	a, err := New(ctx, addr)

	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = a.Close() }()
}

func TestGet_stringKey(t *testing.T) {
	addr := os.Getenv("REDIS_URL")

	if addr == "" {
		t.Skip("REDIS_URL not set — skipping integration test")
	}

	ctx := context.Background()

	a, err := New(ctx, addr)

	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = a.Close() }()

	// Write a plain string value.
	if err = a.client.Set(ctx, "cx:test:str", "hello", 0).Err(); err != nil {
		t.Fatalf("SET: %v", err)
	}

	defer func() { _ = a.client.Del(ctx, "cx:test:str").Err() }()

	result, err := a.Get(ctx, "cx:test:str")

	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if result["value"] != "hello" {
		t.Errorf("expected value=hello, got %v", result["value"])
	}
}

func TestGet_jsonStringKey(t *testing.T) {
	addr := os.Getenv("REDIS_URL")

	if addr == "" {
		t.Skip("REDIS_URL not set — skipping integration test")
	}

	ctx := context.Background()

	a, err := New(ctx, addr)

	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = a.Close() }()

	if err = a.client.Set(ctx, "cx:test:json", `{"status":"pending"}`, 0).Err(); err != nil {
		t.Fatalf("SET: %v", err)
	}

	defer func() { _ = a.client.Del(ctx, "cx:test:json").Err() }()

	result, err := a.Get(ctx, "cx:test:json")

	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if result["status"] != "pending" {
		t.Errorf("expected status=pending, got %v", result["status"])
	}
}

func TestGet_hashKey(t *testing.T) {
	addr := os.Getenv("REDIS_URL")

	if addr == "" {
		t.Skip("REDIS_URL not set — skipping integration test")
	}

	ctx := context.Background()

	a, err := New(ctx, addr)

	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = a.Close() }()

	if err = a.client.HSet(ctx, "cx:test:hash", "status", "pending", "qty", "3").Err(); err != nil {
		t.Fatalf("HSET: %v", err)
	}

	defer func() { _ = a.client.Del(ctx, "cx:test:hash").Err() }()

	result, err := a.Get(ctx, "cx:test:hash")

	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if result["status"] != "pending" {
		t.Errorf("expected status=pending, got %v", result["status"])
	}

	if result["qty"] != "3" {
		t.Errorf("expected qty=3, got %v", result["qty"])
	}
}

func TestGet_missingKey(t *testing.T) {
	addr := os.Getenv("REDIS_URL")

	if addr == "" {
		t.Skip("REDIS_URL not set — skipping integration test")
	}

	ctx := context.Background()

	a, err := New(ctx, addr)

	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = a.Close() }()

	_, err = a.Get(ctx, "cx:test:no-such-key")

	if err == nil {
		t.Fatal("expected error for missing key")
	}
}
