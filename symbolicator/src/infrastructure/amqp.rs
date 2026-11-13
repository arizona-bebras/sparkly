use std::{future::Future, time::Duration};

use anyhow::{Context, Result, bail};
use futures_lite::StreamExt;
use lapin::{
    BasicProperties, Channel, Confirmation, Connection, ConnectionProperties,
    message::Delivery,
    options::{
        BasicAckOptions, BasicCancelOptions, BasicConsumeOptions, BasicPublishOptions,
        BasicQosOptions, BasicRejectOptions, ConfirmSelectOptions, QueueDeclareOptions,
    },
    types::FieldTable,
};

use super::messages::{decode_request, encode_result};
use crate::{
    application::{SymbolProvider, SymbolicationService},
    domain::SymbolicatedBatch,
};

const PUBLISH_TIMEOUT: Duration = Duration::from_secs(10);

pub struct QueueNames {
    pub requests: String,
    pub results: String,
}

impl Default for QueueNames {
    fn default() -> Self {
        Self {
            requests: "symbolication_requests".into(),
            results: "symbolication_results".into(),
        }
    }
}

/// AMQP adapter: one in-flight request and a separate confirmed result channel.
pub struct AmqpWorker {
    connection: Connection,
    consumer_channel: Channel,
    publisher_channel: Channel,
    queues: QueueNames,
}

impl AmqpWorker {
    pub async fn connect(url: &str, queues: QueueNames) -> Result<Self> {
        let connection = tokio::time::timeout(
            PUBLISH_TIMEOUT,
            Connection::connect(url, ConnectionProperties::default()),
        )
        .await
        .context("connect to RabbitMQ timed out")?
        .context("connect to RabbitMQ")?;
        let setup = Self::open_channels(&connection, &queues).await;
        match setup {
            Ok((consumer_channel, publisher_channel)) => Ok(Self {
                connection,
                consumer_channel,
                publisher_channel,
                queues,
            }),
            Err(error) => {
                let _ = connection.close(200, "worker setup failed".into()).await;
                Err(error)
            }
        }
    }

    async fn open_channels(
        connection: &Connection,
        queues: &QueueNames,
    ) -> Result<(Channel, Channel)> {
        let consumer = connection
            .create_channel()
            .await
            .context("open request channel")?;
        let publisher = connection
            .create_channel()
            .await
            .context("open result channel")?;
        for name in [&queues.requests, &queues.results] {
            consumer
                .queue_declare(
                    name.clone().into(),
                    QueueDeclareOptions {
                        durable: true,
                        ..Default::default()
                    },
                    FieldTable::default(),
                )
                .await
                .context("declare symbolication queue")?;
        }
        consumer
            .basic_qos(1, BasicQosOptions::default())
            .await
            .context("set request prefetch")?;
        publisher
            .confirm_select(ConfirmSelectOptions::default())
            .await
            .context("enable result confirmations")?;
        Ok((consumer, publisher))
    }

    pub async fn run<P: SymbolProvider>(
        &self,
        service: &mut SymbolicationService<P>,
        shutdown: impl Future<Output = Result<()>>,
    ) -> Result<()> {
        let mut consumer = self
            .consumer_channel
            .basic_consume(
                self.queues.requests.clone().into(),
                "".into(),
                BasicConsumeOptions::default(),
                FieldTable::default(),
            )
            .await
            .context("consume symbolication requests")?;
        tokio::pin!(shutdown);
        loop {
            tokio::select! {
                biased;
                result = &mut shutdown => {
                    result?;
                    self.consumer_channel.basic_cancel(consumer.tag(), BasicCancelOptions::default()).await.context("cancel symbolication consumer")?;
                    return Ok(());
                }
                delivery = consumer.next() => {
                    let delivery = delivery.context("symbolication request stream closed")?.context("receive symbolication request")?;
                    self.process_delivery(service, delivery).await?;
                }
            }
        }
    }

    async fn process_delivery<P: SymbolProvider>(
        &self,
        service: &mut SymbolicationService<P>,
        delivery: Delivery,
    ) -> Result<()> {
        let task = match decode_request(
            &delivery.data,
            delivery
                .properties
                .content_type()
                .as_ref()
                .map(|s| s.as_str()),
            delivery
                .properties
                .message_id()
                .as_ref()
                .map(|s| s.as_str()),
        ) {
            Ok(task) => task,
            Err(error) => {
                eprintln!("reject symbolication request: {error:#}");
                delivery
                    .reject(BasicRejectOptions { requeue: false })
                    .await
                    .context("reject invalid request")?;
                return Ok(());
            }
        };
        let batch_id = task.context.batch_id();
        let batch = service
            .execute(task)
            .await
            .with_context(|| format!("symbolicate batch {batch_id}"))?;
        self.publish_result(&batch)
            .await
            .with_context(|| format!("publish result for batch {batch_id}"))?;
        delivery
            .ack(BasicAckOptions::default())
            .await
            .context("acknowledge symbolication request")
            .map(|_| ())
    }

    async fn publish_result(&self, batch: &SymbolicatedBatch) -> Result<()> {
        let body = encode_result(batch)?;
        let id = batch.context.batch_id().to_string();
        let confirmation = tokio::time::timeout(PUBLISH_TIMEOUT, async {
            self.publisher_channel
                .basic_publish(
                    "".into(),
                    self.queues.results.clone().into(),
                    BasicPublishOptions {
                        mandatory: true,
                        ..Default::default()
                    },
                    &body,
                    BasicProperties::default()
                        .with_content_type("application/json".into())
                        .with_delivery_mode(2)
                        .with_message_id(id.clone().into())
                        .with_correlation_id(id.into()),
                )
                .await?
                .await
        })
        .await
        .context("result confirmation timed out")?
        .context("publish symbolication result")?;
        match confirmation {
            Confirmation::Ack(None) => Ok(()),
            Confirmation::Ack(Some(_)) => bail!("symbolication result was unroutable"),
            Confirmation::Nack(_) => bail!("broker rejected symbolication result"),
            Confirmation::NotRequested => bail!("result publisher confirmation was not enabled"),
        }
    }

    /// Closing the connection requeues any request whose result was not confirmed.
    pub async fn close(&self) -> Result<()> {
        self.connection
            .close(200, "symbolicator stopping".into())
            .await
            .context("close symbolicator connection")
    }
}
