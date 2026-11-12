# Host sensor example

Workspace application `sparkly-host-example`, using `sparkly-linux` for clock,
stack capture, HTTP transport and rustls identity enrollment, and `sparkly-agent` for telemetry.
The simulation, export scheduling, CA/environment configuration and defmt linker
settings belong to this crate.

From the agent workspace:

```sh
export SPARKLY_CA_CERT="$PWD/../step-ca/local/root_ca.crt"
cargo build -p sparkly-cli --bins
cargo rustc -p sparkly-host-example -- -C linker=target/debug/sparkly-link -C linker-flavor=gcc
SPARKLY_ENROLLMENT_TOKEN="$PARK_TOKEN" SPARKLY_INGEST_URL=https://localhost:10001/ingest target/debug/sparkly-host-example
```

`SPARKLY_CA_CERT` is the build-time path to the trusted PEM CA bundle. The token
is required at runtime only for first enrollment. Keep `.sparkly-host/` between
runs, or set `SPARKLY_DEVICE_DIR` to another private credential directory.
`SPARKLY_ENROLLMENT_URL` defaults to `https://localhost:10002/devices/reg`.
See the workspace README for service setup and enrollment details.

The binary uses a non-PIE Linux link and retains the defmt dictionary so string
IDs match ELF addresses. Log filtering defaults to `info`; override `DEFMT_LOG`
at build time. Keep the exact finalized ELF for the symbolicator. Test with
`cargo test -p sparkly-host-example`; this also requires a CA path at compile time
but does not contact the enrollment/ingest services.
