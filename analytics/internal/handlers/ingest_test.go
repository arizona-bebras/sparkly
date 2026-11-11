package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"sparkly/analytics/internal/domain"
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
	NewIngestHandler(&store).ServeHTTP(response, authenticatedIngestRequest(body))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if store.calls != 1 || store.batch.ReceivedAt.IsZero() || store.batch.ReceivedAt.Location() != time.UTC {
		t.Fatalf("store calls = %d, received at = %v", store.calls, store.batch.ReceivedAt)
	}
	if store.batch.Identity.DeviceID().String() != testDeviceID || store.batch.Identity.ParkID().String() != testParkID {
		t.Fatalf("stored identity = %s/%s", store.batch.Identity.ParkID(), store.batch.Identity.DeviceID())
	}
	var got agentv1.Parcel
	if err := protojson.Unmarshal(store.batch.Payload, &got); err != nil {
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
			NewIngestHandler(&store).ServeHTTP(response, authenticatedIngestRequest(tc.body))
			if response.Code != tc.code || store.calls != 0 {
				t.Fatalf("status = %d, store calls = %d", response.Code, store.calls)
			}
		})
	}
}

type recordingStore struct {
	batch domain.RawBatch
	calls int
	err   error
}

func (s *recordingStore) StoreRawBatch(_ context.Context, batch domain.RawBatch) error {
	s.calls++
	s.batch = batch
	s.batch.Payload = bytes.Clone(batch.Payload)
	return s.err
}

func TestIngestStorageFailure(t *testing.T) {
	response := httptest.NewRecorder()
	NewIngestHandler(&recordingStore{err: errors.New("database unavailable")}).ServeHTTP(response, authenticatedIngestRequest([]byte{0x0a, 1, 1}))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", response.Code)
	}
}

const (
	testDeviceID = "01900000-0000-7000-8000-00000000000a"
	testParkID   = "01900000-0000-7000-8000-00000000000b"
)

func authenticatedIngestRequest(body []byte) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
	request.Header.Set("X-Device-Id", testDeviceID)
	request.Header.Set("X-Park-Id", testParkID)
	return request
}

func TestIngestRejectsInvalidIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, header string
		values       []string
	}{
		{name: "no identity"},
		{name: "missing device", header: "X-Device-Id"},
		{name: "missing park", header: "X-Park-Id"},
		{name: "invalid device", header: "X-Device-Id", values: []string{"invalid"}},
		{name: "invalid park", header: "X-Park-Id", values: []string{"invalid"}},
		{name: "empty device", header: "X-Device-Id", values: []string{""}},
		{name: "empty park", header: "X-Park-Id", values: []string{""}},
		{name: "nil device", header: "X-Device-Id", values: []string{uuid.Nil.String()}},
		{name: "nil park", header: "X-Park-Id", values: []string{uuid.Nil.String()}},
		{name: "noncanonical device", header: "X-Device-Id", values: []string{"0190000000007000800000000000000a"}},
		{name: "uppercase park", header: "X-Park-Id", values: []string{"01900000-0000-7000-8000-00000000000B"}},
		{name: "duplicate device", header: "X-Device-Id", values: []string{testDeviceID, testDeviceID}},
		{name: "conflicting park", header: "X-Park-Id", values: []string{testParkID, testDeviceID}},
		{name: "combined values", header: "X-Device-Id", values: []string{testDeviceID + ", " + testParkID}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := authenticatedIngestRequest([]byte{0x0a, 1, 1})
			if tc.header == "" {
				request.Header = make(http.Header)
			} else {
				request.Header[http.CanonicalHeaderKey(tc.header)] = tc.values
			}
			var store recordingStore
			response := httptest.NewRecorder()
			NewIngestHandler(&store).ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized || store.calls != 0 {
				t.Fatalf("status = %d, store calls = %d", response.Code, store.calls)
			}
		})
	}
}
