-- +goose NO TRANSACTION
-- +goose Up
CREATE TABLE IF NOT EXISTS raw_batches (
    batch_id UUID,
    device_id UUID,
    park_id UUID,
    group_ids Array(UUID),
    received_at DateTime64(3, 'UTC'),
    payload_raw JSON
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(received_at)
ORDER BY (park_id, device_id, received_at, batch_id);

-- +goose Down
DROP TABLE IF EXISTS raw_batches;
