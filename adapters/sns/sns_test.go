package sns

import (
	"encoding/json"
	"testing"
)

func TestUnwrapSNS_jsonPayload(t *testing.T) {
	inner, _ := json.Marshal(map[string]any{"orderId": "ord_1", "status": "pending"})
	envelope, _ := json.Marshal(map[string]any{
		"Type":    "Notification",
		"Message": string(inner),
	})

	result, err := unwrapSNS(string(envelope))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["orderId"] != "ord_1" {
		t.Errorf("expected orderId=ord_1, got %v", result["orderId"])
	}

	if result["status"] != "pending" {
		t.Errorf("expected status=pending, got %v", result["status"])
	}
}

func TestUnwrapSNS_plainStringPayload(t *testing.T) {
	envelope, _ := json.Marshal(map[string]any{
		"Type":    "Notification",
		"Message": "hello world",
	})

	result, err := unwrapSNS(string(envelope))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["message"] != "hello world" {
		t.Errorf("expected message=hello world, got %v", result["message"])
	}
}

func TestUnwrapSNS_notEnvelope(t *testing.T) {
	_, err := unwrapSNS(`{"orderId":"ord_1"}`)

	if err == nil {
		t.Fatal("expected error for non-SNS body")
	}
}

func TestUnwrapSNS_invalidJSON(t *testing.T) {
	_, err := unwrapSNS("not json at all")

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestAssert_matchFound(t *testing.T) {
	msgs := []map[string]any{
		{"orderId": "ord_1", "status": "pending"},
	}
	expect := map[string]any{"status": "pending"}

	failures := Assert(msgs, expect)

	if len(failures) != 0 {
		t.Errorf("unexpected failures: %v", failures)
	}
}

func TestAssert_noMatch(t *testing.T) {
	msgs := []map[string]any{
		{"status": "shipped"},
	}
	expect := map[string]any{"status": "pending"}

	failures := Assert(msgs, expect)

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}
}

func TestAssert_emptyQueue(t *testing.T) {
	failures := Assert(nil, map[string]any{"status": "pending"})

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for empty queue, got %d", len(failures))
	}
}
