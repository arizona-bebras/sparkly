use micropb::MessageDecode;
use sparkly_agent::*;
use sparkly_protocol::wire::sparkly_::agent_::v1_ as pb;
use std::{
    cell::Cell,
    fmt,
    sync::atomic::{AtomicU64, Ordering},
};

struct TestClock(AtomicU64);
impl Clock for TestClock {
    fn now_us(&self) -> u64 {
        self.0.load(Ordering::Relaxed)
    }
}
struct TestBacktrace;
impl Backtrace for TestBacktrace {
    fn capture(&self) -> StackTrace {
        StackTrace {
            pcs: [0x42000100, 0x42000200].into_iter().collect(),
            status: TraceStatus::Complete,
        }
    }
}
// Fixture Boot: framing/identity/string lengths plus the current agent version.
const BOOT_BYTES: usize = 64 + env!("CARGO_PKG_VERSION").len();
const METRIC_BYTES: usize = 28;

fn recorder<const N: usize>() -> Recorder<RamStorage<N>, TestClock, TestBacktrace> {
    Recorder::new(
        Session::new([7; 16], "0123456789abcdef", "test", "power_on").unwrap(),
        RamStorage::new(),
        TestClock(AtomicU64::new(123)),
        TestBacktrace,
    )
    .unwrap()
}
fn decode(batch: &Batch) -> pb::Parcel {
    let mut bytes = [0; 8192];
    let len = ProtobufEncoder.encode(batch, &mut bytes).unwrap();
    let mut parcel = pb::Parcel::default();
    parcel.decode_from_bytes(&bytes[..len]).unwrap();
    parcel
}

#[test]
fn byte_queue_eviction_keeps_newest_records_and_counts_evicted_boot() {
    let recorder = recorder::<{ BOOT_BYTES + 2 * METRIC_BYTES }>();
    for value in 0..4 {
        recorder
            .metric(1, MetricValue::CounterDelta(value))
            .unwrap();
    }
    let batch = recorder.take_batch().unwrap().unwrap();
    assert_eq!(batch.records.len(), 4);
    for (record, expected) in batch.records.iter().zip(0..4) {
        assert!(matches!(record.payload, Payload::Metric {
            value: MetricValue::CounterDelta(value), ..
        } if value == expected));
    }
    assert_eq!(recorder.statistics().evicted, 1);
    let parcel = decode(&batch);
    assert_eq!(parcel.dropped_records, 1);
    assert_eq!(parcel.initial_boot.boot_id.as_slice(), [7; 16]);
}

#[test]
fn boot_event_is_exported_once_with_context_in_later_headers() {
    let recorder = recorder::<1024>();
    let boot = recorder.take_batch().unwrap().unwrap();
    assert_eq!(decode(&boot).events.len(), 1);
    assert!(recorder.take_batch().unwrap().is_none());
    recorder.metric(1, MetricValue::Gauge(1.0)).unwrap();
    let next = recorder.take_batch().unwrap().unwrap();
    assert_ne!(boot.id, next.id);
    let parcel = decode(&next);
    assert_eq!(parcel.initial_boot.build_id, "0123456789abcdef");
    assert_eq!(parcel.initial_boot.boot_id.as_slice(), [7; 16]);
    assert_eq!(parcel.events.len(), 1);
    assert!(matches!(
        parcel.events[0].payload,
        Some(pb::Event_::Payload::Metric(_))
    ));
}

#[test]
fn restored_history_precedes_the_new_reset_in_the_same_parcel() {
    let mut old_session =
        Session::new([7; 16], "0123456789abcdef", "old-firmware", "power_on").unwrap();
    old_session.agent_version = "old-agent".try_into().unwrap();
    let old = Recorder::new(
        old_session,
        RamStorage::<1024>::new(),
        TestClock(AtomicU64::new(123)),
        TestBacktrace,
    )
    .unwrap();
    old.metric(1, MetricValue::Gauge(10.0)).unwrap();
    let next = Recorder::new(
        Session::new([8; 16], "new-build", "new-firmware", "software_reset").unwrap(),
        old.into_storage(),
        TestClock(AtomicU64::new(5)),
        TestBacktrace,
    )
    .unwrap();
    next.metric(1, MetricValue::Gauge(20.0)).unwrap();

    let parcel = decode(&next.take_batch().unwrap().unwrap());
    assert_eq!(parcel.events.len(), 4);
    assert!(
        matches!(&parcel.events[0].payload, Some(pb::Event_::Payload::Boot(boot))
        if boot.boot_id.as_slice() == [7; 16] && boot.build_id == "0123456789abcdef"
            && boot.firmware_version == "old-firmware" && boot.agent_version == "old-agent")
    );
    assert!(
        matches!(&parcel.events[1].payload, Some(pb::Event_::Payload::Metric(metric))
        if metric.measurement == Some(pb::Metric_::Measurement::Gauge(10.0)))
    );
    assert!(
        matches!(&parcel.events[2].payload, Some(pb::Event_::Payload::Boot(boot))
        if boot.boot_id.as_slice() == [8; 16] && boot.build_id == "new-build")
    );
    assert!(
        matches!(&parcel.events[3].payload, Some(pb::Event_::Payload::Metric(metric))
        if metric.measurement == Some(pb::Metric_::Measurement::Gauge(20.0)))
    );
    assert_eq!(parcel.events[1].monotonic_us, 123);
    assert_eq!(parcel.events[2].monotonic_us, 5);
}

#[test]
fn eviction_removes_oldest_records_including_resets_across_boots() {
    // Second Boot uses "new-build", "test", "reset" metadata.
    let old =
        recorder::<{ BOOT_BYTES + 54 + env!("CARGO_PKG_VERSION").len() + 2 * METRIC_BYTES }>();
    old.metric(1, MetricValue::Gauge(10.0)).unwrap();
    let next = Recorder::new(
        Session::new([8; 16], "new-build", "test", "reset").unwrap(),
        old.into_storage(),
        TestClock(AtomicU64::new(0)),
        TestBacktrace,
    )
    .unwrap();
    for value in 20..24 {
        next.metric(1, MetricValue::Gauge(value as f64)).unwrap();
    }

    let parcel = decode(&next.take_batch().unwrap().unwrap());
    assert_eq!(parcel.dropped_records, 2);
    assert_eq!(parcel.events.len(), 5);
    assert!(
        matches!(&parcel.events[0].payload, Some(pb::Event_::Payload::Boot(boot))
        if boot.boot_id.as_slice() == [8; 16])
    );
    for (event, expected) in parcel.events[1..].iter().zip([20.0, 21.0, 22.0, 23.0]) {
        assert!(
            matches!(&event.payload, Some(pb::Event_::Payload::Metric(metric))
            if metric.measurement == Some(pb::Metric_::Measurement::Gauge(expected)))
        );
    }
}

#[test]
fn startup_evicts_oldest_boot_when_history_fills_storage() {
    // Second Boot uses "second", "test", "reset" metadata.
    let old = recorder::<{ BOOT_BYTES + 51 + env!("CARGO_PKG_VERSION").len() }>();
    let next = Recorder::new(
        Session::new([8; 16], "second", "test", "reset").unwrap(),
        old.into_storage(),
        TestClock(AtomicU64::new(0)),
        TestBacktrace,
    )
    .unwrap();
    let newest = Recorder::new(
        Session::new([9; 16], "third", "test", "reset").unwrap(),
        next.into_storage(),
        TestClock(AtomicU64::new(0)),
        TestBacktrace,
    )
    .unwrap();
    assert_eq!(newest.statistics().evicted, 1);
    let mut storage = newest.into_storage();
    for id in [8, 9] {
        assert!(
            matches!(storage.pop().unwrap().payload, Payload::Boot(boot) if boot.boot_id == [id; 16])
        );
    }
    assert!(storage.pop().is_none());
}

#[test]
fn startup_returns_existing_stream_when_new_boot_exceeds_byte_budget() {
    let old = recorder::<BOOT_BYTES>();
    let error = Recorder::new(
        Session::new([8; 16], &"b".repeat(64), &"f".repeat(32), &"r".repeat(64)).unwrap(),
        old.into_storage(),
        TestClock(AtomicU64::new(0)),
        TestBacktrace,
    )
    .err()
    .expect("oversized Boot must fail without eviction");
    assert_eq!(error.error, StorageFull);
    let mut storage = error.storage;
    assert!(
        matches!(storage.pop().unwrap().payload, Payload::Boot(boot) if boot.boot_id == [7; 16])
    );
    assert!(storage.pop().is_none());
}

#[test]
fn telemetry_can_overwrite_an_undelivered_reset() {
    let recorder = recorder::<BOOT_BYTES>();
    recorder.metric(1, MetricValue::Gauge(1.0)).unwrap();
    let parcel = decode(&recorder.take_batch().unwrap().unwrap());
    assert_eq!(parcel.events.len(), 1);
    assert!(matches!(
        parcel.events[0].payload,
        Some(pb::Event_::Payload::Metric(_))
    ));
    assert_eq!(parcel.dropped_records, 1);
}

#[test]
fn events_continue_in_order_across_parcels_without_repeating_the_boot() {
    let recorder = recorder::<1024>();
    for value in 0..12 {
        recorder
            .metric(1, MetricValue::CounterDelta(value))
            .unwrap();
    }
    let first = decode(&recorder.take_batch().unwrap().unwrap());
    let second = decode(&recorder.take_batch().unwrap().unwrap());
    assert_eq!(first.events.len(), MAX_BATCH_RECORDS);
    assert_eq!(second.events.len(), 5);
    let mut values = Vec::new();
    let mut boots = 0;
    for event in first.events.iter().chain(second.events.iter()) {
        match &event.payload {
            Some(pb::Event_::Payload::Boot(_)) => boots += 1,
            Some(pb::Event_::Payload::Metric(metric)) => {
                let Some(pb::Metric_::Measurement::CounterDelta(value)) = metric.measurement else {
                    panic!()
                };
                values.push(value);
            }
            _ => panic!(),
        }
    }
    assert_eq!(boots, 1);
    assert_eq!(values, (0..12).collect::<Vec<_>>());
}

#[test]
fn loss_counts_are_parcel_deltas_instead_of_per_boot_snapshots() {
    let recorder = recorder::<{ BOOT_BYTES + 2 * METRIC_BYTES }>();
    for value in 0..4 {
        recorder
            .metric(1, MetricValue::CounterDelta(value))
            .unwrap();
    }
    assert_eq!(
        decode(&recorder.take_batch().unwrap().unwrap()).dropped_records,
        1
    );
    for value in 4..9 {
        recorder
            .metric(1, MetricValue::CounterDelta(value))
            .unwrap();
    }
    assert_eq!(
        decode(&recorder.take_batch().unwrap().unwrap()).dropped_records,
        1
    );
    assert_eq!(recorder.statistics().evicted, 2);
    assert!(recorder.take_batch().unwrap().is_none());
}

#[test]
fn timestamps_follow_commit_order_when_metrics_arrive_during_formatting() {
    struct TickClock(AtomicU64);
    impl Clock for TickClock {
        fn now_us(&self) -> u64 {
            self.0.fetch_add(1, Ordering::Relaxed)
        }
    }
    struct Formatting<'a>(&'a dyn Sink);
    impl fmt::Display for Formatting<'_> {
        fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
            self.0.metric(1, MetricValue::Gauge(1.0)).unwrap();
            out.write_str("formatted")
        }
    }
    let recorder = Recorder::new(
        Session::new([7; 16], "build", "test", "reset").unwrap(),
        RamStorage::<1024>::new(),
        TickClock(AtomicU64::new(1)),
        NoBacktrace,
    )
    .unwrap();
    recorder
        .text(
            Level::Info,
            "test",
            format_args!("{}", Formatting(&recorder)),
        )
        .unwrap();
    let batch = recorder.take_batch().unwrap().unwrap();
    assert_eq!(
        batch
            .records
            .iter()
            .map(|r| r.monotonic_us)
            .collect::<Vec<_>>(),
        [1, 2, 3]
    );
}

#[test]
fn text_truncates_at_utf8_boundary_and_captures_backtrace_at_callsite() {
    let recorder = recorder::<1024>();
    recorder
        .text(Level::Warn, "sensor", format_args!("{}", "€".repeat(100)))
        .unwrap();
    let batch = recorder.take_batch().unwrap().unwrap();
    let Payload::Text(log) = &batch.records[1].payload else {
        panic!()
    };
    assert_eq!(log.message.len(), 255);
    assert!(log.truncated);
    assert_eq!(log.trace.pcs.as_slice(), &[0x42000100, 0x42000200]);
    assert_eq!(batch.records[1].monotonic_us, 123);
}

#[test]
fn recursive_formatting_is_rejected_without_deadlock() {
    struct Recursive<'a>(&'a dyn Sink);
    impl fmt::Display for Recursive<'_> {
        fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
            assert_eq!(
                self.0.text(Level::Info, "nested", format_args!("nested")),
                Err(CaptureError::Busy)
            );
            out.write_str("outer")
        }
    }
    let recorder = recorder::<1024>();
    recorder
        .text(
            Level::Info,
            "outer",
            format_args!("{}", Recursive(&recorder)),
        )
        .unwrap();
    assert_eq!(recorder.statistics().rejected, 1);
    assert_eq!(recorder.statistics().accepted, 1);
}

#[test]
fn protobuf_preserves_typed_metrics_and_complete_defmt_frames() {
    let recorder = recorder::<1024>();
    metric!("cpu.temperature")
        .set_on(&recorder, 23.123456789)
        .unwrap();
    counter!("requests.completed")
        .increment_on(&recorder, u64::MAX)
        .unwrap();
    recorder.defmt(&[1, 2, 0, 3, 0]).unwrap();
    let parcel = decode(&recorder.take_batch().unwrap().unwrap());
    assert!(
        matches!(&parcel.events[1].payload, Some(pb::Event_::Payload::Metric(m))
        if m.measurement == Some(pb::Metric_::Measurement::Gauge(23.123456789)))
    );
    assert!(
        matches!(&parcel.events[2].payload, Some(pb::Event_::Payload::Metric(m))
        if m.measurement == Some(pb::Metric_::Measurement::CounterDelta(u64::MAX)))
    );
    assert!(
        matches!(&parcel.events[3].payload, Some(pb::Event_::Payload::Log(log))
        if matches!(&log.content, Some(pb::Log_::Content::DefmtLog(frame)) if frame.as_slice() == [1, 2, 0, 3, 0]))
    );
}

#[test]
fn oversized_defmt_frame_is_dropped_whole() {
    let recorder = recorder::<1024>();
    assert_eq!(
        recorder.defmt(&[0; MAX_DEFMT + 1]),
        Err(CaptureError::OversizedFrame)
    );
    let batch = recorder.take_batch().unwrap().unwrap();
    assert_eq!(batch.records.len(), 1);
    assert!(matches!(batch.records[0].payload, Payload::Boot(_)));
    assert_eq!(batch.dropped_records, 1);
}

#[test]
fn losses_after_a_boot_only_batch_are_still_exported() {
    let recorder = recorder::<BOOT_BYTES>();
    recorder.take_batch().unwrap().unwrap();
    assert!(recorder.take_batch().unwrap().is_none());
    recorder.reject();
    let batch = recorder.take_batch().unwrap().unwrap();
    assert!(batch.records.is_empty());
    assert_eq!(decode(&batch).dropped_records, 1);
    assert!(recorder.take_batch().unwrap().is_none());
}

#[derive(Default)]
struct RetryingTransport {
    calls: Vec<(BatchId, Vec<u8>)>,
}
impl Transport for RetryingTransport {
    type Error = &'static str;
    fn send(&mut self, batch: OutboundBatch<'_>) -> Result<Acknowledgment, Self::Error> {
        self.calls.push((batch.id, batch.bytes.to_vec()));
        match self.calls.len() {
            1 => Err("offline"),
            2 => Ok(Acknowledgment { batch_id: [0; 24] }),
            _ => Ok(Acknowledgment { batch_id: batch.id }),
        }
    }
}

#[test]
fn failures_and_wrong_ack_retry_identical_bytes_while_capture_continues() {
    let recorder = recorder::<1024>();
    recorder.metric(1, MetricValue::Gauge(1.0)).unwrap();
    let mut exporter =
        Exporter::<_, _, 8192>::new(&recorder, RetryingTransport::default(), ProtobufEncoder, 1);
    assert!(matches!(
        exporter.flush(),
        Err(ExportError::Transport("offline"))
    ));
    recorder.metric(2, MetricValue::Gauge(2.0)).unwrap();
    assert_eq!(exporter.flush(), Err(ExportError::WrongAcknowledgment));
    assert!(exporter.has_pending());
    exporter.flush().unwrap();
    let calls = &exporter.transport_mut().calls;
    assert_eq!(calls[0], calls[1]);
    assert_eq!(calls[1], calls[2]);
    assert!(!exporter.has_pending());
    exporter.flush().unwrap();
    let calls = &exporter.transport_mut().calls;
    assert_ne!(calls[2].0, calls[3].0);
}

struct FailOnce(Cell<bool>);
impl BatchEncoder for FailOnce {
    fn encode(&self, batch: &Batch, bytes: &mut [u8]) -> Result<usize, EncodeError> {
        if self.0.replace(false) {
            Err(EncodeError::BufferTooSmall)
        } else {
            ProtobufEncoder.encode(batch, bytes)
        }
    }
}

#[test]
fn encoder_failure_does_not_lose_drained_records() {
    let recorder = recorder::<1024>();
    recorder.metric(1, MetricValue::Gauge(1.0)).unwrap();
    let mut exporter = Exporter::<_, _, 8192>::new(
        &recorder,
        RetryingTransport::default(),
        FailOnce(Cell::new(true)),
        1,
    );
    assert_eq!(
        exporter.flush(),
        Err(ExportError::Encode(EncodeError::BufferTooSmall))
    );
    assert!(exporter.has_pending());
    assert!(matches!(exporter.flush(), Err(ExportError::Transport(_))));
    let bytes = &exporter.transport_mut().calls[0].1;
    let mut parcel = pb::Parcel::default();
    parcel.decode_from_bytes(bytes).unwrap();
    assert_eq!(parcel.events.len(), 2);
}

#[test]
fn tiny_output_buffer_fails_instead_of_emitting_partial_parcel() {
    let recorder = recorder::<BOOT_BYTES>();
    let batch = recorder.take_batch().unwrap().unwrap();
    assert_eq!(
        ProtobufEncoder.encode(&batch, &mut [0; 8]),
        Err(EncodeError::BufferTooSmall)
    );
}

#[test]
fn periodic_poll_does_not_busy_retry() {
    let recorder = recorder::<BOOT_BYTES>();
    let mut exporter = Exporter::<_, _, 8192>::new(
        &recorder,
        RetryingTransport::default(),
        ProtobufEncoder,
        1_000_000,
    );
    assert!(exporter.poll().is_err());
    assert_eq!(exporter.poll(), Ok(FlushOutcome::Idle));
    assert_eq!(exporter.transport_mut().calls.len(), 1);
}

#[test]
fn cancelled_async_send_keeps_identical_pending_batch() {
    use std::{
        future::Future,
        task::{Context, Poll, Waker},
    };
    struct Suspended {
        calls: Vec<(BatchId, Vec<u8>)>,
    }
    impl AsyncTransport for Suspended {
        type Error = ();
        async fn send(&mut self, batch: OutboundBatch<'_>) -> Result<Acknowledgment, Self::Error> {
            self.calls.push((batch.id, batch.bytes.to_vec()));
            if self.calls.len() == 1 {
                std::future::pending::<()>().await;
            }
            Ok(Acknowledgment { batch_id: batch.id })
        }
    }
    let recorder = recorder::<BOOT_BYTES>();
    let mut exporter = Exporter::<_, _, 8192>::new(
        &recorder,
        Suspended { calls: Vec::new() },
        ProtobufEncoder,
        1,
    );
    let mut context = Context::from_waker(Waker::noop());
    {
        let mut send = std::pin::pin!(exporter.flush_async());
        assert!(send.as_mut().poll(&mut context).is_pending());
    }
    assert!(exporter.has_pending());
    {
        let mut retry = std::pin::pin!(exporter.flush_async());
        assert!(matches!(
            retry.as_mut().poll(&mut context),
            Poll::Ready(Ok(FlushOutcome::Acknowledged(_)))
        ));
    }
    let calls = &exporter.transport_mut().calls;
    assert_eq!(calls[0], calls[1]);
}

#[test]
fn recorder_accepts_borrowed_and_heap_backed_rings() {
    fn capture<S: RecordStorage>(storage: S)
    where
        S::Error: fmt::Debug,
    {
        let recorder = Recorder::new(
            Session::new([7; 16], "build", "test", "reset").unwrap(),
            storage,
            TestClock(AtomicU64::new(123)),
            NoBacktrace,
        )
        .unwrap();
        recorder.metric(1, MetricValue::CounterDelta(42)).unwrap();
        let parcel = decode(&recorder.take_batch().unwrap().unwrap());
        assert_eq!(parcel.events.len(), 2);
        assert!(matches!(
            &parcel.events[0].payload,
            Some(pb::Event_::Payload::Boot(_))
        ));
        assert!(
            matches!(&parcel.events[1].payload, Some(pb::Event_::Payload::Metric(metric))
            if metric.measurement == Some(pb::Metric_::Measurement::CounterDelta(42)))
        );
    }
    let mut memory = [0_u8; 256];
    capture(RingStorage::with_storage(&mut memory[..]));
    capture(RingStorage::with_storage(
        vec![0_u8; 256].into_boxed_slice(),
    ));
}

#[test]
fn header_retains_old_context_before_a_restored_boot_transition() {
    let old = recorder::<2048>();
    old.take_batch().unwrap().unwrap();
    old.metric(1, MetricValue::Gauge(10.0)).unwrap();
    let next = Recorder::new(
        Session::new([8; 16], "new-build", "new-firmware", "reset").unwrap(),
        old.into_storage(),
        TestClock(AtomicU64::new(0)),
        TestBacktrace,
    )
    .unwrap();
    next.metric(1, MetricValue::Gauge(20.0)).unwrap();
    let batch = next.take_batch().unwrap().unwrap();
    let parcel = decode(&batch);
    assert_eq!(parcel.initial_boot.boot_id.as_slice(), [7; 16]);
    assert!(
        matches!(&parcel.events[1].payload, Some(pb::Event_::Payload::Boot(b)) if b.boot_id.as_slice() == [8;16])
    );
    next.metric(1, MetricValue::Gauge(30.0)).unwrap();
    let later = decode(&next.take_batch().unwrap().unwrap());
    assert_eq!(later.initial_boot.boot_id.as_slice(), [8; 16]);
    assert_eq!(decode(&batch), parcel);
}
