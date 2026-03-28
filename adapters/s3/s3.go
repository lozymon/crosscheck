// Package s3 provides an S3 adapter for crosscheck service assertions.
// It fetches an object and asserts its JSON body against an expect block.
//
// YAML example:
//
//	services:
//	  - adapter: s3
//	    bucket: my-bucket
//	    key: "orders/{{ orderId }}.json"
//	    expect:
//	      status: pending
package s3

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

// Failure describes a field assertion that did not pass.
type Failure struct {
	Field    string
	Expected string
	Actual   string
}

func (f Failure) Error() string {
	return fmt.Sprintf("%s: expected %q, got %q", f.Field, f.Expected, f.Actual)
}

// Adapter holds an S3 client.
type Adapter struct {
	client *awss3.Client
}

// New creates an Adapter from an AWS config.
func New(cfg aws.Config) *Adapter {
	return &Adapter{client: awss3.NewFromConfig(cfg)}
}

// GetObject fetches bucket/key and returns its contents as a map.
// JSON objects are parsed directly; plain text is returned as {"content": value}.
func (a *Adapter) GetObject(ctx context.Context, bucket, key string) (map[string]any, error) {
	out, err := a.client.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return nil, fmt.Errorf("s3 get s3://%s/%s: %w", bucket, key, err)
	}

	defer func() { _ = out.Body.Close() }()

	body, err := io.ReadAll(out.Body)

	if err != nil {
		return nil, fmt.Errorf("s3 read s3://%s/%s: %w", bucket, key, err)
	}

	var parsed map[string]any

	if json.Unmarshal(body, &parsed) == nil {
		return parsed, nil
	}

	return map[string]any{"content": string(body)}, nil
}

// Assert compares actual object fields against the expect block.
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
