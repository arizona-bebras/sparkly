//! Linux host stack capture. PCs are translated to ELF virtual addresses so the
//! single-ELF symbolicator can resolve them even when ASLR relocates a PIE.
use sparkly_core::{Backtrace, StackTrace, TraceStatus};
use std::io;

pub struct LinuxBacktrace {
    load_bias: usize,
    executable_ranges: heapless::Vec<(usize, usize), 16>,
}

impl LinuxBacktrace {
    pub fn new() -> io::Result<Self> {
        #[cfg(target_os = "linux")]
        {
            let mut result = Self {
                load_bias: 0,
                executable_ranges: heapless::Vec::new(),
            };
            // SAFETY: the callback only borrows result during this synchronous
            // call. It reads loader-owned program headers and never retains them.
            unsafe {
                libc::dl_iterate_phdr(Some(main_executable), (&mut result as *mut Self).cast());
            }
            if result.executable_ranges.is_empty() {
                return Err(io::Error::other(
                    "cannot locate host executable code segments",
                ));
            }
            // Initialize the unwinder before installing the recorder; capture
            // only visits raw frames and writes to fixed-capacity storage.
            backtrace::trace(|_| true);
            Ok(result)
        }
        #[cfg(not(target_os = "linux"))]
        Err(io::Error::new(
            io::ErrorKind::Unsupported,
            "host stack capture currently requires Linux",
        ))
    }

    fn elf_pc(&self, return_address: usize) -> Option<u64> {
        // Unwind IPs point after the call instruction. Resolve inside the call
        // site, including when a return address is exactly at a range boundary.
        let address = return_address.checked_sub(1)?;
        self.executable_ranges
            .iter()
            .any(|&(start, end)| (start..end).contains(&address))
            .then(|| (address - self.load_bias) as u64)
    }
}

#[cfg(target_os = "linux")]
unsafe extern "C" fn main_executable(
    info: *mut libc::dl_phdr_info,
    _size: libc::size_t,
    data: *mut libc::c_void,
) -> libc::c_int {
    // SAFETY: dl_iterate_phdr provides a valid info pointer and a NUL-terminated
    // module name for the duration of the callback.
    let info = unsafe { &*info };
    if !unsafe { std::ffi::CStr::from_ptr(info.dlpi_name) }.is_empty() {
        return 0;
    }
    // SAFETY: data points to the exclusive LinuxBacktrace passed by new().
    let result = unsafe { &mut *data.cast::<LinuxBacktrace>() };
    result.load_bias = info.dlpi_addr as usize;
    // SAFETY: the loader supplies dlpi_phnum readable program headers.
    let headers = unsafe { std::slice::from_raw_parts(info.dlpi_phdr, info.dlpi_phnum as usize) };
    for header in headers {
        if header.p_type == libc::PT_LOAD && header.p_flags & libc::PF_X != 0 {
            let Some(start) = result.load_bias.checked_add(header.p_vaddr as usize) else {
                continue;
            };
            let Some(end) = start.checked_add(header.p_memsz as usize) else {
                continue;
            };
            if result.executable_ranges.push((start, end)).is_err() {
                result.executable_ranges.clear();
                break;
            }
        }
    }
    1 // Main executable found; do not collect shared-library ranges.
}

impl Backtrace for LinuxBacktrace {
    #[inline(never)]
    fn capture(&self) -> StackTrace {
        let mut trace = StackTrace {
            status: TraceStatus::Partial,
            ..StackTrace::default()
        };
        let mut visited = 0;
        backtrace::trace(|frame| {
            visited += 1;
            if let Some(pc) = self.elf_pc(frame.ip() as usize)
                && trace.pcs.push(pc).is_err()
            {
                trace.status = TraceStatus::Truncated;
                return false;
            }
            // Bound the walk even when most frames belong to shared libraries.
            if visited >= 128 {
                trace.status = TraceStatus::Truncated;
                return false;
            }
            true
        });
        trace
    }
}

#[cfg(all(test, target_os = "linux"))]
mod tests {
    use super::*;

    #[test]
    fn capture_produces_elf_addresses_from_the_current_executable() {
        let capture = LinuxBacktrace::new().unwrap();
        let trace = capture.capture();
        assert!(!trace.pcs.is_empty());
        assert_ne!(trace.status, TraceStatus::Unsupported);
        let mut resolved_source = false;
        for pc in trace.pcs {
            let runtime_pc = pc as usize + capture.load_bias;
            assert!(
                capture
                    .executable_ranges
                    .iter()
                    .any(|&(start, end)| (start..end).contains(&runtime_pc))
            );
            backtrace::resolve(runtime_pc as *mut std::ffi::c_void, |symbol| {
                resolved_source |= symbol
                    .filename()
                    .is_some_and(|path| path.ends_with("stacktrace.rs"));
            });
        }
        assert!(
            resolved_source,
            "captured PCs must resolve to this source file"
        );
    }

    #[test]
    fn normalizes_pie_return_addresses_and_excludes_other_modules() {
        let capture = LinuxBacktrace {
            load_bias: 0x10000,
            executable_ranges: [(0x12000, 0x13000)].into_iter().collect(),
        };
        assert_eq!(capture.elf_pc(0x12010), Some(0x200f));
        assert_eq!(capture.elf_pc(0x13000), Some(0x2fff));
        assert_eq!(capture.elf_pc(0x12000), None);
        assert_eq!(capture.elf_pc(0x50000), None);
        assert_eq!(capture.elf_pc(0), None);
    }

    #[inline(never)]
    fn deep_capture(capture: &LinuxBacktrace, depth: usize) -> StackTrace {
        if depth == 0 {
            capture.capture()
        } else {
            // Keep recursion frames alive even in optimized test builds.
            std::hint::black_box(deep_capture(capture, depth - 1))
        }
    }

    #[test]
    fn deep_stacks_are_bounded_and_marked_truncated() {
        let capture = LinuxBacktrace::new().unwrap();
        let trace = deep_capture(&capture, 32);
        assert_eq!(trace.pcs.len(), sparkly_core::MAX_FRAMES);
        assert_eq!(trace.status, TraceStatus::Truncated);
    }
}
