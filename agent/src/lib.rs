//! Portable, allocation-free telemetry capture and explicit background export.
#![no_std]

extern crate self as sparkly_agent;
#[cfg(test)]
extern crate std;

mod build_id;
#[cfg(feature = "defmt")]
mod defmt_logger;
mod exporter;
mod global;
#[cfg(feature = "log")]
mod log_logger;
mod metrics;
mod recorder;

pub use build_id::{BuildIdError, build_id};
pub use exporter::*;
pub use global::{InstallError, install, sink};
#[cfg(feature = "log")]
pub use log_logger::install_log;
pub use metrics::{Counter, Gauge};
pub use recorder::*;
pub use sparkly_core::*;
pub use sparkly_macros::{counter, metric};
pub use sparkly_protocol::ProtobufEncoder;
