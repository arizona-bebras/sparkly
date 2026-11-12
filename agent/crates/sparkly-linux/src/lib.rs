//! Linux clock, stack capture, HTTP transport and rustls identity adapters.
//! Applications own recorder installation, transport scheduling and log setup.
mod clock;
pub mod identity;
mod stacktrace;
mod transport;

pub use clock::MonotonicClock;
pub use stacktrace::LinuxBacktrace;
pub use transport::HttpTransport;
