package lambda

import "testing"

func TestAssert_pass(t *testing.T) {
	actual := map[string]any{"result": "success", "statusCode": "200"}
	expect := map[string]any{"result": "success"}

	failures := Assert(actual, expect)

	if len(failures) != 0 {
		t.Errorf("unexpected failures: %v", failures)
	}
}

func TestAssert_fieldMismatch(t *testing.T) {
	actual := map[string]any{"result": "error"}
	expect := map[string]any{"result": "success"}

	failures := Assert(actual, expect)

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}

	if failures[0].Field != "result" {
		t.Errorf("expected field=result, got %q", failures[0].Field)
	}

	if failures[0].Expected != "success" {
		t.Errorf("expected Expected=success, got %q", failures[0].Expected)
	}

	if failures[0].Actual != "error" {
		t.Errorf("expected Actual=error, got %q", failures[0].Actual)
	}
}

func TestAssert_missingField(t *testing.T) {
	actual := map[string]any{"result": "success"}
	expect := map[string]any{"result": "success", "statusCode": "200"}

	failures := Assert(actual, expect)

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for missing field, got %d", len(failures))
	}

	if failures[0].Actual != "(field not found)" {
		t.Errorf("expected Actual=(field not found), got %q", failures[0].Actual)
	}
}

func TestAssert_emptyExpect(t *testing.T) {
	actual := map[string]any{"result": "success"}

	failures := Assert(actual, map[string]any{})

	if len(failures) != 0 {
		t.Errorf("empty expect should always pass, got %v", failures)
	}
}
