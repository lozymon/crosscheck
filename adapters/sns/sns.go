// Package sns provides an SNS adapter for crosscheck service assertions.
// SNS messages are read via an SQS queue subscribed to the topic — the SNS
// notification envelope is unwrapped transparently so expect fields are
// matched against the inner message payload.
//
// YAML example:
//
//	services:
//	  - adapter: sns
//	    queue: "https://sqs.us-east-1.amazonaws.com/123456789/my-topic-sub-queue"
//	    expect:
//	      orderId: "ord_123"
//	      status: pending
package sns

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
)

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

// snsEnvelope is the JSON structure of a message delivered to SQS from SNS.
type snsEnvelope struct {
	Type    string `json:"Type"`
	Message string `json:"Message"` // the actual payload (JSON string)
}

// Adapter reads from an SQS queue that is subscribed to an SNS topic.
type Adapter struct {
	sqsClient *awssqs.Client
}

// New creates an Adapter from an AWS config.
func New(cfg aws.Config) *Adapter {
	return &Adapter{sqsClient: awssqs.NewFromConfig(cfg)}
}

// Peek receives up to maxPeekMessages from queueURL, unwraps the SNS envelope,
// and returns the inner message payloads as maps.
func (a *Adapter) Peek(ctx context.Context, queueURL string) ([]map[string]any, error) {
	out, err := a.sqsClient.ReceiveMessage(ctx, &awssqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: maxPeekMessages,
		VisibilityTimeout:   0,
		WaitTimeSeconds:     0,
	})

	if err != nil {
		return nil, fmt.Errorf("sns/sqs receive %s: %w", queueURL, err)
	}

	result := make([]map[string]any, 0, len(out.Messages))

	for _, msg := range out.Messages {
		body := aws.ToString(msg.Body)

		payload, err := unwrapSNS(body)

		if err != nil {
			// Not an SNS envelope — treat raw body as the payload.
			var raw map[string]any

			if json.Unmarshal([]byte(body), &raw) == nil {
				result = append(result, raw)
			} else {
				result = append(result, map[string]any{"body": body})
			}

			continue
		}

		result = append(result, payload)
	}

	return result, nil
}

// Assert checks that at least one message in msgs satisfies all expect fields.
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

// unwrapSNS parses an SNS notification envelope and returns the inner payload.
func unwrapSNS(body string) (map[string]any, error) {
	var env snsEnvelope

	if err := json.Unmarshal([]byte(body), &env); err != nil {
		return nil, err
	}

	if env.Type != "Notification" || env.Message == "" {
		return nil, fmt.Errorf("not an SNS notification envelope")
	}

	var payload map[string]any

	if err := json.Unmarshal([]byte(env.Message), &payload); err != nil {
		// Message is a plain string, not JSON.
		return map[string]any{"message": env.Message}, nil
	}

	return payload, nil
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
