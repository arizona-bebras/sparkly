use core::{
    cell::{Cell, RefCell},
    fmt::{self, Write},
};
use critical_section::Mutex;
use heapless::{String, Vec};
use sparkly_core::*;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum CaptureError {
    NotInstalled,
    Busy,
    Storage,
    OversizedFrame,
    BatchIdsExhausted,
}

/// Startup could not append the new reset (for example, the byte budget is too
/// small). The existing stream is returned intact for recovery.
pub struct RecorderInitError<S: RecordStorage> {
    pub storage: S,
    pub error: S::Error,
}

impl<S: RecordStorage> fmt::Debug for RecorderInitError<S>
where
    S::Error: fmt::Debug,
{
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        out.debug_struct("RecorderInitError")
            .field("error", &self.error)
            .finish_non_exhaustive()
    }
}

/// Object-safe capture interface shared by logging facades and metric handles.
pub trait Sink: Sync {
    fn text(
        &self,
        level: Level,
        target: &str,
        message: fmt::Arguments<'_>,
    ) -> Result<(), CaptureError>;
    fn defmt(&self, frame: &[u8]) -> Result<(), CaptureError>;
    fn metric(&self, id: u64, value: MetricValue) -> Result<(), CaptureError>;
    fn reject(&self);
}

/// Only one exporter may consume a source. Capture may continue on other contexts.
pub trait BatchSource: Sync {
    fn take_batch(&self) -> Result<Option<Batch>, CaptureError>;
    fn now_us(&self) -> u64;
}

struct State<S> {
    storage: S,
    next_batch: u64,
    statistics: Statistics,
    reported_losses: u64,
}

/// Owns storage and platform services. No runtime, transport or global registration
/// is required. The application's critical-section provider must support SMP.
pub struct Recorder<S, C, B> {
    batch_namespace: BootId,
    clock: C,
    backtrace: B,
    state: Mutex<RefCell<State<S>>>,
    capturing_log: Mutex<Cell<bool>>,
}

impl<S: RecordStorage, C: Clock, B: Backtrace> Recorder<S, C, B> {
    /// Appends exactly one Boot after any restored history, before accepting
    /// telemetry. Capacity pressure may evict older records, including Boots.
    pub fn new(
        session: Session,
        mut storage: S,
        clock: C,
        backtrace: B,
    ) -> Result<Self, RecorderInitError<S>> {
        let batch_namespace = session.boot_id;
        let outcome = match storage.push(Record {
            monotonic_us: clock.now_us(),
            payload: Payload::Boot(session),
        }) {
            Ok(outcome) => outcome,
            Err(error) => return Err(RecorderInitError { storage, error }),
        };
        Ok(Self {
            batch_namespace,
            clock,
            backtrace,
            state: Mutex::new(RefCell::new(State {
                storage,
                next_batch: 0,
                statistics: Statistics {
                    evicted: outcome.evicted,
                    ..Statistics::default()
                },
                reported_losses: 0,
            })),
            capturing_log: Mutex::new(Cell::new(false)),
        })
    }

    /// Returns queued records in stream order. A separate pending exporter batch
    /// must also be preserved by applications that implement durable recovery.
    pub fn into_storage(self) -> S {
        self.state.into_inner().into_inner().storage
    }

    pub fn statistics(&self) -> Statistics {
        critical_section::with(|cs| self.state.borrow(cs).borrow().statistics)
    }

    fn push(&self, payload: Payload) -> Result<(), CaptureError> {
        critical_section::with(|cs| {
            let mut state = self.state.borrow(cs).borrow_mut();
            match state.storage.push(Record {
                // Sampling at commit preserves timestamp order even if another
                // context records a metric while a raw log is being formatted.
                monotonic_us: self.clock.now_us(),
                payload,
            }) {
                Ok(outcome) => {
                    state.statistics.accepted = state.statistics.accepted.saturating_add(1);
                    state.statistics.evicted =
                        state.statistics.evicted.saturating_add(outcome.evicted);
                    Ok(())
                }
                Err(_) => {
                    state.statistics.rejected = state.statistics.rejected.saturating_add(1);
                    Err(CaptureError::Storage)
                }
            }
        })
    }

    fn enter_log(&self) -> Result<LogGuard<'_>, CaptureError> {
        let busy = critical_section::with(|cs| self.capturing_log.borrow(cs).replace(true));
        if busy {
            self.reject();
            return Err(CaptureError::Busy);
        }
        Ok(LogGuard(&self.capturing_log))
    }
}

struct LogGuard<'a>(&'a Mutex<Cell<bool>>);
impl Drop for LogGuard<'_> {
    fn drop(&mut self) {
        critical_section::with(|cs| self.0.borrow(cs).set(false));
    }
}

impl<S: RecordStorage, C: Clock, B: Backtrace> Sink for Recorder<S, C, B> {
    fn text(
        &self,
        level: Level,
        target: &str,
        message: fmt::Arguments<'_>,
    ) -> Result<(), CaptureError> {
        // Formatting may itself log. Drop nested/concurrent raw logs rather than
        // deadlock or hold a critical section while executing user Display code.
        let _guard = self.enter_log()?;
        let mut text = BoundedText::<MAX_TEXT>::new();
        let format_failed = text.write_fmt(message).is_err();
        let mut target_text = BoundedText::<MAX_TARGET>::new();
        let _ = target_text.write_str(target);
        let trace = self.backtrace.capture();
        self.push(Payload::Text(TextLog {
            level,
            target: target_text.text,
            message: text.text,
            truncated: text.truncated || target_text.truncated || format_failed,
            trace,
        }))
    }

    fn defmt(&self, frame: &[u8]) -> Result<(), CaptureError> {
        let _guard = self.enter_log()?;
        let Ok(frame) = Vec::from_slice(frame) else {
            self.reject();
            return Err(CaptureError::OversizedFrame);
        };
        self.push(Payload::Defmt(DefmtLog {
            frame,
            trace: self.backtrace.capture(),
        }))
    }

    fn metric(&self, id: u64, value: MetricValue) -> Result<(), CaptureError> {
        self.push(Payload::Metric { id, value })
    }

    fn reject(&self) {
        critical_section::with(|cs| {
            let mut state = self.state.borrow(cs).borrow_mut();
            state.statistics.rejected = state.statistics.rejected.saturating_add(1);
        });
    }
}

impl<S: RecordStorage, C: Clock, B: Backtrace> BatchSource for Recorder<S, C, B> {
    fn now_us(&self) -> u64 {
        self.clock.now_us()
    }

    fn take_batch(&self) -> Result<Option<Batch>, CaptureError> {
        critical_section::with(|cs| {
            let mut state = self.state.borrow(cs).borrow_mut();
            let next = state
                .next_batch
                .checked_add(1)
                .ok_or(CaptureError::BatchIdsExhausted)?;
            let initial_boot = state.storage.initial_boot().cloned();
            let mut records = Vec::new();
            while !records.is_full() {
                match state.storage.pop() {
                    Some(record) => {
                        let _ = records.push(record);
                    }
                    None => break,
                }
            }
            let losses = state
                .statistics
                .evicted
                .saturating_add(state.statistics.rejected);
            let dropped_records = losses.saturating_sub(state.reported_losses);
            if records.is_empty() && dropped_records == 0 {
                return Ok(None);
            }
            let mut id = [0; 24];
            id[..16].copy_from_slice(&self.batch_namespace);
            id[16..].copy_from_slice(&state.next_batch.to_le_bytes());
            state.next_batch = next;
            state.reported_losses = losses;
            Ok(Some(Batch {
                id,
                initial_boot,
                records,
                dropped_records,
            }))
        })
    }
}

/// Truncates at UTF-8 boundaries while allowing formatting to finish successfully.
struct BoundedText<const N: usize> {
    text: String<N>,
    truncated: bool,
}

impl<const N: usize> BoundedText<N> {
    fn new() -> Self {
        Self {
            text: String::new(),
            truncated: false,
        }
    }
}

impl<const N: usize> Write for BoundedText<N> {
    fn write_str(&mut self, value: &str) -> fmt::Result {
        if self.truncated {
            return Ok(());
        }
        let mut end = value.len().min(N - self.text.len());
        while !value.is_char_boundary(end) {
            end -= 1;
        }
        let _ = self.text.push_str(&value[..end]);
        self.truncated = end != value.len();
        Ok(())
    }
}
