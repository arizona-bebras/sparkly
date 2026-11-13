package msgq

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"sparkly/analytics/internal/domain"
	"sparkly/analytics/internal/services"
)

func TestRawWorkersRejectInvalidCount(t *testing.T) {
	for _, count := range []int{0, -1} {
		if _, err := NewRawBatchesWorkers(nil, nil, count); err == nil {
			t.Fatalf("accepted worker count %d", count)
		}
	}
}

func TestRawWorkersRabbitMQ(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	fixture := newParcelBrokerFixture(t)
	broker, publisher, ch, raw := fixture.broker, fixture.publisher, fixture.channel, fixture.raw
	started := make(chan uuid.UUID, 8)
	release := make(chan struct{})
	store := rawBatchStoreFunc(func(ctx context.Context, batch domain.RawBatch) error {
		started <- batch.ID
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	pool, err := newRawBatchesWorkers(broker, services.NewRawBatchPersistence(store), 4, raw)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for i, worker := range pool.workers {
		for _, other := range pool.workers[:i] {
			if worker.ch == other.ch {
				t.Fatal("workers share a channel")
			}
		}
	}
	if err := pool.Listen(ctx); err != nil {
		t.Fatal(err)
	}
	if err := pool.Listen(ctx); err == nil {
		t.Fatal("workers started twice")
	}
	for range 8 {
		message := validParcelMessage()
		identity, err := domain.NewDeviceIdentity(message.DeviceID, message.ParkID)
		if err != nil {
			t.Fatal(err)
		}
		if err := publisher.PublishRawBatch(ctx, domain.RawBatch{ID: message.BatchID, Identity: identity, ReceivedAt: message.ReceivedAt, Payload: message.Payload}); err != nil {
			t.Fatal(err)
		}
	}
	seen := make(map[uuid.UUID]bool)
	for range 4 {
		select {
		case id := <-started:
			seen[id] = true
		case <-ctx.Done():
			t.Fatal("four writes did not start concurrently")
		}
	}
	if len(seen) != 4 {
		t.Fatal("workers did not process distinct parcels")
	}
	q, err := ch.QueueInspect(raw)
	if err != nil {
		t.Fatal(err)
	}
	if q.Consumers != 4 || q.Messages != 4 {
		t.Fatalf("expected four active consumers and four queued parcels, got %+v", q)
	}
	close(release)
	for range 4 {
		select {
		case id := <-started:
			if seen[id] {
				t.Fatal("parcel processed twice")
			}
			seen[id] = true
		case <-ctx.Done():
			t.Fatal("remaining parcels did not progress")
		}
	}
	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
	q, err = ch.QueueInspect(raw)
	if err != nil || q.Consumers != 0 {
		t.Fatalf("workers did not stop: %+v, %v", q, err)
	}
}
