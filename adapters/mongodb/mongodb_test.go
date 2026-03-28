package mongodb

import (
	"context"
	"os"
	"testing"
)

// Assert and databaseFromURI are pure functions — no MongoDB needed.

// --- databaseFromURI ---

func TestDatabaseFromURI_present(t *testing.T) {
	db, err := databaseFromURI("mongodb://localhost:27017/testdb")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if db != "testdb" {
		t.Errorf("expected testdb, got %q", db)
	}
}

func TestDatabaseFromURI_withCredentials(t *testing.T) {
	db, err := databaseFromURI("mongodb://user:pass@localhost:27017/mydb")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if db != "mydb" {
		t.Errorf("expected mydb, got %q", db)
	}
}

func TestDatabaseFromURI_missing(t *testing.T) {
	_, err := databaseFromURI("mongodb://localhost:27017")

	if err == nil {
		t.Fatal("expected error for missing database")
	}
}

func TestDatabaseFromURI_emptyPath(t *testing.T) {
	_, err := databaseFromURI("mongodb://localhost:27017/")

	if err == nil {
		t.Fatal("expected error for empty database path")
	}
}

// --- Assert ---

func TestAssert_pass(t *testing.T) {
	docs := []map[string]any{
		{"status": "pending", "quantity": "2"},
	}
	expect := []map[string]any{
		{"status": "pending", "quantity": "2"},
	}

	failures := Assert(docs, expect)

	if len(failures) != 0 {
		t.Errorf("unexpected failures: %v", failures)
	}
}

func TestAssert_fieldMismatch(t *testing.T) {
	docs := []map[string]any{
		{"status": "shipped"},
	}
	expect := []map[string]any{
		{"status": "pending"},
	}

	failures := Assert(docs, expect)

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

func TestAssert_missingDoc(t *testing.T) {
	docs := []map[string]any{}
	expect := []map[string]any{
		{"status": "pending"},
	}

	failures := Assert(docs, expect)

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for missing doc, got %d", len(failures))
	}

	if failures[0].Actual != "(missing)" {
		t.Errorf("expected Actual=(missing), got %q", failures[0].Actual)
	}
}

func TestAssert_missingField(t *testing.T) {
	docs := []map[string]any{
		{"status": "pending"},
	}
	expect := []map[string]any{
		{"status": "pending", "quantity": "2"},
	}

	failures := Assert(docs, expect)

	if len(failures) != 1 {
		t.Fatalf("expected 1 failure for missing field, got %d", len(failures))
	}

	if failures[0].Field != "quantity" {
		t.Errorf("expected field=quantity, got %q", failures[0].Field)
	}
}

func TestAssert_emptyExpect(t *testing.T) {
	docs := []map[string]any{{"status": "pending"}}

	failures := Assert(docs, []map[string]any{})

	if len(failures) != 0 {
		t.Errorf("empty expect should always pass, got %v", failures)
	}
}

// Integration tests — only run when MONGODB_URL is set.

func TestNew_ping(t *testing.T) {
	uri := os.Getenv("MONGODB_URL")

	if uri == "" {
		t.Skip("MONGODB_URL not set — skipping integration test")
	}

	ctx := context.Background()

	a, err := New(ctx, uri)

	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = a.Close(ctx) }()
}

func TestQuery_insertAndFind(t *testing.T) {
	uri := os.Getenv("MONGODB_URL")

	if uri == "" {
		t.Skip("MONGODB_URL not set — skipping integration test")
	}

	ctx := context.Background()

	a, err := New(ctx, uri)

	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = a.Close(ctx) }()

	// Insert a test document.
	coll := a.client.Database(a.dbName).Collection("cx_test")

	_, err = coll.InsertOne(ctx, map[string]any{"status": "pending", "ref": "cx-test-1"})

	if err != nil {
		t.Fatalf("InsertOne: %v", err)
	}

	defer func() { _, _ = coll.DeleteMany(ctx, map[string]any{"ref": "cx-test-1"}) }()

	docs, err := a.Query(ctx, "cx_test", map[string]any{"ref": "cx-test-1"})

	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}

	if docs[0]["status"] != "pending" {
		t.Errorf("expected status=pending, got %v", docs[0]["status"])
	}
}
