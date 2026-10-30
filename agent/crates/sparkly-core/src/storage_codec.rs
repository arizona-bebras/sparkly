//! Private RAM representation, not a persistence or wire format. Only bytes
//! written here are decoded here; lengths and tags are therefore trusted.
use crate::*;
use heapless::{String, Vec};

pub(crate) const HEADER: usize = 11; // u16 total length, u8 tag, u64 timestamp

pub(crate) fn visit(record: &Record, mut write: impl FnMut(&[u8])) {
    fn string(value: &str, write: &mut impl FnMut(&[u8])) {
        write(&(value.len() as u16).to_le_bytes());
        write(value.as_bytes());
    }
    fn trace(value: &StackTrace, write: &mut impl FnMut(&[u8])) {
        write(&[
            match value.status {
                TraceStatus::Unsupported => 0,
                TraceStatus::Complete => 1,
                TraceStatus::Truncated => 2,
                TraceStatus::Partial => 3,
            },
            value.pcs.len() as u8,
        ]);
        for pc in &value.pcs {
            write(&pc.to_le_bytes());
        }
    }
    write(&[match record.payload {
        Payload::Boot(_) => 0,
        Payload::Text(_) => 1,
        Payload::Defmt(_) => 2,
        Payload::Metric { .. } => 3,
    }]);
    write(&record.monotonic_us.to_le_bytes());
    match &record.payload {
        Payload::Boot(session) => {
            write(&session.boot_id);
            for value in [
                &session.build_id[..],
                &session.firmware_version[..],
                &session.agent_version[..],
                &session.reset_reason[..],
            ] {
                string(value, &mut write);
            }
            write(&[u8::from(session.raw_reset_reason.is_some())]);
            if let Some(reason) = session.raw_reset_reason {
                write(&reason.to_le_bytes());
            }
        }
        Payload::Text(log) => {
            write(&[
                match log.level {
                    Level::Trace => 0,
                    Level::Debug => 1,
                    Level::Info => 2,
                    Level::Warn => 3,
                    Level::Error => 4,
                },
                u8::from(log.truncated),
            ]);
            string(&log.target, &mut write);
            string(&log.message, &mut write);
            trace(&log.trace, &mut write);
        }
        Payload::Defmt(log) => {
            write(&(log.frame.len() as u16).to_le_bytes());
            write(&log.frame);
            trace(&log.trace, &mut write);
        }
        Payload::Metric { id, value } => {
            write(&id.to_le_bytes());
            match value {
                MetricValue::Gauge(value) => {
                    write(&[0]);
                    write(&value.to_le_bytes());
                }
                MetricValue::CounterDelta(value) => {
                    write(&[1]);
                    write(&value.to_le_bytes());
                }
            }
        }
    }
}

// Read wrapped records directly from two slices without a full-record scratch
// buffer or moving queued bytes. Tags and lengths come only from visit().
struct Reader<'a>(core::iter::Chain<core::slice::Iter<'a, u8>, core::slice::Iter<'a, u8>>);
impl Reader<'_> {
    fn byte(&mut self) -> u8 {
        *self.0.next().unwrap()
    }
    fn array<const N: usize>(&mut self) -> [u8; N] {
        core::array::from_fn(|_| self.byte())
    }
    fn u16(&mut self) -> usize {
        u16::from_le_bytes(self.array()) as usize
    }
    fn u64(&mut self) -> u64 {
        u64::from_le_bytes(self.array())
    }
    fn vec<const N: usize>(&mut self, len: usize) -> Vec<u8, N> {
        (0..len).map(|_| self.byte()).collect()
    }
    fn string<const N: usize>(&mut self) -> String<N> {
        let len = self.u16();
        String::from_utf8(self.vec(len)).unwrap()
    }
    fn trace(&mut self) -> StackTrace {
        let status = match self.byte() {
            0 => TraceStatus::Unsupported,
            1 => TraceStatus::Complete,
            2 => TraceStatus::Truncated,
            3 => TraceStatus::Partial,
            _ => unreachable!(),
        };
        let count = self.byte();
        let mut pcs = Vec::new();
        for _ in 0..count {
            pcs.push(self.u64()).unwrap();
        }
        StackTrace { pcs, status }
    }
}

pub(crate) fn decode(first: &[u8], second: &[u8]) -> Record {
    let mut reader = Reader(first.iter().chain(second.iter()));
    let _len = reader.u16();
    let tag = reader.byte();
    let monotonic_us = reader.u64();
    let payload = match tag {
        0 => Payload::Boot(Session {
            boot_id: reader.array(),
            build_id: reader.string(),
            firmware_version: reader.string(),
            agent_version: reader.string(),
            reset_reason: reader.string(),
            raw_reset_reason: if reader.byte() != 0 {
                Some(u32::from_le_bytes(reader.array()))
            } else {
                None
            },
        }),
        1 => Payload::Text(TextLog {
            level: match reader.byte() {
                0 => Level::Trace,
                1 => Level::Debug,
                2 => Level::Info,
                3 => Level::Warn,
                4 => Level::Error,
                _ => unreachable!(),
            },
            truncated: reader.byte() != 0,
            target: reader.string(),
            message: reader.string(),
            trace: reader.trace(),
        }),
        2 => Payload::Defmt(DefmtLog {
            frame: {
                let len = reader.u16();
                reader.vec(len)
            },
            trace: reader.trace(),
        }),
        3 => Payload::Metric {
            id: reader.u64(),
            value: match reader.byte() {
                0 => MetricValue::Gauge(f64::from_bits(reader.u64())),
                1 => MetricValue::CounterDelta(reader.u64()),
                _ => unreachable!(),
            },
        },
        _ => unreachable!(),
    };
    debug_assert!(reader.0.next().is_none());
    Record {
        monotonic_us,
        payload,
    }
}
