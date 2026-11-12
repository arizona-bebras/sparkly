-- +goose NO TRANSACTION
-- +goose Up
CREATE TABLE IF NOT EXISTS group_membership (
    device_id UUID,
    group_ids Array(UUID),
    version DateTime64(3, 'UTC')
)
ENGINE = ReplacingMergeTree(version)
ORDER BY device_id;

-- +goose Down
DROP TABLE IF EXISTS group_membership;
