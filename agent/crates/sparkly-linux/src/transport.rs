use sparkly_core::{Acknowledgment, OutboundBatch, Transport};
use std::io;

/// A 204 acknowledges ClickHouse buffering, not durable delivery.
/// The TLS client authenticates using the locally persisted device identity.
pub struct HttpTransport {
    endpoint: String,
    client: ureq::Agent,
}

impl HttpTransport {
    pub fn new(endpoint: String, client: ureq::Agent) -> Self {
        Self { endpoint, client }
    }
}

impl Transport for HttpTransport {
    type Error = io::Error;

    fn send(&mut self, batch: OutboundBatch<'_>) -> Result<Acknowledgment, Self::Error> {
        let response = self
            .client
            .post(&self.endpoint)
            .header("Content-Type", "application/x-protobuf")
            .send(batch.bytes)
            .map_err(io::Error::other)?;
        if response.status() != 204 {
            return Err(io::Error::other(format!(
                "ingest returned {}; expected 204 No Content",
                response.status()
            )));
        }
        Ok(Acknowledgment { batch_id: batch.id })
    }
}
