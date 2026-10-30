# Agent

Rust workspace for bounded telemetry capture on `no_std` devices and Linux.
Raw `log` records, encoded `defmt` frames, metrics and boot metadata are implemented.
Crashes and production networking/authentication are future work.

Each startup appends one Boot event to storage. Parcels preserve the ordered
retained stream and may span multiple boots. The byte ring evicts oldest complete
records, including Boots, so capacity pressure can lose boot/build context for
later telemetry. Evicted Boots count toward loss statistics. Initialization
returns the original storage on failure. The provided RAM queue and pending
exporter batch are volatile; durable flash/outbox storage remains an extension
to implement.

`RamStorage<N>` now takes a **byte budget**, not a record count. It stores actual
payload lengths in a circular buffer; a metric occupies 28 bytes. Both examples
use `RamStorage<2048>`.
Update existing capacities when migrating. The pending exporter batch still
reserves fixed-size records separately from the queue and encoding buffer.
`RamStorage<N>` is an alias for `RingStorage<[u8; N]>`. Use
`RingStorage::with_storage(backing)` for borrowed slices, preallocated Vec/Box
buffers, or a custom `ByteStorage` provider. This selects the ring's byte memory;
raw flash needs RAM staging and a separate persistence worker, not direct
mutable-slice writes. Flash persistence/recovery remains future work.

- [Architecture, extension contracts and memory model](../docs/architecture/agent.md)
- [Implementation roadmap](../docs/architecture/agent-roadmap.md)
- [ESP32-S3 integration and example](crates/sparkly-esp/README.md)
- [CLI setup and linker integration](crates/sparkly-cli/README.md)

Requires Rust 1.88+ for the portable crates, a GNU-compatible native C compiler
for host linker tests, and `protoc` on the build host. The
ESP example uses the separately installed Espressif Rust toolchain and its own
lockfile. Host commands below are run from `agent/`:

```sh
cargo test --workspace --features std,defmt
cargo clippy --workspace --all-targets --features std,defmt -- -D warnings
cargo fmt --all -- --check
cargo build -p sparkly-cli --bins
cargo rustc --example host --features std -- -C linker=target/debug/sparkly-link -C linker-flavor=gcc
SPARKLY_INGEST_URL=http://localhost:8081/ingest target/debug/examples/host
target/debug/sparkly inspect target/debug/examples/host
```

The host example requires `SPARKLY_INGEST_URL` and posts binary protobuf over HTTP
with a 10-second timeout. Start the analytics ingest receiver before running it.
Only `204 No Content` counts as a demo acknowledgment. This example assumes the
development endpoint prints parcels; it does not provide durable delivery or
authentication. The HTTP client is a development dependency and does not affect
the portable library.
`sparkly-link` finalizes the runtime build ID automatically after linking.
`sparkly inspect` extracts and verifies build IDs and the metric dictionary.
Run it on the finalized, unstripped ELF before releasing/uploading firmware.

Application code can use `metric!("cpu.temperature").set(23.5)` and
`counter!("requests.completed").increment(1)`. Metric names reside in an ELF-only
section when the application links the provided script. Global facades require a
static recorder and explicit installation. For isolated recorders/tests, use the
handle's `set_on` / `increment_on` methods or the `Sink` trait directly.

The application selects its `critical-section` implementation. Enable `std` only
for a host application; embedded applications use their HAL or RTOS provider.
Enable `defmt` only when selecting Sparkly as the application's sole defmt logger.
