//! Host demonstration: writes a protobuf parcel to disk and acknowledges the file.
use sparkly_agent::*;
use std::{io::Write, sync::OnceLock, time::Instant};

struct HostClock(Instant);
impl Clock for HostClock {
    fn now_us(&self) -> u64 {
        self.0.elapsed().as_micros() as u64
    }
}

/// An explicit local recipient for the demo, not a backend transport.
struct FileTransport;
impl Transport for FileTransport {
    type Error = std::io::Error;
    fn send(&mut self, batch: OutboundBatch<'_>) -> Result<Acknowledgment, Self::Error> {
        let mut file = std::fs::File::create("target/host-parcel.bin")?;
        file.write_all(batch.bytes)?;
        file.sync_all()?;
        println!(
            "wrote {} bytes to target/host-parcel.bin",
            batch.bytes.len()
        );
        Ok(Acknowledgment { batch_id: batch.id })
    }
}

static RECORDER: OnceLock<Recorder<RamStorage<16>, HostClock, NoBacktrace>> = OnceLock::new();

fn main() {
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
    let mut exporter =
        Exporter::<_, _, 8192>::new(recorder, FileTransport, ProtobufEncoder, 1_000_000);
    exporter.flush().unwrap();
}
