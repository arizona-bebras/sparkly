use anyhow::{Context, Result};
use serde::Serialize;
use sha2::{Digest, Sha256};
use std::{collections::BTreeMap, path::Path};

#[allow(clippy::useless_borrows_in_formatting)] // pbjson-generated code
pub mod wire {
    include!(concat!(env!("OUT_DIR"), "/sparkly.agent.v1.rs"));
    include!(concat!(env!("OUT_DIR"), "/sparkly.agent.v1.serde.rs"));
}

/// Parse analytics' protobuf JSON without losing integer precision.
pub fn parse_parcel_json(bytes: &[u8]) -> serde_json::Result<wire::Parcel> {
    let mut value: serde_json::Value = serde_json::from_slice(bytes)?;
    // defmt-decoder enables serde_json/arbitrary_precision globally. Its numeric
    // representation conflicts with pbjson's untagged floating-point visitor.
    // Protobuf JSON accepts numeric strings: normalize gauge numbers to that
    // equivalent representation, including exponent notation, before decoding.
    if let Some(events) = value.get_mut("events").and_then(|v| v.as_array_mut()) {
        for event in events {
            if let Some(gauge) = event.get_mut("metric").and_then(|m| m.get_mut("gauge"))
                && gauge.is_number()
            {
                *gauge = serde_json::Value::String(gauge.to_string());
            }
        }
    }
    serde_json::from_value(value)
}

#[derive(Debug, Serialize)]
pub struct Common {
    pub event_index: usize,
    pub boot_id: Option<String>,
    pub monotonic_us: u64,
    pub build_id: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct BootRow {
    #[serde(flatten)]
    pub common: Common,
    pub reset_reason: String,
    pub raw_reset_reason: Option<u32>,
    pub firmware_version: String,
    pub agent_version: String,
}

#[derive(Debug, Serialize)]
pub struct MetricRow {
    #[serde(flatten)]
    pub common: Common,
    pub metric_id: u64,
    pub name: Option<String>,
    pub kind: Option<String>,
    pub value: Option<f64>,
    // Keep integer deltas exact, including values above f64's integer precision.
    pub counter_delta: Option<u64>,
    pub decode_status: String,
}

#[derive(Debug, Serialize)]
pub struct Frame {
    pub pc: u64,
    pub module: Option<String>,
    pub function: Option<String>,
    pub file: Option<String>,
    pub line: Option<u32>,
}

#[derive(Debug, Serialize)]
pub struct LogRow {
    #[serde(flatten)]
    pub common: Common,
    pub fingerprint: String,
    pub level: Option<String>,
    pub message: Option<String>,
    pub target: Option<String>,
    pub file: Option<String>,
    pub line: Option<u32>,
    pub truncated: bool,
    pub trace_status: String,
    pub frames: Vec<Frame>,
    pub defmt_raw_hex: Option<String>,
    pub decode_status: String,
}

#[derive(Debug, Serialize)]
pub struct CrashRow {
    #[serde(flatten)]
    pub common: Common,
    pub sp: Option<u64>,
    pub stack: String,
    pub registers: BTreeMap<String, u64>,
    pub reason: String,
    pub frames: Vec<Frame>,
}

#[derive(Debug, Serialize)]
pub struct Diagnostic {
    pub event_index: Option<usize>,
    pub message: String,
}

#[derive(Debug, Default, Serialize)]
pub struct Output {
    pub producer_batch_id: String,
    pub dropped_records: u64,
    pub symbol_build_id: String,
    pub boots: Vec<BootRow>,
    pub metrics: Vec<MetricRow>,
    pub logs: Vec<LogRow>,
    pub crashes: Vec<CrashRow>,
    pub diagnostics: Vec<Diagnostic>,
}

pub struct Symbols {
    build_id: String,
    metrics: Vec<sparkly_build::MetricDefinition>,
    loader: addr2line::Loader,
    defmt: Option<defmt_decoder::Table>,
    defmt_locations: defmt_decoder::Locations,
    defmt_error: Option<String>,
}

impl Symbols {
    pub fn load(path: impl AsRef<Path>) -> Result<Self> {
        let path = path.as_ref();
        let artifact = sparkly_build::inspect_elf(path)
            .map_err(|e| anyhow::anyhow!(e.to_string()))
            .context("validate symbol ELF")?;
        let bytes = std::fs::read(path).context("read symbol ELF")?;
        let (defmt, mut defmt_error) = match defmt_decoder::Table::parse(&bytes) {
            Ok(table) => (table, None),
            Err(error) => (None, Some(error.to_string())),
        };
        let defmt_locations = match defmt.as_ref().map(|table| table.get_locations(&bytes)) {
            Some(Ok(locations)) => locations,
            Some(Err(error)) => {
                defmt_error = Some(format!("source locations unavailable: {error}"));
                Default::default()
            }
            None => Default::default(),
        };
        Ok(Self {
            build_id: hex::encode(artifact.build_id),
            metrics: artifact.metrics,
            loader: addr2line::Loader::new(path).map_err(|e| anyhow::anyhow!(e.to_string()))?,
            defmt,
            defmt_locations,
            defmt_error,
        })
    }

    fn frames(
        &self,
        pcs: &[wire::Frame],
        matches: bool,
        event: usize,
        out: &mut Output,
    ) -> Vec<Frame> {
        let mut result = Vec::new();
        for pc in pcs {
            let start = result.len();
            if matches {
                let resolved = (|| -> Result<()> {
                    let mut frames = self
                        .loader
                        .find_frames(pc.pc)
                        .map_err(|e| anyhow::anyhow!(e.to_string()))?;
                    while let Some(frame) = frames.next()? {
                        result.push(Frame {
                            pc: pc.pc,
                            module: Some(self.build_id.clone()),
                            function: frame
                                .function
                                .and_then(|f| f.demangle().ok().map(|s| s.into_owned())),
                            file: frame
                                .location
                                .as_ref()
                                .and_then(|l| l.file.map(str::to_owned)),
                            line: frame.location.as_ref().and_then(|l| l.line),
                        });
                    }
                    Ok(())
                })();
                if let Err(error) = resolved {
                    diagnostic(out, event, format!("PC {:#x}: {error}", pc.pc));
                }
            }
            if result.len() == start {
                result.push(Frame {
                    pc: pc.pc,
                    module: matches.then(|| self.build_id.clone()),
                    function: matches
                        .then(|| self.loader.find_symbol(pc.pc).map(str::to_owned))
                        .flatten(),
                    file: None,
                    line: None,
                });
            }
        }
        result
    }
}

fn diagnostic(out: &mut Output, event: usize, message: impl Into<String>) {
    out.diagnostics.push(Diagnostic {
        event_index: Some(event),
        message: message.into(),
    });
}

fn common(index: usize, event: &wire::Event, boot: Option<&wire::Boot>) -> Common {
    Common {
        event_index: index,
        boot_id: boot
            .filter(|b| !b.boot_id.is_empty())
            .map(|b| hex::encode(&b.boot_id)),
        monotonic_us: event.monotonic_us,
        build_id: boot
            .filter(|b| !b.build_id.is_empty())
            .map(|b| b.build_id.clone()),
    }
}

pub fn symbolicate(parcel: &wire::Parcel, symbols: &Symbols) -> Output {
    let mut out = Output {
        producer_batch_id: hex::encode(&parcel.batch_id),
        dropped_records: parcel.dropped_records,
        symbol_build_id: symbols.build_id.clone(),
        ..Output::default()
    };
    if let Some(error) = &symbols.defmt_error {
        out.diagnostics.push(Diagnostic {
            event_index: None,
            message: format!("defmt metadata: {error}"),
        });
    }
    let mut boot = parcel.initial_boot.as_ref();
    for (index, event) in parcel.events.iter().enumerate() {
        if let Some(wire::event::Payload::Boot(next)) = &event.payload {
            boot = Some(next);
        }
        let common = common(index, event, boot);
        let matches = common.build_id.as_deref() == Some(symbols.build_id.as_str());
        if !matches {
            diagnostic(
                &mut out,
                index,
                "missing build context or build ID differs from supplied ELF; symbol decoding skipped",
            );
        }
        match &event.payload {
            Some(wire::event::Payload::Boot(boot)) => out.boots.push(BootRow {
                common,
                reset_reason: boot.reset_reason.clone(),
                raw_reset_reason: boot.raw_reset_reason,
                firmware_version: boot.firmware_version.clone(),
                agent_version: boot.agent_version.clone(),
            }),
            Some(wire::event::Payload::Metric(metric)) => {
                let definition = matches
                    .then(|| symbols.metrics.iter().find(|m| m.id == metric.metric_id))
                    .flatten();
                let (value, counter_delta, kind) = match metric.measurement {
                    Some(wire::metric::Measurement::Gauge(v)) => {
                        (v.is_finite().then_some(v), None, Some("gauge"))
                    }
                    Some(wire::metric::Measurement::CounterDelta(v)) => {
                        (None, Some(v), Some("counter_delta"))
                    }
                    None => (None, None, None),
                };
                let valid = definition.is_some_and(|d| {
                    matches!(
                        (d.kind, kind),
                        (1, Some("gauge")) | (2, Some("counter_delta"))
                    )
                }) && (value.is_some() || counter_delta.is_some());
                if !valid {
                    diagnostic(
                        &mut out,
                        index,
                        "metric dictionary/measurement missing, mismatched, or non-finite",
                    );
                }
                out.metrics.push(MetricRow {
                    common,
                    metric_id: metric.metric_id,
                    name: definition.map(|d| d.name.clone()),
                    kind: kind.map(str::to_owned),
                    value,
                    counter_delta,
                    decode_status: if valid { "decoded" } else { "unresolved" }.into(),
                });
            }
            Some(wire::event::Payload::Log(log)) => {
                let frames = symbols.frames(&log.frames, matches, index, &mut out);
                let mut row = LogRow {
                    common,
                    fingerprint: String::new(),
                    level: None,
                    message: None,
                    target: None,
                    file: None,
                    line: None,
                    truncated: false,
                    trace_status: wire::TraceStatus::try_from(log.trace_status)
                        .map(|s| {
                            s.as_str_name()
                                .trim_start_matches("TRACE_STATUS_")
                                .to_owned()
                        })
                        .unwrap_or_else(|_| "UNKNOWN".into()),
                    frames,
                    defmt_raw_hex: None,
                    decode_status: "unresolved".into(),
                };
                match &log.content {
                    Some(wire::log::Content::RawLog(raw)) => {
                        row.level = wire::LogLevel::try_from(raw.level)
                            .ok()
                            .map(|l| l.as_str_name().trim_start_matches("LOG_LEVEL_").to_owned());
                        row.message = Some(raw.message.clone());
                        row.target = Some(raw.target.clone());
                        row.truncated = raw.truncated;
                        row.decode_status = "decoded".into();
                    }
                    Some(wire::log::Content::DefmtLog(bytes)) => {
                        row.defmt_raw_hex = Some(hex::encode(bytes));
                        if matches && let Some(table) = &symbols.defmt {
                            let mut decoder = table.new_stream_decoder();
                            decoder.received(bytes);
                            match decoder.decode() {
                                Ok(frame) => {
                                    row.message = Some(frame.display_message().to_string());
                                    if let Some(location) =
                                        symbols.defmt_locations.get(&frame.index())
                                    {
                                        row.target = Some(location.module.clone());
                                        row.file = Some(location.file.display().to_string());
                                        row.line = location.line.try_into().ok();
                                    }
                                    row.level =
                                        frame.level().map(|l| format!("{l:?}").to_uppercase());
                                    row.decode_status = "decoded".into();
                                }
                                Err(error) => {
                                    diagnostic(&mut out, index, format!("defmt frame: {error}"))
                                }
                            }
                        }
                        if row.message.is_none() {
                            diagnostic(
                                &mut out,
                                index,
                                "defmt message unavailable; encoded bytes preserved",
                            );
                        }
                    }
                    None => diagnostic(&mut out, index, "log has no content"),
                }
                // Deterministic local fingerprint; not an incident grouping policy.
                row.fingerprint = hex::encode(Sha256::digest(
                    serde_json::to_vec(&(
                        &row.common.build_id,
                        &row.level,
                        &row.message,
                        &row.target,
                        &row.defmt_raw_hex,
                        log.frames.iter().map(|f| f.pc).collect::<Vec<_>>(),
                    ))
                    .expect("serializable fingerprint fields"),
                ));
                out.logs.push(row);
            }
            Some(wire::event::Payload::Crash(crash)) => {
                let frames = symbols.frames(&crash.frames, matches, index, &mut out);
                out.crashes.push(CrashRow {
                    common,
                    sp: crash.sp,
                    stack: hex::encode(&crash.stack),
                    registers: crash
                        .registers
                        .iter()
                        .map(|(k, v)| (k.clone(), *v))
                        .collect(),
                    reason: crash.reason.clone(),
                    frames,
                });
            }
            None => diagnostic(&mut out, index, "event has no supported payload"),
        }
    }
    out
}
