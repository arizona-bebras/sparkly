fn main() {
    println!("cargo:rerun-if-changed=../agent/proto/parcel.proto");
    let descriptors = std::path::PathBuf::from(std::env::var_os("OUT_DIR").unwrap())
        .join("parcel-descriptor.bin");
    prost_build::Config::new()
        .file_descriptor_set_path(&descriptors)
        .compile_protos(&["../agent/proto/parcel.proto"], &["../agent/proto"])
        .expect("generate telemetry types (requires protoc)");
    pbjson_build::Builder::new()
        .register_descriptors(&std::fs::read(descriptors).expect("read parcel descriptors"))
        .expect("register parcel descriptors")
        .build(&[".sparkly.agent.v1"])
        .expect("generate protobuf JSON mapping");
}
