use std::{path::Path, process::Command, sync::OnceLock};

pub fn fixture() -> &'static Path {
    static DIRECTORY: OnceLock<tempfile::TempDir> = OnceLock::new();
    DIRECTORY
        .get_or_init(|| {
            let dir = tempfile::tempdir().unwrap();
            let status = Command::new("python3")
                .arg(concat!(
                    env!("CARGO_MANIFEST_DIR"),
                    "/tools/make_fixture.py"
                ))
                .arg(dir.path())
                .status()
                .unwrap();
            assert!(status.success());
            dir
        })
        .path()
}
