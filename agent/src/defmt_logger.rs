//! One global defmt channel. The application's SMP-safe critical section remains
//! held from acquire through release, as required by defmt's logger contract.
use crate::{MAX_DEFMT, sink};
use core::cell::{Cell, RefCell};
use critical_section::{Mutex, RestoreState};
use heapless::Vec;

#[defmt::global_logger]
struct Logger;

struct Frame {
    encoder: defmt::Encoder,
    bytes: Vec<u8, MAX_DEFMT>,
    overflow: bool,
}

impl Frame {
    const fn new() -> Self {
        Self {
            encoder: defmt::Encoder::new(),
            bytes: Vec::new(),
            overflow: false,
        }
    }
}

static FRAME: Mutex<RefCell<Frame>> = Mutex::new(RefCell::new(Frame::new()));
static RESTORE: Mutex<Cell<Option<RestoreState>>> = Mutex::new(Cell::new(None));

fn append(bytes: &mut Vec<u8, MAX_DEFMT>, overflow: &mut bool, fragment: &[u8]) {
    if !*overflow && bytes.extend_from_slice(fragment).is_err() {
        *overflow = true;
    }
}

// SAFETY: acquire enters the global critical section and release exits it on the
// same context. All shared state is accessed under that section. Recursive acquire
// panics as required by defmt; no other execution context can interleave a frame.
unsafe impl defmt::Logger for Logger {
    fn acquire() {
        // SAFETY: the restore token is saved below and consumed exactly once in release.
        let restore = unsafe { critical_section::acquire() };
        let recursive = critical_section::with(|cs| RESTORE.borrow(cs).get().is_some());
        if recursive {
            // SAFETY: balance this recursive acquisition before reporting the error.
            unsafe {
                critical_section::release(restore);
            }
            panic!("recursive defmt logging");
        }
        critical_section::with(|cs| {
            RESTORE.borrow(cs).set(Some(restore));
            let mut frame = FRAME.borrow(cs).borrow_mut();
            frame.bytes.clear();
            frame.overflow = false;
            let Frame {
                encoder,
                bytes,
                overflow,
            } = &mut *frame;
            encoder.start_frame(|fragment| append(bytes, overflow, fragment));
        });
    }

    unsafe fn write(fragment: &[u8]) {
        critical_section::with(|cs| {
            let mut frame = FRAME.borrow(cs).borrow_mut();
            let Frame {
                encoder,
                bytes,
                overflow,
            } = &mut *frame;
            encoder.write(fragment, |fragment| append(bytes, overflow, fragment));
        });
    }

    unsafe fn release() {
        let restore = critical_section::with(|cs| {
            let mut frame = FRAME.borrow(cs).borrow_mut();
            let Frame {
                encoder,
                bytes,
                overflow,
            } = &mut *frame;
            encoder.end_frame(|fragment| append(bytes, overflow, fragment));
            if let Some(recorder) = sink() {
                if *overflow {
                    recorder.reject();
                } else {
                    let _ = recorder.defmt(bytes);
                }
            }
            RESTORE
                .borrow(cs)
                .take()
                .expect("defmt release without acquire")
        });
        // SAFETY: this is the token from the matching acquire on this context.
        unsafe {
            critical_section::release(restore);
        }
    }

    // Best effort: buffered frames remain queued. Network work cannot safely run
    // while defmt holds its critical section; the exporter is driven separately.
    unsafe fn flush() {}
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{BatchSource, Clock, NoBacktrace, Payload, RamStorage, Recorder, Session};
    use defmt::Logger as _;
    use std::sync::OnceLock;

    struct TestClock;
    impl Clock for TestClock {
        fn now_us(&self) -> u64 {
            1
        }
    }
    static RECORDER: OnceLock<Recorder<RamStorage<4>, TestClock, NoBacktrace>> = OnceLock::new();

    #[test]
    fn fragments_commit_as_one_frame_and_overflow_recovers_on_next_frame() {
        let recorder = RECORDER.get_or_init(|| {
            Recorder::new(
                Session::new([1; 16], "test", "test", "test").unwrap(),
                RamStorage::new(),
                TestClock,
                NoBacktrace,
            )
            .unwrap()
        });
        crate::install(recorder).unwrap();
        Logger::acquire();
        // SAFETY: matching acquire/release on this thread, as defmt's macros do.
        unsafe {
            Logger::write(&[1, 2]);
            Logger::write(&[3, 4]);
            Logger::release();
        }
        Logger::acquire();
        unsafe {
            Logger::write(&[1; MAX_DEFMT * 2]);
            Logger::release();
        }
        Logger::acquire();
        unsafe {
            Logger::write(&[5, 6]);
            Logger::release();
        }
        let batch = recorder.take_batch().unwrap().unwrap();
        assert_eq!(batch.records.len(), 3);
        assert!(matches!(batch.records[0].payload, Payload::Boot(_)));
        assert_eq!(batch.dropped_records, 1);
        let mut reference = defmt::Encoder::new();
        let mut expected = std::vec::Vec::new();
        reference.start_frame(|b| expected.extend_from_slice(b));
        reference.write(&[1, 2, 3, 4], |b| expected.extend_from_slice(b));
        reference.end_frame(|b| expected.extend_from_slice(b));
        let Payload::Defmt(log) = &batch.records[1].payload else {
            panic!()
        };
        assert_eq!(log.frame.as_slice(), expected.as_slice());
    }
}
