use micropb_gen::{Config, Generator};

fn main() {
    let proto = "../../proto/parcel.proto";
    println!("cargo:rerun-if-changed={proto}");
    let mut generator = Generator::new();
    generator
        .add_protoc_arg("-I../../proto")
        .use_container_heapless()
        .configure(".", Config::new().max_len(64).max_bytes(64))
        .configure(".sparkly.agent.v1.Parcel.events", Config::new().max_len(8))
        .configure(
            ".sparkly.agent.v1.Parcel.batch_id",
            Config::new().max_bytes(24),
        )
        .configure(
            ".sparkly.agent.v1.Boot.boot_id",
            Config::new().max_bytes(16),
        )
        .configure(
            ".sparkly.agent.v1.RawLog.message",
            Config::new().max_bytes(256),
        )
        .configure(
            ".sparkly.agent.v1.Log.defmt_log",
            Config::new().max_bytes(512),
        )
        .configure(".sparkly.agent.v1.Log.frames", Config::new().max_len(16))
        // Keep the existing crash wire contract; this implementation does not emit it.
        .configure(
            ".sparkly.agent.v1.Crash.stack",
            Config::new().max_bytes(256),
        )
        .configure(".sparkly.agent.v1.Crash.frames", Config::new().max_len(16))
        .configure(
            ".sparkly.agent.v1.Crash.registers",
            Config::new().max_len(8),
        );
    generator
        .compile_protos(
            &[proto],
            std::path::PathBuf::from(std::env::var_os("OUT_DIR").unwrap()).join("parcel.rs"),
        )
        .expect("generate bounded telemetry types (requires protoc)");
}
