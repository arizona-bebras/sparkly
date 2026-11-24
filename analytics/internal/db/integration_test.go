package db_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"sparkly/analytics/internal/db"
	"sparkly/analytics/internal/domain"
	"sparkly/analytics/internal/repos"
	agentv1 "sparkly/analytics/proto/agent/v1"
)

func testClickHouse(t *testing.T) (context.Context, *sql.DB) {
	t.Helper()
	if os.Getenv("CLICKHOUSE_TEST") != "1" {
		t.Skip("set CLICKHOUSE_TEST=1 to test against ClickHouse")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	admin, err := db.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })
	name := "analytics_test_" + uuid.New().String()[:8]
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(cleanup, "DROP DATABASE "+name); err != nil {
			t.Error(err)
		}
	})
	t.Setenv("CLICKHOUSE_DB", name)
	conn, err := db.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return ctx, conn
}

func TestClickHouseRawBatchRepository(t *testing.T) {
	ctx, conn := testClickHouse(t)
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
	payloadJSON, err := protojson.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	before := time.Now().UTC().Truncate(time.Millisecond)
	deviceID, parkID := uuid.New(), uuid.New()
	identity, err := domain.NewDeviceIdentity(deviceID, parkID)
	if err != nil {
		t.Fatal(err)
	}
	wantID := uuid.Must(uuid.NewV7())
	if err := repos.NewRawBatchesRepo(conn).StoreRawBatch(ctx, domain.RawBatch{
		ID: wantID, Identity: identity, ReceivedAt: before, Payload: payloadJSON,
	}); err != nil {
		t.Fatal(err)
	}
	var id, device, park, payload string
	var groups uint64
	var received time.Time
	// A successful write must be immediately visible, without polling for a flush.
	err = conn.QueryRowContext(ctx, `SELECT toString(batch_id), toString(device_id), toString(park_id),
   length(group_ids), received_at, toJSONString(payload_raw) FROM raw_batches`).Scan(
		&id, &device, &park, &groups, &received, &payload)
	if err != nil {
		t.Fatal(err)
	}
	if id != wantID.String() {
		t.Fatalf("batch ID = %s, want %s", id, wantID)
	}
	batchID, err := uuid.Parse(id)
	if err != nil || batchID.Version() != 7 {
		t.Fatalf("batch ID = %q, error = %v", id, err)
	}
	if device != deviceID.String() || park != parkID.String() || groups != 0 {
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

func TestClickHouseSymbolicatedEventsMigration(t *testing.T) {
	ctx, conn := testClickHouse(t)
	migrations, err := goose.NewProvider(goose.DialectClickHouse, conn, os.DirFS("migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := migrations.UpTo(ctx, 2); err != nil {
		t.Fatal(err)
	}

	assertEventSchemaCount := func(want uint64) {
		t.Helper()
		const query = `SELECT count() FROM system.tables WHERE database = currentDatabase()
			AND name IN ('logs', 'metrics', 'boots', 'crashes', 'logs_deduplicated',
				'metrics_deduplicated', 'boots_deduplicated', 'crashes_deduplicated')`
		var got uint64
		if err := conn.QueryRowContext(ctx, query).Scan(&got); err != nil || got != want {
			t.Fatalf("event schema objects: got %d, want %d, error %v", got, want, err)
		}
	}
	assertEventSchemaCount(0)
	if err := db.Migrate(ctx, conn); err != nil {
		t.Fatal(err)
	}
	assertEventSchemaCount(8)
	if _, err := migrations.DownTo(ctx, 2); err != nil {
		t.Fatal(err)
	}
	assertEventSchemaCount(0)
}
