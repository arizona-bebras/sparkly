use prost::Message;
use sparkly_symbolicator::{Symbols, parse_parcel_json, symbolicate, wire};
use std::{path::Path, process::Command, sync::OnceLock};

fn fixture() -> &'static Path {
    static DIRECTORY: OnceLock<tempfile::TempDir> = OnceLock::new();
    DIRECTORY
        .get_or_init(|| {
            let dir = tempfile::tempdir().unwrap();
            let status = Command::new("python3")
                .arg(concat!(
                    env!("CARGO_MANIFEST_DIR"),
                    "/tools/make_fixture.py"
                ))
                .arg(dir.path())
                .status()
                .unwrap();
            assert!(status.success());
            dir
        })
        .path()
}

fn inputs() -> (wire::Parcel, Symbols) {
    let dir = fixture();
    (
        parse_parcel_json(&std::fs::read(dir.join("parcel.json")).unwrap()).unwrap(),
        Symbols::load(dir.join("firmware.elf")).unwrap(),
    )
}

#[test]
fn decodes_dictionary_dwarf_defmt_and_crashes_from_a_real_elf() {
    let (parcel, symbols) = inputs();
    let out = symbolicate(&parcel, &symbols);
    assert!(out.diagnostics.is_empty(), "{:?}", out.diagnostics);
    assert_eq!(out.boots.len(), 1, "initial_boot is context, not a reset");
    assert_eq!(out.metrics[0].name.as_deref(), Some("device.temperature"));
    assert_eq!(out.metrics[0].value, Some(23.5));
    assert_eq!(out.logs[0].message.as_deref(), Some("sensor ready"));
    let frame = &out.logs[0].frames[0];
    assert_eq!(frame.function.as_deref(), Some("sampled_function"));
    assert!(frame.file.as_ref().unwrap().ends_with("firmware.c"));
    assert!(frame.line.unwrap() > 0);
    assert_eq!(out.logs[1].message.as_deref(), Some("temperature=42"));
    assert_eq!(out.logs[1].level.as_deref(), Some("INFO"));
    assert_eq!(out.crashes[0].stack, "0102");
    assert_eq!(out.crashes[0].registers["a0"], 42);
    assert_eq!(
        serde_json::to_value(&out).unwrap()["logs"][0]["event_index"],
        1
    );
    assert_eq!(
        serde_json::to_value(&out).unwrap(),
        serde_json::to_value(symbolicate(&parcel, &symbols)).unwrap()
    );
}

#[test]
fn boot_transition_switches_symbols_without_guessing_for_mismatched_builds() {
    let (mut parcel, symbols) = inputs();
    let correct = parcel.initial_boot.clone().unwrap();
    parcel.initial_boot.as_mut().unwrap().build_id = "different-build".into();
    // Raw text remains usable, but frames and metric names must not use another ELF.
    parcel.events.insert(
        2,
        wire::Event {
            monotonic_us: 0,
            payload: Some(wire::event::Payload::Boot(correct)),
        },
    );
    let out = symbolicate(&parcel, &symbols);
    assert!(out.metrics[0].name.is_none());
    assert_eq!(out.logs[0].message.as_deref(), Some("sensor ready"));
    assert!(out.logs[0].frames[0].function.is_none());
    assert_eq!(out.logs[1].message.as_deref(), Some("temperature=42"));
    assert!(out.crashes[0].frames[0].function.is_some());
    assert!(!out.diagnostics.is_empty());
}

#[test]
fn malformed_defmt_does_not_prevent_later_frames_from_decoding() {
    let (mut parcel, symbols) = inputs();
    let valid = parcel.events[2].clone();
    let Some(wire::event::Payload::Log(log)) = &mut parcel.events[2].payload else {
        panic!()
    };
    log.content = Some(wire::log::Content::DefmtLog(vec![255]));
    parcel.events.push(valid);
    let out = symbolicate(&parcel, &symbols);
    assert!(out.logs[1].message.is_none());
    assert_eq!(out.logs[1].defmt_raw_hex.as_deref(), Some("ff"));
    assert_eq!(out.logs[2].message.as_deref(), Some("temperature=42"));
    assert!(!out.diagnostics.is_empty());
}

#[test]
fn missing_context_and_unknown_integer_metrics_preserve_original_values() {
    let (mut parcel, symbols) = inputs();
    parcel.initial_boot = None;
    let Some(wire::event::Payload::Metric(metric)) = &mut parcel.events[0].payload else {
        panic!()
    };
    metric.measurement = Some(wire::metric::Measurement::CounterDelta(u64::MAX));
    let out = symbolicate(&parcel, &symbols);
    assert!(out.metrics[0].name.is_none());
    assert!(out.metrics[0].value.is_none());
    assert_eq!(out.metrics[0].counter_delta, Some(u64::MAX));
    assert_eq!(
        serde_json::to_value(out).unwrap()["metrics"][0]["counter_delta"].as_u64(),
        Some(u64::MAX)
    );
}

#[test]
fn agent_encoder_header_survives_the_json_handoff() {
    use sparkly_core::{Batch, BatchEncoder, Session};
    let session = Session::new([7; 16], "abcd", "v1", "reset").unwrap();
    let batch = Batch {
        id: [1; 24],
        initial_boot: Some(session),
        records: Default::default(),
        dropped_records: 2,
    };
    let mut bytes = [0; 1024];
    let len = sparkly_protocol::ProtobufEncoder
        .encode(&batch, &mut bytes)
        .unwrap();
    let parcel = wire::Parcel::decode(&bytes[..len]).unwrap();
    let json = serde_json::to_vec(&parcel).unwrap();
    let parcel: wire::Parcel = serde_json::from_slice(&json).unwrap();
    assert_eq!(parcel.initial_boot.unwrap().build_id, "abcd");
    assert!(parcel.events.is_empty());
    assert_eq!(parcel.dropped_records, 2);
}

#[test]
fn analytics_json_mapping_preserves_uint64_bytes_enums_and_oneofs() {
    // Same field layout and scalar mapping as Go protojson.Marshal.
    let parcel: wire::Parcel = serde_json::from_str(
        r#"{
        "batchId": "AQI=",
        "initialBoot": {"buildId": "abcd", "bootId": "AwQ="},
        "droppedRecords": "18446744073709551615",
        "events": [
            {"monotonicUs": "18446744073709551615", "metric": {
                "metricId": "18446744073709551615", "counterDelta": "18446744073709551615"}},
            {"log": {"rawLog": {"level": "LOG_LEVEL_WARN", "message": "test"},
                "traceStatus": "TRACE_STATUS_PARTIAL", "frames": [{"pc": "18446744073709551615"}]}},
            {"crash": {"sp": "0", "stack": "AP8=", "registers": {"pc": "18446744073709551615"}}}
        ]
    }"#,
    )
    .unwrap();
    assert_eq!(parcel.batch_id, [1, 2]);
    assert_eq!(parcel.initial_boot.unwrap().boot_id, [3, 4]);
    assert_eq!(parcel.dropped_records, u64::MAX);
    assert_eq!(parcel.events[0].monotonic_us, u64::MAX);
    let Some(wire::event::Payload::Metric(metric)) = &parcel.events[0].payload else {
        panic!()
    };
    assert_eq!(metric.metric_id, u64::MAX);
    assert_eq!(
        metric.measurement,
        Some(wire::metric::Measurement::CounterDelta(u64::MAX))
    );
    let Some(wire::event::Payload::Log(log)) = &parcel.events[1].payload else {
        panic!()
    };
    assert_eq!(log.trace_status, wire::TraceStatus::Partial as i32);
    assert_eq!(log.frames[0].pc, u64::MAX);
    let Some(wire::log::Content::RawLog(raw)) = &log.content else {
        panic!()
    };
    assert_eq!(raw.level, wire::LogLevel::Warn as i32);
    let Some(wire::event::Payload::Crash(crash)) = &parcel.events[2].payload else {
        panic!()
    };
    assert_eq!(crash.sp, Some(0));
    assert_eq!(crash.stack, [0, 255]);
    assert_eq!(crash.registers["pc"], u64::MAX);
}

#[test]
fn json_accepts_proto_field_names_and_rejects_invalid_values() {
    let parcel: wire::Parcel = serde_json::from_str(
        r#"{
        "initial_boot": {"build_id": "abcd"},
        "events": [{"monotonic_us": "42", "log": {"raw_log": {"level": 3}}}]
    }"#,
    )
    .unwrap();
    assert_eq!(parcel.initial_boot.unwrap().build_id, "abcd");
    assert_eq!(parcel.events[0].monotonic_us, 42);
    for json in [
        r#"{"batchId":"!invalid-base64"}"#,
        r#"{"droppedRecords":"18446744073709551616"}"#,
        r#"{"events":[{"metric":{},"log":{}}]}"#,
        r#"{"events":[{"log":{"rawLog":{"level":"INVALID_LEVEL"}}}]}"#,
        "not json",
    ] {
        assert!(
            serde_json::from_str::<wire::Parcel>(json).is_err(),
            "{json}"
        );
    }
}

#[test]
fn json_gauges_accept_fractional_exponent_and_special_values() {
    for (input, expected) in [
        ("23.5", 23.5),
        ("2e1", 20.0),
        ("-1.25e-3", -0.00125),
        ("0", 0.0),
        ("\"Infinity\"", f64::INFINITY),
    ] {
        let json = format!(r#"{{"events":[{{"metric":{{"gauge":{input}}}}}]}}"#);
        let parcel = parse_parcel_json(json.as_bytes()).unwrap();
        let Some(wire::event::Payload::Metric(metric)) = &parcel.events[0].payload else {
            panic!()
        };
        assert_eq!(
            metric.measurement,
            Some(wire::metric::Measurement::Gauge(expected))
        );
    }
}
