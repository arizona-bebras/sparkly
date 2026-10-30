//! Simulated IoT sensor: runs until interrupted, exporting to SPARKLY_INGEST_URL.
use sparkly_agent::*;
use std::{
    io,
    sync::OnceLock,
    thread,
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

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
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
        log::info!(target: "device", "state {:?} -> {:?}", self.state, next);
        counter!("device.state_transitions").increment(1);
        self.state = next;
        self.state_ticks = 0;
    }

    fn tick(&mut self, uptime: Duration) {
        self.state_ticks += 1;
        let cpu_load = match self.state {
            DeviceState::Booting => {
                if self.state_ticks >= 3 {
                    log::info!(target: "device", "self-test passed; sensor initialized");
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
                    log::error!(target: "sensor", "mock sensor timeout; scheduling bus reset");
                    counter!("sensor.errors").increment(1);
                    self.transition(DeviceState::Recovering);
                } else {
                    let humidity = self.random.range(35.0, 65.0);
                    metric!("sensor.humidity_percent").set(humidity);
                    counter!("sensor.samples").increment(1);
                    log::info!(target: "sensor", "sample: temperature={:.1} C humidity={humidity:.1}%", self.temperature);
                    if self.temperature >= 35.0 {
                        log::warn!(target: "device", "temperature limit reached; enabling cooling fan");
                        self.transition(DeviceState::Cooling);
                    }
                }
                self.random.range(65.0, 95.0)
            }
            DeviceState::Cooling => {
                self.temperature -= self.random.range(0.8, 1.6);
                self.battery -= 0.03;
                if self.temperature <= 26.0 {
                    log::info!(target: "device", "temperature normal; disabling cooling fan");
                    self.transition(DeviceState::Idle);
                }
                self.random.range(10.0, 20.0)
            }
            DeviceState::Recovering => {
                self.temperature += (24.0 - self.temperature) * 0.1;
                if self.state_ticks >= 4 {
                    log::info!(target: "sensor", "bus reset complete; sensor responding");
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
    log::info!(target: "device", "booting mock environmental sensor");
    let mut device = Device::new(u64::from_le_bytes(boot_id[..8].try_into().unwrap()));
    let mut exporter = Exporter::<_, _, 8192>::new(
        recorder,
        HttpTransport::new(endpoint),
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
