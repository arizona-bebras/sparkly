package repos

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"

	"sparkly/analytics/internal/domain"
)

type RawBatchesRepo struct {
	db *sql.DB
}

func NewRawBatchesRepo(db *sql.DB) *RawBatchesRepo {
	return &RawBatchesRepo{db: db}
}

// StoreRawBatch queues the protobuf JSON representation of a device parcel.
// Device and park ownership comes from the authenticated gateway identity.
func (r *RawBatchesRepo) StoreRawBatch(ctx context.Context, batch domain.RawBatch) error {
	if err := batch.Identity.Validate(); err != nil {
		return fmt.Errorf("store raw batch: %w", err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate batch ID: %w", err)
	}
	// Acknowledge once ClickHouse accepts the batch into its async buffer.
	ctx = clickhouse.Context(ctx, clickhouse.WithAsync(false))
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO raw_batches (batch_id, device_id, park_id, group_ids, received_at, payload_raw)
		VALUES (toUUID(?), toUUID(?), toUUID(?), CAST([], 'Array(UUID)'), fromUnixTimestamp64Milli(?, 'UTC'), CAST(? AS JSON))`,
		id.String(), batch.Identity.DeviceID().String(), batch.Identity.ParkID().String(), batch.ReceivedAt.UnixMilli(), string(batch.Payload))
	if err != nil {
		return fmt.Errorf("insert raw batch: %w", err)
	}
	return nil
}
