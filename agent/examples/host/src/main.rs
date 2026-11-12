//! Simulated IoT sensor: enrolls once, then exports over mTLS until interrupted.
use sparkly_linux::{HttpTransport, LinuxBacktrace, MonotonicClock, identity};

use sparkly_agent::*;
use std::{
    io,
    sync::OnceLock,
    thread,
    time::{Duration, Instant},
};

const ROOT_CA_PEM: &[u8] = include_bytes!(env!("SPARKLY_CA_CERT"));

static RECORDER: OnceLock<Recorder<RamStorage<2048>, MonotonicClock, LinuxBacktrace>> =
    OnceLock::new();

#[derive(Clone, Copy, Debug, PartialEq, Eq, defmt::Format)]
enum DeviceState {
    Booting,
    Idle,
    Sampling,
    Cooling,
    Recovering,
}

/// Small, non-cryptographic PRNG, seeded separately on every simulated boot.
struct MockRandom(u64);

impl MockRandom {
    fn range(&mut self, min: f64, max: f64) -> f64 {
        self.0 ^= self.0 << 13;
        self.0 ^= self.0 >> 7;
        self.0 ^= self.0 << 17;
        let fraction = (self.0 >> 11) as f64 / (1_u64 << 53) as f64;
        min + fraction * (max - min)
    }
}

struct Device {
    state: DeviceState,
    state_ticks: u32,
    temperature: f64,
    battery: f64,
    random: MockRandom,
}

impl Device {
    fn new(seed: u64) -> Self {
        Self {
            state: DeviceState::Booting,
            state_ticks: 0,
            temperature: 24.0,
            battery: 85.0,
            random: MockRandom(seed.max(1)),
        }
    }

    fn transition(&mut self, next: DeviceState) {
        defmt::info!("state {} -> {}", self.state, next);
        counter!("device.state_transitions").increment(1);
        self.state = next;
        self.state_ticks = 0;
    }

    fn tick(&mut self, uptime: Duration) {
        self.state_ticks += 1;
        let cpu_load = match self.state {
            DeviceState::Booting => {
                if self.state_ticks >= 3 {
                    defmt::info!("self-test passed; sensor initialized");
                    self.transition(DeviceState::Idle);
                }
                self.random.range(40.0, 65.0)
            }
            DeviceState::Idle => {
                self.temperature += (24.0 - self.temperature) * 0.15;
                // Pretend a solar panel charges the battery while the sensor sleeps.
                self.battery += self.random.range(0.05, 0.15);
                if self.state_ticks >= 6 {
                    self.transition(DeviceState::Sampling);
                }
                self.random.range(1.0, 5.0)
            }
            DeviceState::Sampling => {
                self.temperature += self.random.range(0.6, 1.4);
                self.battery -= self.random.range(0.05, 0.12);
                if self.random.range(0.0, 1.0) < 0.05 {
                    defmt::error!("mock sensor timeout; scheduling bus reset");
                    counter!("sensor.errors").increment(1);
                    self.transition(DeviceState::Recovering);
                } else {
                    let humidity = self.random.range(35.0, 65.0);
                    metric!("sensor.humidity_percent").set(humidity);
                    counter!("sensor.samples").increment(1);
                    defmt::info!(
                        "sample: temperature={=f64} C humidity={=f64}%",
                        self.temperature,
                        humidity
                    );
                    if self.temperature >= 35.0 {
                        defmt::warn!("temperature limit reached; enabling cooling fan");
                        self.transition(DeviceState::Cooling);
                    }
                }
                self.random.range(65.0, 95.0)
            }
            DeviceState::Cooling => {
                self.temperature -= self.random.range(0.8, 1.6);
                self.battery -= 0.03;
                if self.temperature <= 26.0 {
                    defmt::info!("temperature normal; disabling cooling fan");
                    self.transition(DeviceState::Idle);
                }
                self.random.range(10.0, 20.0)
            }
            DeviceState::Recovering => {
                self.temperature += (24.0 - self.temperature) * 0.1;
                if self.state_ticks >= 4 {
                    defmt::info!("bus reset complete; sensor responding");
                    counter!("sensor.recoveries").increment(1);
                    self.transition(DeviceState::Idle);
                }
                self.random.range(20.0, 35.0)
            }
        };
        self.battery = self.battery.clamp(0.0, 100.0);
        metric!("device.uptime_seconds").set(uptime.as_secs_f64());
        metric!("cpu.temperature").set(self.temperature);
        metric!("cpu.load_percent").set(cpu_load);
        metric!("battery.charge_percent").set(self.battery);
        metric!("wifi.rssi_dbm").set(self.random.range(-80.0, -45.0));
    }
}

fn main() -> io::Result<()> {
    let endpoint = std::env::var("SPARKLY_INGEST_URL")
        .expect("set SPARKLY_INGEST_URL, e.g. https://localhost:10001/ingest");
    let enrollment_token = std::env::var("SPARKLY_ENROLLMENT_TOKEN").unwrap_or_else(|_| "".into());
    let build_id = build_id().expect("build this example using sparkly-link");
    let enrollment_url = std::env::var("SPARKLY_ENROLLMENT_URL")
        .unwrap_or_else(|_| "https://localhost:10002/devices/reg".into());
    let directory = std::env::var_os("SPARKLY_DEVICE_DIR")
        .map(std::path::PathBuf::from)
        .unwrap_or_else(|| ".sparkly-host".into());
    identity::require_https(&endpoint)?;
    identity::require_https(&enrollment_url)?;
    let client =
        identity::load_or_enroll(&enrollment_url, &directory, ROOT_CA_PEM, &enrollment_token)?;
    let mut boot_id = [0; 16];
    std::io::Read::read_exact(
        &mut std::fs::File::open("/dev/urandom").unwrap(),
        &mut boot_id,
    )
    .unwrap();
    let backtrace = LinuxBacktrace::new()?;
    let recorder = RECORDER.get_or_init(|| {
        Recorder::new(
            Session::new(boot_id, &build_id, "host-example", "process_start").unwrap(),
            RamStorage::new(),
            MonotonicClock::new(),
            backtrace,
        )
        .unwrap()
    });
    install(recorder).unwrap();
    defmt::info!("booting mock environmental sensor");
    let mut device = Device::new(u64::from_le_bytes(boot_id[..8].try_into().unwrap()));
    let mut exporter = Exporter::<_, _, 8192>::new(
        recorder,
        HttpTransport::new(endpoint, client),
        ProtobufEncoder,
        1_000_000,
    );
    let started = Instant::now();
    let export_interval = Duration::from_secs(2);
    let mut last_export = started;
    println!("simulating an IoT sensor; press Ctrl+C to stop");
    loop {
        device.tick(started.elapsed());
        if last_export.elapsed() >= export_interval {
            // Each parcel holds only a few records. Drain the queue, keeping the
            // exporter's pending parcel intact on failure for the next attempt.
            loop {
                match exporter.flush() {
                    Ok(FlushOutcome::Acknowledged(_)) => {}
                    Ok(FlushOutcome::Idle) => break,
                    Err(error) => {
                        eprintln!("telemetry export failed; retrying in 2 seconds: {error:?}");
                        break;
                    }
                }
            }
            last_export = Instant::now();
        }
        thread::sleep(Duration::from_secs(1));
    }
}

#[cfg(all(test, target_os = "linux"))]
mod tests {
    use super::*;

    #[test]
    fn defmt_sensor_logs_decode_with_the_host_elf_and_include_frames() {
        let recorder = RECORDER.get_or_init(|| {
            Recorder::new(
                Session::new([1; 16], "test", "host-example", "test").unwrap(),
                RamStorage::new(),
                MonotonicClock::new(),
                LinuxBacktrace::new().unwrap(),
            )
            .unwrap()
        });
        install(recorder).unwrap();
        let mut device = Device::new(1);
        device.transition(DeviceState::Sampling);
        // Exercise typed floating-point formatting as well as the state enum.
        defmt::warn!("sample: temperature={=f64} C humidity={=f64}%", 24.5, 50.0);
        let batch = recorder.take_batch().unwrap().unwrap();
        let elf = std::fs::read(std::env::current_exe().unwrap()).unwrap();
        let table = defmt_decoder::Table::parse(&elf).unwrap().unwrap();
        let mut messages = Vec::new();
        for record in &batch.records {
            match &record.payload {
                Payload::Boot(_) | Payload::Metric { .. } => {}
                Payload::Defmt(log) => {
                    assert!(!log.trace.pcs.is_empty());
                    assert!(matches!(
                        log.trace.status,
                        TraceStatus::Partial | TraceStatus::Truncated
                    ));
                    let mut decoder = table.new_stream_decoder();
                    decoder.received(&log.frame);
                    messages.push(decoder.decode().unwrap().display_message().to_string());
                }
                _ => panic!("expected defmt logs rather than raw text"),
            }
        }
        assert!(
            messages.iter().any(|s| s == "state Booting -> Sampling"),
            "{messages:?}"
        );
        assert!(
            messages
                .iter()
                .any(|s| s == "sample: temperature=24.5 C humidity=50.0%"),
            "{messages:?}"
        );
    }
}
