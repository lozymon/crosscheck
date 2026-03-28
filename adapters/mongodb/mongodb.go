// Package mongodb provides a MongoDB adapter for crosscheck database assertions.
// It connects via mongo-driver/v2 and finds documents matching a filter.
//
// YAML mapping:
//
//	adapter: mongodb
//	query:  orders          # collection name
//	params:
//	  orderId: "{{ orderId }}"   # filter fields
//	expect:
//	  - status: pending
//
// The database name is taken from the URL path: mongodb://host:27017/mydb.
package mongodb

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lozymon/crosscheck/config"
)

// Failure describes a single document field assertion that did not pass.
type Failure struct {
	Doc      int
	Field    string
	Expected string
	Actual   string
}

func (f Failure) Error() string {
	return fmt.Sprintf("doc %d: %s: expected %q, got %q", f.Doc, f.Field, f.Expected, f.Actual)
}

// Adapter holds a MongoDB client and target database name.
type Adapter struct {
	client *mongo.Client
	dbName string
}

// New connects to MongoDB and returns an Adapter.
// uri must include the database name in the path, e.g.
// "mongodb://user:pass@localhost:27017/testdb".
func New(ctx context.Context, uri string) (*Adapter, error) {
	dbName, err := databaseFromURI(uri)

	if err != nil {
		return nil, err
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))

	if err != nil {
		return nil, fmt.Errorf("mongodb connect: %w", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)

		return nil, fmt.Errorf("mongodb ping: %w", err)
	}

	return &Adapter{client: client, dbName: dbName}, nil
}

// Close disconnects the MongoDB client.
func (a *Adapter) Close(ctx context.Context) error {
	return a.client.Disconnect(ctx)
}

// Query finds documents in collection matching filter and returns them as
// a slice of maps. collection is the value of the YAML `query:` field;
// filter is the interpolated `params:` map.
func (a *Adapter) Query(ctx context.Context, collection string, filter map[string]any) ([]map[string]any, error) {
	coll := a.client.Database(a.dbName).Collection(collection)

	cur, err := coll.Find(ctx, filter)

	if err != nil {
		return nil, fmt.Errorf("mongodb find %s: %w", collection, err)
	}

	defer func() { _ = cur.Close(ctx) }()

	var docs []bson.M

	if err = cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongodb decode %s: %w", collection, err)
	}

	result := make([]map[string]any, len(docs))

	for i, d := range docs {
		result[i] = flattenDoc(d)
	}

	return result, nil
}

// Assert compares actual documents returned by Query against the expect block.
// Returns a slice of Failure — empty means all assertions passed.
func Assert(docs []map[string]any, expect []map[string]any) []Failure {
	var failures []Failure

	for i, expectedDoc := range expect {
		if i >= len(docs) {
			failures = append(failures, Failure{
				Doc:      i,
				Field:    "(doc)",
				Expected: fmt.Sprintf("%v", expectedDoc),
				Actual:   "(missing)",
			})

			continue
		}

		actualDoc := docs[i]

		for field, expectedVal := range expectedDoc {
			actual, ok := actualDoc[field]

			if !ok {
				failures = append(failures, Failure{
					Doc:      i,
					Field:    field,
					Expected: fmt.Sprintf("%v", expectedVal),
					Actual:   "(field not returned)",
				})

				continue
			}

			actualStr := fmt.Sprintf("%v", actual)
			expectedStr := fmt.Sprintf("%v", expectedVal)

			if actualStr != expectedStr {
				failures = append(failures, Failure{
					Doc:      i,
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
		return nil, fmt.Errorf("mongodb wait_for: invalid timeout %q: %w", assert.WaitFor.Timeout, err)
	}

	interval, err := time.ParseDuration(assert.WaitFor.Interval)

	if err != nil {
		return nil, fmt.Errorf("mongodb wait_for: invalid interval %q: %w", assert.WaitFor.Interval, err)
	}

	deadline := time.Now().Add(timeout)

	for {
		docs, queryErr := a.Query(ctx, assert.Query, assert.Params)

		if queryErr != nil {
			return nil, queryErr
		}

		failures := Assert(docs, assert.Expect)

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

// databaseFromURI extracts the database name from a MongoDB URI path.
// Returns an error if no database is present.
func databaseFromURI(uri string) (string, error) {
	u, err := url.Parse(uri)

	if err != nil {
		return "", fmt.Errorf("mongodb parse uri: %w", err)
	}

	db := strings.TrimPrefix(u.Path, "/")
	db = strings.SplitN(db, "?", 2)[0]

	if db == "" {
		return "", fmt.Errorf(
			"mongodb uri must include a database name in the path, e.g. mongodb://host:27017/mydb",
		)
	}

	return db, nil
}

// flattenDoc converts a bson.M to map[string]any, stringifying ObjectID and
// other BSON-specific types so Assert comparisons work with plain YAML values.
func flattenDoc(doc bson.M) map[string]any {
	out := make(map[string]any, len(doc))

	for k, v := range doc {
		switch val := v.(type) {
		case bson.M:
			out[k] = fmt.Sprintf("%v", val)
		default:
			out[k] = fmt.Sprintf("%v", val)
		}
	}

	return out
}
