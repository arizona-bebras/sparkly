# Agent implementation roadmap

## Implemented foundation

- Portable `no_std` domain model, synchronous capture, explicit synchronization.
- Pluggable storage, clock, backtrace provider, encoder and sync/async transport.
- Raw `log` capture with bounded text and immediate stack capture.
- Complete encoded defmt frames, with whole-frame overflow handling.
- Static metric IDs, gauge/counter-delta handles and ELF-only dictionary entries.
- Once-per-startup Boot events, random boot IDs and ordered mixed-boot parcels.
- Byte-bounded circular storage, oldest-record eviction (including Boots), and
  per-parcel loss deltas.
- Bounded micropb encoding and retained retries with acknowledgment validation.
- ESP32-S3 hardware adapter, build-ID linker integration and standalone example.
- Linux demonstration, ELF inspector, portable tests and protocol round trips.
- CLI project setup, transparent linker delegation, and reproducible post-link
  build-ID embedding without HAL flash-map overrides.

## Next: verify hardware and resource budgets

1. Flash the example and compare the emitted build ID with the uploaded ELF.
2. Decode actual defmt frames with the matching ELF, including a deliberately
   dropped/oversized frame followed by valid data.
3. Symbolicate raw-log PCs from optimized firmware and document frame-pointer,
   inlining and stack-walker limitations.
4. Exercise dual-core and interrupt logging. Measure critical-section duration,
   RAM, task stack high-water marks and per-log capture cost.
5. Test reset reason reporting across power-on, software reset and watchdog.
   This does not require implementing crash capture.

## Next: production delivery and enrollment

1. Implement ordered per-device ingestion: atomically apply events, advance
   active boot/build context and record parcel IDs before acknowledgment. Define
   maximum batch size and deduplication retention; never apply a duplicate reset.
2. Choose and measure the ESP32-S3 TLS/crypto stack with actual certificate chains.
   Verify hostname/root/date checks, client authentication, CSR generation,
   certificate renewal and peak handshake RAM before committing to a library.
3. Add pluggable credential storage and a private-key signer. Implement scoped
   token enrollment, idempotent CSR retries and atomic credential replacement.
4. Add direct HTTPS/mTLS transport using embedded I/O/network traits where useful.
5. Specify signed bridge envelopes, key lookup, replay rules and trusted/signed
   acknowledgments. Add USB/BLE framing and a bridge implementation.
6. Add jittered backoff, batching thresholds and wakeups without coupling the core
   to an executor. Authentication errors must not trigger a tight retry loop.

## Storage and observability improvements

- Measure byte-ring capture/pop latency on hardware. Compact the exporter's
  pending batch as well,
  preserving records on encoding failure and identical bytes on send retries.
- Design a separate durable outbox contract and flash worker, including erase
  alignment, wear, power-fail recovery and pending-batch acknowledgment state.
  Persist queue-to-parcel transfers atomically, replay old pending parcels before
  new ones, and retain loss deltas with the original parcel IDs. Test power loss
  at each commit/ack boundary and startup with a full ring.
- Preserve `RecordStorage::initial_boot` metadata in future persistent storage;
  the RAM ring now retains head context across Boot eviction and parcels carry it
  in `initial_boot`. Loss deltas still cannot reconstruct evicted telemetry.
- Add cumulative counters, bounded labels, units and histograms with explicit
  overflow/reset semantics and dictionary versioning.
- Add configurable backtrace depth/filtering, structured log fields and optional
  file/line metadata. Current behavior captures a trace for every accepted log.
- Consider per-core/per-context defmt staging if measured latency requires it.
- Use the shared ELF extraction library in the backend; add CLI upload of exact
  finalized ELFs, store artifact hashes and reject conflicting build IDs.
- Broaden CLI init profiles beyond ESP32-S3 and validate more linker/host combinations.

## Additional platforms

- Linux: add per-frame module identity for shared libraries. `sparkly-linux`
  now provides reusable clock, stack and rustls identity adapters; the separate
  host application owns its linker/defmt configuration. It captures main-executable
  stacks with ELF address normalization; its defmt build uses a non-PIE binary
  to preserve string IDs. Add optional `metrics` facade support.
- STM32: choose a first MCU family, implement reset decoding, a suitable stack
  walker and linker note mapping; reuse the existing recorder/exporter.
- Zephyr: expose a C ABI with opaque handles, pointer/length buffers and status
  codes. Host callbacks supply time, synchronization, storage staging and I/O.
  Keep Rust futures/trait objects out of the ABI; provide a host-driven step API.

## Deferred: crashes

No crash hooks, panic persistence, crash transport or fault-safe storage are
implemented. A future design needs a separate fault path, retained capsule,
original build/boot identity and recovery on the next boot. It must not reuse
ordinary recorder locks or attempt networking from a fault handler.
