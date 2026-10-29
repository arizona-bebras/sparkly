use crate::{BuildIdError, build_id};
use core::fmt::{self, Write};
use heapless::String;
use sparkly_agent::{
    Backtrace, BootId, Clock, RecordStorage, Recorder, Session, SessionError, StackTrace,
    TraceStatus,
};

pub struct EspClock;
impl Clock for EspClock {
    fn now_us(&self) -> u64 {
        esp_hal::time::Instant::now()
            .duration_since_epoch()
            .as_micros()
    }
}

pub struct EspBacktrace;
impl Backtrace for EspBacktrace {
    #[inline(never)]
    fn capture(&self) -> StackTrace {
        let captured = esp_backtrace::Backtrace::capture();
        let mut trace = StackTrace {
            status: TraceStatus::Partial,
            ..Default::default()
        };
        for frame in captured.frames() {
            if trace.pcs.push(frame.program_counter() as u64).is_err() {
                trace.status = TraceStatus::Truncated;
                break;
            }
        }
        // The dependency does not distinguish a terminated walk from an invalid
        // stack. At its configured bound, conservatively report truncation.
        let limit = option_env!("ESP_BACKTRACE_CONFIG_BACKTRACE_FRAMES")
            .unwrap_or("10")
            .parse::<usize>()
            .unwrap_or(10);
        if captured.frames().len() >= limit {
            trace.status = TraceStatus::Truncated;
        }
        trace
    }
}

pub type EspRecorder<S> = Recorder<S, EspClock, EspBacktrace>;

#[derive(Debug)]
pub enum InitErrorReason<E> {
    BuildId(BuildIdError),
    Session(SessionError),
    Storage(E),
}

/// Retains restored history on every initialization failure.
pub struct InitError<S: RecordStorage> {
    pub storage: S,
    pub reason: InitErrorReason<S::Error>,
}

impl<S: RecordStorage> fmt::Debug for InitError<S>
where
    S::Error: fmt::Debug,
{
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        out.debug_struct("InitError")
            .field("reason", &self.reason)
            .finish_non_exhaustive()
    }
}

/// Call after esp_hal::init. Supply a fresh boot ID from a real entropy source or
/// a durable device-specific counter scheme; this function takes no peripherals.
pub fn recorder<S: RecordStorage>(
    storage: S,
    boot_id: BootId,
    firmware_version: &str,
) -> Result<EspRecorder<S>, InitError<S>> {
    let build_id = match build_id() {
        Ok(id) => id,
        Err(error) => {
            return Err(InitError {
                storage,
                reason: InitErrorReason::BuildId(error),
            });
        }
    };
    let reason = esp_hal::system::reset_reason();
    let mut description = String::<64>::new();
    match reason {
        Some(reason) => {
            let _ = write!(description, "{reason:?}");
        }
        None => {
            let _ = description.push_str("unknown");
        }
    }
    let mut session = match Session::new(boot_id, &build_id, firmware_version, &description) {
        Ok(session) => session,
        Err(error) => {
            return Err(InitError {
                storage,
                reason: InitErrorReason::Session(error),
            });
        }
    };
    session.raw_reset_reason = reason.map(|reason| reason as u32);
    Recorder::new(session, storage, EspClock, EspBacktrace).map_err(|error| InitError {
        storage: error.storage,
        reason: InitErrorReason::Storage(error.error),
    })
}
