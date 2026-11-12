package msgq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sparkly/analytics/internal/domain"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type GroupMembershipService interface {
	AssociateGroupsToDevice(ctx context.Context, deviceIDStr string, groupIDsStr []string, versionStr string) error
}

type DeviceGroupsQueueConsumer struct {
	queueName string
	ch        *amqp.Channel
	service   GroupMembershipService
}

func NewDeviceGroupsQueueConsumer(broker *MessageBroker, service GroupMembershipService) (*DeviceGroupsQueueConsumer, error) {
	if service == nil {
		return nil, fmt.Errorf("group membership service cannot be nil")
	}

	qName := "group_membership"
	_, err := broker.ch.QueueDeclare(qName, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare group_membership queue: %w", err)
	}

	return &DeviceGroupsQueueConsumer{
		queueName: qName,
		ch:        broker.ch,
		service:   service,
	}, nil
}

func (c *DeviceGroupsQueueConsumer) processMsg(ctx context.Context, brokerMsg amqp.Delivery) error {
	if brokerMsg.ContentType != "application/json" {
		return fmt.Errorf("message broker, group_membership queue error: %w", domain.ErrWrongBrokerMsgContentType)
	}

	r := bytes.NewReader(brokerMsg.Body)

	var msgParams GroupMembershipMsg
	if err := json.NewDecoder(r).Decode(&msgParams); err != nil {
		return fmt.Errorf("failed to decode broker message: %w: %w", domain.ErrInvalidBrokerMsg, err)
	}

	err := c.service.AssociateGroupsToDevice(ctx, msgParams.DeviceID, msgParams.GroupIDs, msgParams.Version)
	if err != nil {
		return fmt.Errorf("failed to associate groups to device: %w", err)
	}

	return nil
}

func (c *DeviceGroupsQueueConsumer) handleDelivery(ctx context.Context, brokerMsg amqp.Delivery) bool {
	if err := c.processMsg(ctx, brokerMsg); err != nil {
		log.Printf("message broker, group_membership queue error: %v", err)
		requeue := !errors.Is(err, domain.ErrWrongBrokerMsgContentType) && !errors.Is(err, domain.ErrInvalidBrokerMsg)
		if nackErr := brokerMsg.Nack(false, requeue); nackErr != nil {
			log.Printf("message broker, failed to nack message: %v", nackErr)
		}
		return requeue
	}

	if ackErr := brokerMsg.Ack(false); ackErr != nil {
		log.Printf("message broker, failed to ack message: %v", ackErr)
	}
	return false
}

func (c *DeviceGroupsQueueConsumer) consumeLoop(ctx context.Context, msgs <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			if c.ch != nil {
				c.ch.Close()
			}
			return
		case brokerMsg, ok := <-msgs:
			if !ok {
				return
			}
			if c.handleDelivery(ctx, brokerMsg) {
				select {
				case <-ctx.Done():
					if c.ch != nil {
						c.ch.Close()
					}
					return
				case <-time.After(time.Second):
				}
			}
		}
	}
}

func (c *DeviceGroupsQueueConsumer) Listen(ctx context.Context) error {
	msgs, err := c.ch.Consume(c.queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to register rabbitmq group_membership consumer: %w", err)
	}

	go c.consumeLoop(ctx, msgs)

	return nil
}
