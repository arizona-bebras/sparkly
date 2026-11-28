package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sparkly/analytics/internal/db"
	"sparkly/analytics/internal/domain"
	"sparkly/analytics/internal/parcel"
	"sparkly/analytics/internal/services"
)

func TestRoutes(t *testing.T) {
	var store routeStore
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, _, err := db.Open(ctx)
	if err != nil {
		cancel()
		t.Fatalf("db init failed: %v", err)
	}
	mux := newMux(services.NewIngestService(parcel.Codec{}, &store), conn)
	request := httptest.NewRequest(http.MethodPost, "/sparkly.analytics.v1.AnalyticsService/GetSummary", strings.NewReader("{}"))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusNotImplemented || !strings.Contains(response.Body.String(), `"unimplemented"`) {
		t.Fatalf("RPC status = %d, body = %s", response.Code, response.Body)
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ingest", nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /ingest status = %d", response.Code)
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader([]byte{0x0a, 1, 1}))
	request.Header.Set("X-Device-Id", "01900000-0000-7000-8000-000000000001")
	request.Header.Set("X-Park-Id", "01900000-0000-7000-8000-000000000002")
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || len(store.payload) == 0 {
		t.Fatalf("POST /ingest status = %d, payload = %q", response.Code, store.payload)
	}
}

type routeStore struct{ payload []byte }

func (s *routeStore) PublishRawBatch(_ context.Context, batch domain.RawBatch) error {
	s.payload = append([]byte(nil), batch.Payload...)
	return nil
}
