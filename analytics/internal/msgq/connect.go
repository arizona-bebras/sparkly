package msgq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type MessageBroker struct {
	ch *amqp.Channel
}

func NewMessageBroker(host string, port string) (*MessageBroker, error) {
	conn, err := amqp.Dial(fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq service: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to get channel from rabbitmq connect: %w", err)
	}

	return &MessageBroker{
		ch: ch,
	}, nil
}