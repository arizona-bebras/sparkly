package services

import (
	"context"
	"errors"
	"testing"

	"sparkly/analytics/internal/domain"
)

type rawBatchRepositoryFunc func(context.Context, domain.RawBatch) error

func (f rawBatchRepositoryFunc) StoreRawBatch(ctx context.Context, batch domain.RawBatch) error {
	return f(ctx, batch)
}

func TestRawPersistenceRejectsInvalidBatchBeforeRepository(t *testing.T) {
	service := NewRawBatchPersistence(rawBatchRepositoryFunc(func(context.Context, domain.RawBatch) error { t.Fatal("invalid batch reached repository"); return nil }))
	if err := service.PersistRawBatch(context.Background(), domain.RawBatch{}); !errors.Is(err, domain.ErrInvalidRawBatch) {
		t.Fatalf("err=%v", err)
	}
}
