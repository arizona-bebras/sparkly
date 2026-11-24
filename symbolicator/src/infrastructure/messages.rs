use anyhow::{Context, Result, ensure};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::value::RawValue;
use uuid::Uuid;

use crate::{
    Output,
    application::SymbolicationTask,
    domain::{BatchContext, SymbolicatedBatch},
    parse_parcel_json,
};

pub const MAX_REQUEST_BYTES: usize = 8 << 20;
const SCHEMA_VERSION: u32 = 1;
const RESULT_VERSION: u32 = 5;

// Mirrors analytics/internal/msgq/parcel_message.go. Keep transport DTOs out of the domain.
#[derive(Deserialize)]
struct RequestMessage {
    schema_version: u32,
    batch_id: Uuid,
    device_id: Uuid,
    park_id: Uuid,
    received_at: DateTime<Utc>,
    payload: Box<RawValue>,
}

#[derive(Serialize)]
struct ResultMessage<'a> {
    schema_version: u32,
    result_version: u32,
    batch_id: Uuid,
    device_id: Uuid,
    park_id: Uuid,
    received_at: DateTime<Utc>,
    result: &'a Output,
}

pub fn decode_request(
    body: &[u8],
    content_type: Option<&str>,
    message_id: Option<&str>,
) -> Result<SymbolicationTask> {
    ensure!(
        content_type == Some("application/json"),
        "expected application/json"
    );
    ensure!(body.len() <= MAX_REQUEST_BYTES, "request exceeds 8 MiB");
    let message: RequestMessage = serde_json::from_slice(body).context("decode parcel envelope")?;
    ensure!(
        message.schema_version == SCHEMA_VERSION,
        "unsupported parcel schema version"
    );
    if let Some(id) = message_id.filter(|id| !id.is_empty()) {
        ensure!(
            id == message.batch_id.to_string(),
            "AMQP message ID differs from batch ID"
        );
    }
    let context = BatchContext::new(
        message.batch_id,
        message.device_id,
        message.park_id,
        message.received_at,
    )?;
    ensure!(
        message.payload.get().starts_with('{'),
        "payload must be a parcel object"
    );
    let parcel = parse_parcel_json(message.payload.get().as_bytes())
        .context("decode protobuf JSON parcel")?;
    Ok(SymbolicationTask { context, parcel })
}

pub fn encode_result(batch: &SymbolicatedBatch) -> Result<Vec<u8>> {
    let context = &batch.context;
    serde_json::to_vec(&ResultMessage {
        schema_version: SCHEMA_VERSION,
        result_version: RESULT_VERSION,
        batch_id: context.batch_id(),
        device_id: context.device_id(),
        park_id: context.park_id(),
        received_at: context.received_at(),
        result: &batch.output,
    })
    .context("encode symbolication result")
}
