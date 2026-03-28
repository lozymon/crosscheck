// Package dynamodb provides a DynamoDB adapter for crosscheck service assertions.
// It fetches a single item by primary key and asserts its fields against an expect block.
//
// YAML example:
//
//	services:
//	  - adapter: dynamodb
//	    table: orders
//	    key_name: orderId       # partition key attribute name (default: "id")
//	    key: "{{ orderId }}"    # partition key value
//	    expect:
//	      status: pending
//
// For tables with a sort key:
//
//	sort_key_name: createdAt
//	sort_key: "2024-01-01"
package dynamodb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	awsddb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// defaultKeyName is used when key_name is not specified in the YAML.
const defaultKeyName = "id"

// Failure describes a field assertion that did not pass.
type Failure struct {
	Field    string
	Expected string
	Actual   string
}

func (f Failure) Error() string {
	return fmt.Sprintf("%s: expected %q, got %q", f.Field, f.Expected, f.Actual)
}

// Adapter holds a DynamoDB client.
type Adapter struct {
	client *awsddb.Client
}

// New creates an Adapter from an AWS config.
func New(cfg aws.Config) *Adapter {
	return &Adapter{client: awsddb.NewFromConfig(cfg)}
}

// GetItem fetches a single item from table by its primary key.
// keyName is the partition key attribute name (defaults to "id" if empty).
// If sortKeyName and sortKey are non-empty, they are included in the key.
// Returns the item as a flat map of stringified values.
func (a *Adapter) GetItem(ctx context.Context, table, keyName, keyVal, sortKeyName, sortKey string) (map[string]any, error) {
	if keyName == "" {
		keyName = defaultKeyName
	}

	key := map[string]types.AttributeValue{
		keyName: &types.AttributeValueMemberS{Value: keyVal},
	}

	if sortKeyName != "" && sortKey != "" {
		key[sortKeyName] = &types.AttributeValueMemberS{Value: sortKey}
	}

	out, err := a.client.GetItem(ctx, &awsddb.GetItemInput{
		TableName: aws.String(table),
		Key:       key,
	})

	if err != nil {
		return nil, fmt.Errorf("dynamodb get %s/%s=%s: %w", table, keyName, keyVal, err)
	}

	if out.Item == nil {
		return nil, fmt.Errorf("dynamodb item not found: table=%s %s=%s", table, keyName, keyVal)
	}

	var raw map[string]any

	if err = attributevalue.UnmarshalMap(out.Item, &raw); err != nil {
		return nil, fmt.Errorf("dynamodb unmarshal %s: %w", table, err)
	}

	// Stringify all values so Assert comparisons work with plain YAML values.
	result := make(map[string]any, len(raw))

	for k, v := range raw {
		result[k] = fmt.Sprintf("%v", v)
	}

	return result, nil
}

// Assert compares actual item fields against the expect block.
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
