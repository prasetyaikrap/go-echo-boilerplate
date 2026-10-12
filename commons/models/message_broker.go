package models

import (
	"context"
	"encoding/json"
)

// BrokerMessage is a protocol-agnostic message received from a message broker
type BrokerMessage struct {
	// Destination the message was sent to (queue or topic)
	Destination string
	// ContentType of Body, e.g. "application/json"
	ContentType string
	// Headers sent with the message, including broker-specific ones (message-id, etc.)
	Headers map[string]string
	// Body is the raw message payload
	Body []byte
}

// Bind decodes a JSON Body into dest, which must be a pointer
func (m BrokerMessage) Bind(dest any) error {
	return json.Unmarshal(m.Body, dest)
}

// BrokerMessageHandler processes a received message. Returning nil acknowledges the message,
// returning an error rejects it so the broker can redeliver it (or dead-letter it, depending on broker setup).
// Delivery is at-least-once, so handlers should be idempotent.
type BrokerMessageHandler func(ctx context.Context, msg BrokerMessage) error
