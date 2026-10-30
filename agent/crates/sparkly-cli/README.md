# Sparkly CLI

The package installs `sparkly` and `sparkly-link`. The CLI currently supports
project setup, ELF inspection and post-link finalization. Upload, enrollment
and device-management commands are future work.

## Install and integrate

From this checkout:

```sh
cargo install --path agent/crates/sparkly-cli --locked
```

In an existing ESP32-S3 Rust firmware project:

```sh
sparkly init --chip esp32s3 --sdk-path /path/to/sparkly/agent --defmt
source ~/export-esp.sh
cargo +esp build --release
```

The SDK crates are not published yet; `--sdk-path` adds local path dependencies.
When they are published, omitting that flag selects versioned dependencies.
Omit `--defmt` to use raw logs without selecting Sparkly's global defmt logger.

`init` adds the Sparkly/logging dependencies, selects the chip's default target,
and configures the target linker,
frame pointers, HAL/defmt link arguments and build-std. It preserves unrelated
settings, comments, dependency options, the existing target runner and build.rs.
It copies build.rustflags into target.rustflags when necessary to preserve their
effect. Existing target linkers are recorded as the real driver; override with
`--real-linker` and `--linker-flavor gcc|ld`. Re-running init is idempotent.

Only `Cargo.toml` and `.cargo/config.toml` are edited; each is replaced atomically
after all input validation. They are not a cross-file transaction. Review them
in version control. Run init at the Cargo configuration root for the application.
Virtual workspaces and legacy `.cargo/config` need explicit application selection
or migration. The first setup profile supports ESP32-S3 and a single underlying
linker configuration per project; other targets can use the wrapper explicitly.

No Sparkly-specific build.rs, copied HAL linker scripts, or explicit firmware
finalization commands are needed for ordinary configured Cargo builds. Existing
hardware initialization and application task scheduling remain application code.

## Linker behavior

Cargo invokes `sparkly-link`, which delegates to the configured real linker:

| Setting | Host default | ESP32-S3 setup |
| --- | --- | --- |
| `SPARKLY_REAL_LINKER` | `cc` | `xtensa-esp32s3-elf-gcc` |
| `SPARKLY_LINKER_FLAVOR` | `gcc` | `gcc` |
| `SPARKLY_BARE_METAL` | `false` | `true` |

The `ld` flavor accepts a GNU-compatible direct linker. A `rust-lld` executable
is invoked with its GNU flavor. Cargo's rustc linker flavor must match the real
driver; init configures this explicitly. These settings are scoped through Cargo's
project environment and can be overridden for custom toolchains.

The wrapper forwards the original argument vector without a shell. It inspects
GNU response files (including nested response files and quoted output paths),
adds its ELF-only metadata script and `--build-id=sha1`, invokes the real linker,
then finalizes the ELF. Real linker failures preserve their exit code. Recursion,
malformed metadata, corrupt slots, and finalization failures fail the Cargo build.
Uninstrumented executables/shared libraries can pass through without a runtime ID.

The target must produce an ELF using a GNU-compatible linker interface. Native
Linux and ESP32-S3 are validated; Mach-O/MSVC formats are not supported. Keep
symbol stripping disabled until after finalization. In particular Cargo's
`strip = "symbols"` is rejected because it would hide the runtime slot's symbol.

## Inspect and finalize

```sh
sparkly inspect path/to/firmware.elf
sparkly finalize path/to/manually-linked.elf
```

Inspection prints the GNU ID, verifies the runtime copy when present, and prints
metric ID/name/kind mappings. It rejects allocated dictionaries, malformed entries,
name-hash mismatches, collisions, conflicting kinds and mismatched runtime IDs.
Finalization patches only the reserved runtime slot and is idempotent. It must
run before stripping, flash-image conversion, compression or signing. Upload the
finalized, unstripped ELF to the backend.

## Development without installation

From `agent/`, build both tools and the native example:

```sh
cargo build -p sparkly-cli --bins
cargo rustc --example host --features std -- -C linker=target/debug/sparkly-link -C linker-flavor=gcc
target/debug/examples/host
target/debug/sparkly inspect target/debug/examples/host
```

For the ESP example, put `agent/target/debug` on PATH, load `export-esp.sh`, and
run `cargo +esp build` from its directory. Both binaries are also available via
`cargo run -p sparkly-cli --bin sparkly -- ...` for setup/inspection.
