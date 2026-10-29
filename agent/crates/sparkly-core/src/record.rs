use heapless::{String, Vec};

pub const MAX_TEXT: usize = 256;
pub const MAX_TARGET: usize = 64;
pub const MAX_DEFMT: usize = 512;
pub const MAX_FRAMES: usize = 16;
/// Maximum stored events in a parcel, including any boot transitions.
pub const MAX_BATCH_RECORDS: usize = 8;

/// Must be unique for every boot of a device, including after a factory reset.
pub type BootId = [u8; 16];
/// Producer boot ID followed by a local counter, used only for deduplication.
/// This identifies the parcel, not the boot(s) to which its events belong.
pub type BatchId = [u8; 24];

/// Boot metadata captured at startup and owned by its stored reset event.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Session {
    pub boot_id: BootId,
    pub build_id: String<64>,
    pub firmware_version: String<32>,
    pub agent_version: String<32>,
    pub reset_reason: String<64>,
    pub raw_reset_reason: Option<u32>,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SessionError {
    EmptyBuildId,
    FieldTooLong,
}

impl Session {
    pub fn new(
        boot_id: BootId,
        build_id: &str,
        firmware_version: &str,
        reset_reason: &str,
    ) -> Result<Self, SessionError> {
        if build_id.is_empty() {
            return Err(SessionError::EmptyBuildId);
        }
        Ok(Self {
            boot_id,
            build_id: build_id
                .try_into()
                .map_err(|_| SessionError::FieldTooLong)?,
            firmware_version: firmware_version
                .try_into()
                .map_err(|_| SessionError::FieldTooLong)?,
            agent_version: env!("CARGO_PKG_VERSION")
                .try_into()
                .map_err(|_| SessionError::FieldTooLong)?,
            reset_reason: reset_reason
                .try_into()
                .map_err(|_| SessionError::FieldTooLong)?,
            raw_reset_reason: None,
        })
    }
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum TraceStatus {
    #[default]
    Unsupported,
    Complete,
    Truncated,
    /// Addresses were captured, but the walker cannot certify a complete stack.
    Partial,
}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct StackTrace {
    pub pcs: Vec<u64, MAX_FRAMES>,
    pub status: TraceStatus,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Level {
    Trace,
    Debug,
    Info,
    Warn,
    Error,
}

#[derive(Clone, Debug, PartialEq)]
pub struct TextLog {
    pub level: Level,
    pub target: String<MAX_TARGET>,
    pub message: String<MAX_TEXT>,
    /// True if either the target or formatted message exceeded its capacity.
    pub truncated: bool,
    pub trace: StackTrace,
}

#[derive(Clone, Debug, PartialEq)]
pub struct DefmtLog {
    /// One complete frame, encoded using defmt::Encoder.
    pub frame: Vec<u8, MAX_DEFMT>,
    pub trace: StackTrace,
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub enum MetricValue {
    Gauge(f64),
    /// An increment event, not an absolute cumulative counter value.
    CounterDelta(u64),
}

#[derive(Clone, Debug, PartialEq)]
pub enum Payload {
    /// A real reset, stored once before telemetry from this boot.
    Boot(Session),
    Text(TextLog),
    Defmt(DefmtLog),
    Metric {
        id: u64,
        value: MetricValue,
    },
}

#[derive(Clone, Debug, PartialEq)]
pub struct Record {
    pub monotonic_us: u64,
    pub payload: Payload,
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct Statistics {
    pub accepted: u64,
    pub evicted: u64,
    pub rejected: u64,
}

/// Owns records removed from storage. Keep it until encoding and delivery succeed.
#[derive(Clone, Debug, PartialEq)]
pub struct Batch {
    pub id: BatchId,
    pub records: Vec<Record, MAX_BATCH_RECORDS>,
    /// Loss delta since the previous batch was assembled, counted once on ingest.
    pub dropped_records: u64,
}
