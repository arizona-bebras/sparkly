use crate::{Level, sink};

struct Logger;
static LOGGER: Logger = Logger;

/// Selects Sparkly as the sole global log backend. Returns an error if the
/// application already installed another backend; it is never silently replaced.
pub fn install_log(level: log::LevelFilter) -> Result<(), log::SetLoggerError> {
    #[cfg(target_has_atomic = "ptr")]
    {
        log::set_logger(&LOGGER)?;
        log::set_max_level(level);
        Ok(())
    }
    #[cfg(not(target_has_atomic = "ptr"))]
    critical_section::with(|_| {
        // SAFETY: on targets without atomics the application supplies a global
        // critical section excluding threads/interrupts. All other calls to log's
        // unsafe setters must also uphold their no-concurrent-setter contract.
        unsafe {
            log::set_logger_racy(&LOGGER)?;
            log::set_max_level_racy(level);
        }
        Ok(())
    })
}

impl log::Log for Logger {
    fn enabled(&self, metadata: &log::Metadata<'_>) -> bool {
        metadata.level() <= log::max_level()
    }

    fn log(&self, record: &log::Record<'_>) {
        if !self.enabled(record.metadata()) {
            return;
        }
        let Some(recorder) = sink() else {
            return;
        };
        let level = match record.level() {
            log::Level::Error => Level::Error,
            log::Level::Warn => Level::Warn,
            log::Level::Info => Level::Info,
            log::Level::Debug => Level::Debug,
            log::Level::Trace => Level::Trace,
        };
        let _ = recorder.text(level, record.target(), *record.args());
    }

    // Capture has no transport access. Applications explicitly drive Exporter.
    fn flush(&self) {}
}
