#![cfg(target_os = "linux")]
use object::{Object, ObjectSection, ObjectSymbol};
use sparkly_build::{finalize_elf, inspect_elf};
use std::{
    fs,
    path::{Path, PathBuf},
    process::{Command, Output},
};

fn source(dir: &Path, valid: bool) -> PathBuf {
    let name = "cpu.temperature";
    let id = name.bytes().fold(0xcbf29ce484222325_u64, |hash, byte| {
        (hash ^ u64::from(byte)).wrapping_mul(0x100000001b3)
    });
    let mut dictionary = b"SPKM".to_vec();
    dictionary.extend_from_slice(&[1, 1]);
    dictionary.extend_from_slice(&(name.len() as u16).to_le_bytes());
    dictionary.extend_from_slice(&id.to_le_bytes());
    dictionary.extend_from_slice(name.as_bytes());
    let array = dictionary
        .iter()
        .map(u8::to_string)
        .collect::<Vec<_>>()
        .join(",");
    let path = dir.join("fixture.c");
    fs::write(
        &path,
        format!(
            r#"
#include <stdio.h>
__attribute__((used, section(".rodata.sparkly.build_id")))
const unsigned char __sparkly_build_id[40] = {{ '{magic}', 'P', 'I', 'D', 1 }};
__attribute__((used, section(".sparkly.metrics")))
const unsigned char metrics[] = {{ {array} }};
int main(void) {{
    const volatile unsigned char *slot = __sparkly_build_id;
    if (slot[5] != 20) return 9;
    for (int i = 8; i < 28; ++i) printf("%02x", slot[i]);
    return 0;
}}
"#,
            magic = if valid { 'S' } else { 'X' }
        ),
    )
    .unwrap();
    path
}

fn wrapper() -> Command {
    let mut command = Command::new(env!("CARGO_BIN_EXE_sparkly-link"));
    command
        .env("SPARKLY_REAL_LINKER", "cc")
        .env("SPARKLY_LINKER_FLAVOR", "gcc")
        .env("SPARKLY_BARE_METAL", "false")
        .env_remove("SPARKLY_LINK_ACTIVE");
    command
}

fn success(output: Output) {
    assert!(
        output.status.success(),
        "{}",
        String::from_utf8_lossy(&output.stderr)
    );
}

#[test]
fn wrapper_builds_reproducible_executables_and_runtime_reads_the_patched_id() {
    let dir = tempfile::tempdir().unwrap();
    let source = source(dir.path(), true);
    let first = dir.path().join("first.elf");
    let second = dir.path().join("second.elf");
    for path in [&first, &second] {
        success(
            wrapper()
                .arg(&source)
                .args(["-g", "-O2", "-o"])
                .arg(path)
                .output()
                .unwrap(),
        );
    }
    let artifact = inspect_elf(&first).unwrap();
    assert_eq!(artifact.build_id, inspect_elf(&second).unwrap().build_id);
    assert_eq!(artifact.runtime_id.as_ref(), Some(&artifact.build_id));
    assert_eq!(artifact.metrics[0].name, "cpu.temperature");
    let output = Command::new(&first).output().unwrap();
    assert!(output.status.success());
    let id = artifact
        .build_id
        .iter()
        .map(|b| format!("{b:02x}"))
        .collect::<String>();
    assert_eq!(String::from_utf8(output.stdout).unwrap(), id);
    let before = fs::read(&first).unwrap();
    assert!(!finalize_elf(&first).unwrap().changed);
    assert_eq!(fs::read(&first).unwrap(), before);
}

#[test]
fn response_files_and_output_paths_with_spaces_are_forwarded() {
    let dir = tempfile::tempdir().unwrap();
    let source = source(dir.path(), true);
    let output = dir.path().join("output with spaces.elf");
    let response = dir.path().join("linker arguments.rsp");
    fs::write(
        &response,
        format!("'{}' -o '{}'", source.display(), output.display()),
    )
    .unwrap();
    success(
        wrapper()
            .arg(format!("@{}", response.display()))
            .output()
            .unwrap(),
    );
    assert!(inspect_elf(output).unwrap().runtime_id.is_some());
}

#[test]
fn finalizer_changes_only_the_fixed_slot_and_preserves_permissions() {
    use std::os::unix::fs::PermissionsExt;
    let dir = tempfile::tempdir().unwrap();
    let source = source(dir.path(), true);
    let output = dir.path().join("manual.elf");
    let script = dir.path().join("metadata.x");
    fs::write(&script, sparkly_build::metadata_script(false)).unwrap();
    success(
        Command::new("cc")
            .arg(&source)
            .arg("-o")
            .arg(&output)
            .arg("-Wl,--build-id=sha1")
            .arg("-Xlinker")
            .arg("-T")
            .arg("-Xlinker")
            .arg(&script)
            .output()
            .unwrap(),
    );
    let before = fs::read(&output).unwrap();
    assert!(
        inspect_elf(&output)
            .unwrap_err()
            .to_string()
            .contains("not finalized")
    );
    let elf = object::File::parse(before.as_slice()).unwrap();
    let symbol = elf
        .symbols()
        .find(|s| s.name() == Ok("__sparkly_build_id"))
        .unwrap();
    let section = elf
        .section_by_index(symbol.section_index().unwrap())
        .unwrap();
    let offset = (section.file_range().unwrap().0 + symbol.address() - section.address()) as usize;
    let mode = fs::metadata(&output).unwrap().permissions().mode();
    assert!(finalize_elf(&output).unwrap().changed);
    let after = fs::read(&output).unwrap();
    assert_eq!(before.len(), after.len());
    assert_eq!(&before[..offset], &after[..offset]);
    assert_eq!(&before[offset + 40..], &after[offset + 40..]);
    assert_eq!(mode, fs::metadata(&output).unwrap().permissions().mode());
    let mut corrupted = after;
    corrupted[offset + 8] ^= 1;
    fs::write(&output, &corrupted).unwrap();
    assert!(finalize_elf(&output).is_err());
    assert_eq!(fs::read(output).unwrap(), corrupted);
}

#[test]
fn invalid_slot_fails_the_link_instead_of_silently_emitting_bad_firmware() {
    let dir = tempfile::tempdir().unwrap();
    let source = source(dir.path(), false);
    let output = dir.path().join("bad.elf");
    let result = wrapper()
        .arg(source)
        .arg("-o")
        .arg(output)
        .output()
        .unwrap();
    assert!(!result.status.success());
    assert!(
        String::from_utf8_lossy(&result.stderr).contains("invalid runtime build-ID slot header")
    );
}

#[test]
fn missing_note_is_rejected_without_modifying_the_file() {
    let dir = tempfile::tempdir().unwrap();
    let source = source(dir.path(), true);
    let output = dir.path().join("no-note.elf");
    success(
        Command::new("cc")
            .arg(source)
            .arg("-o")
            .arg(&output)
            .arg("-Wl,--build-id=none")
            .output()
            .unwrap(),
    );
    let before = fs::read(&output).unwrap();
    assert!(
        finalize_elf(&output)
            .unwrap_err()
            .to_string()
            .contains("missing GNU build ID")
    );
    assert_eq!(fs::read(output).unwrap(), before);
}

#[test]
fn driver_exit_code_is_preserved_and_recursion_is_rejected() {
    use std::os::unix::fs::PermissionsExt;
    let dir = tempfile::tempdir().unwrap();
    let driver = dir.path().join("driver");
    fs::write(&driver, "#!/bin/sh\nexit 23\n").unwrap();
    fs::set_permissions(&driver, fs::Permissions::from_mode(0o755)).unwrap();
    assert_eq!(
        wrapper()
            .env("SPARKLY_REAL_LINKER", driver)
            .arg("input.o")
            .output()
            .unwrap()
            .status
            .code(),
        Some(23)
    );
    let recursive = wrapper()
        .env("SPARKLY_REAL_LINKER", env!("CARGO_BIN_EXE_sparkly-link"))
        .output()
        .unwrap();
    assert!(!recursive.status.success());
    assert!(String::from_utf8_lossy(&recursive.stderr).contains("real linker cannot"));
}

#[test]
fn symbol_stripping_is_rejected_before_it_hides_the_runtime_slot() {
    let result = wrapper()
        .args(["input.o", "-Wl,--strip-all", "-o", "unused.elf"])
        .output()
        .unwrap();
    assert!(!result.status.success());
    assert!(
        String::from_utf8_lossy(&result.stderr).contains("strip symbols only after finalization")
    );
}

#[test]
fn uninstrumented_linker_outputs_pass_through() {
    let dir = tempfile::tempdir().unwrap();
    let source = dir.path().join("plain.c");
    let output = dir.path().join("plain.elf");
    fs::write(&source, "int main(void) { return 0; }\n").unwrap();
    success(
        wrapper()
            .arg(source)
            .arg("-o")
            .arg(&output)
            .output()
            .unwrap(),
    );
    assert!(inspect_elf(output).unwrap().runtime_id.is_none());
}

#[test]
fn direct_gnu_ld_is_supported_without_a_compiler_driver() {
    let dir = tempfile::tempdir().unwrap();
    let source = dir.path().join("direct.c");
    let object = dir.path().join("direct.o");
    let output = dir.path().join("direct.elf");
    fs::write(
        &source,
        r#"
__attribute__((used, section(".rodata.sparkly.build_id")))
const unsigned char __sparkly_build_id[40] = {'S', 'P', 'I', 'D', 1};
void _start(void) {
    const volatile unsigned char *slot = __sparkly_build_id;
    while (slot[5]) {}
}
"#,
    )
    .unwrap();
    success(
        Command::new("cc")
            .arg("-c")
            .arg(source)
            .arg("-o")
            .arg(&object)
            .output()
            .unwrap(),
    );
    success(
        wrapper()
            .env("SPARKLY_REAL_LINKER", "ld")
            .env("SPARKLY_LINKER_FLAVOR", "ld")
            .arg(object)
            .arg("-o")
            .arg(&output)
            .output()
            .unwrap(),
    );
    let artifact = inspect_elf(output).unwrap();
    assert_eq!(artifact.runtime_id, Some(artifact.build_id));
}
