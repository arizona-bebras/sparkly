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
	var chErr error
	if b.ch != nil {
		chErr = b.ch.Close()
	}
	var connErr error
	if b.conn != nil {
		connErr = b.conn.Close()
	}
	if chErr != nil {
		return chErr
	}
	return connErr
}
