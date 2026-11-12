use crate::{Payload, Record, Session, storage_codec};

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct PushOutcome {
    pub evicted: u64,
}

/// A record is larger than the queue byte budget.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct StorageFull;

/// Synchronous capture storage. Methods run inside the recorder critical section:
/// they must be bounded, nonblocking, non-reentrant, and must never log.
///
/// Preserve FIFO order among retained records. Any record, including a Boot,
/// may be evicted when capacity is needed.
/// A failed push must leave existing records intact. pop transfers ownership to
/// the ordered exporter; it is not an acknowledgment. A durable implementation
/// must recover pending parcels before exporting later records after a reboot.
/// Persistent flash/network I/O
/// belongs behind a separate staging worker, not in these methods.
pub trait RecordStorage: Send {
    type Error;

    fn push(&mut self, record: Record) -> Result<PushOutcome, Self::Error>;
    fn pop(&mut self) -> Option<Record>;

    /// Boot context immediately before the queue head. Preserve this alongside
    /// restored history and advance it when a Boot is popped or evicted.
    /// Storage without recoverable context must return None rather than guess.
    fn initial_boot(&self) -> Option<&Session> {
        None
    }
}

/// Byte memory used by the ring. Its length must remain positive and unchanged
/// while owned by a ring. Both views must refer to the same exclusively owned
/// memory; writes must be immediately readable. Access must be bounded,
/// nonblocking, infallible and must not allocate, log or re-enter the recorder.
///
/// Arrays, mutable slices, Vec<u8> and Box<[u8]> work through the blanket impl.
/// Heap buffers must be allocated/sized before capture. Raw flash is not byte
/// memory: use RAM staging and a separate flash worker/RecordStorage adapter.
pub trait ByteStorage: Send {
    fn as_bytes(&self) -> &[u8];
    fn as_bytes_mut(&mut self) -> &mut [u8];
}

impl<T: AsRef<[u8]> + AsMut<[u8]> + Send> ByteStorage for T {
    fn as_bytes(&self) -> &[u8] {
        self.as_ref()
    }
    fn as_bytes_mut(&mut self) -> &mut [u8] {
        self.as_mut()
    }
}

/// A bounded FIFO of packed records over interchangeable byte memory. Records
/// may wrap across the end; push/pop never move other queued records. Capacity
/// pressure evicts oldest whole records, including Boots.
pub struct RingStorage<B> {
    backing: B,
    head: usize,
    used: usize,
    initial_boot: Option<Session>,
}

/// An inline RAM ring with a compile-time byte budget. For borrowed or heap
/// memory, use RingStorage::with_storage instead.
pub type RamStorage<const N: usize> = RingStorage<[u8; N]>;

impl<const N: usize> RingStorage<[u8; N]> {
    pub const fn new() -> Self {
        assert!(N > 0, "byte capacity must be positive");
        Self {
            backing: [0; N],
            head: 0,
            used: 0,
            initial_boot: None,
        }
    }
}

impl<B: ByteStorage> RingStorage<B> {
    /// Starts an EMPTY ring in existing byte memory. Previous bytes are ignored;
    /// this does not restore queue metadata or persistent records after reboot.
    /// Panics for a zero-length buffer. No allocation or clearing is performed.
    pub fn with_storage(backing: B) -> Self {
        assert!(
            !backing.as_bytes().is_empty(),
            "byte capacity must be positive"
        );
        Self {
            backing,
            head: 0,
            used: 0,
            initial_boot: None,
        }
    }

    /// Returns the byte provider, discarding queue metadata. This is not a
    /// persistent snapshot; retain the entire RingStorage to hand off a queue.
    pub fn into_backing(self) -> B {
        self.backing
    }

    /// Bytes occupied by queued records, including framing.
    pub fn used_bytes(&self) -> usize {
        self.used
    }

    pub fn capacity_bytes(&self) -> usize {
        self.backing.as_bytes().len()
    }

    /// Exact packed footprint of this record, independent of target layout.
    pub fn record_size(record: &Record) -> usize {
        let mut len = 2;
        storage_codec::visit(record, |bytes| len += bytes.len());
        len
    }

    fn head_length(&self) -> usize {
        u16::from_le_bytes([
            self.backing.as_bytes()[self.head],
            self.backing.as_bytes()[(self.head + 1) % self.capacity_bytes()],
        ]) as usize
    }

    fn discard_head(&mut self, len: usize) {
        self.head = (self.head + len) % self.capacity_bytes();
        self.used -= len;
    }

    fn write_at(&mut self, offset: usize, bytes: &[u8]) -> usize {
        let capacity = self.capacity_bytes();
        let first = bytes.len().min(capacity - offset);
        let memory = self.backing.as_bytes_mut();
        memory[offset..offset + first].copy_from_slice(&bytes[..first]);
        memory[..bytes.len() - first].copy_from_slice(&bytes[first..]);
        (offset + bytes.len()) % capacity
    }
}

impl<const N: usize> Default for RamStorage<N> {
    fn default() -> Self {
        Self::new()
    }
}

impl<B: ByteStorage> RecordStorage for RingStorage<B> {
    type Error = StorageFull;

    fn push(&mut self, record: Record) -> Result<PushOutcome, Self::Error> {
        let capacity = self.capacity_bytes();
        let len = Self::record_size(&record);
        if len > capacity || len > u16::MAX as usize {
            return Err(StorageFull);
        }

        let mut evicted = 0;
        while len > capacity - self.used {
            self.pop();
            evicted += 1;
        }
        let start = (self.head + self.used) % capacity;
        let mut end = self.write_at(start, &(len as u16).to_le_bytes());
        storage_codec::visit(&record, |bytes| {
            end = self.write_at(end, bytes);
        });
        debug_assert_eq!(end, (start + len) % capacity);
        debug_assert!(len >= storage_codec::HEADER);
        self.used += len;
        Ok(PushOutcome { evicted })
    }

    fn initial_boot(&self) -> Option<&Session> {
        self.initial_boot.as_ref()
    }

    fn pop(&mut self) -> Option<Record> {
        if self.used == 0 {
            return None;
        }
        let len = self.head_length();
        let first = len.min(self.capacity_bytes() - self.head);
        let memory = self.backing.as_bytes();
        let record = storage_codec::decode(
            &memory[self.head..self.head + first],
            &memory[..len - first],
        );
        self.discard_head(len);
        if let Payload::Boot(boot) = &record.payload {
            self.initial_boot = Some(boot.clone());
        }
        Some(record)
    }
}
