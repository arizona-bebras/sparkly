package msgq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sparkly/analytics/internal/domain"
	"sparkly/analytics/internal/services"

	amqp "github.com/rabbitmq/amqp091-go"
)


type DeviceGroupsQueueConsumer struct {
	queueName string
	ch        *amqp.Channel
	service services.GroupMembershipService
}

func NewDeviceGroupsQueueConsumer(broker *MessageBroker) (*DeviceGroupsQueueConsumer, error) {
	qName := "group_membership"
	_, err := broker.ch.QueueDeclare(qName, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare group_membership queue: %w", err)
	}

	return &DeviceGroupsQueueConsumer{
		queueName: qName,
		ch: broker.ch,
	}, nil
}

func (c *DeviceGroupsQueueConsumer) processMsg(ctx context.Context, brokerMsg amqp.Delivery) error {
	if brokerMsg.ContentType != "application/json" {
		return fmt.Errorf("message broker, group_membership queue error: %w", domain.ErrWrongBrokerMsgContentType)
	}
	
	r := bytes.NewReader(brokerMsg.Body)
	
	var msgParams GroupMembershipMsg
	if err := json.NewDecoder(r).Decode(&msgParams); err != nil {
		return fmt.Errorf("failed to decode broker message: %w", err)
	}

	err := c.service.AssociateGroupsToDevice(ctx, msgParams.DeviceID, msgParams.GroupIDs, msgParams.Version)
	if err != nil {
		return fmt.Errorf("failed to associate groups to device: %w", err)
	}

	return nil
}

func (c *DeviceGroupsQueueConsumer) Listen(ctx context.Context) error {
	msgs, err := c.ch.Consume(c.queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to register rabbitmq group_membership consumer: %w", err)
	}

	go func() {
		select {
		case <-ctx.Done():
			c.ch.Close()
			return
		case brokerMsg := <-msgs:
			if err := c.processMsg(ctx, brokerMsg); err != nil {
				log.Printf("message broker, group_membership queue error: %w", err)
			}
		}
	}()

	return nil
}
