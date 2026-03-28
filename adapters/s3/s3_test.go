package s3

import "testing"

func TestAssert_pass(t *testing.T) {
	actual := map[string]any{"status": "pending", "orderId": "ord_1"}
	expect := map[string]any{"status": "pending"}

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
	expect := map[string]any{"status": "pending", "orderId": "ord_1"}

	failures := Assert(actual, expect)

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for missing field, got %d", len(failures))
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
