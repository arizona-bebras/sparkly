package msgq

import (
	"context"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"sparkly/analytics/internal/db"
	"sparkly/analytics/internal/domain"
	"sparkly/analytics/internal/repos"
)

type symbolicatedStoreFunc func(context.Context, domain.SymbolicatedBatch) error

func (f symbolicatedStoreFunc) StoreSymbolicatedBatch(ctx context.Context, batch domain.SymbolicatedBatch) error {
	return f(ctx, batch)
}

func TestSymbolicationResultsRabbitMQClickHouse(t *testing.T) {
	if os.Getenv("CLICKHOUSE_TEST") != "1" {
		t.Skip("set CLICKHOUSE_TEST=1 and RABBITMQ_TEST_URL for end-to-end persistence")
	}
	fixture := newParcelBrokerFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := db.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	name := "analytics_test_" + uuid.NewString()[:8]
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(ctx, "DROP DATABASE "+name); err != nil {
			t.Error(err)
		}
	}()
	t.Setenv("CLICKHOUSE_DB", name)
	conn, err := db.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := db.Migrate(ctx, conn); err != nil {
		t.Fatal(err)
	}

	// Fail the last insert so the delivery must wait for the crash table.
	for _, query := range []string{"SYSTEM STOP MERGES logs", "RENAME TABLE crashes TO crashes_unavailable"} {
		if _, err := conn.ExecContext(ctx, query); err != nil {
			t.Fatal(err)
		}
	}

	firstError := make(chan error, 1)
	successes := make(chan struct{}, 2)
	repo := repos.NewSymbolicatedEventsRepo(conn)
	store := symbolicatedStoreFunc(func(ctx context.Context, batch domain.SymbolicatedBatch) error {
		err := repo.StoreSymbolicatedBatch(ctx, batch)
		if err != nil {
			select {
			case firstError <- err:
			default:
			}
		} else {
			successes <- struct{}{}
		}
		return err
	})
	consumer, err := newSymbolicatedResultsConsumer(fixture.broker, store, fixture.results)
	if err != nil {
		t.Fatal(err)
	}
	defer consumer.Close()
	failure, err := consumer.Listen(ctx)
	if err != nil {
		t.Fatal(err)
	}
	publish := func(msg amqp.Delivery) {
		t.Helper()
		err := fixture.channel.PublishWithContext(ctx, "", fixture.results, true, false, amqp.Publishing{
			ContentType: msg.ContentType, MessageId: msg.MessageId, CorrelationId: msg.CorrelationId,
			Body: msg.Body, DeliveryMode: amqp.Persistent,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	publish(symbolicationV5FixtureDelivery())

	select {
	case err := <-firstError:
		if !strings.Contains(err.Error(), "insert crashes") {
			t.Fatal(err)
		}
	case err := <-failure:
		t.Fatal(err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	assertCount := func(query string, want uint64) {
		t.Helper()
		var count uint64
		if err := conn.QueryRowContext(ctx, query).Scan(&count); err != nil || count != want {
			t.Fatalf("%s returned %d, want %d, err %v", query, count, want, err)
		}
	}
	assertCount("SELECT count() FROM logs_deduplicated", 1)
	assertCount("SELECT count() FROM boots_deduplicated", 1)
	assertCount("SELECT count() FROM metrics_deduplicated", 1)

	if _, err := conn.ExecContext(ctx, "RENAME TABLE crashes_unavailable TO crashes"); err != nil {
		t.Fatal(err)
	}
	publish(symbolicationV4FixtureDelivery()) // The equivalent legacy result with the same batch ID.
	for range 2 {
		select {
		case <-successes:
		case err := <-failure:
			t.Fatal(err)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	consumer.Close()
	queue, err := fixture.channel.QueueInspect(fixture.results)
	if err != nil || queue.Messages != 0 || queue.Consumers != 0 {
		t.Fatalf("expected both deliveries acknowledged: %+v, err %v", queue, err)
	}
	assertCount("SELECT count() FROM logs_deduplicated WHERE message = 'ready'", 1)
	assertCount("SELECT count() FROM logs_deduplicated", 1)
	assertCount("SELECT count() FROM crashes_deduplicated", 1)
	assertCount("SELECT count() FROM metrics_deduplicated", 1)
	assertCount("SELECT count() FROM boots_deduplicated WHERE reset_reason = 'power_on' AND raw_reset_reason = 17", 1)
	assertCount("SELECT count() FROM raw_batches", 0)
	assertCount(`SELECT count() FROM logs_deduplicated WHERE length(frames) = 1
		AND module = 'app::sensor'
		AND tupleElement(frames[1], 'pc') = toUInt64('18446744073709551615')`, 1)

	var copies uint64
	if err := conn.QueryRowContext(ctx, "SELECT count() FROM logs").Scan(&copies); err != nil || copies < 3 {
		t.Fatalf("expected physical log duplicates with merges disabled: %d, err %v", copies, err)
	}
	var counter, monotonic uint64
	var receivedAt time.Time
	var diagnostic string
	err = conn.QueryRowContext(ctx, `SELECT counter_delta, monotonic_us, received_at, diagnostics[1]
		FROM metrics_deduplicated`).Scan(&counter, &monotonic, &receivedAt, &diagnostic)
	if err != nil || counter != math.MaxUint64 || monotonic != math.MaxUint64 ||
		receivedAt.Nanosecond() != 123456789 || diagnostic != "unknown metric" {
		t.Fatalf("unexpected metric: %d %d %s %q, err %v", counter, monotonic, receivedAt, diagnostic, err)
	}
	var pc, register uint64
	var reason string
	err = conn.QueryRowContext(ctx, `SELECT tupleElement(frames[1], 'pc'), registers['pc'], reason
		FROM crashes_deduplicated`).Scan(&pc, &register, &reason)
	if err != nil || pc != math.MaxUint64 || register != math.MaxUint64 || reason != "panic" {
		t.Fatalf("unexpected crash: %d %d %q, err %v", pc, register, reason, err)
	}
}
