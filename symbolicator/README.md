# Symbolicator

An AMQP worker for analytics parcels, plus an offline Rust decoder. The worker
consumes `symbolication_requests`, symbolicates each parcel, and publishes a
persistent JSON result to `symbolication_results`.

From `symbolicator/`, generate a runnable native fixture and start the worker:

```sh
python3 tools/make_fixture.py
RABBITMQ_URL=amqp://sparkly:sparkly@localhost:5672/sparkly cargo run --locked -- worker
```

`cargo run --locked` also starts the worker. `RABBITMQ_URL` defaults to
`amqp://localhost:5672/`; use the same broker and vhost as analytics.
The worker loads the hardcoded `fixtures/firmware.elf` path once at startup,
relative to the working directory. For real device data, put the finalized,
unstripped firmware ELF at that path. Missing or invalid ELF files fail startup.
There is no S3 integration yet.

The offline command remains available and accepts an explicit ELF path:

```sh
cargo run --locked -- fixtures/parcel.json fixtures/firmware.elf > /tmp/symbolicated.json
cargo test --locked
cargo clippy --locked --all-targets -- -D warnings
cargo fmt -- --check
```

Set `RABBITMQ_TEST_URL` when running tests to exercise the live broker. Tests use
uniquely named queues, verify correlated results, rejection of malformed requests,
and redelivery after an unroutable result; they remove their queues afterward.

Requires Rust, `protoc`, Python 3, a GNU-compatible C compiler, and GNU binutils
(`objcopy`, `readelf`, `nm`) for fixture generation/tests. The fixture contains a
real ELF build ID, DWARF, a Sparkly metric dictionary and a small raw-encoded defmt
table. Tests also cover missing/mismatched builds, mixed-boot batches, malformed
defmt frames, exact uint64 deltas, and agent/service protobuf compatibility.

The worker preserves analytics' server batch ID, authenticated device/park identity,
and receipt timestamp. It does not infer ownership from the parcel. Analytics
will add group membership, event timestamps, and boot sequences when result
persistence is implemented. The worker does not write to ClickHouse.

The request envelope limit is 8 MiB, allowing for base64 and JSON expansion of
analytics' 1 MiB binary parcels. Per-event decoding failures preserve usable fields
and append diagnostics instead of aborting the batch.

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

## Queue contract and delivery

Both queues are durable and shared by all instances. Each worker has a request
channel with prefetch 1 and a separate result publisher channel. Requests use the
analytics version-1 envelope: `schema_version`, `batch_id`, `device_id`, `park_id`,
`received_at`, and nested protobuf JSON `payload`. Content type must be
`application/json`; an AMQP message ID, when supplied, must equal `batch_id`.

Results contain `schema_version: 1`, `result_version: 1`, the same correlation and
ownership fields, `status` (`complete` or `partial` when diagnostics exist),
`symbol_build_ids` (the artifacts actually used), and `result` (the decoder output
described above). AMQP `message_id` and `correlation_id` both equal `batch_id`.
Results are published through the default exchange with mandatory routing,
persistent delivery mode, and publisher confirms, with a 10-second publish timeout.

A request is acknowledged only after its result is confirmed and routed. Invalid
requests are logged and rejected without requeue; RabbitMQ discards them unless
a dead-letter policy is configured. A symbol-provider error, failed/uncertain
publication, acknowledgement failure, or broken connection stops the worker.
The connection closes and unacknowledged requests become available for redelivery.
Restart the process (or use a process supervisor) to resume; automatic reconnect
is not implemented. SIGINT/SIGTERM finish the current task before stopping intake.

Delivery is at least once. A crash after publishing but before acknowledging can
produce duplicate results with the same batch ID. The worker does not deduplicate.
Analytics' result consumer has not been implemented, so results remain queued.

## Code boundaries and symbol providers

- `domain`: validated batch context and symbolicated batch data.
- `application`: symbolication use case and the asynchronous `SymbolProvider` port.
- `infrastructure/messages`: analytics envelope decoding and result serialization.
- `infrastructure/amqp`: topology, consumption, publication, and acknowledgement policy.
- `infrastructure/fixed_elf`: the temporary local ELF provider.
- `config` and `main`: settings, adapter composition, and process lifecycle.
- The existing library decoder remains usable independently of the worker.

`SymbolProvider::symbols_for(build_id)` returns a shared decoded ELF, `None` for a
missing artifact, or an error for a failed lookup. The use case splits parcels at
boot transitions, requests symbols by effective build ID, caches lookups within
the task, and retains original event indices. Returned ELF build IDs are checked.
Missing artifacts produce unresolved rows and diagnostics; transient lookup
errors leave the request retryable. The result lists all artifacts used; the
legacy `result.symbol_build_id` is set when exactly one artifact was used.

To add S3, implement that port in a new infrastructure adapter, including fetching,
ELF validation and cache policy, and inject it in `main`. The fixed provider,
AMQP adapter, queue contract, and symbolication use case need no storage-specific
branches. The port is asynchronous so S3 I/O does not require changing the use case.
