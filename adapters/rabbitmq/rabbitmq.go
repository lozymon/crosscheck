// Package rabbitmq provides a RabbitMQ adapter for crosscheck service assertions.
// It connects via amqp091-go, peeks messages from a named queue without consuming
// them, and asserts their JSON payload against an expect block.
//
// YAML example:
//
//	services:
//	  - adapter: rabbitmq
//	    queue: assert-orders
//	    wait_for: { timeout: 5s, interval: 200ms }
//	    expect:
//	      event: order.placed
//	      orderId: "ord_123"
package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// maxPeekMessages is the maximum number of messages read per Peek call.
const maxPeekMessages = 10

// Failure describes a single assertion that did not pass.
type Failure struct {
	Field    string
	Expected string
	Actual   string
}

func (f Failure) Error() string {
	return fmt.Sprintf("%s: expected %q, got %q", f.Field, f.Expected, f.Actual)
}

// Adapter holds an AMQP connection and channel.
type Adapter struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// New dials a RabbitMQ broker and returns an Adapter.
// url must be a valid AMQP URL, e.g. "amqp://guest:guest@localhost:5672".
func New(_ context.Context, url string) (*Adapter, error) {
	conn, err := amqp.Dial(url)

	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial %s: %w", url, err)
	}

	ch, err := conn.Channel()

	if err != nil {
		_ = conn.Close()

		return nil, fmt.Errorf("rabbitmq open channel: %w", err)
	}

	return &Adapter{conn: conn, channel: ch}, nil
}

// Close releases the channel and connection.
func (a *Adapter) Close() error {
	_ = a.channel.Close()

	return a.conn.Close()
}

// Peek reads up to maxPeekMessages from queue without permanently consuming them.
// Each message is basic.get'd with autoAck=false and then nack'd with requeue=true
// so all messages remain available for the real subscriber services.
// Each message body is JSON-parsed; non-JSON bodies fall back to {"body": rawString}.
func (a *Adapter) Peek(_ context.Context, queue string) ([]map[string]any, error) {
	var result []map[string]any
	var deliveries []amqp.Delivery

	for range maxPeekMessages {
		msg, ok, err := a.channel.Get(queue, false /* autoAck */)

		if err != nil {
			return nil, fmt.Errorf("rabbitmq get %s: %w", queue, err)
		}

		if !ok {
			break
		}

		deliveries = append(deliveries, msg)

		var parsed map[string]any

		if json.Unmarshal(msg.Body, &parsed) == nil {
			result = append(result, parsed)
		} else {
			result = append(result, map[string]any{"body": string(msg.Body)})
		}
	}

	// Requeue all messages so they remain in the queue for service consumers.
	for _, d := range deliveries {
		_ = d.Nack(false /* multiple */, true /* requeue */)
	}

	return result, nil
}

// Assert checks that at least one message in msgs satisfies all expect fields.
// Returns a single Failure if no matching message is found; nil slice means pass.
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

// matchesAll returns true when actual contains all key/value pairs in expect.
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
