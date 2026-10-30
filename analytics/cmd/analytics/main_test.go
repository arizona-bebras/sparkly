package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRoutes(t *testing.T) {
	var store routeStore
	mux := newMux(&store)
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
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader([]byte{0x0a, 1, 1})))
	if response.Code != http.StatusNoContent || len(store.payload) == 0 {
		t.Fatalf("POST /ingest status = %d, payload = %q", response.Code, store.payload)
	}
}

type routeStore struct{ payload []byte }

func (s *routeStore) StoreRawBatch(_ context.Context, payload []byte, _ time.Time) error {
	s.payload = append([]byte(nil), payload...)
	return nil
}
