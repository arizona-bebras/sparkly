package msgq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type DeviceGroupsQueueProducer struct {
	queueName string
	ch        *amqp.Channel
}

func NewDeviceGroupsQueueProducer(broker *MessageBroker) (*DeviceGroupsQueueProducer, error) {
	qName := "group_membership"
	_, err := broker.ch.QueueDeclare(qName, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare group_membership queue: %w", err)
	}

	return &DeviceGroupsQueueProducer{
		queueName: qName,
		ch:        broker.ch,
	}, nil
}

func (q *DeviceGroupsQueueProducer) SendMsg(ctx context.Context, msg GroupMembershipMsg) error {
	msg.ID = uuid.New().String()
	binMsg, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal group membership message: %w", err)
	}

	if err := q.ch.PublishWithContext(ctx, "", q.queueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    msg.ID,
		Body:         binMsg,
	}); err != nil {
		return fmt.Errorf("failed to produce group membership message: %w", err)
	}

	return nil
}
