use sparkly_core::Clock;
use std::time::Instant;

pub struct MonotonicClock(Instant);

impl MonotonicClock {
    pub fn new() -> Self {
        Self(Instant::now())
    }
}

impl Default for MonotonicClock {
    fn default() -> Self {
        Self::new()
    }
}

impl Clock for MonotonicClock {
    fn now_us(&self) -> u64 {
        self.0.elapsed().as_micros().min(u64::MAX as u128) as u64
    }
}
