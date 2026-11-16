package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidRawBatch = errors.New("invalid raw batch")

// RawBatch contains the parcel's JSON representation and server-assigned metadata.
// Identity comes from the authenticated gateway, never from the parcel payload.
type RawBatch struct {
	ID         uuid.UUID
	Identity   DeviceIdentity
	Payload    []byte
	ReceivedAt time.Time
}

// Validate enforces batch invariants independently of HTTP, AMQP, or ClickHouse.
// Payload encoding is validated by the parcel adapter at the system boundary.
func (b RawBatch) Validate() error {
	if err := b.Identity.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidRawBatch, err)
	}
	if b.ID == uuid.Nil {
		return fmt.Errorf("%w: batch ID is required", ErrInvalidRawBatch)
	}
	if b.ReceivedAt.IsZero() {
		return fmt.Errorf("%w: receipt time is required", ErrInvalidRawBatch)
	}
	if len(b.Payload) == 0 {
		return fmt.Errorf("%w: payload is required", ErrInvalidRawBatch)
	}
	return nil
}
