//! Platform-independent telemetry values and extension contracts.
#![no_std]
#![forbid(unsafe_code)]

pub mod record;
pub mod storage;
pub mod transport;

pub use record::*;
pub use storage::{PushOutcome, RamStorage, RecordStorage, StorageFull};
pub use transport::*;

/// Microseconds since boot. Implementations must be monotonic, bounded,
/// interrupt-safe, and must not log. Called under the recorder critical section.
pub trait Clock: Sync {
    fn now_us(&self) -> u64;
}

/// Captures addresses synchronously in the caller's execution context.
/// Implementations must not log, allocate, or access the telemetry recorder.
pub trait Backtrace: Sync {
    fn capture(&self) -> StackTrace;
}

/// Explicit fallback for platforms without a stack walker.
pub struct NoBacktrace;

impl Backtrace for NoBacktrace {
    fn capture(&self) -> StackTrace {
        StackTrace::default()
    }
}
