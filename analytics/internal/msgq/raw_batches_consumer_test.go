package msgq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"sparkly/analytics/internal/domain"
	"sparkly/analytics/internal/services"
)

type rawBatchStoreFunc func(context.Context, domain.RawBatch) error

func (f rawBatchStoreFunc) StoreRawBatch(ctx context.Context, batch domain.RawBatch) error {
	return f(ctx, batch)
}

func validParcelMessage() ParcelMessage {
	return ParcelMessage{
		SchemaVersion: 1, BatchID: uuid.Must(uuid.NewV7()),
		DeviceID: uuid.New(), ParkID: uuid.New(), ReceivedAt: time.Now().UTC(),
		Payload: json.RawMessage(`{"batchId":"AQ==","events":[{"monotonicUs":"18446744073709551615","metric":{"metricId":"18446744073709551615","counterDelta":"18446744073709551615"}}]}`),
	}
}

func TestRawConsumerAcknowledgesOnlyAfterStorage(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "storage failure"}[fail], func(t *testing.T) {
			want := validParcelMessage()
			body, err := json.Marshal(want)
			if err != nil {
				t.Fatal(err)
			}
			ack := &mockAcknowledger{}
			calls := 0
			consumer := &RawBatchesConsumer{persister: services.NewRawBatchPersistence(rawBatchStoreFunc(func(ctx context.Context, got domain.RawBatch) error {
				calls++
				if len(ack.acks) != 0 || len(ack.nacks) != 0 {
					t.Fatal("delivery acknowledged before storage")
				}
				if got.ID != want.BatchID || got.Identity.DeviceID() != want.DeviceID || got.Identity.ParkID() != want.ParkID || !got.ReceivedAt.Equal(want.ReceivedAt) || !bytes.Equal(got.Payload, want.Payload) {
					t.Fatalf("stored batch differs: %+v", got)
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("write has no timeout")
				}
				if fail {
					return errors.New("ClickHouse unavailable")
				}
				return nil
			}))}
			retry, err := consumer.handleDelivery(context.Background(), amqp.Delivery{Acknowledger: ack, DeliveryTag: 7, MessageId: want.BatchID.String(), ContentType: "application/json", Body: body})
			if err != nil || retry != fail || calls != 1 {
				t.Fatalf("retry=%v err=%v calls=%d", retry, err, calls)
			}
			if fail {
				if len(ack.acks) != 0 || len(ack.nacks) != 1 || ack.nacks[0] != (nackRecord{tag: 7, requeue: true}) {
					t.Fatalf("acks=%v nacks=%v", ack.acks, ack.nacks)
				}
			} else if len(ack.acks) != 1 || ack.acks[0] != 7 || len(ack.nacks) != 0 {
				t.Fatalf("acks=%v nacks=%v", ack.acks, ack.nacks)
			}
		})
	}
}

func TestRawConsumerRejectsInvalidEnvelope(t *testing.T) {
	for _, name := range []string{"content type", "invalid JSON", "trailing JSON", "version", "batch ID", "device ID", "park ID", "receipt time", "missing payload", "null payload", "array payload", "invalid protobuf JSON", "mismatched message ID"} {
		t.Run(name, func(t *testing.T) {
			message := validParcelMessage()
			switch name {
			case "version":
				message.SchemaVersion = 2
			case "batch ID":
				message.BatchID = uuid.Nil
			case "device ID":
				message.DeviceID = uuid.Nil
			case "park ID":
				message.ParkID = uuid.Nil
			case "receipt time":
				message.ReceivedAt = time.Time{}
			case "missing payload":
				message.Payload = nil
			case "null payload":
				message.Payload = json.RawMessage(`null`)
			case "array payload":
				message.Payload = json.RawMessage(`[]`)
			case "invalid protobuf JSON":
				message.Payload = json.RawMessage(`{"events":"bad"}`)
			}
			body, err := json.Marshal(message)
			if err != nil {
				t.Fatal(err)
			}
			ack := &mockAcknowledger{}
			delivery := amqp.Delivery{Acknowledger: ack, DeliveryTag: 1, ContentType: "application/json", Body: body}
			switch name {
			case "content type":
				delivery.ContentType = "text/plain"
			case "invalid JSON":
				delivery.Body = []byte("{")
			case "trailing JSON":
				delivery.Body = append(body, []byte(` {}`)...)
			case "mismatched message ID":
				delivery.MessageId = uuid.NewString()
			}
			consumer := &RawBatchesConsumer{persister: services.NewRawBatchPersistence(rawBatchStoreFunc(func(context.Context, domain.RawBatch) error { t.Fatal("invalid batch reached storage"); return nil }))}
			retry, err := consumer.handleDelivery(context.Background(), delivery)
			if err != nil || retry || len(ack.acks) != 0 || len(ack.nacks) != 1 || ack.nacks[0].requeue {
				t.Fatalf("retry=%v err=%v acks=%v nacks=%v", retry, err, ack.acks, ack.nacks)
			}
		})
	}
}
