package msgq

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"sparkly/analytics/internal/db"
	"sparkly/analytics/internal/domain"
	"sparkly/analytics/internal/repos"
	"sparkly/analytics/internal/services"
)

func TestRawConsumerRabbitMQClickHouse(t *testing.T) {
	url := os.Getenv("RABBITMQ_TEST_URL")
	if url == "" || os.Getenv("CLICKHOUSE_TEST") != "1" {
		t.Skip("set RABBITMQ_TEST_URL and CLICKHOUSE_TEST=1 for end-to-end persistence")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, _, err := db.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	name := "analytics_test_" + uuid.NewString()[:8]
	if err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := admin.Exec(cleanup, "DROP DATABASE "+name); err != nil {
			t.Error(err)
		}
	}()
	t.Setenv("CLICKHOUSE_DB", name)
	conn, migrationDB, err := db.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := db.Migrate(ctx, migrationDB); err != nil {
		t.Fatal(err)
	}
	fixture := newParcelBrokerFixture(t)
	broker, publisher, ch := fixture.broker, fixture.publisher, fixture.channel
	raw, requests, results := fixture.raw, fixture.requests, fixture.results
	consumer, err := newRawBatchesWorkers(broker, services.NewRawBatchPersistence(repos.NewRawBatchesRepo(conn)), 4, raw)
	if err != nil {
		t.Fatal(err)
	}
	defer consumer.Close()
	if err := consumer.Listen(ctx); err != nil {
		t.Fatal(err)
	}
	want := validParcelMessage()
	want.ReceivedAt = want.ReceivedAt.Truncate(time.Millisecond)
	identity, err := domain.NewDeviceIdentity(want.DeviceID, want.ParkID)
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.PublishRawBatch(ctx, domain.RawBatch{ID: want.BatchID, Identity: identity, ReceivedAt: want.ReceivedAt, Payload: want.Payload}); err != nil {
		t.Fatal(err)
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		var count uint64
		if err := conn.QueryRow(ctx, "SELECT count() FROM raw_batches").Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count == 1 {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-ticker.C:
		}
	}
	// Stop and close the consumer channel: any unacknowledged delivery would be requeued.
	if err := consumer.Close(); err != nil {
		t.Fatal(err)
	}
	var id, device, park, payload string
	var received time.Time
	if err := conn.QueryRow(ctx, `SELECT toString(batch_id), toString(device_id), toString(park_id), received_at, toJSONString(payload_raw) FROM raw_batches`).Scan(&id, &device, &park, &received, &payload); err != nil {
		t.Fatal(err)
	}
	if id != want.BatchID.String() || device != want.DeviceID.String() || park != want.ParkID.String() || !received.Equal(want.ReceivedAt) {
		t.Fatalf("stored metadata differs: %s %s %s %v", id, device, park, received)
	}
	var gotPayload, wantPayload any
	if err := json.Unmarshal([]byte(payload), &gotPayload); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want.Payload, &wantPayload); err != nil {
		t.Fatal(err)
	}
	gotJSON, _ := json.Marshal(gotPayload)
	wantJSON, _ := json.Marshal(wantPayload)
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("stored payload = %s, want %s", gotJSON, wantJSON)
	}
	for queue, expected := range map[string]int{raw: 0, requests: 1, results: 0} {
		q, err := ch.QueueInspect(queue)
		if err != nil {
			t.Fatal(err)
		}
		if q.Messages != expected || q.Consumers != 0 {
			t.Fatalf("queue %s: %+v, expected %d messages and no consumers", queue, q, expected)
		}
	}
}
