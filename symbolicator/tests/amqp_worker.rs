mod common;
#[path = "common/request.rs"]
mod request;

use std::{future::pending, time::Duration};

use anyhow::Result;
use futures_lite::StreamExt;
use lapin::{
    BasicProperties, Channel, Confirmation, Connection, ConnectionProperties, options::*,
    types::FieldTable,
};
use serde_json::Value;
use sparkly_symbolicator::{
    application::SymbolicationService,
    infrastructure::{
        amqp::{AmqpWorker, QueueNames},
        fixed_elf::FixedElfProvider,
    },
};
use tokio::{sync::oneshot, time::timeout};

struct BrokerFixture {
    connection: Connection,
    channel: Channel,
    worker: AmqpWorker,
    requests: String,
    results: String,
}

impl BrokerFixture {
    async fn connect() -> Option<Self> {
        let Ok(url) = std::env::var("RABBITMQ_TEST_URL") else {
            eprintln!("set RABBITMQ_TEST_URL to run AMQP integration tests");
            return None;
        };
        let directory = tempfile::tempdir().unwrap();
        let prefix = format!(
            "symbolicator-test-{}-{}",
            std::process::id(),
            directory.path().file_name().unwrap().to_string_lossy()
        );
        let requests = format!("{prefix}-requests");
        let results = format!("{prefix}-results");
        let worker = AmqpWorker::connect(
            &url,
            QueueNames {
                requests: requests.clone(),
                results: results.clone(),
            },
        )
        .await
        .unwrap();
        let connection = Connection::connect(&url, ConnectionProperties::default())
            .await
            .unwrap();
        let channel = connection.create_channel().await.unwrap();
        channel
            .confirm_select(ConfirmSelectOptions::default())
            .await
            .unwrap();
        Some(Self {
            connection,
            channel,
            worker,
            requests,
            results,
        })
    }

    async fn publish(&self, body: &[u8], content_type: &str) {
        let confirmation = self
            .channel
            .basic_publish(
                "".into(),
                self.requests.clone().into(),
                BasicPublishOptions {
                    mandatory: true,
                    ..Default::default()
                },
                body,
                BasicProperties::default()
                    .with_content_type(content_type.into())
                    .with_delivery_mode(2)
                    .with_message_id("01900000-0000-7000-8000-000000000001".into()),
            )
            .await
            .unwrap()
            .await
            .unwrap();
        assert!(matches!(confirmation, Confirmation::Ack(None)));
    }

    async fn cleanup(self) {
        for queue in [self.requests, self.results] {
            self.channel
                .queue_delete(queue.into(), QueueDeleteOptions::default())
                .await
                .unwrap();
        }
        self.connection
            .close(200, "test complete".into())
            .await
            .unwrap();
    }
}

#[tokio::test]
async fn publishes_correlated_results_and_rejects_invalid_requests() {
    let Some(fixture) = BrokerFixture::connect().await else {
        return;
    };
    let provider = FixedElfProvider::load(common::fixture().join("firmware.elf")).unwrap();
    let mut service = SymbolicationService::new(provider);
    let message = request::request();
    fixture.publish(b"not json", "application/json").await;
    fixture
        .publish(&serde_json::to_vec(&message).unwrap(), "application/json")
        .await;
    let mut results = fixture
        .channel
        .basic_consume(
            fixture.results.clone().into(),
            "".into(),
            BasicConsumeOptions::default(),
            FieldTable::default(),
        )
        .await
        .unwrap();
    let (stop, stopped) = oneshot::channel();
    let (processing, body) = timeout(Duration::from_secs(20), async {
        tokio::join!(
            fixture.worker.run(&mut service, async {
                stopped.await?;
                Ok(())
            }),
            async {
                let delivery = results.next().await.unwrap().unwrap();
                assert_eq!(
                    delivery
                        .properties
                        .content_type()
                        .as_ref()
                        .unwrap()
                        .as_str(),
                    "application/json"
                );
                assert_eq!(*delivery.properties.delivery_mode(), Some(2));
                assert_eq!(
                    delivery.properties.message_id().as_ref().unwrap().as_str(),
                    message["batch_id"].as_str().unwrap()
                );
                assert_eq!(
                    delivery.properties.correlation_id(),
                    delivery.properties.message_id()
                );
                let body: Value = serde_json::from_slice(&delivery.data).unwrap();
                delivery.ack(BasicAckOptions::default()).await.unwrap();
                stop.send(()).unwrap();
                body
            }
        )
    })
    .await
    .unwrap();
    processing.unwrap();
    fixture.worker.close().await.unwrap();
    for field in ["batch_id", "device_id", "park_id", "received_at"] {
        assert_eq!(body[field], message[field]);
    }
    assert_eq!(body["result_version"], 5);
    assert_eq!(body["result"]["diagnostics"], serde_json::json!([]));
    for field in ["status", "symbol_build_ids"] {
        assert!(body.get(field).is_none());
    }
    assert!(body["result"].get("symbol_build_id").is_none());
    assert_eq!(body["result"]["logs"][1]["message"], "temperature=42");
    let log = &body["result"]["logs"][0];
    assert_eq!(log["module"], "fixture");
    assert_eq!(log["frames"].as_array().unwrap().len(), 1);
    assert!(log["frames"][0]["pc"].as_u64().is_some());
    assert_eq!(log["frames"][0]["function"], "sampled_function");
    for field in [
        "source",
        "target",
        "file",
        "line",
        "fingerprint",
        "decode_status",
    ] {
        assert!(log.get(field).is_none());
    }
    let queue = fixture
        .channel
        .queue_declare(
            fixture.requests.clone().into(),
            QueueDeclareOptions {
                passive: true,
                ..Default::default()
            },
            FieldTable::default(),
        )
        .await
        .unwrap();
    assert_eq!(queue.message_count(), 0);
    assert_eq!(queue.consumer_count(), 0);
    fixture.cleanup().await;
}

#[tokio::test]
async fn unroutable_results_leave_the_request_available_for_redelivery() {
    let Some(fixture) = BrokerFixture::connect().await else {
        return;
    };
    let provider = FixedElfProvider::load(common::fixture().join("firmware.elf")).unwrap();
    let mut service = SymbolicationService::new(provider);
    // Remove the destination after startup to force a mandatory return.
    fixture
        .channel
        .queue_delete(
            fixture.results.clone().into(),
            QueueDeleteOptions::default(),
        )
        .await
        .unwrap();
    fixture
        .publish(
            &serde_json::to_vec(&request::request()).unwrap(),
            "application/json",
        )
        .await;
    let processing = timeout(
        Duration::from_secs(20),
        fixture.worker.run(&mut service, pending::<Result<()>>()),
    )
    .await
    .unwrap();
    assert!(format!("{:#}", processing.unwrap_err()).contains("unroutable"));
    fixture.worker.close().await.unwrap();
    let delivery = fixture
        .channel
        .basic_get(fixture.requests.clone().into(), BasicGetOptions::default())
        .await
        .unwrap()
        .expect("unacknowledged request was lost");
    assert!(delivery.redelivered);
    assert_eq!(
        delivery.properties.message_id().as_ref().unwrap().as_str(),
        "01900000-0000-7000-8000-000000000001"
    );
    delivery.ack(BasicAckOptions::default()).await.unwrap();
    // Restore the intentionally removed queue so fixture cleanup is uniform.
    fixture
        .channel
        .queue_declare(
            fixture.results.clone().into(),
            QueueDeclareOptions {
                durable: true,
                ..Default::default()
            },
            FieldTable::default(),
        )
        .await
        .unwrap();
    fixture.cleanup().await;
}
