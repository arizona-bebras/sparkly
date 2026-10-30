# ESP32-S3 hardware support

`sparkly-esp` supplies the clock, reset reason, program-counter capture and
firmware build ID. Storage, encoding, transport and executor remain application
choices. It does not initialize the chip or claim application peripherals.

The adapter currently pins `esp-hal = 1.0.0` and `esp-backtrace = 0.18.1` because
it uses unstable reset/RNG APIs. Upgrade these dependencies together with
cross-build verification. The HAL's own flash mapping is used unchanged.

## Build the end-user example

Install the Espressif toolchain (named `esp`), its Xtensa linker, `protoc`, and
both Sparkly CLI binaries:

```sh
cargo install --path agent/crates/sparkly-cli --locked
```

From the repository root:

```sh
cd agent/crates/sparkly-esp/examples/esp32s3
source ~/export-esp.sh
cargo +esp build
cargo +esp build --release
```

Run from this directory: Cargo discovers the local `.cargo/config.toml` based
on the current directory, not `--manifest-path` alone. The configuration enables
frame pointers, defmt linking, 16 backtrace frames, and `build-std = ["core"]`.
The example is a separate workspace so Xtensa settings never affect host builds.

The example uses `StaticCell`, installs both logging facades, records a sample
temperature, uses a 2048-byte circular RAM queue, and periodically drives an
exporter. Its `Offline` transport
deliberately returns an error and keeps the first batch pending. Replace it with
an authenticated application transport; the example does not pretend that a
successful USB write is backend acceptance. Queued records eventually overwrite
the oldest records while offline, including
Boot markers, with loss statistics retained. Evicting an unexported Boot loses
the context needed to interpret later telemetry. The example starts with empty
RAM;
applications restoring a durable stream append the new reset after that history.

The example briefly enables the ADC entropy source to create a fresh boot ID.
Applications with existing entropy/radio management can instead supply their
own 128-bit boot ID. `recorder()` does not initialize RNG hardware.

Inspect the ELF from `agent/`:

```sh
sparkly inspect crates/sparkly-esp/examples/esp32s3/target/xtensa-esp32s3-none-elf/debug/sparkly-esp32s3-example
```

## Integrating into existing firmware

1. Run `sparkly init --chip esp32s3 --sdk-path /path/to/sparkly/agent --defmt`.
   This adds dependencies and configures the linker wrapper. Omit `--defmt` if
   the application does not select Sparkly's global defmt logger.
2. Initialize `esp-hal` and create a unique boot ID.
3. Call `sparkly_esp::recorder(storage, boot_id, firmware_version)` and retain it
   in static storage if using global facades. It appends the new Boot immediately;
   on failure, `InitError` returns the existing storage alongside its reason.
4. Call `sparkly_agent::install(recorder)`, then optionally `install_log(level)`.
5. Drive one `Exporter` with your transport. Use `poll_async()` for an async
   transport or `poll()` for a synchronous one; the application owns scheduling.

No Sparkly build.rs or HAL linker-script override is required. The wrapper calls
the real Xtensa driver, generates a GNU build ID, then patches its value into a
40-byte ordinary `.rodata` slot. The GNU note and metric dictionary remain
ELF-only metadata. `sparkly_esp::build_id()` delegates to the portable volatile
runtime reader, preventing constant-folding of the placeholder under LTO.
Missing finalization is reported during initialization as `NotFinalized`.

No panic handler is installed by the library. The example's handler only halts.
Backtraces are best effort: the upstream walker does not certify why a walk
ended, so the adapter reports `Partial` or `Truncated`, rather than promising a
complete stack. Backend symbolication uses the exact uploaded ELF and the
already-adjusted program counters from `esp-backtrace`.

## Validation boundary

Cross-compilation and ELF inspection establish linking and metadata layout.
On-device checks still need to measure stack use and logging latency, verify
frame accuracy, exercise both cores and interrupt priorities, and compare the
reported build ID with the ELF. No hardware is required for the portable tests.
