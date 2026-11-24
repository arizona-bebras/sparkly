package msgq

import (
	"bytes"
	"context"
	"math"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

// The legacy fixture tests queued v1 results, including their plain JSON UInt64 values.
const symbolicationResultFixture = `{
  "schema_version":1, "result_version":1,
  "batch_id":"0193aee0-3b00-7000-8000-000000000001",
  "device_id":"0193aee0-3b00-7000-8000-000000000002",
  "park_id":"0193aee0-3b00-7000-8000-000000000003",
  "received_at":"2025-01-02T03:04:05.123456789Z",
  "status":"partial", "symbol_build_ids":["abcd"],
  "result":{
    "producer_batch_id":"0102", "dropped_records":0, "symbol_build_id":"abcd",
    "boots":[{"event_index":0,"boot_id":"1234","build_id":"abcd","monotonic_us":0,"reset_reason":"power_on","raw_reset_reason":17,"firmware_version":"1.2.3","agent_version":"0.1.0"}],
    "logs":[{"event_index":1,"boot_id":"1234","build_id":"abcd","monotonic_us":1,"level":"INFO","message":"ready","target":"app::sensor","file":"src/sensor.rs","line":42,"frames":[{"pc":18446744073709551615,"module":"abcd","function":"entry","file":null,"line":null}],"fingerprint":"fingerprint","truncated":false,"trace_status":"UNSUPPORTED","defmt_raw_hex":null,"decode_status":"decoded"}],
    "crashes":[{"event_index":2,"boot_id":"1234","build_id":"abcd","monotonic_us":2,"reason":"panic","sp":18446744073709551615,"stack":"abcd","registers":{"pc":18446744073709551615},"frames":[{"pc":18446744073709551615,"module":"abcd","function":"entry","file":null,"line":null}]}],
    "metrics":[{"event_index":3,"boot_id":"1234","build_id":"abcd","monotonic_us":18446744073709551615,"metric_id":18446744073709551615,"name":null,"kind":"counter_delta","value":null,"counter_delta":18446744073709551615,"decode_status":"unresolved"}],
    "diagnostics":[{"event_index":3,"message":"unknown metric"},{"event_index":4,"message":"unsupported event"},{"event_index":null,"message":"ELF metadata warning"}]
  }
}`

const symbolicationFixtureBatchID = "0193aee0-3b00-7000-8000-000000000001"

func symbolicationFixtureDelivery() amqp.Delivery {
	return amqp.Delivery{ContentType: "application/json", MessageId: symbolicationFixtureBatchID,
		CorrelationId: symbolicationFixtureBatchID, Body: []byte(symbolicationResultFixture)}
}

func symbolicationV2FixtureDelivery() amqp.Delivery {
	msg := symbolicationFixtureDelivery()
	msg.Body = bytes.Replace(msg.Body, []byte(`"result_version":1`), []byte(`"result_version":2`), 1)
	msg.Body = bytes.Replace(msg.Body, []byte(`"target":"app::sensor","file":"src/sensor.rs","line":42,"frames":[`),
		[]byte(`"frames":[{"pc":null,"module":"app::sensor","function":null,"file":"src/sensor.rs","line":42},`), 1)
	msg.Body = bytes.ReplaceAll(msg.Body, []byte(`"module":"abcd"`), []byte(`"module":null`))
	return msg
}

func symbolicationV3FixtureDelivery() amqp.Delivery {
	msg := symbolicationV2FixtureDelivery()
	msg.Body = bytes.Replace(msg.Body, []byte(`"result_version":2`), []byte(`"result_version":3`), 1)
	for _, field := range []string{
		`"status":"partial", `, `"symbol_build_ids":["abcd"],`, `, "symbol_build_id":"abcd"`,
		`"fingerprint":"fingerprint",`, `,"decode_status":"decoded"`, `,"module":null`,
	} {
		msg.Body = bytes.ReplaceAll(msg.Body, []byte(field), nil)
	}
	return msg
}

func symbolicationV4FixtureDelivery() amqp.Delivery {
	msg := symbolicationV3FixtureDelivery()
	msg.Body = bytes.Replace(msg.Body, []byte(`"result_version":3`), []byte(`"result_version":4`), 1)
	msg.Body = bytes.Replace(msg.Body, []byte(`"frames":[{"pc":null,"module":"app::sensor","function":null,"file":"src/sensor.rs","line":42},`),
		[]byte(`"source":{"module":"app::sensor","file":"src/sensor.rs","line":42},"frames":[`), 1)
	return msg
}

func symbolicationV5FixtureDelivery() amqp.Delivery {
	msg := symbolicationV4FixtureDelivery()
	msg.Body = bytes.Replace(msg.Body, []byte(`"result_version":4`), []byte(`"result_version":5`), 1)
	msg.Body = bytes.Replace(msg.Body, []byte(`"source":{"module":"app::sensor","file":"src/sensor.rs","line":42}`),
		[]byte(`"module":"app::sensor"`), 1)
	return msg
}

func TestDecodeSymbolicatedBatch(t *testing.T) {
	for name, delivery := range map[string]amqp.Delivery{
		"v1": symbolicationFixtureDelivery(), "v2": symbolicationV2FixtureDelivery(),
		"v3": symbolicationV3FixtureDelivery(), "v4": symbolicationV4FixtureDelivery(), "v5": symbolicationV5FixtureDelivery(),
	} {
		t.Run(name, func(t *testing.T) {
			batch, err := decodeSymbolicatedBatch(delivery)
			if err != nil {
				t.Fatal(err)
			}
			metric := batch.Output.Metrics[0]
			crash := batch.Output.Crashes[0]
			if *metric.CounterDelta != math.MaxUint64 || *metric.MetricID != math.MaxUint64 ||
				*metric.MonotonicUS != math.MaxUint64 || *crash.Frames[0].PC != math.MaxUint64 ||
				crash.Registers["pc"] != math.MaxUint64 || batch.ReceivedAt.Nanosecond() != 123456789 {
				t.Fatalf("numeric precision lost: %+v", batch)
			}
			if batch.Output.Diagnostics[0].Unattached || !batch.Output.Diagnostics[1].Unattached {
				t.Fatalf("unexpected diagnostic attachment: %+v", batch.Output.Diagnostics)
			}
			entry := batch.Output.Logs[0]
			frames := entry.Frames
			if entry.Module == nil || *entry.Module != "app::sensor" ||
				len(frames) != 1 || frames[0].PC == nil || *frames[0].PC != math.MaxUint64 {
				t.Fatalf("unexpected module or stack frames: %+v", entry)
			}
		})
	}
}

func TestDecodeEmptySymbolicationResult(t *testing.T) {
	msg := symbolicationFixtureDelivery()
	msg.Body = []byte(`{
		"schema_version":1, "result_version":3,
		"batch_id":"0193aee0-3b00-7000-8000-000000000001",
		"device_id":"0193aee0-3b00-7000-8000-000000000002",
		"park_id":"0193aee0-3b00-7000-8000-000000000003",
		"received_at":"2025-01-02T03:04:05Z",
		"result":{"boots":[],"logs":[],"metrics":[],"crashes":[],"diagnostics":[]}
	}`)
	if _, err := decodeSymbolicatedBatch(msg); err != nil {
		t.Fatal(err)
	}
}

type resultRejectRecorder struct {
	mockAcknowledger
	rejected bool
	requeue  bool
}

func (r *resultRejectRecorder) Reject(_ uint64, requeue bool) error {
	r.rejected, r.requeue = true, requeue
	return nil
}

func TestSymbolicationResultsRejectMalformedDelivery(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*amqp.Delivery)
	}{
		{"wrong content type", func(msg *amqp.Delivery) { msg.ContentType = "text/plain" }},
		{"wrong correlation ID", func(msg *amqp.Delivery) { msg.CorrelationId = "unexpected" }},
		{"trailing JSON", func(msg *amqp.Delivery) { msg.Body = append(msg.Body, []byte(" {}")...) }},
		{"unsupported version", func(msg *amqp.Delivery) {
			msg.Body = bytes.Replace(msg.Body, []byte(`"result_version":1`), []byte(`"result_version":6`), 1)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			msg := symbolicationFixtureDelivery()
			tc.change(&msg)
			ack := &resultRejectRecorder{}
			msg.Acknowledger = ack
			consumer := &SymbolicatedResultsConsumer{}
			if err := consumer.handleDelivery(context.Background(), msg); err != nil {
				t.Fatal(err)
			}
			if !ack.rejected || ack.requeue || len(ack.acks) != 0 {
				t.Fatalf("unexpected acknowledgement: %+v", ack)
			}
		})
	}
}
