package msgq

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"sparkly/analytics/internal/domain"
	"sparkly/analytics/internal/parcel"
)

const parcelSchemaVersion = 1

var errInvalidParcelMessage = errors.New("invalid parcel message")

// ParcelMessage is shared by raw persistence and symbolication consumers.
// Payload follows parcel.proto's protobuf JSON mapping, not ordinary Go JSON.
type ParcelMessage struct {
	SchemaVersion int             `json:"schema_version"`
	BatchID       uuid.UUID       `json:"batch_id"`
	DeviceID      uuid.UUID       `json:"device_id"`
	ParkID        uuid.UUID       `json:"park_id"`
	ReceivedAt    time.Time       `json:"received_at"`
	Payload       json.RawMessage `json:"payload"`
}

func encodeParcelMessage(batch domain.RawBatch) ([]byte, error) {
	if err := batch.Validate(); err != nil {
		return nil, err
	}
	body, err := json.Marshal(ParcelMessage{
		SchemaVersion: parcelSchemaVersion, BatchID: batch.ID,
		DeviceID: batch.Identity.DeviceID(), ParkID: batch.Identity.ParkID(),
		ReceivedAt: batch.ReceivedAt.UTC(), Payload: batch.Payload,
	})
	if err != nil {
		return nil, fmt.Errorf("encode parcel envelope: %w", err)
	}
	return body, nil
}

func decodeRawBatch(msg amqp.Delivery) (domain.RawBatch, error) {
	invalid := func(reason string) (domain.RawBatch, error) {
		return domain.RawBatch{}, fmt.Errorf("%w: %s", errInvalidParcelMessage, reason)
	}
	if msg.ContentType != "application/json" {
		return invalid("expected application/json")
	}
	var envelope ParcelMessage
	if err := json.Unmarshal(msg.Body, &envelope); err != nil {
		return invalid(err.Error())
	}
	if envelope.SchemaVersion != parcelSchemaVersion {
		return invalid("unsupported schema version")
	}
	if msg.MessageId != "" && msg.MessageId != envelope.BatchID.String() {
		return invalid("AMQP message ID differs from batch ID")
	}
	identity, err := domain.NewDeviceIdentity(envelope.DeviceID, envelope.ParkID)
	if err != nil {
		return invalid(err.Error())
	}
	if err := parcel.ValidateJSON(envelope.Payload); err != nil {
		return invalid(err.Error())
	}
	batch := domain.RawBatch{ID: envelope.BatchID, Identity: identity, ReceivedAt: envelope.ReceivedAt.UTC(), Payload: envelope.Payload}
	if err := batch.Validate(); err != nil {
		return invalid(err.Error())
	}
	return batch, nil
}
