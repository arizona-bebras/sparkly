use crate::{BatchSource, CaptureError};
use sparkly_core::*;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum FlushOutcome {
    Idle,
    Acknowledged(BatchId),
}

#[derive(Debug, PartialEq, Eq)]
pub enum ExportError<E> {
    Capture(CaptureError),
    Encode(EncodeError),
    Transport(E),
    WrongAcknowledgment,
}

/// Single-consumer export worker. Call poll periodically, or flush explicitly.
/// It owns the pending batch and exact encoded bytes until a matching ack arrives.
/// Dropping the exporter loses its in-flight RAM batch.
pub struct Exporter<'a, T, E, const N: usize = 8192> {
    source: &'a dyn BatchSource,
    transport: T,
    encoder: E,
    interval_us: u64,
    last_attempt: Option<u64>,
    pending: Option<Batch>,
    encoded_len: Option<usize>,
    bytes: [u8; N],
}

impl<'a, T, E: BatchEncoder, const N: usize> Exporter<'a, T, E, N> {
    pub fn new(source: &'a dyn BatchSource, transport: T, encoder: E, interval_us: u64) -> Self {
        assert!(interval_us > 0, "flush interval must be positive");
        Self {
            source,
            transport,
            encoder,
            interval_us,
            last_attempt: None,
            pending: None,
            encoded_len: None,
            bytes: [0; N],
        }
    }

    pub fn has_pending(&self) -> bool {
        self.pending.is_some()
    }
    pub fn transport_mut(&mut self) -> &mut T {
        &mut self.transport
    }

    fn due(&self) -> bool {
        self.last_attempt
            .is_none_or(|last| self.source.now_us().saturating_sub(last) >= self.interval_us)
    }

    fn prepare<X>(&mut self) -> Result<bool, ExportError<X>> {
        self.last_attempt = Some(self.source.now_us());
        if self.pending.is_none() {
            self.pending = self.source.take_batch().map_err(ExportError::Capture)?;
        }
        let Some(batch) = &self.pending else {
            return Ok(false);
        };
        if self.encoded_len.is_none() {
            self.encoded_len = Some(
                self.encoder
                    .encode(batch, &mut self.bytes)
                    .map_err(ExportError::Encode)?,
            );
        }
        Ok(true)
    }

    fn finish<X>(&mut self, ack: Acknowledgment) -> Result<FlushOutcome, ExportError<X>> {
        if self.pending.as_ref().map(|batch| batch.id) != Some(ack.batch_id) {
            return Err(ExportError::WrongAcknowledgment);
        }
        self.pending = None;
        self.encoded_len = None;
        Ok(FlushOutcome::Acknowledged(ack.batch_id))
    }
}

impl<T: Transport, E: BatchEncoder, const N: usize> Exporter<'_, T, E, N> {
    pub fn poll(&mut self) -> Result<FlushOutcome, ExportError<T::Error>> {
        if !self.due() {
            return Ok(FlushOutcome::Idle);
        }
        self.flush()
    }

    pub fn flush(&mut self) -> Result<FlushOutcome, ExportError<T::Error>> {
        if !self.prepare()? {
            return Ok(FlushOutcome::Idle);
        }
        let batch = self.pending.as_ref().expect("prepared batch");
        let ack = self
            .transport
            .send(OutboundBatch {
                id: batch.id,
                bytes: &self.bytes[..self.encoded_len.expect("encoded batch")],
            })
            .map_err(ExportError::Transport)?;
        self.finish(ack)
    }
}

impl<T: AsyncTransport, E: BatchEncoder, const N: usize> Exporter<'_, T, E, N> {
    pub async fn poll_async(&mut self) -> Result<FlushOutcome, ExportError<T::Error>> {
        if !self.due() {
            return Ok(FlushOutcome::Idle);
        }
        self.flush_async().await
    }

    pub async fn flush_async(&mut self) -> Result<FlushOutcome, ExportError<T::Error>> {
        if !self.prepare()? {
            return Ok(FlushOutcome::Idle);
        }
        let batch = self.pending.as_ref().expect("prepared batch");
        let ack = self
            .transport
            .send(OutboundBatch {
                id: batch.id,
                bytes: &self.bytes[..self.encoded_len.expect("encoded batch")],
            })
            .await
            .map_err(ExportError::Transport)?;
        self.finish(ack)
    }
}
