use crate::{Batch, BatchId};

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum EncodeError {
    BufferTooSmall,
    InvalidRecord,
}

/// An interchangeable batch serializer. It must not retain references to input.
pub trait BatchEncoder {
    fn encode(&self, batch: &Batch, output: &mut [u8]) -> Result<usize, EncodeError>;
}

#[derive(Clone, Copy, Debug)]
pub struct OutboundBatch<'a> {
    pub id: BatchId,
    /// Immutable serialized bytes. Retries must transmit these exact bytes.
    pub bytes: &'a [u8],
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Acknowledgment {
    pub batch_id: BatchId,
}

/// Delivery boundary: return an acknowledgment only after the configured durable
/// recipient accepts the batch. A socket write or USB transfer is not an ack.
/// Implementations own connection setup, authentication, framing and timeouts.
/// Preserve one ordered stream per device: boot/build context carries across
/// parcels. The recipient must atomically apply events/context and deduplicate
/// the ID before acknowledging; a retry must not create a second reset.
pub trait Transport {
    type Error;

    fn send(&mut self, batch: OutboundBatch<'_>) -> Result<Acknowledgment, Self::Error>;
}

/// Executor-independent async equivalent, usable with Embassy or a host runtime.
/// Cancellation must not fabricate an ack; a subsequent call may retry the batch.
#[allow(async_fn_in_trait)]
pub trait AsyncTransport {
    type Error;

    async fn send(&mut self, batch: OutboundBatch<'_>) -> Result<Acknowledgment, Self::Error>;
}
