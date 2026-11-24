-- +goose NO TRANSACTION
-- +goose Up
CREATE TABLE IF NOT EXISTS logs (
    park_id UUID,
    device_id UUID,
    batch_id UUID,
    event_index UInt64,
    received_at DateTime64(9, 'UTC'),
    boot_id Nullable(String),
    build_id Nullable(String),
    monotonic_us UInt64,
    diagnostics Array(String),
    level Nullable(String),
    message Nullable(String),
    module Nullable(String),
    frames Array(Tuple(pc UInt64, function Nullable(String), file Nullable(String), line Nullable(UInt32))),
    truncated Bool,
    trace_status String,
    defmt_raw_hex Nullable(String)
)
ENGINE = ReplacingMergeTree
ORDER BY (park_id, batch_id, event_index);

CREATE TABLE IF NOT EXISTS metrics (
    park_id UUID,
    device_id UUID,
    batch_id UUID,
    event_index UInt64,
    received_at DateTime64(9, 'UTC'),
    boot_id Nullable(String),
    build_id Nullable(String),
    monotonic_us UInt64,
    diagnostics Array(String),
    metric_id UInt64,
    name Nullable(String),
    kind Nullable(String),
    value Nullable(Float64),
    counter_delta Nullable(UInt64),
    decode_status String
)
ENGINE = ReplacingMergeTree
ORDER BY (park_id, batch_id, event_index);

CREATE TABLE IF NOT EXISTS boots (
    park_id UUID,
    device_id UUID,
    batch_id UUID,
    event_index UInt64,
    received_at DateTime64(9, 'UTC'),
    boot_id Nullable(String),
    build_id Nullable(String),
    monotonic_us UInt64,
    diagnostics Array(String),
    reset_reason String,
    raw_reset_reason Nullable(UInt32),
    firmware_version String,
    agent_version String
)
ENGINE = ReplacingMergeTree
ORDER BY (park_id, batch_id, event_index);

CREATE TABLE IF NOT EXISTS crashes (
    park_id UUID,
    device_id UUID,
    batch_id UUID,
    event_index UInt64,
    received_at DateTime64(9, 'UTC'),
    boot_id Nullable(String),
    build_id Nullable(String),
    monotonic_us UInt64,
    diagnostics Array(String),
    frames Array(Tuple(pc UInt64, function Nullable(String), file Nullable(String), line Nullable(UInt32))),
    sp Nullable(UInt64),
    reason Nullable(String),
    stack_hex Nullable(String),
    registers Map(String, UInt64)
)
ENGINE = ReplacingMergeTree
ORDER BY (park_id, batch_id, event_index);

CREATE VIEW IF NOT EXISTS logs_deduplicated AS SELECT * FROM logs FINAL;
CREATE VIEW IF NOT EXISTS metrics_deduplicated AS SELECT * FROM metrics FINAL;
CREATE VIEW IF NOT EXISTS boots_deduplicated AS SELECT * FROM boots FINAL;
CREATE VIEW IF NOT EXISTS crashes_deduplicated AS SELECT * FROM crashes FINAL;

-- +goose Down
DROP VIEW IF EXISTS crashes_deduplicated;
DROP VIEW IF EXISTS boots_deduplicated;
DROP VIEW IF EXISTS metrics_deduplicated;
DROP VIEW IF EXISTS logs_deduplicated;
DROP TABLE IF EXISTS crashes;
DROP TABLE IF EXISTS boots;
DROP TABLE IF EXISTS metrics;
DROP TABLE IF EXISTS logs;
