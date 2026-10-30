use crate::Sink;
use core::cell::Cell;
use critical_section::Mutex;

static SINK: Mutex<Cell<Option<&'static dyn Sink>>> = Mutex::new(Cell::new(None));

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum InstallError {
    AlreadyInstalled,
}

/// Installs the process-wide destination for metric macros and the defmt logger.
/// Call install_log separately when selecting Sparkly as the log facade backend.
pub fn install(recorder: &'static dyn Sink) -> Result<(), InstallError> {
    critical_section::with(|cs| {
        let slot = SINK.borrow(cs);
        if slot.get().is_some() {
            return Err(InstallError::AlreadyInstalled);
        }
        slot.set(Some(recorder));
        Ok(())
    })
}

pub fn sink() -> Option<&'static dyn Sink> {
    critical_section::with(|cs| SINK.borrow(cs).get())
}
