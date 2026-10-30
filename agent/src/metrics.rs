use crate::{CaptureError, MetricValue, Sink, sink};

/// A compile-time metric identifier. Names live only in the ELF dictionary.
#[derive(Clone, Copy, Debug)]
pub struct Gauge {
    id: u64,
}

impl Gauge {
    pub const fn new(id: u64) -> Self {
        Self { id }
    }
    pub const fn id(self) -> u64 {
        self.id
    }
    pub fn set(self, value: f64) {
        let _ = self.try_set(value);
    }
    pub fn try_set(self, value: f64) -> Result<(), CaptureError> {
        self.set_on(sink().ok_or(CaptureError::NotInstalled)?, value)
    }
    pub fn set_on(self, recorder: &dyn Sink, value: f64) -> Result<(), CaptureError> {
        recorder.metric(self.id, MetricValue::Gauge(value))
    }
}

/// Emits exact integer increments; the backend applies each event once.
#[derive(Clone, Copy, Debug)]
pub struct Counter {
    id: u64,
}

impl Counter {
    pub const fn new(id: u64) -> Self {
        Self { id }
    }
    pub const fn id(self) -> u64 {
        self.id
    }
    pub fn increment(self, amount: u64) {
        let _ = self.try_increment(amount);
    }
    pub fn try_increment(self, amount: u64) -> Result<(), CaptureError> {
        self.increment_on(sink().ok_or(CaptureError::NotInstalled)?, amount)
    }
    pub fn increment_on(self, recorder: &dyn Sink, amount: u64) -> Result<(), CaptureError> {
        recorder.metric(self.id, MetricValue::CounterDelta(amount))
    }
}
