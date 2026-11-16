use anyhow::{Result, ensure};
use chrono::{DateTime, Utc};
use uuid::Uuid;

use crate::Output;

// Go serializes an uninitialized time.Time as 0001-01-01T00:00:00Z.
const ZERO_RECEIPT_TIME_SECONDS: i64 = -62_135_596_800;

/// Correlation and ownership assigned by analytics, never inferred from the parcel.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct BatchContext {
    batch_id: Uuid,
    device_id: Uuid,
    park_id: Uuid,
    received_at: DateTime<Utc>,
}

impl BatchContext {
    pub fn new(
        batch_id: Uuid,
        device_id: Uuid,
        park_id: Uuid,
        received_at: DateTime<Utc>,
    ) -> Result<Self> {
        ensure!(!batch_id.is_nil(), "batch ID is required");
        ensure!(!device_id.is_nil(), "device ID is required");
        ensure!(!park_id.is_nil(), "park ID is required");
        ensure!(
            received_at.timestamp() != ZERO_RECEIPT_TIME_SECONDS,
            "receipt time is required"
        );
        Ok(Self {
            batch_id,
            device_id,
            park_id,
            received_at,
        })
    }

    pub fn batch_id(&self) -> Uuid {
        self.batch_id
    }
    pub fn device_id(&self) -> Uuid {
        self.device_id
    }
    pub fn park_id(&self) -> Uuid {
        self.park_id
    }
    pub fn received_at(&self) -> DateTime<Utc> {
        self.received_at
    }
}

pub struct SymbolicatedBatch {
    pub context: BatchContext,
    pub output: Output,
    pub symbol_build_ids: Vec<String>,
}
