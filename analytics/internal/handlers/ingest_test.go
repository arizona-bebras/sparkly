package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	agentv1 "sparkly/analytics/proto/agent/v1"
)

func TestIngestStoresParcel(t *testing.T) {
	want := &agentv1.Parcel{
		BatchId: []byte("batch-id"),
		Events: []*agentv1.Event{
			{Payload: &agentv1.Event_Boot{Boot: &agentv1.Boot{BuildId: "firmware-build", ResetReason: "power_on"}}},
			{MonotonicUs: 10, Payload: &agentv1.Event_Metric{Metric: &agentv1.Metric{MetricId: 42, Measurement: &agentv1.Metric_Gauge{Gauge: 23.5}}}},
		},
	}
	body, err := proto.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var store recordingStore
	response := httptest.NewRecorder()
	NewIngestHandler(&store).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body)))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if store.calls != 1 || store.receivedAt.IsZero() || store.receivedAt.Location() != time.UTC {
		t.Fatalf("store calls = %d, received at = %v", store.calls, store.receivedAt)
	}
	var got agentv1.Parcel
	if err := protojson.Unmarshal(store.payload, &got); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(want, &got) {
		t.Fatalf("stored parcel = %v, want %v", &got, want)
	}
}

func TestIngestRejectsInvalidBodies(t *testing.T) {
	for _, tc := range []struct {
		name string
		body []byte
		code int
	}{
		{"empty", nil, http.StatusBadRequest},
		{"malformed", []byte{0x0a, 0xff}, http.StatusBadRequest},
		{"oversized", make([]byte, maxParcelBytes+1), http.StatusRequestEntityTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var store recordingStore
			response := httptest.NewRecorder()
			NewIngestHandler(&store).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(tc.body)))
			if response.Code != tc.code || store.calls != 0 {
				t.Fatalf("status = %d, store calls = %d", response.Code, store.calls)
			}
		})
	}
}

type recordingStore struct {
	payload    []byte
	calls      int
	receivedAt time.Time
	err        error
}

func (s *recordingStore) StoreRawBatch(ctx context.Context, payload []byte, receivedAt time.Time) error {
	s.calls++
	s.payload = append([]byte(nil), payload...)
	s.receivedAt = receivedAt
	return s.err
}

func TestIngestStorageFailure(t *testing.T) {
	response := httptest.NewRecorder()
	NewIngestHandler(&recordingStore{err: errors.New("database unavailable")}).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader([]byte{0x0a, 1, 1})))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", response.Code)
	}
}
