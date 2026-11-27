package msgq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type MessageBroker struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewMessageBroker(url string) (*MessageBroker, error) {
	if url == "" {
		url = "amqp://localhost:5672/"
	}

	fmt.Println(url)
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq service: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get channel from rabbitmq connect: %w", err)
	}

	return &MessageBroker{
		conn: conn,
		ch:   ch,
	}, nil
}

func (b *MessageBroker) Close() error {
	if b.conn != nil {
		return b.conn.Close()
	}
	return nil
}
