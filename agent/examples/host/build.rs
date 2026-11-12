fn main() {
    println!("cargo:rerun-if-changed=defmt-host.x");
    println!("cargo:rerun-if-env-changed=DEFMT_LOG");
    println!("cargo:rerun-if-env-changed=SPARKLY_CA_CERT");
    if let Some(ca) = std::env::var_os("SPARKLY_CA_CERT") {
        println!(
            "cargo:rerun-if-changed={}",
            std::path::Path::new(&ca).display()
        );
    }
    if std::env::var("CARGO_CFG_TARGET_OS").as_deref() == Ok("linux") {
        let filter = std::env::var("DEFMT_LOG").unwrap_or_else(|_| "info".into());
        println!("cargo:rustc-env=DEFMT_LOG={filter}");
        let script = std::path::PathBuf::from(std::env::var_os("CARGO_MANIFEST_DIR").unwrap())
            .join("defmt-host.x");
        println!(
            "cargo:rustc-link-arg-bin=sparkly-host-example=-T{}",
            script.display()
        );
        // defmt string IDs are linked addresses cast to u16. They must not be
        // relocated by PIE/ASLR. This applies only to the Linux host example.
        println!("cargo:rustc-link-arg-bin=sparkly-host-example=-no-pie");
    }
}
