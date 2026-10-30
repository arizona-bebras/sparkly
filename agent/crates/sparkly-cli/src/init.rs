use crate::Result;
use clap::{Args, ValueEnum};
use std::{
    fs,
    io::Write,
    path::{Path, PathBuf},
};
use toml_edit::{Array, DocumentMut, InlineTable, Item, Table, Value, value};

#[derive(Clone, Copy, Debug, ValueEnum)]
pub enum Chip {
    Esp32s3,
}

#[derive(Clone, Copy, Debug, ValueEnum)]
pub enum LinkerFlavor {
    Gcc,
    Ld,
}

#[derive(Debug, Args)]
pub struct InitOptions {
    #[arg(long, value_enum)]
    pub chip: Chip,
    /// Existing application directory. Its build.rs, if any, is left intact.
    #[arg(long, default_value = ".")]
    pub path: PathBuf,
    /// Local agent/ workspace, required for unpublished SDK dependencies.
    #[arg(long)]
    pub sdk_path: Option<PathBuf>,
    /// Override the existing target linker or ESP's default GCC driver.
    #[arg(long)]
    pub real_linker: Option<String>,
    #[arg(long, value_enum)]
    pub linker_flavor: Option<LinkerFlavor>,
    /// Select Sparkly's global defmt logger in addition to raw log capture.
    #[arg(long)]
    pub defmt: bool,
}

/// Validate all edits before writing either file. TOML editing preserves unrelated
/// settings/comments, and each file is replaced atomically. Re-running is idempotent.
pub fn initialize(options: &InitOptions) -> Result<PathBuf> {
    let root = options.path.canonicalize()?;
    let manifest_path = root.join("Cargo.toml");
    let config_path = root.join(".cargo/config.toml");
    if root.join(".cargo/config").exists() {
        return Err(
            "found legacy .cargo/config; migrate it to config.toml before running init".into(),
        );
    }
    let mut manifest = fs::read_to_string(&manifest_path)?.parse::<DocumentMut>()?;
    if !manifest.get("package").is_some_and(Item::is_table_like) {
        return Err("init requires an application package, not a virtual workspace".into());
    }
    let config_text = if config_path.exists() {
        fs::read_to_string(&config_path)?
    } else {
        String::new()
    };
    let mut config = config_text.parse::<DocumentMut>()?;
    let sdk = options
        .sdk_path
        .as_ref()
        .map(|path| path.canonicalize())
        .transpose()?;
    if let Some(sdk) = &sdk
        && (!sdk.join("Cargo.toml").is_file()
            || !sdk.join("crates/sparkly-esp/Cargo.toml").is_file())
    {
        return Err("--sdk-path must point to the agent/ workspace".into());
    }
    let use_defmt = options.defmt
        || dependency_features(&manifest, "sparkly-agent")
            .iter()
            .any(|f| f == "defmt");
    add_dependency(
        &mut manifest,
        "sparkly-agent",
        sdk.as_deref(),
        &root,
        if use_defmt {
            &["log", "defmt"]
        } else {
            &["log"]
        },
    )?;
    add_dependency(
        &mut manifest,
        "sparkly-esp",
        sdk.as_ref()
            .map(|p| p.join("crates/sparkly-esp"))
            .as_deref(),
        &root,
        &["esp32s3"],
    )?;
    add_dependency(&mut manifest, "log", None, &root, &[])?;
    if use_defmt {
        add_dependency(&mut manifest, "defmt", None, &root, &[])?;
    }

    let target = match options.chip {
        Chip::Esp32s3 => "xtensa-esp32s3-none-elf",
    };
    let existing_linker = config
        .get("target")
        .and_then(|t| t.get(target))
        .and_then(|t| t.get("linker"))
        .and_then(Item::as_str)
        .filter(|name| *name != "sparkly-link");
    let real = options
        .real_linker
        .as_deref()
        .or(existing_linker)
        .or_else(|| env_setting(&config, "SPARKLY_REAL_LINKER"))
        .unwrap_or("xtensa-esp32s3-elf-gcc")
        .to_owned();
    if matches!(
        Path::new(&real).file_stem().and_then(|p| p.to_str()),
        Some("sparkly-link" | "sparkly")
    ) {
        return Err("--real-linker must identify the underlying toolchain, not Sparkly".into());
    }
    let flavor = match options.linker_flavor {
        Some(LinkerFlavor::Gcc) => "gcc",
        Some(LinkerFlavor::Ld) => "ld",
        None => env_setting(&config, "SPARKLY_LINKER_FLAVOR").unwrap_or("gcc"),
    }
    .to_owned();
    if !matches!(flavor.as_str(), "gcc" | "ld") {
        return Err("linker flavor must be gcc or ld".into());
    }

    let target_flags = config
        .get("target")
        .and_then(|t| t.get(target))
        .and_then(|t| t.get("rustflags"));
    let inherited_flags = config.get("build").and_then(|b| b.get("rustflags"));
    let mut flags = read_flags(target_flags.or(inherited_flags))?;
    if flags
        .iter()
        .any(|f| f.starts_with("linker=") || f.starts_with("-Clinker="))
    {
        return Err("move -C linker= from rustflags to target.linker before running init".into());
    }
    ensure_codegen(&mut flags, &format!("linker-flavor={flavor}"));
    ensure_codegen(&mut flags, "force-frame-pointers=yes");
    ensure_codegen(&mut flags, "link-arg=-Tlinkall.x");
    if use_defmt {
        ensure_codegen(&mut flags, "link-arg=-Tdefmt.x");
    }

    table(&mut config, "target")?;
    if config["target"].get(target).is_none() {
        config["target"][target] = Item::Table(Table::new());
    }
    if !config["target"][target].is_table_like() {
        return Err("target settings must be a table".into());
    }
    config["target"][target]["linker"] = value("sparkly-link");
    let mut array = Array::new();
    for flag in flags {
        array.push(flag);
    }
    config["target"][target]["rustflags"] = value(array);
    table(&mut config, "build")?;
    config["build"]["target"] = value(target);
    table(&mut config, "unstable")?;
    if config["unstable"].get("build-std").is_none() {
        let mut array = Array::new();
        array.push("core");
        config["unstable"]["build-std"] = value(array);
    }
    set_env(
        &mut config,
        "SPARKLY_REAL_LINKER",
        &real,
        Path::new(&real).components().count() > 1 && !Path::new(&real).is_absolute(),
    )?;
    set_env(&mut config, "SPARKLY_LINKER_FLAVOR", &flavor, false)?;
    set_env(&mut config, "SPARKLY_BARE_METAL", "true", false)?;
    if env_setting(&config, "ESP_BACKTRACE_CONFIG_BACKTRACE_FRAMES").is_none() {
        set_env(
            &mut config,
            "ESP_BACKTRACE_CONFIG_BACKTRACE_FRAMES",
            "16",
            false,
        )?;
    }
    if use_defmt && env_setting(&config, "DEFMT_LOG").is_none() {
        set_env(&mut config, "DEFMT_LOG", "info", false)?;
    }

    fs::create_dir_all(config_path.parent().unwrap())?;
    let manifest_file = stage(&manifest_path, manifest.to_string().as_bytes())?;
    let config_file = stage(&config_path, config.to_string().as_bytes())?;
    manifest_file.persist(&manifest_path).map_err(|e| e.error)?;
    config_file.persist(&config_path).map_err(|e| e.error)?;
    Ok(root)
}

fn table(document: &mut DocumentMut, name: &str) -> Result<()> {
    if document.get(name).is_none() {
        document[name] = Item::Table(Table::new());
    }
    if !document[name].is_table_like() {
        return Err(format!("{name} must be a table").into());
    }
    Ok(())
}

fn dependency_features(manifest: &DocumentMut, name: &str) -> Vec<String> {
    manifest
        .get("dependencies")
        .and_then(|d| d.get(name))
        .and_then(|d| d.get("features"))
        .and_then(Item::as_array)
        .map(|a| {
            a.iter()
                .filter_map(|v| v.as_str().map(str::to_owned))
                .collect()
        })
        .unwrap_or_default()
}

fn add_dependency(
    manifest: &mut DocumentMut,
    name: &str,
    sdk: Option<&Path>,
    root: &Path,
    features: &[&str],
) -> Result<()> {
    table(manifest, "dependencies")?;
    // Avoid adding a second copy under the standard name when the user renamed it.
    if manifest["dependencies"]
        .as_table_like()
        .unwrap()
        .iter()
        .any(|(key, item)| key != name && item.get("package").and_then(Item::as_str) == Some(name))
    {
        return Err(format!(
            "dependency {name} is renamed; configure that alias manually before init"
        )
        .into());
    }
    let entry = &mut manifest["dependencies"][name];
    if entry.is_none() {
        let mut dependency = InlineTable::new();
        if let Some(path) = sdk {
            let path = relative_path(root, path);
            dependency.insert(
                "path",
                Value::from(
                    path.to_str()
                        .ok_or("dependency paths must be representable as UTF-8 TOML strings")?
                        .replace('\\', "/"),
                ),
            );
        } else {
            dependency.insert(
                "version",
                Value::from(match name {
                    "log" => "0.4",
                    "defmt" => "1",
                    _ => env!("CARGO_PKG_VERSION"),
                }),
            );
        }
        *entry = Item::Value(Value::InlineTable(dependency));
    } else if let Some(version) = entry.as_str() {
        let mut dependency = InlineTable::new();
        dependency.insert("version", Value::from(version));
        *entry = Item::Value(Value::InlineTable(dependency));
    }
    let dependency = entry
        .as_table_like_mut()
        .ok_or("dependency must be a string or table")?;
    if !features.is_empty() {
        if dependency.get("features").is_none() {
            dependency.insert("features", value(Array::new()));
        }
        let array = dependency
            .get_mut("features")
            .and_then(Item::as_array_mut)
            .ok_or("dependency features must be an array")?;
        for feature in features {
            if !array.iter().any(|value| value.as_str() == Some(feature)) {
                array.push(*feature);
            }
        }
    }
    Ok(())
}

fn relative_path(from: &Path, to: &Path) -> PathBuf {
    let from: Vec<_> = from.components().collect();
    let to_components: Vec<_> = to.components().collect();
    if from.first() != to_components.first() {
        return to.to_owned();
    }
    let common = from
        .iter()
        .zip(&to_components)
        .take_while(|(a, b)| a == b)
        .count();
    let mut path = PathBuf::new();
    for _ in common..from.len() {
        path.push("..");
    }
    for component in &to_components[common..] {
        path.push(component.as_os_str());
    }
    if path.as_os_str().is_empty() {
        path.push(".");
    }
    path
}

fn read_flags(item: Option<&Item>) -> Result<Vec<String>> {
    match item {
        None => Ok(Vec::new()),
        Some(item) if item.as_str().is_some() => Ok(item
            .as_str()
            .unwrap()
            .split_whitespace()
            .map(str::to_owned)
            .collect()),
        Some(item) => item
            .as_array()
            .ok_or("rustflags must be a string or array")?
            .iter()
            .map(|value| {
                value
                    .as_str()
                    .map(str::to_owned)
                    .ok_or_else(|| "rustflags must contain strings".into())
            })
            .collect(),
    }
}

fn ensure_codegen(flags: &mut Vec<String>, setting: &str) {
    if !flags
        .iter()
        .any(|flag| flag == setting || flag == &format!("-C{setting}"))
    {
        flags.extend(["-C".to_owned(), setting.to_owned()]);
    }
}

fn env_setting<'a>(config: &'a DocumentMut, name: &str) -> Option<&'a str> {
    let item = config.get("env")?.get(name)?;
    item.as_str()
        .or_else(|| item.get("value").and_then(Item::as_str))
}

fn set_env(config: &mut DocumentMut, name: &str, setting: &str, relative: bool) -> Result<()> {
    table(config, "env")?;
    if config["env"].get(name).is_some_and(Item::is_table_like) {
        config["env"][name]["value"] = value(setting);
        if relative {
            config["env"][name]["relative"] = value(true);
        } else {
            config["env"][name]
                .as_table_like_mut()
                .unwrap()
                .remove("relative");
        }
    } else if relative {
        let mut item = InlineTable::new();
        item.insert("value", Value::from(setting));
        item.insert("relative", Value::from(true));
        config["env"][name] = value(item);
    } else {
        config["env"][name] = value(setting);
    }
    Ok(())
}

fn stage(path: &Path, contents: &[u8]) -> Result<tempfile::NamedTempFile> {
    let mut file = tempfile::NamedTempFile::new_in(path.parent().unwrap())?;
    if path.exists() {
        file.as_file()
            .set_permissions(fs::metadata(path)?.permissions())?;
    }
    file.write_all(contents)?;
    file.as_file().sync_all()?;
    Ok(file)
}
