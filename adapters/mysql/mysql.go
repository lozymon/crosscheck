// Package mysql provides a MySQL adapter for crosscheck database assertions.
// It connects via database/sql + go-sql-driver/mysql, rewrites named params
// (`:varName` → `?`), executes queries, and asserts rows against an expect block.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // register mysql driver

	"github.com/lozymon/crosscheck/config"
)

// Failure describes a single row assertion that did not pass.
type Failure struct {
	Row      int
	Field    string
	Expected string
	Actual   string
}

func (f Failure) Error() string {
	return fmt.Sprintf("row %d: %s: expected %q, got %q", f.Row, f.Field, f.Expected, f.Actual)
}

// Adapter holds a database connection for a single MySQL instance.
type Adapter struct {
	db *sql.DB
}

// New connects to MySQL and returns an Adapter.
// dsn may be a go-sql-driver DSN ("user:pass@tcp(host:port)/db")
// or a mysql:// URL ("mysql://user:pass@host:port/db") — both are normalised.
func New(ctx context.Context, dsn string) (*Adapter, error) {
	db, err := sql.Open("mysql", normaliseDSN(dsn))

	if err != nil {
		return nil, fmt.Errorf("mysql open: %w", err)
	}

	if err = db.PingContext(ctx); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("mysql ping: %w", err)
	}

	return &Adapter{db: db}, nil
}

// Close releases the database connection.
func (a *Adapter) Close() error {
	return a.db.Close()
}

// normaliseDSN converts a mysql:// URL to go-sql-driver's native DSN format.
// If the string doesn't start with "mysql://" it is returned unchanged.
func normaliseDSN(dsn string) string {
	const prefix = "mysql://"

	if !strings.HasPrefix(dsn, prefix) {
		return dsn
	}

	// Strip scheme: "mysql://user:pass@host:port/db" → "user:pass@host:port/db"
	rest := dsn[len(prefix):]

	// Split on the last '@' to separate credentials from host/db.
	at := strings.LastIndex(rest, "@")

	if at < 0 {
		return rest
	}

	creds := rest[:at]    // "user:pass"
	hostDB := rest[at+1:] // "host:port/db"

	// Split host from database path.
	slash := strings.Index(hostDB, "/")

	var host, db string

	if slash < 0 {
		host = hostDB
		db = ""
	} else {
		host = hostDB[:slash]
		db = hostDB[slash:]
	}

	// Wrap host in tcp(...) as required by go-sql-driver.
	return fmt.Sprintf("%s@tcp(%s)%s", creds, host, db)
}

// Query rewrites named params, executes the query, and returns all rows
// as a slice of maps keyed by column name.
func (a *Adapter) Query(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	rewritten, args, err := rewriteParams(query, params)

	if err != nil {
		return nil, fmt.Errorf("mysql rewrite params: %w", err)
	}

	rows, err := a.db.QueryContext(ctx, rewritten, args...)

	if err != nil {
		return nil, fmt.Errorf("mysql query: %w", err)
	}

	defer func() { _ = rows.Close() }()

	return collectRows(rows)
}

// Assert compares actual rows returned by Query against the expect block.
// Returns a slice of Failure — empty means all assertions passed.
func Assert(rows []map[string]any, expect []map[string]any) []Failure {
	var failures []Failure

	for i, expectedRow := range expect {
		if i >= len(rows) {
			failures = append(failures, Failure{
				Row:      i,
				Field:    "(row)",
				Expected: fmt.Sprintf("%v", expectedRow),
				Actual:   "(missing)",
			})

			continue
		}

		actualRow := rows[i]

		for field, expectedVal := range expectedRow {
			actual, ok := actualRow[field]

			if !ok {
				failures = append(failures, Failure{
					Row:      i,
					Field:    field,
					Expected: fmt.Sprintf("%v", expectedVal),
					Actual:   "(column not returned)",
				})

				continue
			}

			actualStr := fmt.Sprintf("%v", actual)
			expectedStr := fmt.Sprintf("%v", expectedVal)

			if actualStr != expectedStr {
				failures = append(failures, Failure{
					Row:      i,
					Field:    field,
					Expected: expectedStr,
					Actual:   actualStr,
				})
			}
		}
	}

	return failures
}

// WaitFor polls Query + Assert until all assertions pass or the timeout is reached.
func (a *Adapter) WaitFor(ctx context.Context, assert *config.DBAssert) ([]Failure, error) {
	timeout, err := time.ParseDuration(assert.WaitFor.Timeout)

	if err != nil {
		return nil, fmt.Errorf("mysql wait_for: invalid timeout %q: %w", assert.WaitFor.Timeout, err)
	}

	interval, err := time.ParseDuration(assert.WaitFor.Interval)

	if err != nil {
		return nil, fmt.Errorf("mysql wait_for: invalid interval %q: %w", assert.WaitFor.Interval, err)
	}

	deadline := time.Now().Add(timeout)

	for {
		rows, queryErr := a.Query(ctx, assert.Query, assert.Params)

		if queryErr != nil {
			return nil, queryErr
		}

		failures := Assert(rows, assert.Expect)

		if len(failures) == 0 {
			return nil, nil
		}

		if time.Now().After(deadline) {
			return failures, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// namedParamRe matches :identifier placeholders (word chars only).
var namedParamRe = regexp.MustCompile(`:([A-Za-z_][A-Za-z0-9_]*)`)

// rewriteParams converts `:varName` placeholders to `?` positional params
// and returns the rewritten query along with the ordered argument slice.
// MySQL uses `?` for all placeholders (no numbering).
func rewriteParams(query string, params map[string]any) (string, []any, error) {
	var args []any

	var rewriteErr error

	rewritten := namedParamRe.ReplaceAllStringFunc(query, func(match string) string {
		if rewriteErr != nil {
			return match
		}

		name := strings.TrimPrefix(match, ":")

		val, ok := params[name]

		if !ok {
			rewriteErr = fmt.Errorf("param %q referenced in query but not provided", name)

			return match
		}

		args = append(args, val)

		return "?"
	})

	if rewriteErr != nil {
		return "", nil, rewriteErr
	}

	return rewritten, args, nil
}

// collectRows reads all rows from a *sql.Rows into a slice of maps.
func collectRows(rows *sql.Rows) ([]map[string]any, error) {
	cols, err := rows.Columns()

	if err != nil {
		return nil, fmt.Errorf("mysql columns: %w", err)
	}

	var result []map[string]any

	for rows.Next() {
		values := make([]any, len(cols))
		valuePtrs := make([]any, len(cols))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err = rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("mysql scan row: %w", err)
		}

		row := make(map[string]any, len(cols))

		for i, col := range cols {
			// MySQL driver returns []byte for string columns — convert to string.
			if b, ok := values[i].([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = values[i]
			}
		}

		result = append(result, row)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql rows: %w", err)
	}

	return result, nil
}
