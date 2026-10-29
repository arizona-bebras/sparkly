//! Bounded protobuf codec. Generated types are kept behind a stable domain model.
#![no_std]

#[allow(clippy::all, nonstandard_style, unused, irrefutable_let_patterns)]
pub mod wire {
    include!(concat!(env!("OUT_DIR"), "/parcel.rs"));
}

mod encode;
pub use encode::ProtobufEncoder;
