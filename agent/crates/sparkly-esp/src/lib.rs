//! ESP hardware services. Select `esp32s3` in the final Xtensa application.
#![no_std]

pub use sparkly_agent::{BuildIdError, build_id};

#[cfg(all(feature = "esp32s3", target_arch = "xtensa"))]
mod esp32s3;
#[cfg(all(feature = "esp32s3", target_arch = "xtensa"))]
pub use esp32s3::*;
