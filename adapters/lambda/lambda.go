// Package lambda provides a Lambda adapter for crosscheck service assertions.
// It invokes a Lambda function directly and asserts the JSON response against
// an expect block.
//
// YAML example:
//
//	services:
//	  - adapter: lambda
//	    key: process-order-fn   # function name or ARN
//	    payload:
//	      orderId: "{{ orderId }}"
//	    expect:
//	      result: success
package lambda

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awslambda "github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
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

// Adapter holds a Lambda client.
type Adapter struct {
	client *awslambda.Client
}

// New creates an Adapter from an AWS config.
func New(cfg aws.Config) *Adapter {
	return &Adapter{client: awslambda.NewFromConfig(cfg)}
}

// Invoke calls function synchronously with payload and returns the response
// body parsed as a map. payload may be nil for functions that require no input.
// Returns an error if the function errors or the response cannot be parsed.
func (a *Adapter) Invoke(ctx context.Context, function string, payload map[string]any) (map[string]any, error) {
	var payloadBytes []byte

	if len(payload) > 0 {
		var err error

		payloadBytes, err = json.Marshal(payload)

		if err != nil {
			return nil, fmt.Errorf("lambda marshal payload: %w", err)
		}
	}

	out, err := a.client.Invoke(ctx, &awslambda.InvokeInput{
		FunctionName:   aws.String(function),
		InvocationType: types.InvocationTypeRequestResponse,
		Payload:        payloadBytes,
	})

	if err != nil {
		return nil, fmt.Errorf("lambda invoke %s: %w", function, err)
	}

	if out.FunctionError != nil {
		return nil, fmt.Errorf(
			"lambda %s returned function error %q: %s",
			function, aws.ToString(out.FunctionError), string(out.Payload),
		)
	}

	var result map[string]any

	if err = json.Unmarshal(out.Payload, &result); err != nil {
		return map[string]any{"response": string(out.Payload)}, nil
	}

	return result, nil
}

// Assert compares actual response fields against the expect block.
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
