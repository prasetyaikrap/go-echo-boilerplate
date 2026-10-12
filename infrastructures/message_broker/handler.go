package message_broker

import (
	"context"
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/infrastructures/configurations"
	"go-serviceboilerplate/infrastructures/message_broker/stomp"
)

// MessageBroker is the protocol-agnostic contract every broker implementation (STOMP, AMQP, Kafka, NATS, ...) satisfies.
// Depend on this interface in repositories and use cases instead of a concrete protocol.
type MessageBroker interface {
	// Publish sends payload to a destination. []byte is sent as-is, string as text, anything else as JSON
	Publish(ctx context.Context, destination string, payload any, headers map[string]string) error
	// Subscribe consumes a destination in the background. handler returning nil acknowledges the message
	Subscribe(destination string, handler models.BrokerMessageHandler, headers map[string]string) error
	// Close stops subscriptions, waits for in-flight handlers, and disconnects
	Close() error
}

var _ MessageBroker = (*stomp.StompInstance)(nil)

// MessageBrokerInstance holds every message broker connection the service uses. Add a field per extra connection
type MessageBrokerInstance struct {
	MainStomp MessageBroker
}

func NewMessageBrokerInstance(configs *configurations.Configs) *MessageBrokerInstance {
	mainStomp := stomp.NewStompInstance(configs, configs.Envs.MainStomp)

	return &MessageBrokerInstance{
		MainStomp: mainStomp,
	}
}

func (m *MessageBrokerInstance) Close() {
	m.MainStomp.Close()
}
