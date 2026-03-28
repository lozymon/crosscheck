// Package redis provides a Redis adapter for crosscheck service assertions.
// It connects via go-redis/v9, fetches keys (string or hash), and asserts
// values against an expect block.
package redis

import (
	"context"
	"encoding/json"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

// Failure describes a single field assertion that did not pass.
type Failure struct {
	Field    string
	Expected string
	Actual   string
}

func (f Failure) Error() string {
	return fmt.Sprintf("%s: expected %q, got %q", f.Field, f.Expected, f.Actual)
}

// Adapter holds a Redis client for a single instance.
type Adapter struct {
	client *goredis.Client
}

// New connects to Redis and returns an Adapter.
// addr is a Redis URL, e.g. "redis://localhost:6379" or "redis://:password@localhost:6379/0".
func New(ctx context.Context, addr string) (*Adapter, error) {
	opts, err := goredis.ParseURL(addr)

	if err != nil {
		return nil, fmt.Errorf("redis parse url: %w", err)
	}

	client := goredis.NewClient(opts)

	if err = client.Ping(ctx).Err(); err != nil {
		_ = client.Close()

		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return &Adapter{client: client}, nil
}

// Close releases the Redis connection.
func (a *Adapter) Close() error {
	return a.client.Close()
}

// Get fetches key and returns its contents as a flat map.
//
// Supported key types:
//   - hash   — HGETALL, returned as-is
//   - string — GET, then JSON-unmarshalled if possible; falls back to {"value": rawString}
//
// Returns an error if the key does not exist or has an unsupported type.
func (a *Adapter) Get(ctx context.Context, key string) (map[string]any, error) {
	keyType, err := a.client.Type(ctx, key).Result()

	if err != nil {
		return nil, fmt.Errorf("redis TYPE %s: %w", key, err)
	}

	switch keyType {
	case "hash":
		result, err := a.client.HGetAll(ctx, key).Result()

		if err != nil {
			return nil, fmt.Errorf("redis HGETALL %s: %w", key, err)
		}

		out := make(map[string]any, len(result))

		for k, v := range result {
			out[k] = v
		}

		return out, nil

	case "string":
		val, err := a.client.Get(ctx, key).Result()

		if err != nil {
			return nil, fmt.Errorf("redis GET %s: %w", key, err)
		}

		var parsed map[string]any

		if json.Unmarshal([]byte(val), &parsed) == nil {
			return parsed, nil
		}

		return map[string]any{"value": val}, nil

	case "none":
		return nil, fmt.Errorf("redis key %q does not exist", key)

	default:
		return nil, fmt.Errorf(
			"redis key %q has unsupported type %q (supported: string, hash)",
			key, keyType,
		)
	}
}

// Assert compares actual key data against the expect block.
// Returns a slice of Failure — empty means all assertions passed.
func Assert(actual map[string]any, expect map[string]any) []Failure {
	var failures []Failure

	for field, expectedVal := range expect {
		actualVal, ok := actual[field]

		if !ok {
			failures = append(failures, Failure{
				Field:    field,
				Expected: fmt.Sprintf("%v", expectedVal),
				Actual:   "(field not found)",
			})

			continue
		}

		actualStr := fmt.Sprintf("%v", actualVal)
		expectedStr := fmt.Sprintf("%v", expectedVal)

		if actualStr != expectedStr {
			failures = append(failures, Failure{
				Field:    field,
				Expected: expectedStr,
				Actual:   actualStr,
			})
		}
	}

	return failures
}
