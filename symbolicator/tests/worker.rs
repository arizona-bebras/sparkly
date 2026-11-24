mod common;
#[path = "common/request.rs"]
mod request;

use std::{cell::RefCell, rc::Rc};

use anyhow::{Result, bail};
use request::request;
use serde_json::{Value, json};
use sparkly_symbolicator::{
    Symbols,
    application::{SymbolProvider, SymbolicationService},
    infrastructure::{
        fixed_elf::FixedElfProvider,
        messages::{MAX_REQUEST_BYTES, decode_request, encode_result},
    },
    symbolicate,
};

#[test]
fn request_validation_rejects_poison_messages() {
    for field in [
        "schema_version",
        "batch_id",
        "device_id",
        "park_id",
        "received_at",
        "payload",
    ] {
        let mut message = request();
        message.as_object_mut().unwrap().remove(field);
        assert!(
            decode_request(
                &serde_json::to_vec(&message).unwrap(),
                Some("application/json"),
                None
            )
            .is_err(),
            "missing {field}"
        );
    }
    for (field, value) in [
        ("schema_version", json!(2)),
        ("batch_id", json!("00000000-0000-0000-0000-000000000000")),
        ("device_id", json!("00000000-0000-0000-0000-000000000000")),
        ("park_id", json!("bad")),
        ("received_at", json!("0001-01-01T00:00:00Z")),
        ("received_at", json!("yesterday")),
        ("payload", Value::Null),
        ("payload", json!([])),
        ("payload", json!({"events":"bad"})),
        ("payload", json!({"batchId":"invalid base64!"})),
    ] {
        let mut message = request();
        message[field] = value;
        assert!(
            decode_request(
                &serde_json::to_vec(&message).unwrap(),
                Some("application/json"),
                None
            )
            .is_err(),
            "invalid {field}"
        );
    }
    let body = serde_json::to_vec(&request()).unwrap();
    assert!(decode_request(&body, Some("text/plain"), None).is_err());
    assert!(decode_request(&body, Some("application/json"), Some("another-id")).is_err());
    assert!(
        decode_request(
            &vec![b' '; MAX_REQUEST_BYTES + 1],
            Some("application/json"),
            None
        )
        .is_err()
    );
    let trailing = [body.as_slice(), b" {}"].concat();
    assert!(decode_request(&trailing, Some("application/json"), None).is_err());
}

#[tokio::test]
async fn fixed_provider_matches_existing_decoder_and_preserves_metadata() {
    let message = request();
    let task = decode_request(
        &serde_json::to_vec(&message).unwrap(),
        Some("application/json"),
        message["batch_id"].as_str(),
    )
    .unwrap();
    let expected = symbolicate(
        &task.parcel,
        &Symbols::load(common::fixture().join("firmware.elf")).unwrap(),
    );
    let provider = FixedElfProvider::load(common::fixture().join("firmware.elf")).unwrap();
    let result = SymbolicationService::new(provider)
        .execute(task)
        .await
        .unwrap();
    assert_eq!(
        serde_json::to_value(&result.output).unwrap(),
        serde_json::to_value(expected).unwrap()
    );
    let encoded: Value = serde_json::from_slice(&encode_result(&result).unwrap()).unwrap();
    for field in ["batch_id", "device_id", "park_id", "received_at"] {
        assert_eq!(encoded[field], message[field]);
    }
    assert_eq!(encoded["schema_version"], 1);
    assert_eq!(encoded["result_version"], 3);
    assert_eq!(encoded["result"]["diagnostics"], json!([]));
}

struct RecordingProvider {
    symbols: Rc<Symbols>,
    calls: Rc<RefCell<Vec<String>>>,
    fail: bool,
}

impl SymbolProvider for RecordingProvider {
    async fn symbols_for(&mut self, build_id: &str) -> Result<Option<Rc<Symbols>>> {
        self.calls.borrow_mut().push(build_id.to_owned());
        if self.fail {
            bail!("artifact store unavailable")
        }
        Ok((self.symbols.build_id() == build_id).then(|| Rc::clone(&self.symbols)))
    }
}

#[tokio::test]
async fn provider_is_replaceable_and_boot_transitions_keep_original_event_indices() {
    let mut message = request();
    let build_id = message["payload"]["initialBoot"]["buildId"]
        .as_str()
        .unwrap()
        .to_owned();
    message["payload"]["initialBoot"]["buildId"] = json!("missing-build");
    let log = message["payload"]["events"][1].clone();
    message["payload"]["events"]
        .as_array_mut()
        .unwrap()
        .push(log);
    let repeated_boot = message["payload"]["events"][4].clone();
    message["payload"]["events"]
        .as_array_mut()
        .unwrap()
        .push(repeated_boot);
    let calls = Rc::new(RefCell::new(Vec::new()));
    let provider = RecordingProvider {
        symbols: Rc::new(Symbols::load(common::fixture().join("firmware.elf")).unwrap()),
        calls: Rc::clone(&calls),
        fail: false,
    };
    let task = decode_request(
        &serde_json::to_vec(&message).unwrap(),
        Some("application/json"),
        None,
    )
    .unwrap();
    let result = SymbolicationService::new(provider)
        .execute(task)
        .await
        .unwrap();
    assert_eq!(*calls.borrow(), vec!["missing-build", &build_id]);
    assert!(result.output.logs[0].frames[1].function.is_none());
    assert!(result.output.logs[2].frames[1].function.is_some());
    assert_eq!(result.output.logs[2].common.event_index, 5);
    assert_eq!(result.output.boots[0].common.event_index, 4);
    assert!(!result.output.diagnostics.is_empty());
    let encoded: Value = serde_json::from_slice(&encode_result(&result).unwrap()).unwrap();
    assert!(
        !encoded["result"]["diagnostics"]
            .as_array()
            .unwrap()
            .is_empty()
    );
}

#[tokio::test]
async fn transient_provider_failures_abort_the_use_case() {
    let provider = RecordingProvider {
        symbols: Rc::new(Symbols::load(common::fixture().join("firmware.elf")).unwrap()),
        calls: Rc::default(),
        fail: true,
    };
    let task = decode_request(
        &serde_json::to_vec(&request()).unwrap(),
        Some("application/json"),
        None,
    )
    .unwrap();
    let error = SymbolicationService::new(provider)
        .execute(task)
        .await
        .err()
        .unwrap();
    assert!(format!("{error:#}").contains("artifact store unavailable"));
}

#[tokio::test]
async fn unresolved_results_preserve_exact_counters_without_symbols() {
    let mut message = request();
    message["payload"] = json!({"batchId":"AQI=","droppedRecords":"18446744073709551615","events":[{"monotonicUs":"18446744073709551615","metric":{"metricId":"18446744073709551615","counterDelta":"18446744073709551615"}}]});
    let provider = FixedElfProvider::load(common::fixture().join("firmware.elf")).unwrap();
    let task = decode_request(
        &serde_json::to_vec(&message).unwrap(),
        Some("application/json"),
        None,
    )
    .unwrap();
    let result = SymbolicationService::new(provider)
        .execute(task)
        .await
        .unwrap();
    let encoded: Value = serde_json::from_slice(&encode_result(&result).unwrap()).unwrap();
    assert_eq!(
        encoded["result"]["dropped_records"].as_u64(),
        Some(u64::MAX)
    );
    assert_eq!(
        encoded["result"]["metrics"][0]["counter_delta"].as_u64(),
        Some(u64::MAX)
    );
    assert_eq!(
        encoded["result"]["metrics"][0]["monotonic_us"].as_u64(),
        Some(u64::MAX)
    );
    assert_eq!(encoded["result"]["producer_batch_id"], "0102");
    assert!(
        !encoded["result"]["diagnostics"]
            .as_array()
            .unwrap()
            .is_empty()
    );
}
