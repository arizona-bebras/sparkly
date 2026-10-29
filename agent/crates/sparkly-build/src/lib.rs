//! Host-side ELF validation and fixed-size post-link build-ID finalization.
//! No application build.rs or platform memory-map replacement is required.
mod elf;
mod metrics;

pub use elf::{Artifact, FinalizedArtifact, finalize_elf, inspect_elf, needs_finalization};
pub use metrics::MetricDefinition;

pub type Error = Box<dyn std::error::Error + Send + Sync>;
pub type Result<T> = std::result::Result<T, Error>;

/// ELF-only metadata. Host linkers need INSERT to retain their default layout;
/// bare-metal applications already supply a complete linker script.
pub fn metadata_script(bare_metal: bool) -> String {
    let layout = include_str!("../linker/metadata.x");
    if bare_metal {
        layout.replace("INSERT AFTER .comment;", "")
    } else {
        layout.to_owned()
    }
}
