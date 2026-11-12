use crate::wire::sparkly_::agent_::v1_ as pb;
use micropb::{MessageEncode, PbEncoder, Tag, WIRE_TYPE_LEN, WIRE_TYPE_VARINT};
use sparkly_core::*;

/// Streams one generated Event at a time, avoiding a second full Batch on stack.
pub struct ProtobufEncoder;

impl BatchEncoder for ProtobufEncoder {
    fn encode(&self, batch: &Batch, output: &mut [u8]) -> Result<usize, EncodeError> {
        let capacity = output.len();
        let mut encoder = PbEncoder::new(output);
        encode_batch(batch, &mut encoder).map_err(|()| EncodeError::BufferTooSmall)?;
        Ok(capacity - encoder.as_writer().len())
    }
}

fn encode_batch(batch: &Batch, encoder: &mut PbEncoder<&mut [u8]>) -> Result<(), ()> {
    // Only the Parcel envelope is written explicitly; generated codecs encode all
    // event payloads. The round-trip tests guard these schema field numbers.
    encoder.encode_tag(Tag::from_parts(1, WIRE_TYPE_LEN))?;
    encoder.encode_bytes(&batch.id)?;
    encoder.encode_tag(Tag::from_parts(3, WIRE_TYPE_VARINT))?;
    encoder.encode_varint64(batch.dropped_records)?;
    if let Some(boot) = &batch.initial_boot {
        let boot = make_boot(boot)?;
        encoder.encode_tag(Tag::from_parts(4, WIRE_TYPE_LEN))?;
        encoder.encode_varint32(boot.compute_size() as u32)?;
        boot.encode(encoder)?;
    }
    for record in &batch.records {
        let payload = match &record.payload {
            Payload::Boot(session) => pb::Event_::Payload::Boot(make_boot(session)?),
            Payload::Metric { id, value } => pb::Event_::Payload::Metric(pb::Metric {
                metric_id: *id,
                measurement: Some(match value {
                    MetricValue::Gauge(value) => pb::Metric_::Measurement::Gauge(*value),
                    MetricValue::CounterDelta(value) => {
                        pb::Metric_::Measurement::CounterDelta(*value)
                    }
                }),
            }),
            Payload::Text(log) => pb::Event_::Payload::Log(make_log(
                pb::Log_::Content::RawLog(pb::RawLog {
                    level: match log.level {
                        Level::Trace => pb::LogLevel::Trace,
                        Level::Debug => pb::LogLevel::Debug,
                        Level::Info => pb::LogLevel::Info,
                        Level::Warn => pb::LogLevel::Warn,
                        Level::Error => pb::LogLevel::Error,
                    },
                    message: log.message.clone(),
                    target: log.target.clone(),
                    truncated: log.truncated,
                }),
                &log.trace,
            )),
            Payload::Defmt(log) => pb::Event_::Payload::Log(make_log(
                pb::Log_::Content::DefmtLog(log.frame.clone()),
                &log.trace,
            )),
        };
        write_event(
            encoder,
            &pb::Event {
                monotonic_us: record.monotonic_us,
                payload: Some(payload),
            },
        )?;
    }
    Ok(())
}

fn make_log(content: pb::Log_::Content, trace: &StackTrace) -> pb::Log {
    pb::Log {
        content: Some(content),
        frames: trace.pcs.iter().map(|pc| pb::Frame { pc: *pc }).collect(),
        trace_status: match trace.status {
            TraceStatus::Unsupported => pb::TraceStatus::Unsupported,
            TraceStatus::Complete => pb::TraceStatus::Complete,
            TraceStatus::Truncated => pb::TraceStatus::Truncated,
            TraceStatus::Partial => pb::TraceStatus::Partial,
        },
    }
}

fn write_event(encoder: &mut PbEncoder<&mut [u8]>, event: &pb::Event) -> Result<(), ()> {
    encoder.encode_tag(Tag::from_parts(2, WIRE_TYPE_LEN))?;
    encoder.encode_varint32(event.compute_size() as u32)?;
    event.encode(encoder)
}

fn make_boot(session: &Session) -> Result<pb::Boot, ()> {
    let mut boot = pb::Boot {
        boot_id: session.boot_id.as_slice().try_into().map_err(|_| ())?,
        build_id: session.build_id.clone(),
        firmware_version: session
            .firmware_version
            .as_str()
            .try_into()
            .map_err(|_| ())?,
        agent_version: session.agent_version.as_str().try_into().map_err(|_| ())?,
        reset_reason: session.reset_reason.clone(),
        ..Default::default()
    };
    if let Some(reason) = session.raw_reset_reason {
        boot.set_raw_reset_reason(reason);
    }
    Ok(boot)
}
