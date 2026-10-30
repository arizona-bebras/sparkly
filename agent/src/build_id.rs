use core::{fmt::Write, ptr};
use heapless::String;

/// Versioned slot patched by sparkly-link after the real linker succeeds.
/// Normal .rodata placement is owned entirely by the application's HAL/linker.
/// Layout: SPID, version=1, ID length, reserved(2), ID bytes padded to 32.
#[used]
#[unsafe(export_name = "__sparkly_build_id")]
#[unsafe(link_section = ".rodata.sparkly.build_id")]
static BUILD_ID_SLOT: [u8; 40] = [
    b'S', b'P', b'I', b'D', 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
];

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum BuildIdError {
    /// The application was linked without Sparkly's finalization step.
    NotFinalized,
    InvalidSlot,
}

/// Reads the ID from the finalized executable, without opening an ELF at runtime.
pub fn build_id() -> Result<String<64>, BuildIdError> {
    // SAFETY: the immutable static is allocated and aligned for its entire type.
    // Volatile is essential: LLVM must load the post-link bytes rather than
    // substituting the zero initializer, including under LTO.
    let slot = unsafe { ptr::read_volatile(ptr::addr_of!(BUILD_ID_SLOT)) };
    if &slot[..4] != b"SPID" || slot[4] != 1 || slot[6..8] != [0, 0] {
        return Err(BuildIdError::InvalidSlot);
    }
    let len = usize::from(slot[5]);
    if len == 0 {
        return Err(BuildIdError::NotFinalized);
    }
    if len > 32 || slot[8 + len..].iter().any(|byte| *byte != 0) {
        return Err(BuildIdError::InvalidSlot);
    }
    let mut id = String::new();
    for byte in &slot[8..8 + len] {
        write!(id, "{byte:02x}").unwrap();
    }
    Ok(id)
}
