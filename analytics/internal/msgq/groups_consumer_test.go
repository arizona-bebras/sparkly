package msgq

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"sparkly/analytics/internal/domain"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type mockGroupMembershipService struct {
	mu    sync.Mutex
	calls []GroupMembershipMsg
	err   error
}

func (m *mockGroupMembershipService) AssociateGroupsToDevice(ctx context.Context, deviceIDStr string, groupIDsStr []string, versionStr string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.calls = append(m.calls, GroupMembershipMsg{
		DeviceID: deviceIDStr,
		GroupIDs: groupIDsStr,
		Version:  versionStr,
	})
	return nil
}

type mockAcknowledger struct {
	mu    sync.Mutex
	acks  []uint64
	nacks []nackRecord
}

type nackRecord struct {
	tag     uint64
	requeue bool
}

func (a *mockAcknowledger) Ack(tag uint64, multiple bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.acks = append(a.acks, tag)
	return nil
}

func (a *mockAcknowledger) Nack(tag uint64, multiple, requeue bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.nacks = append(a.nacks, nackRecord{tag: tag, requeue: requeue})
	return nil
}

func (a *mockAcknowledger) Reject(tag uint64, requeue bool) error {
	return nil
}

func TestDeviceGroupsQueueConsumerConsumeMultipleMessages(t *testing.T) {
	svc := &mockGroupMembershipService{}
	consumer := &DeviceGroupsQueueConsumer{
		queueName: "group_membership",
		service:   svc,
	}

	ack := &mockAcknowledger{}
	msgs := make(chan amqp.Delivery, 2)

	for i := range 2 {
		body, err := json.Marshal(GroupMembershipMsg{
			ID:       uuid.New().String(),
			DeviceID: uuid.New().String(),
			Version:  time.Now().UTC().Format(time.RFC3339Nano),
			GroupIDs: []string{uuid.New().String()},
		})
		if err != nil {
			t.Fatal(err)
		}
		msgs <- amqp.Delivery{
			Acknowledger: ack,
			DeliveryTag:  uint64(i + 1),
			ContentType:  "application/json",
			Body:         body,
		}
	}
	close(msgs)

	consumer.consumeLoop(context.Background(), msgs)

	if len(svc.calls) != 2 {
		t.Fatalf("expected 2 processed messages, got %d", len(svc.calls))
	}
	if len(ack.acks) != 2 {
		t.Fatalf("expected 2 acks, got %d", len(ack.acks))
	}
	if len(ack.nacks) != 0 {
		t.Fatalf("expected 0 nacks, got %d", len(ack.nacks))
	}
}

func TestDeviceGroupsQueueConsumerNackBehavior(t *testing.T) {
	svc := &mockGroupMembershipService{}
	consumer := &DeviceGroupsQueueConsumer{
		queueName: "group_membership",
		service:   svc,
	}

	ack := &mockAcknowledger{}

	requeue := consumer.handleDelivery(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		DeliveryTag:  1,
		ContentType:  "text/plain",
		Body:         []byte(`{}`),
	})
	if requeue {
		t.Fatal("expected wrong content type not to requeue")
	}

	requeue = consumer.handleDelivery(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		DeliveryTag:  2,
		ContentType:  "application/json",
		Body:         []byte(`not-json`),
	})
	if requeue {
		t.Fatal("expected invalid json not to requeue")
	}

	svc.err = domain.ErrInvalidBrokerMsg
	requeue = consumer.handleDelivery(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		DeliveryTag:  3,
		ContentType:  "application/json",
		Body:         []byte(`{"device_id":"bad"}`),
	})
	if requeue {
		t.Fatal("expected invalid broker msg not to requeue")
	}

	svc.err = errors.New("clickhouse unavailable")
	requeue = consumer.handleDelivery(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		DeliveryTag:  4,
		ContentType:  "application/json",
		Body:         []byte(`{"device_id":"00000000-0000-0000-0000-000000000001"}`),
	})
	if !requeue {
		t.Fatal("expected transient error to requeue")
	}

	if len(ack.acks) != 0 {
		t.Fatalf("expected 0 acks, got %d", len(ack.acks))
	}
	expectedNacks := []nackRecord{
		{tag: 1, requeue: false},
		{tag: 2, requeue: false},
		{tag: 3, requeue: false},
		{tag: 4, requeue: true},
	}
	if len(ack.nacks) != len(expectedNacks) {
		t.Fatalf("expected %d nacks, got %d", len(expectedNacks), len(ack.nacks))
	}
	for i, want := range expectedNacks {
		if ack.nacks[i] != want {
			t.Fatalf("nack[%d] = %+v, want %+v", i, ack.nacks[i], want)
		}
	}
}
