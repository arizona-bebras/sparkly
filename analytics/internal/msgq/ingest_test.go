package msgq

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"sparkly/analytics/internal/handlers"
	"sparkly/analytics/internal/parcel"
	"sparkly/analytics/internal/services"
)

// Uses uniquely named queues and exchange; existing application queues are untouched.
func TestIngestRabbitMQ(t *testing.T) {
	fixture := newParcelBrokerFixture(t)
	publisher, ch := fixture.publisher, fixture.channel
	exchange := publisher.exchange
	raw, requests, results := fixture.raw, fixture.requests, fixture.results

	// Concurrent requests must get distinct IDs, with identical copies in both queues.
	const count = 4
	var wg sync.WaitGroup
	for range count {
		wg.Go(func() {
			r := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader([]byte{0x0a, 1, 1}))
			r.Header.Set("X-Device-Id", "01900000-0000-7000-8000-000000000001")
			r.Header.Set("X-Park-Id", "01900000-0000-7000-8000-000000000002")
			w := httptest.NewRecorder()
			handlers.NewIngestHandler(services.NewIngestService(parcel.Codec{}, publisher)).ServeHTTP(w, r)
			if w.Code != http.StatusNoContent {
				t.Errorf("status = %d: %s", w.Code, w.Body)
			}
		})
	}
	wg.Wait()
	seen := make(map[string][]byte)
	for _, queue := range []string{raw, requests} {
		q, err := ch.QueueInspect(queue)
		if err != nil {
			t.Fatal(err)
		}
		if q.Messages != count || q.Consumers != 0 {
			t.Fatalf("queue %s: %+v", queue, q)
		}
		for range count {
			msg, ok, err := ch.Get(queue, false)
			if err != nil || !ok {
				t.Fatalf("get %s: ok=%v err=%v", queue, ok, err)
			}
			var envelope ParcelMessage
			if err := json.Unmarshal(msg.Body, &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.SchemaVersion != 1 || envelope.BatchID.Version() != 7 || envelope.BatchID.String() != msg.MessageId || msg.DeliveryMode != 2 || msg.ContentType != "application/json" {
				t.Fatalf("invalid envelope or AMQP metadata: %+v / %+v", envelope, msg)
			}
			if envelope.ReceivedAt.IsZero() || envelope.DeviceID.String() != "01900000-0000-7000-8000-000000000001" || envelope.ParkID.String() != "01900000-0000-7000-8000-000000000002" || string(envelope.Payload) != `{"batchId":"AQ=="}` {
				t.Fatalf("invalid parcel metadata: %+v", envelope)
			}
			if queue == raw {
				if _, exists := seen[msg.MessageId]; exists {
					t.Fatal("duplicate batch ID")
				}
				seen[msg.MessageId] = msg.Body
			} else {
				if !bytes.Equal(seen[msg.MessageId], msg.Body) {
					t.Fatal("queue copies differ")
				}
				delete(seen, msg.MessageId)
			}
			if err := msg.Ack(false); err != nil {
				t.Fatal(err)
			}
		}
	}
	q, err := ch.QueueInspect(results)
	if err != nil || q.Messages != 0 || q.Consumers != 0 {
		t.Fatalf("results queue: %+v, %v", q, err)
	}
	for _, queue := range []string{raw, requests} {
		if err := ch.QueueUnbind(queue, "", exchange, nil); err != nil {
			t.Fatal(err)
		}
	}
	r := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader([]byte{0x0a, 1, 1}))
	r.Header.Set("X-Device-Id", "01900000-0000-7000-8000-000000000001")
	r.Header.Set("X-Park-Id", "01900000-0000-7000-8000-000000000002")
	w := httptest.NewRecorder()
	handlers.NewIngestHandler(services.NewIngestService(parcel.Codec{}, publisher)).ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("unroutable publication status = %d", w.Code)
	}
}
