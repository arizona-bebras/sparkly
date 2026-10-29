use crate::{Result, response};
use std::{
    env,
    ffi::{OsStr, OsString},
    io::Write,
    path::Path,
    process::Command,
};

/// Environment is scoped by Cargo's project configuration. In a host invocation,
/// defaults match GNU-compatible cc. Init records ESP's real driver explicitly.
pub fn run(arguments: Vec<OsString>) -> Result<i32> {
    let real = env::var_os("SPARKLY_REAL_LINKER").unwrap_or_else(|| "cc".into());
    let flavor = env::var("SPARKLY_LINKER_FLAVOR").unwrap_or_else(|_| "gcc".into());
    if !matches!(flavor.as_str(), "gcc" | "ld") {
        return Err("SPARKLY_LINKER_FLAVOR must be gcc or ld".into());
    }
    let bare_metal = match env::var("SPARKLY_BARE_METAL").as_deref() {
        Ok("true" | "1") => true,
        Ok("false" | "0") | Err(_) => false,
        _ => return Err("SPARKLY_BARE_METAL must be true or false".into()),
    };
    // Also catches aliases/symlinks or accidental recursion through another shim.
    if env::var_os("SPARKLY_LINK_ACTIVE").is_some() {
        return Err("recursive linker configuration".into());
    }
    let name = Path::new(&real)
        .file_stem()
        .and_then(OsStr::to_str)
        .unwrap_or("");
    if matches!(name, "sparkly-link" | "sparkly") {
        return Err("real linker cannot be a Sparkly entry point".into());
    }
    let expanded = response::expand(&arguments)?;
    if expanded.iter().any(|arg| {
        arg.to_str().is_some_and(|arg| {
            matches!(arg, "-s" | "--strip-all")
                || arg
                    .strip_prefix("-Wl,")
                    .is_some_and(|args| args.split(',').any(|a| matches!(a, "-s" | "--strip-all")))
        })
    }) {
        return Err(
            "strip symbols only after finalization; disable Cargo profile strip=\"symbols\"".into(),
        );
    }
    let output = response::output_path(&expanded)?;
    // Linker probes must work without creating or trying to inspect an artifact.
    if output.is_none()
        && expanded
            .iter()
            .any(|arg| arg == "--version" || arg == "--help" || arg == "-v")
    {
        let status = Command::new(&real)
            .args(&arguments)
            .env("SPARKLY_LINK_ACTIVE", "1")
            .status()?;
        return Ok(status.code().unwrap_or(1));
    }

    let mut script = tempfile::Builder::new()
        .prefix("sparkly-link-")
        .suffix(".x")
        .tempfile()?;
    script.write_all(sparkly_build::metadata_script(bare_metal).as_bytes())?;
    script.flush()?;
    let mut command = Command::new(&real);
    if name == "rust-lld" {
        command.args(["-flavor", "gnu"]);
    }
    command.args(&arguments).env("SPARKLY_LINK_ACTIVE", "1");
    if flavor == "gcc" {
        // -Xlinker avoids splitting paths containing commas (unlike -Wl,...).
        command
            .arg("-Xlinker")
            .arg("-T")
            .arg("-Xlinker")
            .arg(script.path())
            .arg("-Xlinker")
            .arg("--build-id=sha1");
    } else {
        command.arg("-T").arg(script.path()).arg("--build-id=sha1");
    }
    let status = command.status()?;
    if !status.success() {
        return Ok(status.code().unwrap_or(1));
    }
    let output = output.unwrap_or_else(|| "a.out".into());
    // A failure must be visible to Cargo, which must not publish an unfinished ELF.
    if sparkly_build::needs_finalization(Path::new(&output))? {
        sparkly_build::finalize_elf(Path::new(&output))?;
    } else {
        // Tests, shared libraries and other uninstrumented outputs may legitimately
        // have no runtime slot, but any metric metadata must still be valid.
        sparkly_build::inspect_elf(Path::new(&output))?;
    }
    Ok(0)
}
