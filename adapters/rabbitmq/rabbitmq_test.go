package rabbitmq

import (
	"testing"
)

// Assert is a pure function — no broker needed.

func TestAssert_pass(t *testing.T) {
	msgs := []map[string]any{
		{"event": "order.placed", "orderId": "ord-1", "productId": "prod-001"},
	}
	expect := map[string]any{"event": "order.placed", "orderId": "ord-1"}

	failures := Assert(msgs, expect)

	if len(failures) != 0 {
		t.Errorf("unexpected failures: %v", failures)
	}
}

func TestAssert_noMessages(t *testing.T) {
	failures := Assert(nil, map[string]any{"event": "order.placed"})

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}

	if failures[0].Actual != "(no messages found)" {
		t.Errorf("unexpected failure message: %s", failures[0].Actual)
	}
}

func TestAssert_fieldMismatch(t *testing.T) {
	msgs := []map[string]any{
		{"event": "order.failed", "orderId": "ord-1"},
	}
	expect := map[string]any{"event": "order.placed"}

	failures := Assert(msgs, expect)

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}
}

func TestAssert_multipleMessages_oneMatches(t *testing.T) {
	msgs := []map[string]any{
		{"event": "order.failed"},
		{"event": "order.placed", "orderId": "ord-2"},
	}
	expect := map[string]any{"event": "order.placed", "orderId": "ord-2"}

	failures := Assert(msgs, expect)

	if len(failures) != 0 {
		t.Errorf("unexpected failures: %v", failures)
	}
}
