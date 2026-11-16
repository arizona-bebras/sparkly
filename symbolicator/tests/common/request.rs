use serde_json::{Value, json};

pub fn request() -> Value {
    json!({
        "schema_version": 1,
        "batch_id": "01900000-0000-7000-8000-000000000001",
        "device_id": "01900000-0000-7000-8000-000000000002",
        "park_id": "01900000-0000-7000-8000-000000000003",
        "received_at": "2026-11-13T12:00:00.123Z",
        "payload": serde_json::from_slice::<Value>(&std::fs::read(crate::common::fixture().join("parcel.json")).unwrap()).unwrap(),
    })
}
