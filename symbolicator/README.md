# Symbolicator

A standalone Rust decoder for one protobuf JSON `Parcel`. It takes an ELF path as its second argument and prints JSON containing `boots`, `metrics`, `logs`,
`crashes`, and diagnostics. The library entry point is independent of file I/O so
a RabbitMQ consumer can call it later. RabbitMQ, S3, and database writes are not
part of this first slice.

From `symbolicator/`, generate a runnable native fixture and decode it:

```sh
python3 tools/make_fixture.py
cargo run -- fixtures/parcel.json fixtures/firmware.elf > /tmp/symbolicated.json
cargo test
cargo clippy --all-targets -- -D warnings
cargo fmt -- --check
```

Requires Rust, `protoc`, Python 3, a GNU-compatible C compiler, and GNU binutils
(`objcopy`, `readelf`, `nm`) for fixture generation/tests. The fixture contains a
real ELF build ID, DWARF, a Sparkly metric dictionary and a small raw-encoded defmt
table. Tests also cover missing/mismatched builds, mixed-boot batches, malformed
defmt frames, exact uint64 deltas, and agent/service protobuf compatibility.

To decode a real batch, supply its protobuf JSON parcel followed by the matching
finalized, unstripped firmware ELF path.
The symbolicator accepts only parcel data and symbols. Analytics adds device/park
identity, group membership, server batch ID, receipt time, event timestamps and
boot sequences when storing the result.

The JSON input limit is 8 MiB to allow for base64 and JSON expansion of the
1 MiB binary parcels accepted by analytics. The supplied ELF must have a valid GNU
build ID; a corrupt ELF or malformed JSON fails the command. Per-event decoding failures preserve
usable fields and append diagnostics instead of aborting the batch.

Input uses the same protobuf JSON mapping as analytics' `protojson.Marshal`:
lowerCamelCase field names, base64 bytes, enum names, and decimal strings for
64-bit integers. The parser also accepts original snake_case field names and
numeric enum values. Types and JSON mappings are generated from `parcel.proto`;
there is no separate input schema to maintain. Library callers use
`parse_parcel_json` followed by `symbolicate`. Binary protobuf decoding happens
in analytics before the handoff to this service.

Output follows the proposed typed ClickHouse tables, with these explicit details:

- Each row retains the original `event_index`, `monotonic_us`, `build_id` and
  hex `boot_id` from the parcel. Producer parcel ID and loss count are returned
  separately at the top level.
- Singular `initial_boot` seeds decoding without generating a boot row. Real Boot
  events create rows and switch context for subsequent events. The supplied ELF is
  used only when its build ID matches the current context exactly.
- Metrics retain numeric `metric_id`, resolved `name` and `kind`. Gauge `value` is
  a float; counter deltas use exact `counter_delta` integers rather than losing
  precision in a float. Unknown IDs and invalid/mismatched measurements produce
  unresolved rows. Non-finite gauge values become null with a diagnostic.
- Logs preserve raw text/target/truncation, or decode a complete defmt frame with
  the ELF's table/encoding. Defmt source module/file/line are returned when the
  debug metadata provides them. Original defmt bytes remain in `defmt_raw_hex`.
  `decode_status` describes message decoding; frame resolution is represented by
  nullable frame fields and diagnostics. `trace_status` retains capture quality.
- Frames retain unsigned PCs, optional function/file/line, and one row per inline
  frame when DWARF provides it. `module` is the matched ELF build ID; the protocol
  does not supply module names or relocation bases. Fixed-address firmware is
  supported; PIE/shared-library rebasing and raw stack unwinding are future work.
- Crash stack bytes use hex `stack`; registers remain uint64 values.
- Log `fingerprint` is a deterministic SHA-256 of build, level, message, target,
  encoded defmt bytes and captured PCs. This is a local placeholder, not a final
  incident-grouping policy. Incidents and group membership require external data.

The output is an intermediate result, not ClickHouse DDL or an insert statement.
No broker acknowledgment or ingestion behavior has changed yet.
