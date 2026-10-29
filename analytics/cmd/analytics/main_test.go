package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoutes(t *testing.T) {
	var output bytes.Buffer
	mux := newMux(&output)
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
	if response.Code != http.StatusNoContent || output.Len() == 0 {
		t.Fatalf("POST /ingest status = %d, output = %q", response.Code, output.String())
	}
}
