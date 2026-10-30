package handlers

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	agentv1 "sparkly/analytics/proto/agent/v1"
)

func TestIngestPrintsParcel(t *testing.T) {
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
	var output bytes.Buffer
	response := httptest.NewRecorder()
	NewIngestHandler(&output).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body)))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	var got agentv1.Parcel
	if err := protojson.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(want, &got) {
		t.Fatalf("printed parcel = %v, want %v", &got, want)
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
			var output bytes.Buffer
			response := httptest.NewRecorder()
			NewIngestHandler(&output).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(tc.body)))
			if response.Code != tc.code || output.Len() != 0 {
				t.Fatalf("status = %d, output = %q", response.Code, output.String())
			}
		})
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestIngestOutputFailure(t *testing.T) {
	response := httptest.NewRecorder()
	NewIngestHandler(failingWriter{}).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader([]byte{0x0a, 1, 1})))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", response.Code)
	}
}
