package db_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"sparkly/analytics/internal/db"
	"sparkly/analytics/internal/handlers"
	"sparkly/analytics/internal/repos"
	agentv1 "sparkly/analytics/proto/agent/v1"
)

func TestClickHouseIngest(t *testing.T) {
	if os.Getenv("CLICKHOUSE_TEST") != "1" {
		t.Skip("set CLICKHOUSE_TEST=1 to test against ClickHouse")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	admin, err := db.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	name := "analytics_test_" + uuid.New().String()[:8]
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(cleanup, "DROP DATABASE "+name); err != nil {
			t.Error(err)
		}
	}()
	t.Setenv("CLICKHOUSE_DB", name)
	conn, err := db.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	for range 2 {
		if err := db.Migrate(ctx, conn); err != nil {
			t.Fatal(err)
		}
	}
	want := &agentv1.Parcel{
		BatchId: []byte("original-batch"),
		Events: []*agentv1.Event{
			{Payload: &agentv1.Event_Boot{
				Boot: &agentv1.Boot{BuildId: "build-'\\\n", ResetReason: "power_on"},
			}},
			{MonotonicUs: 10, Payload: &agentv1.Event_Metric{Metric: &agentv1.Metric{
				MetricId: 42, Measurement: &agentv1.Metric_Gauge{Gauge: 23.5},
			}}},
		},
	}
	body, err := proto.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	before := time.Now().UTC().Truncate(time.Millisecond)
	response := httptest.NewRecorder()
	handlers.NewIngestHandler(repos.NewRawBatchesRepo(conn)).ServeHTTP(response,
		httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body)).WithContext(ctx))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	var id, device, park, payload string
	var groups uint64
	var received time.Time
	// A successful response only acknowledges buffering; wait for the async flush.
	flushCtx, flushCancel := context.WithTimeout(ctx, 15*time.Second)
	defer flushCancel()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		err = conn.QueryRowContext(flushCtx, `SELECT toString(batch_id), toString(device_id), toString(park_id),
			length(group_ids), received_at, toJSONString(payload_raw) FROM raw_batches`).Scan(
			&id, &device, &park, &groups, &received, &payload)
		if err == nil {
			break
		}
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatal(err)
		}
		select {
		case <-flushCtx.Done():
			t.Fatalf("waiting for async batch persistence: %v", flushCtx.Err())
		case <-ticker.C:
		}
	}
	batchID, err := uuid.Parse(id)
	if err != nil || batchID.Version() != 7 {
		t.Fatalf("batch ID = %q, error = %v", id, err)
	}
	if device != uuid.Nil.String() || park != uuid.Nil.String() || groups != 0 {
		t.Fatalf("device = %s, park = %s, groups = %d", device, park, groups)
	}
	if received.Before(before) || received.After(time.Now()) {
		t.Fatalf("received_at = %s, before = %s", received, before)
	}
	var got agentv1.Parcel
	if err := protojson.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(want, &got) {
		t.Fatalf("stored payload = %s, want %v", payload, want)
	}
	var versions uint64
	if err := conn.QueryRowContext(ctx, "SELECT count() FROM goose_db_version WHERE version_id = 1 AND is_applied = 1").Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if versions != 1 {
		t.Fatalf("migration applied %d times", versions)
	}
}
