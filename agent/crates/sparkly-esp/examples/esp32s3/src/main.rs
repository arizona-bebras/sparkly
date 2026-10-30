#![no_std]
#![no_main]

use esp_hal::{
    delay::Delay,
    main,
    rng::{Trng, TrngSource},
};
use sparkly_agent::{
    Acknowledgment, Exporter, OutboundBatch, ProtobufEncoder, RamStorage, Transport, metric,
};
use sparkly_esp::EspRecorder;
use static_cell::StaticCell;

esp_bootloader_esp_idf::esp_app_desc!();

static RECORDER: StaticCell<EspRecorder<RamStorage<2048>>> = StaticCell::new();

#[main]
fn main() -> ! {
    let peripherals = esp_hal::init(esp_hal::Config::default());
    // Enable actual entropy before creating the boot ID. This temporarily owns
    // ADC1; release it before initializing an application ADC or radio.
    let entropy = TrngSource::new(peripherals.RNG, peripherals.ADC1);
    let random = Trng::try_new().unwrap();
    let mut boot_id = [0; 16];
    random.read(&mut boot_id);
    drop(random);
    let _ = entropy.try_disable();

    let recorder = RECORDER.init_with(|| {
        sparkly_esp::recorder(RamStorage::new(), boot_id, env!("CARGO_PKG_VERSION")).unwrap()
    });
    sparkly_agent::install(recorder).unwrap();
    sparkly_agent::install_log(log::LevelFilter::Info).unwrap();

    // Supply your authenticated Wi-Fi or signed bridge Transport here. Offline
    // deliberately never acknowledges data: the pending batch remains in RAM.
    let mut exporter = Exporter::<_, _, 8192>::new(recorder, Offline, ProtobufEncoder, 5_000_000);
    let delay = Delay::new();
    loop {
        log::info!(target: "example", "sampling telemetry");
        defmt::info!("compact sample: {=u32}", 42);
        metric!("example.temperature").set(23.5); // illustrative value, not a sensor read
        let _ = exporter.poll();
        delay.delay_millis(1_000);
    }
}

struct Offline;
impl Transport for Offline {
    type Error = ();
    fn send(&mut self, _: OutboundBatch<'_>) -> Result<Acknowledgment, Self::Error> {
        Err(())
    }
}

// The application owns its panic behavior. Crash reporting is outside this draft.
#[panic_handler]
fn panic(_: &core::panic::PanicInfo<'_>) -> ! {
    loop {
        core::hint::spin_loop();
    }
}
