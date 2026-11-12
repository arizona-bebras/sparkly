# Linux platform support

Reusable adapters for `sparkly-core`, including its identity lifecycle. This crate does not
install a recorder/logger or select an application's linker configuration.

- `MonotonicClock`: microseconds since construction using `Instant`.
- `LinuxBacktrace`: bounded main-executable stack capture, translating runtime
  addresses to ELF virtual addresses for offline symbolication. Other modules
  are omitted; traces are partial or truncated. Initialize before installing
  logging so unwinder initialization happens outside capture.
- `HttpTransport`: sends binary protobuf parcels using a supplied authenticated
  HTTP client. Only `204 No Content` acknowledges the original batch ID; request
  failures or other status codes return errors for the exporter to retry.
- `identity::FileIdentityStore`: `device.key` and `device.crt` storage, exclusive
  advisory lock, Unix permissions, atomic certificate replacement.
- `identity::RustlsIdentity`: P-256 key generation, identity-free signed CSR,
  HTTPS enrollment, certificate validation and an authenticated `ureq::Agent`.

Construct the transport with `HttpTransport::new(ingest_url, client)` using the
client returned by enrollment. The transport does not print or configure logging.

Use both identity adapters with `sparkly_core::identity::load_or_enroll`, or call the
convenience wrapper:

```rust,ignore
let client = sparkly_linux::identity::load_or_enroll(
    enrollment_url, credential_directory, trusted_ca_pem, enrollment_token,
)?;
```

Enrollment disables redirects, limits responses to 128 KiB and uses a 10-second
HTTP timeout. Loaded and returned certificates are checked for trust, validity,
client-auth usage and key correspondence. Expired/corrupt identities return
errors; failed enrollment retains the saved key. Token/configuration comes from
the caller; there are no example-specific environment variables in this crate.

Run `cargo test -p sparkly-linux` from the agent workspace. The runnable example
and defmt linker configuration are in `examples/host/`.
