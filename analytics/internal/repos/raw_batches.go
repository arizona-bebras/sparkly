package repos

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"

	"sparkly/analytics/internal/domain"
)

type RawBatchesRepo struct {
	db *sql.DB
}

func NewRawBatchesRepo(db *sql.DB) *RawBatchesRepo {
	return &RawBatchesRepo{db: db}
}

// StoreRawBatch persists a parcel using its ingest-assigned batch ID.
// Device and park ownership comes from the authenticated gateway identity.
func (r *RawBatchesRepo) StoreRawBatch(ctx context.Context, batch domain.RawBatch) error {
	if err := batch.Validate(); err != nil {
		return fmt.Errorf("store raw batch: %w", err)
	}
	// Wait for the async buffer to flush before acknowledging the broker delivery.
	ctx = clickhouse.Context(ctx, clickhouse.WithAsync(true))
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO raw_batches (batch_id, device_id, park_id, group_ids, received_at, payload_raw)
		VALUES (toUUID(?), toUUID(?), toUUID(?), CAST([], 'Array(UUID)'), fromUnixTimestamp64Milli(?, 'UTC'), CAST(? AS JSON))`,
		batch.ID.String(), batch.Identity.DeviceID().String(), batch.Identity.ParkID().String(), batch.ReceivedAt.UnixMilli(), string(batch.Payload))
	if err != nil {
		return fmt.Errorf("insert raw batch: %w", err)
	}
	return nil
}
