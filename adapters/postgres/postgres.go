// Package postgres provides a Postgres adapter for crosscheck database assertions.
// It connects via pgx/v5, rewrites named params (`:varName` → `$1`),
// executes queries, and asserts rows against an expect block.
package postgres

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

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

// Adapter holds a connection pool for a single Postgres instance.
type Adapter struct {
	pool *pgxpool.Pool
}

// New connects to Postgres and returns an Adapter.
// connStr is a standard DSN, e.g. "postgres://user:pass@localhost/testdb".
func New(ctx context.Context, connStr string) (*Adapter, error) {
	pool, err := pgxpool.New(ctx, connStr)

	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	return &Adapter{pool: pool}, nil
}

// Close releases all connections in the pool.
func (a *Adapter) Close() {
	a.pool.Close()
}

// Query rewrites named params, executes the query, and returns all rows
// as a slice of maps keyed by column name.
func (a *Adapter) Query(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	rewritten, args, err := rewriteParams(query, params)

	if err != nil {
		return nil, fmt.Errorf("postgres rewrite params: %w", err)
	}

	rows, err := a.pool.Query(ctx, rewritten, args...)

	if err != nil {
		return nil, fmt.Errorf("postgres query: %w", err)
	}
	defer rows.Close()

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
// Uses the Timeout and Interval from the config.WaitFor block.
func (a *Adapter) WaitFor(ctx context.Context, assert *config.DBAssert) ([]Failure, error) {
	timeout, err := time.ParseDuration(assert.WaitFor.Timeout)

	if err != nil {
		return nil, fmt.Errorf("postgres wait_for: invalid timeout %q: %w", assert.WaitFor.Timeout, err)
	}

	interval, err := time.ParseDuration(assert.WaitFor.Interval)

	if err != nil {
		return nil, fmt.Errorf("postgres wait_for: invalid interval %q: %w", assert.WaitFor.Interval, err)
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

// namedParamRe matches :identifier placeholders (word chars only, not preceded by colon).
var namedParamRe = regexp.MustCompile(`:([A-Za-z_][A-Za-z0-9_]*)`)

// rewriteParams converts `:varName` placeholders to `$1, $2, …` positional params
// and returns the rewritten query along with the ordered argument slice.
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

		return fmt.Sprintf("$%d", len(args))
	})

	if rewriteErr != nil {
		return "", nil, rewriteErr
	}

	return rewritten, args, nil
}

// collectRows reads all rows from a pgx.Rows into a slice of maps.
func collectRows(rows pgx.Rows) ([]map[string]any, error) {
	fields := rows.FieldDescriptions()

	var result []map[string]any

	for rows.Next() {
		values, err := rows.Values()

		if err != nil {
			return nil, fmt.Errorf("postgres scan row: %w", err)
		}

		row := make(map[string]any, len(fields))

		for i, fd := range fields {
			row[string(fd.Name)] = values[i]
		}

		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres rows: %w", err)
	}

	return result, nil
}
