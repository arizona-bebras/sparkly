package repos

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
)

type RawBatchesRepo struct {
	db *sql.DB
}

func NewRawBatchesRepo(db *sql.DB) *RawBatchesRepo {
	return &RawBatchesRepo{db: db}
}

// StoreRawBatch queues the protobuf JSON representation of a device parcel.
// Device and park ownership will be populated once device authentication exists.
func (r *RawBatchesRepo) StoreRawBatch(ctx context.Context, payload []byte, receivedAt time.Time) error {
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate batch ID: %w", err)
	}
	// Acknowledge once ClickHouse accepts the batch into its async buffer.
	ctx = clickhouse.Context(ctx, clickhouse.WithAsync(false))
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO raw_batches (batch_id, device_id, park_id, group_ids, received_at, payload_raw)
		VALUES (toUUID(?), toUUID(?), toUUID(?), CAST([], 'Array(UUID)'), fromUnixTimestamp64Milli(?, 'UTC'), CAST(? AS JSON))`,
		id.String(), uuid.Nil.String(), uuid.Nil.String(), receivedAt.UnixMilli(), string(payload))
	if err != nil {
		return fmt.Errorf("insert raw batch: %w", err)
	}
	return nil
}
