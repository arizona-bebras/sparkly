use sparkly_core::*;

fn metric(value: u64) -> Record {
    Record {
        monotonic_us: value,
        payload: Payload::Metric {
            id: u64::MAX,
            value: MetricValue::CounterDelta(value),
        },
    }
}
fn boot() -> Record {
    Record {
        monotonic_us: u64::MAX,
        payload: Payload::Boot(Session::new([255; 16], "build", "firmware", "reset").unwrap()),
    }
}

#[test]
fn metrics_use_28_bytes_and_reuse_freed_capacity() {
    let mut storage = RamStorage::<56>::new();
    assert_eq!(storage.capacity_bytes(), 56);
    assert_eq!(RamStorage::<56>::record_size(&metric(0)), 28);
    storage.push(metric(0)).unwrap();
    storage.push(metric(1)).unwrap();
    assert_eq!(storage.used_bytes(), 56);
    assert_eq!(storage.push(metric(2)).unwrap().evicted, 1);
    assert_eq!(storage.pop(), Some(metric(1)));
    storage.push(metric(3)).unwrap();
    assert_eq!(storage.pop(), Some(metric(2)));
    assert_eq!(storage.pop(), Some(metric(3)));
    assert_eq!(storage.used_bytes(), 0);
    assert!(storage.pop().is_none());
}

#[test]
fn large_record_evicts_oldest_records_including_boots() {
    let mut storage = RamStorage::<180>::new();
    storage.push(metric(0)).unwrap();
    storage.push(boot()).unwrap();
    storage.push(metric(1)).unwrap();
    storage.push(metric(2)).unwrap();
    let text = Record {
        monotonic_us: 7,
        payload: Payload::Text(TextLog {
            level: Level::Warn,
            target: "test".try_into().unwrap(),
            message: "x".repeat(80).as_str().try_into().unwrap(),
            truncated: false,
            trace: StackTrace::default(),
        }),
    };
    assert_eq!(storage.push(text.clone()).unwrap().evicted, 2);
    assert_eq!(storage.pop(), Some(metric(1)));
    assert_eq!(storage.pop(), Some(metric(2)));
    assert_eq!(storage.pop(), Some(text));
    assert!(storage.pop().is_none());
}

#[test]
fn failed_push_does_not_partially_evict_history() {
    let mut storage = RamStorage::<128>::new();
    storage.push(metric(0)).unwrap();
    storage.push(boot()).unwrap();
    storage.push(metric(1)).unwrap();
    let used = storage.used_bytes();
    let large = Record {
        monotonic_us: 0,
        payload: Payload::Defmt(DefmtLog {
            frame: [1; 128].as_slice().try_into().unwrap(),
            trace: StackTrace::default(),
        }),
    };
    assert_eq!(storage.push(large), Err(StorageFull));
    assert_eq!(storage.used_bytes(), used);
    assert_eq!(storage.pop(), Some(metric(0)));
    assert_eq!(storage.pop(), Some(boot()));
    assert_eq!(storage.pop(), Some(metric(1)));
    let mut tiny = RamStorage::<27>::new();
    assert_eq!(tiny.push(metric(0)), Err(StorageFull));
    assert_eq!(tiny.used_bytes(), 0);
}

fn roundtrip(record: Record) {
    let mut storage = RamStorage::<1024>::new();
    let size = RamStorage::<1024>::record_size(&record);
    storage.push(record.clone()).unwrap();
    assert_eq!(storage.used_bytes(), size);
    assert_eq!(storage.pop(), Some(record));
    assert_eq!(storage.used_bytes(), 0);
}

#[test]
fn all_payloads_roundtrip_with_maximum_lengths_and_trace_states() {
    let mut session =
        Session::new([255; 16], &"b".repeat(64), &"f".repeat(32), &"r".repeat(64)).unwrap();
    session.agent_version = "a".repeat(32).as_str().try_into().unwrap();
    for reason in [None, Some(u32::MAX)] {
        session.raw_reset_reason = reason;
        roundtrip(Record {
            monotonic_us: u64::MAX,
            payload: Payload::Boot(session.clone()),
        });
    }
    for status in [
        TraceStatus::Unsupported,
        TraceStatus::Complete,
        TraceStatus::Truncated,
        TraceStatus::Partial,
    ] {
        for count in [0, MAX_FRAMES] {
            let trace = StackTrace {
                status,
                pcs: (0..count).map(|i| u64::MAX - i as u64).collect(),
            };
            for level in [
                Level::Trace,
                Level::Debug,
                Level::Info,
                Level::Warn,
                Level::Error,
            ] {
                for truncated in [false, true] {
                    roundtrip(Record {
                        monotonic_us: 42,
                        payload: Payload::Text(TextLog {
                            level,
                            target: "t".repeat(MAX_TARGET).as_str().try_into().unwrap(),
                            message: "é".repeat(MAX_TEXT / 2).as_str().try_into().unwrap(),
                            truncated,
                            trace: trace.clone(),
                        }),
                    });
                }
            }
            for len in [0, MAX_DEFMT] {
                roundtrip(Record {
                    monotonic_us: 0,
                    payload: Payload::Defmt(DefmtLog {
                        frame: vec![255; len].as_slice().try_into().unwrap(),
                        trace: trace.clone(),
                    }),
                });
            }
        }
    }
    roundtrip(metric(u64::MAX));
    for value in [-0.0, f64::INFINITY, f64::NEG_INFINITY, f64::MAX] {
        roundtrip(Record {
            monotonic_us: 1,
            payload: Payload::Metric {
                id: 0,
                value: MetricValue::Gauge(value),
            },
        });
    }
}

#[test]
fn gauge_nan_bits_are_preserved() {
    let bits = 0x7ff8_0000_0000_0042;
    let mut storage = RamStorage::<28>::new();
    storage
        .push(Record {
            monotonic_us: 0,
            payload: Payload::Metric {
                id: 0,
                value: MetricValue::Gauge(f64::from_bits(bits)),
            },
        })
        .unwrap();
    let Payload::Metric {
        value: MetricValue::Gauge(value),
        ..
    } = storage.pop().unwrap().payload
    else {
        panic!()
    };
    assert_eq!(value.to_bits(), bits);
}

#[test]
fn mixed_operations_match_fifo_model_across_wraparound_and_rejection() {
    let mut storage = RamStorage::<512>::new();
    let mut model: Vec<Record> = Vec::new();
    let mut seed = 17_u64;
    for i in 0..2000 {
        seed = seed.wrapping_mul(6364136223846793005).wrapping_add(1);
        if seed.is_multiple_of(5) {
            let expected = if model.is_empty() {
                None
            } else {
                Some(model.remove(0))
            };
            assert_eq!(storage.pop(), expected);
        } else {
            let record = match (seed >> 32) % 4 {
                0 => boot(),
                1 => Record {
                    monotonic_us: i,
                    payload: Payload::Text(TextLog {
                        level: Level::Info,
                        target: "model".try_into().unwrap(),
                        message: "x"
                            .repeat((seed % 200) as usize)
                            .as_str()
                            .try_into()
                            .unwrap(),
                        truncated: false,
                        trace: StackTrace::default(),
                    }),
                },
                2 => Record {
                    monotonic_us: i,
                    payload: Payload::Defmt(DefmtLog {
                        frame: vec![i as u8; (seed % 512) as usize]
                            .as_slice()
                            .try_into()
                            .unwrap(),
                        trace: StackTrace::default(),
                    }),
                },
                _ => metric(i),
            };
            let size = RamStorage::<512>::record_size;
            let expected = if size(&record) > 512 {
                Err(StorageFull)
            } else {
                let mut evicted = 0;
                while model.iter().map(size).sum::<usize>() + size(&record) > 512 {
                    model.remove(0);
                    evicted += 1;
                }
                model.push(record.clone());
                Ok(PushOutcome { evicted })
            };
            assert_eq!(storage.push(record), expected);
        }
        assert_eq!(
            storage.used_bytes(),
            model
                .iter()
                .map(RamStorage::<512>::record_size)
                .sum::<usize>()
        );
    }
    for record in model {
        assert_eq!(storage.pop(), Some(record));
    }
    assert!(storage.pop().is_none());
}

#[test]
fn wrapped_length_header_can_be_read_and_evicted() {
    // Two 28-byte records leave the head at byte 56, the last byte of the ring.
    let mut storage = RamStorage::<57>::new();
    for i in 0..2 {
        storage.push(metric(i)).unwrap();
        assert_eq!(storage.pop(), Some(metric(i)));
    }
    storage.push(metric(2)).unwrap(); // length header crosses the boundary
    storage.push(metric(3)).unwrap();
    assert_eq!(storage.push(metric(4)).unwrap().evicted, 1);
    assert_eq!(storage.pop(), Some(metric(3)));
    assert_eq!(storage.pop(), Some(metric(4)));
    assert!(storage.pop().is_none());
}

#[test]
fn maximum_frame_and_utf8_text_roundtrip_at_every_ring_offset() {
    let records = [
        Record {
            monotonic_us: u64::MAX,
            payload: Payload::Defmt(DefmtLog {
                frame: [255; MAX_DEFMT].as_slice().try_into().unwrap(),
                trace: StackTrace {
                    status: TraceStatus::Truncated,
                    pcs: (0..MAX_FRAMES).map(|i| u64::MAX - i as u64).collect(),
                },
            }),
        },
        Record {
            monotonic_us: 1,
            payload: Payload::Text(TextLog {
                level: Level::Error,
                target: "target".try_into().unwrap(),
                message: "é".repeat(MAX_TEXT / 2).as_str().try_into().unwrap(),
                truncated: true,
                trace: StackTrace::default(),
            }),
        },
    ];
    for record in records {
        // 1021 is prime. Advancing by 28 + record_size visits every head offset,
        // exercising splits inside UTF-8 code points, lengths, and trace addresses.
        let mut storage = RamStorage::<1021>::new();
        for i in 0..1021 {
            storage.push(metric(i)).unwrap();
            assert_eq!(storage.pop(), Some(metric(i)));
            storage.push(record.clone()).unwrap();
            assert_eq!(storage.pop(), Some(record.clone()));
        }
    }
}

fn exercise_byte_storage<B: ByteStorage>(backing: B) -> B {
    let mut storage = RingStorage::with_storage(backing);
    assert_eq!(storage.capacity_bytes(), 57);
    assert_eq!(storage.used_bytes(), 0);
    for i in 0..100 {
        assert_eq!(storage.push(metric(i)).unwrap().evicted, u64::from(i >= 2));
    }
    // An oversized push must preserve the two retained records even after wrap.
    let oversized = Record {
        monotonic_us: 0,
        payload: Payload::Defmt(DefmtLog {
            frame: [0; 64].as_slice().try_into().unwrap(),
            trace: StackTrace::default(),
        }),
    };
    assert_eq!(storage.push(oversized), Err(StorageFull));
    assert_eq!(storage.used_bytes(), 56);
    assert_eq!(storage.pop(), Some(metric(98)));
    assert_eq!(storage.pop(), Some(metric(99)));
    assert!(storage.pop().is_none());
    storage.into_backing()
}

#[test]
fn ring_supports_owned_arrays_borrowed_slices_and_heap_buffers() {
    exercise_byte_storage([0_u8; 57]);
    let mut memory = [0_u8; 57];
    exercise_byte_storage(&mut memory[..]);
    exercise_byte_storage(vec![0_u8; 57].into_boxed_slice());

    // The budget is Vec length, not its potentially larger allocation capacity.
    let mut heap = Vec::with_capacity(128);
    heap.resize(57, 0_u8);
    let pointer = heap.as_ptr();
    let allocation = heap.capacity();
    let heap = exercise_byte_storage(heap);
    assert_eq!(heap.as_ptr(), pointer);
    assert_eq!(heap.capacity(), allocation);
}

#[test]
fn custom_byte_storage_exposes_only_its_reserved_region() {
    struct Region {
        memory: [u8; 89],
    }
    impl ByteStorage for Region {
        fn as_bytes(&self) -> &[u8] {
            &self.memory[16..73]
        }
        fn as_bytes_mut(&mut self) -> &mut [u8] {
            &mut self.memory[16..73]
        }
    }
    let region = exercise_byte_storage(Region { memory: [0xa5; 89] });
    assert_eq!(&region.memory[..16], &[0xa5; 16]);
    assert_eq!(&region.memory[73..], &[0xa5; 16]);
}

#[test]
fn constructing_from_used_backing_starts_empty_without_clearing_memory() {
    let storage = RingStorage::with_storage([0xa5_u8; 57]);
    assert_eq!(storage.used_bytes(), 0);
    let mut storage = RingStorage::with_storage(storage.into_backing());
    assert!(storage.pop().is_none());
    assert_eq!(storage.into_backing(), [0xa5_u8; 57]);
}

#[test]
#[should_panic(expected = "byte capacity must be positive")]
fn empty_backing_is_rejected() {
    RingStorage::with_storage(Vec::<u8>::new());
}
