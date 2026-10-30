//! Host demonstration: posts a protobuf parcel to SPARKLY_INGEST_URL.
use sparkly_agent::*;
use std::{
    io,
    sync::OnceLock,
    time::{Duration, Instant},
};

struct HostClock(Instant);
impl Clock for HostClock {
    fn now_us(&self) -> u64 {
        self.0.elapsed().as_micros() as u64
    }
}

/// Development transport: ingest prints parcels without persisting them, so its
/// 204 response is only a demo acknowledgment, not durable delivery.
struct HttpTransport {
    endpoint: String,
    client: ureq::Agent,
}

impl HttpTransport {
    fn new(endpoint: String) -> Self {
        let config = ureq::Agent::config_builder()
            .timeout_global(Some(Duration::from_secs(10)))
            .max_redirects(0)
            .build();
        Self {
            endpoint,
            client: config.into(),
        }
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
        println!("sent {} bytes to {}", batch.bytes.len(), self.endpoint);
        Ok(Acknowledgment { batch_id: batch.id })
    }
}

static RECORDER: OnceLock<Recorder<RamStorage<2048>, HostClock, NoBacktrace>> = OnceLock::new();

fn main() {
    let endpoint = std::env::var("SPARKLY_INGEST_URL")
        .expect("set SPARKLY_INGEST_URL, e.g. http://localhost:8081/ingest");
    let build_id = build_id().expect("build this example using sparkly-link");
    let mut boot_id = [0; 16];
    std::io::Read::read_exact(
        &mut std::fs::File::open("/dev/urandom").unwrap(),
        &mut boot_id,
    )
    .unwrap();
    let recorder = RECORDER.get_or_init(|| {
        Recorder::new(
            Session::new(boot_id, &build_id, "host-example", "process_start").unwrap(),
            RamStorage::new(),
            HostClock(Instant::now()),
            NoBacktrace,
        )
        .unwrap()
    });
    install(recorder).unwrap();
    install_log(log::LevelFilter::Info).unwrap();
    log::info!(target: "host", "hello from Sparkly");
    metric!("cpu.temperature").set(23.5);
    counter!("requests.completed").increment(1);
    let mut exporter = Exporter::<_, _, 8192>::new(
        recorder,
        HttpTransport::new(endpoint),
        ProtobufEncoder,
        1_000_000,
    );
    exporter.flush().unwrap();
}
