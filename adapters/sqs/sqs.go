// Package sqs provides an SQS adapter for crosscheck service assertions.
// It peeks at messages in a queue (without consuming them) and asserts
// their JSON body against an expect block.
//
// YAML example:
//
//	services:
//	  - adapter: sqs
//	    queue: "https://sqs.us-east-1.amazonaws.com/123456789/my-queue"
//	    expect:
//	      orderId: "ord_123"
//	      status: pending
package sqs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
)

// maxPeekMessages is the maximum SQS allows per ReceiveMessage call.
const maxPeekMessages = 10

// Failure describes a field assertion that did not pass.
type Failure struct {
	Field    string
	Expected string
	Actual   string
}

func (f Failure) Error() string {
	return fmt.Sprintf("%s: expected %q, got %q", f.Field, f.Expected, f.Actual)
}

// Adapter holds an SQS client.
type Adapter struct {
	client *awssqs.Client
}

// New creates an Adapter from an AWS config.
func New(cfg aws.Config) *Adapter {
	return &Adapter{client: awssqs.NewFromConfig(cfg)}
}

// Peek receives up to maxPeekMessages from queueURL without deleting them.
// VisibilityTimeout=0 makes messages immediately visible to other consumers again.
// Each message body is JSON-parsed; non-JSON bodies are returned as {"body": rawString}.
func (a *Adapter) Peek(ctx context.Context, queueURL string) ([]map[string]any, error) {
	out, err := a.client.ReceiveMessage(ctx, &awssqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: maxPeekMessages,
		VisibilityTimeout:   0,
		WaitTimeSeconds:     0,
	})

	if err != nil {
		return nil, fmt.Errorf("sqs receive %s: %w", queueURL, err)
	}

	result := make([]map[string]any, 0, len(out.Messages))

	for _, msg := range out.Messages {
		body := aws.ToString(msg.Body)

		var parsed map[string]any

		if json.Unmarshal([]byte(body), &parsed) == nil {
			result = append(result, parsed)
		} else {
			result = append(result, map[string]any{"body": body})
		}
	}

	return result, nil
}

// Assert checks that at least one message in msgs satisfies all expect fields.
// Returns a single Failure if no matching message is found; empty if one matches.
func Assert(msgs []map[string]any, expect map[string]any) []Failure {
	if len(msgs) == 0 {
		return []Failure{{
			Field:    "(queue)",
			Expected: fmt.Sprintf("%v", expect),
			Actual:   "(no messages found)",
		}}
	}

	for _, msg := range msgs {
		if matchesAll(msg, expect) {
			return nil
		}
	}

	return []Failure{{
		Field:    "(queue)",
		Expected: fmt.Sprintf("%v", expect),
		Actual:   fmt.Sprintf("checked %d message(s), none matched", len(msgs)),
	}}
}

// matchesAll returns true if actual contains all expected key/value pairs.
func matchesAll(actual map[string]any, expect map[string]any) bool {
	for field, expectedVal := range expect {
		actualVal, ok := actual[field]

		if !ok {
			return false
		}

		if fmt.Sprintf("%v", actualVal) != fmt.Sprintf("%v", expectedVal) {
			return false
		}
	}

	return true
}
