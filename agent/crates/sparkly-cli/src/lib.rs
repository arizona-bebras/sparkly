//! Project setup and a transparent GNU ELF linker entry point.
pub mod init;
pub mod linker;
mod response;

pub use sparkly_build::{Error, Result};
