package mysql

import (
	"context"
	"os"
	"testing"
)

// rewriteParams and Assert are pure functions — no DB needed.

// --- rewriteParams ---

func TestRewriteParams_singleParam(t *testing.T) {
	q, args, err := rewriteParams(
		"SELECT * FROM orders WHERE id = :orderId",
		map[string]any{"orderId": "ord_1"},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q != "SELECT * FROM orders WHERE id = ?" {
		t.Errorf("unexpected query: %q", q)
	}

	if len(args) != 1 || args[0] != "ord_1" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestRewriteParams_multipleParams(t *testing.T) {
	q, args, err := rewriteParams(
		"SELECT * FROM orders WHERE status = :status AND quantity = :qty",
		map[string]any{"status": "pending", "qty": 2},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q != "SELECT * FROM orders WHERE status = ? AND quantity = ?" {
		t.Errorf("unexpected query: %q", q)
	}

	if len(args) != 2 {
		t.Errorf("expected 2 args, got %d", len(args))
	}
}

func TestRewriteParams_noParams(t *testing.T) {
	q, args, err := rewriteParams("SELECT 1", map[string]any{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q != "SELECT 1" {
		t.Errorf("unexpected query: %q", q)
	}

	if len(args) != 0 {
		t.Errorf("expected no args, got %v", args)
	}
}

func TestRewriteParams_missingParam(t *testing.T) {
	_, _, err := rewriteParams(
		"SELECT * FROM orders WHERE id = :orderId",
		map[string]any{},
	)

	if err == nil {
		t.Fatal("expected error for missing param")
	}
}

// --- Assert ---

func TestAssert_pass(t *testing.T) {
	rows := []map[string]any{
		{"status": "pending", "quantity": "2"},
	}
	expect := []map[string]any{
		{"status": "pending", "quantity": "2"},
	}

	failures := Assert(rows, expect)

	if len(failures) != 0 {
		t.Errorf("unexpected failures: %v", failures)
	}
}

func TestAssert_fieldMismatch(t *testing.T) {
	rows := []map[string]any{
		{"status": "shipped"},
	}
	expect := []map[string]any{
		{"status": "pending"},
	}

	failures := Assert(rows, expect)

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

func TestAssert_missingRow(t *testing.T) {
	rows := []map[string]any{}
	expect := []map[string]any{
		{"status": "pending"},
	}

	failures := Assert(rows, expect)

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for missing row, got %d", len(failures))
	}

	if failures[0].Actual != "(missing)" {
		t.Errorf("expected Actual=(missing), got %q", failures[0].Actual)
	}
}

func TestAssert_missingColumn(t *testing.T) {
	rows := []map[string]any{
		{"status": "pending"},
	}
	expect := []map[string]any{
		{"status": "pending", "quantity": "2"},
	}

	failures := Assert(rows, expect)

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for missing column, got %d", len(failures))
	}

	if failures[0].Field != "quantity" {
		t.Errorf("expected field=quantity, got %q", failures[0].Field)
	}
}

func TestAssert_emptyExpect(t *testing.T) {
	rows := []map[string]any{{"status": "pending"}}

	failures := Assert(rows, []map[string]any{})

	if len(failures) != 0 {
		t.Errorf("empty expect should always pass, got %v", failures)
	}
}

// Integration tests — only run when MYSQL_URL is set.

func TestNew_ping(t *testing.T) {
	dsn := os.Getenv("MYSQL_URL")

	if dsn == "" {
		t.Skip("MYSQL_URL not set — skipping integration test")
	}

	ctx := context.Background()

	a, err := New(ctx, dsn)

	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = a.Close() }()
}
