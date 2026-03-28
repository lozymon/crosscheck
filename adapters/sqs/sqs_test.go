package sqs

import "testing"

func TestAssert_matchFound(t *testing.T) {
	msgs := []map[string]any{
		{"orderId": "ord_1", "status": "pending"},
		{"orderId": "ord_2", "status": "shipped"},
	}
	expect := map[string]any{"orderId": "ord_1", "status": "pending"}

	failures := Assert(msgs, expect)

	if len(failures) != 0 {
		t.Errorf("unexpected failures: %v", failures)
	}
}

func TestAssert_matchOnSecondMessage(t *testing.T) {
	msgs := []map[string]any{
		{"orderId": "ord_2"},
		{"orderId": "ord_1", "status": "pending"},
	}
	expect := map[string]any{"orderId": "ord_1"}

	failures := Assert(msgs, expect)

	if len(failures) != 0 {
		t.Errorf("unexpected failures: %v", failures)
	}
}

func TestAssert_noMatch(t *testing.T) {
	msgs := []map[string]any{
		{"orderId": "ord_2", "status": "shipped"},
	}
	expect := map[string]any{"orderId": "ord_1"}

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

	if failures[0].Actual != "(no messages found)" {
		t.Errorf("unexpected Actual: %q", failures[0].Actual)
	}
}

func TestAssert_emptyExpect(t *testing.T) {
	msgs := []map[string]any{{"orderId": "ord_1"}}

	failures := Assert(msgs, map[string]any{})

	if len(failures) != 0 {
		t.Errorf("empty expect should always pass, got %v", failures)
	}
}

func TestMatchesAll_missingField(t *testing.T) {
	actual := map[string]any{"status": "pending"}
	expect := map[string]any{"status": "pending", "orderId": "ord_1"}

	if matchesAll(actual, expect) {
		t.Error("expected false for missing field")
	}
}
