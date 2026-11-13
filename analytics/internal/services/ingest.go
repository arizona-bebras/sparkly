package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"sparkly/analytics/internal/domain"
)

var (
	ErrInvalidParcel     = errors.New("invalid protobuf parcel")
	ErrIngestUnavailable = errors.New("parcel publication unavailable")
)

type ParcelDecoder interface {
	DecodeParcel([]byte) ([]byte, error)
}

type RawBatchPublisher interface {
	PublishRawBatch(context.Context, domain.RawBatch) error
}

// IngestService prepares one batch and publishes it to both downstream paths.
type IngestService struct {
	decoder   ParcelDecoder
	publisher RawBatchPublisher
}

func NewIngestService(decoder ParcelDecoder, publisher RawBatchPublisher) *IngestService {
	return &IngestService{decoder: decoder, publisher: publisher}
}

func (s *IngestService) IngestParcel(ctx context.Context, identity domain.DeviceIdentity, body []byte, receivedAt time.Time) error {
	if err := identity.Validate(); err != nil {
		return err
	}
	payload, err := s.decoder.DecodeParcel(body)
	if err != nil {
		return fmt.Errorf("decode parcel: %w", err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("assign batch ID: %w", err)
	}
	batch := domain.RawBatch{ID: id, Identity: identity, ReceivedAt: receivedAt.UTC(), Payload: payload}
	if err := batch.Validate(); err != nil {
		return err
	}
	if err := s.publisher.PublishRawBatch(ctx, batch); err != nil {
		return fmt.Errorf("%w: %w", ErrIngestUnavailable, err)
	}
	return nil
}
