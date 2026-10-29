use heapless::Vec;

use crate::{Payload, Record};

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct PushOutcome {
    pub evicted: u64,
}

/// Every slot holds an undelivered boot transition; none may be evicted.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct StorageFull;

/// Synchronous capture storage. Methods run inside the recorder critical section:
/// they must be bounded, nonblocking, non-reentrant, and must never log.
///
/// Preserve FIFO order and never evict Boot records: the backend relies on every
/// reset to establish context, even when all its ordinary telemetry was dropped.
/// A failed push must leave existing records intact. pop transfers ownership to
/// the ordered exporter; it is not an acknowledgment. A durable implementation
/// must recover pending parcels before exporting later records after a reboot.
/// Persistent flash/network I/O
/// belongs behind a separate staging worker, not in these methods.
pub trait RecordStorage: Send {
    type Error;

    fn push(&mut self, record: Record) -> Result<PushOutcome, Self::Error>;
    fn pop(&mut self) -> Option<Record>;
}

/// A bounded queue of complete records, evicting the oldest non-Boot when full.
/// Boot records stay in FIFO order until transferred to an exporter. If only
/// Boot records remain, push returns StorageFull without modifying the queue.
/// Capacity is in records; each slot reserves space for the largest Payload.
pub struct RamStorage<const N: usize> {
    records: Vec<Record, N>,
}

impl<const N: usize> RamStorage<N> {
    pub const fn new() -> Self {
        assert!(N > 0, "record capacity must be positive");
        Self {
            records: Vec::new(),
        }
    }
}

impl<const N: usize> Default for RamStorage<N> {
    fn default() -> Self {
        Self::new()
    }
}

impl<const N: usize> RecordStorage for RamStorage<N> {
    type Error = StorageFull;

    fn push(&mut self, record: Record) -> Result<PushOutcome, Self::Error> {
        let evicted = if self.records.is_full() {
            let index = self
                .records
                .iter()
                .position(|record| !matches!(record.payload, Payload::Boot(_)))
                .ok_or(StorageFull)?;
            self.records.remove(index);
            1
        } else {
            0
        };
        // The constructor disallows zero capacity; a full queue was freed above.
        let result = self.records.push(record);
        debug_assert!(result.is_ok());
        Ok(PushOutcome { evicted })
    }

    fn pop(&mut self) -> Option<Record> {
        if self.records.is_empty() {
            None
        } else {
            Some(self.records.remove(0))
        }
    }
}
