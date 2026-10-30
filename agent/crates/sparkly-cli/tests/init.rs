use sparkly_cli::init::{Chip, InitOptions, initialize};
use std::fs;
use toml_edit::DocumentMut;

fn project() -> (tempfile::TempDir, InitOptions) {
    let dir = tempfile::tempdir().unwrap();
    fs::write(dir.path().join("Cargo.toml"), "# app comment\n[package]\nname = 'firmware'\nversion = '0.1.0'\nedition = '2024'\n\n[dependencies]\nexisting = { version = '1', default-features = false }\n").unwrap();
    fs::create_dir(dir.path().join(".cargo")).unwrap();
    fs::write(dir.path().join(".cargo/config.toml"), "# cargo comment\n[alias]\nflash = 'run --release'\n\n[build]\nrustflags = ['-C', 'opt-level=s']\n\n[target.xtensa-esp32s3-none-elf]\nlinker = 'custom-gcc'\nrunner = 'my-flasher'\n").unwrap();
    fs::write(
        dir.path().join("build.rs"),
        "fn main() { /* application owned */ }\n",
    )
    .unwrap();
    let options = InitOptions {
        chip: Chip::Esp32s3,
        path: dir.path().to_owned(),
        sdk_path: None,
        real_linker: None,
        linker_flavor: None,
        defmt: true,
    };
    (dir, options)
}

#[test]
fn preserves_application_settings_and_is_idempotent() {
    let (dir, options) = project();
    initialize(&options).unwrap();
    let manifest = fs::read_to_string(dir.path().join("Cargo.toml")).unwrap();
    let config = fs::read_to_string(dir.path().join(".cargo/config.toml")).unwrap();
    let cargo: DocumentMut = config.parse().unwrap();
    assert!(manifest.contains("# app comment"));
    assert!(config.contains("# cargo comment"));
    assert_eq!(
        cargo["target"]["xtensa-esp32s3-none-elf"]["linker"].as_str(),
        Some("sparkly-link")
    );
    assert_eq!(
        cargo["target"]["xtensa-esp32s3-none-elf"]["runner"].as_str(),
        Some("my-flasher")
    );
    assert_eq!(
        cargo["env"]["SPARKLY_REAL_LINKER"].as_str(),
        Some("custom-gcc")
    );
    assert_eq!(cargo["alias"]["flash"].as_str(), Some("run --release"));
    assert!(
        cargo["target"]["xtensa-esp32s3-none-elf"]["rustflags"]
            .as_array()
            .unwrap()
            .iter()
            .any(|v| v.as_str() == Some("opt-level=s"))
    );
    assert!(
        fs::read_to_string(dir.path().join("build.rs"))
            .unwrap()
            .contains("application owned")
    );
    initialize(&options).unwrap();
    assert_eq!(
        fs::read_to_string(dir.path().join("Cargo.toml")).unwrap(),
        manifest
    );
    assert_eq!(
        fs::read_to_string(dir.path().join(".cargo/config.toml")).unwrap(),
        config
    );
}

#[test]
fn local_sdk_dependencies_use_paths_and_existing_features_survive() {
    let (dir, mut options) = project();
    let sdk = tempfile::tempdir().unwrap();
    fs::create_dir_all(sdk.path().join("crates/sparkly-esp")).unwrap();
    fs::write(
        sdk.path().join("Cargo.toml"),
        "[package]\nname='sparkly-agent'\n",
    )
    .unwrap();
    fs::write(
        sdk.path().join("crates/sparkly-esp/Cargo.toml"),
        "[package]\nname='sparkly-esp'\n",
    )
    .unwrap();
    options.sdk_path = Some(sdk.path().to_owned());
    initialize(&options).unwrap();
    let manifest: DocumentMut = fs::read_to_string(dir.path().join("Cargo.toml"))
        .unwrap()
        .parse()
        .unwrap();
    assert!(
        manifest["dependencies"]["sparkly-agent"]["path"]
            .as_str()
            .is_some()
    );
    assert_eq!(
        manifest["dependencies"]["existing"]["default-features"].as_bool(),
        Some(false)
    );
    let features = manifest["dependencies"]["sparkly-agent"]["features"]
        .as_array()
        .unwrap();
    assert!(features.iter().any(|v| v.as_str() == Some("defmt")));
}

#[test]
fn invalid_configuration_is_rejected_before_modifying_either_file() {
    let (dir, options) = project();
    let path = dir.path().join(".cargo/config.toml");
    fs::write(
        &path,
        "[target.xtensa-esp32s3-none-elf]\nrustflags = [123]\n",
    )
    .unwrap();
    let manifest_before = fs::read(dir.path().join("Cargo.toml")).unwrap();
    let config_before = fs::read(&path).unwrap();
    assert!(initialize(&options).is_err());
    assert_eq!(
        fs::read(dir.path().join("Cargo.toml")).unwrap(),
        manifest_before
    );
    assert_eq!(fs::read(path).unwrap(), config_before);
}

#[test]
fn selected_chip_sets_the_default_target_and_preserves_relative_driver_options() {
    let (dir, options) = project();
    fs::write(
        dir.path().join(".cargo/config.toml"),
        r#"
[build]
target = "old-target"
[target.xtensa-esp32s3-none-elf]
linker = "tools/custom-gcc"
[env]
SPARKLY_REAL_LINKER = { value = "previous-driver", force = true }
"#,
    )
    .unwrap();
    initialize(&options).unwrap();
    let path = dir.path().join(".cargo/config.toml");
    let contents = fs::read_to_string(&path).unwrap();
    let config = contents.parse::<DocumentMut>().unwrap();
    assert_eq!(
        config["build"]["target"].as_str(),
        Some("xtensa-esp32s3-none-elf")
    );
    assert_eq!(
        config["env"]["SPARKLY_REAL_LINKER"]["value"].as_str(),
        Some("tools/custom-gcc")
    );
    assert_eq!(
        config["env"]["SPARKLY_REAL_LINKER"]["relative"].as_bool(),
        Some(true)
    );
    assert_eq!(
        config["env"]["SPARKLY_REAL_LINKER"]["force"].as_bool(),
        Some(true)
    );
    initialize(&options).unwrap();
    assert_eq!(fs::read_to_string(path).unwrap(), contents);
}
