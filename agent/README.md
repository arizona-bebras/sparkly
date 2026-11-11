# Agent

Rust workspace for bounded telemetry capture on `no_std` devices and Linux.
Raw `log` records, encoded `defmt` frames, metrics and boot metadata are implemented.
The host example supports token-based enrollment and mTLS; embedded transports and crash capture remain future work.

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
# Required for commands that compile the host example. Use an absolute path.
export SPARKLY_CA_CERT="$PWD/../step-ca/local/root_ca.crt"
cargo test --workspace --features std,defmt
cargo clippy --workspace --all-targets --features std,defmt -- -D warnings
cargo fmt --all -- --check
cargo build -p sparkly-cli --bins
# Use a verified CA certificate and the park's enrollment token.
cargo rustc --example host --features std -- -C linker=target/debug/sparkly-link -C linker-flavor=gcc
SPARKLY_ENROLLMENT_TOKEN="$PARK_TOKEN" SPARKLY_INGEST_URL=https://localhost:10001/ingest target/debug/examples/host
target/debug/sparkly inspect target/debug/examples/host
```

The host example embeds its CA bundle from `SPARKLY_CA_CERT` and its park token
from `SPARKLY_ENROLLMENT_TOKEN` **at build time**. `SPARKLY_CA_CERT` is required
whenever the host example is compiled, including workspace tests and Clippy with
all targets. Use an absolute path: `include_bytes!` resolves relative paths from
`examples/host.rs`, not the working directory. Cargo tracks CA-file and
build-variable changes. Verify the
CA's fingerprint before building; the example never downloads or trusts a CA at
runtime. The binary contains the enrollment token; it is not written to the
credential directory or sent with telemetry. Library-only builds can omit both
variables. CI generates a disposable CA certificate for compiling the example;
no enrollment token is required for tests.

| Variable                   | When read | Purpose                                                                    |
| -------------------------- | --------- | -------------------------------------------------------------------------- |
| `SPARKLY_CA_CERT`          | Build     | Path to the PEM trust bundle to embed                                      |
| `SPARKLY_ENROLLMENT_TOKEN` | Build     | Park enrollment token, required for first enrollment                       |
| `SPARKLY_INGEST_URL`       | Runtime   | Required HTTPS ingest URL, e.g. `https://localhost:10001/ingest`           |
| `SPARKLY_ENROLLMENT_URL`   | Runtime   | HTTPS registration URL; defaults to `https://localhost:10002/devices/reg`  |
| `SPARKLY_DEVICE_DIR`       | Runtime   | Credential directory; defaults to `.sparkly-host` in the working directory |

On first startup, the example generates an ECDSA P-256 private key and persists
it as `device.key`, then sends a signed CSR with no common name or SANs to the
backend's enrollment endpoint. It verifies the returned certificate's trust,
expiry, client-auth usage, and correspondence to the private key, then atomically
saves the PEM chain as `device.crt`. On Unix, newly created directories use 0700
and credential files use 0600. An advisory lock prevents concurrent enrollment
using the same directory. Keep the directory between runs; changing it creates
a different device identity.

Later startups reuse the saved key and certificate without contacting enrollment
or requiring the token. A failed enrollment retains the key for a later retry.
Corrupt, expired, or incomplete saved credentials produce an error rather than
silently registering a new device. Automatic renewal is not implemented. The CA
bundle must trust the enrollment/ingest servers and the issuer of device
certificates; development uses the same root for both. See
[CA setup](../step-ca/README.md) and [gateway configuration](../envoy/README.md).

Start backend, analytics, and Envoy before running the example. Requests use
HTTPS with a 10-second timeout and redirects disabled. Ingest uses the device
certificate for mTLS; the example does not supply device/park identity headers.
Only `204 No Content` counts as an acknowledgment; analytics acknowledges its
ClickHouse async buffer, which is not a guarantee of durable delivery. TLS and
enrollment dependencies are development dependencies and do not affect the
portable library. Run `cargo test --example host --features std` to exercise
credential persistence and validation.

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
