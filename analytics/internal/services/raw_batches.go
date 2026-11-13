package services

import (
	"context"
	"fmt"
	"time"

	"sparkly/analytics/internal/domain"
)

const rawBatchWriteTimeout = 30 * time.Second

type RawBatchRepository interface {
	StoreRawBatch(context.Context, domain.RawBatch) error
}

type RawBatchPersistence struct {
	repository RawBatchRepository
}

func NewRawBatchPersistence(repository RawBatchRepository) *RawBatchPersistence {
	return &RawBatchPersistence{repository: repository}
}

func (s *RawBatchPersistence) PersistRawBatch(ctx context.Context, batch domain.RawBatch) error {
	if err := batch.Validate(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, rawBatchWriteTimeout)
	defer cancel()
	if err := s.repository.StoreRawBatch(ctx, batch); err != nil {
		return fmt.Errorf("persist raw batch %s: %w", batch.ID, err)
	}
	return nil
}
