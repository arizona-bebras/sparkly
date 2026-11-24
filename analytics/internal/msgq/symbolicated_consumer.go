package msgq

import (
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"sparkly/analytics/internal/domain"
)

type SymbolicatedBatchStore interface {
	StoreSymbolicatedBatch(context.Context, domain.SymbolicatedBatch) error
}

type SymbolicatedResultsConsumer struct {
	ch     *amqp.Channel
	queue  string
	store  SymbolicatedBatchStore
	cancel context.CancelFunc
	done   chan struct{}
}

func NewSymbolicatedResultsConsumer(broker *MessageBroker, store SymbolicatedBatchStore) (*SymbolicatedResultsConsumer, error) {
	return newSymbolicatedResultsConsumer(broker, store, SymbolicationResultsQueue)
}

func newSymbolicatedResultsConsumer(broker *MessageBroker, store SymbolicatedBatchStore, queue string) (*SymbolicatedResultsConsumer, error) {
	ch, err := broker.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open symbolication results channel: %w", err)
	}
	if err := ch.Qos(1, 0, false); err != nil {
		ch.Close()
		return nil, fmt.Errorf("set symbolication results prefetch: %w", err)
	}
	return &SymbolicatedResultsConsumer{ch: ch, queue: queue, store: store, done: make(chan struct{})}, nil
}

func (c *SymbolicatedResultsConsumer) Listen(ctx context.Context) (<-chan error, error) {
	if c.cancel != nil {
		return nil, fmt.Errorf("symbolication results consumer already started")
	}
	deliveries, err := c.ch.Consume(c.queue, "", false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("consume symbolication results: %w", err)
	}
	ctx, c.cancel = context.WithCancel(ctx)
	failure := make(chan error, 1)
	go func() {
		defer close(c.done)
		defer c.ch.Close()
		if err := c.consume(ctx, deliveries); err != nil && ctx.Err() == nil {
			failure <- err
		}
	}()
	return failure, nil
}

func (c *SymbolicatedResultsConsumer) Close() {
	if c.cancel == nil {
		c.ch.Close()
		return
	}
	c.cancel()
	<-c.done
}

func (c *SymbolicatedResultsConsumer) consume(ctx context.Context, deliveries <-chan amqp.Delivery) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("symbolication results deliveries closed")
			}
			if err := c.handleDelivery(ctx, msg); err != nil {
				return fmt.Errorf("handle symbolication result: %w", err)
			}
		}
	}
}

func (c *SymbolicatedResultsConsumer) handleDelivery(ctx context.Context, msg amqp.Delivery) error {
	batch, err := decodeSymbolicatedBatch(msg)
	if err != nil {
		log.Printf("reject symbolication result %q: %v", msg.MessageId, err)
		return msg.Reject(false)
	}

	for delay := time.Second; ; delay = min(2*delay, 30*time.Second) {
		if ctx.Err() != nil {
			return nil
		}
		if c.ch.IsClosed() {
			return fmt.Errorf("symbolication results channel closed")
		}
		writeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err = c.store.StoreSymbolicatedBatch(writeCtx, batch)
		cancel()
		if err == nil {
			break
		}
		log.Printf("store symbolication result %s failed, retrying in %s: %v", batch.ID, delay, err)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(delay):
		}
	}

	for _, diagnostic := range batch.Output.Diagnostics {
		if diagnostic.EventIndex == nil {
			log.Printf("symbolication batch %s: %q", batch.ID, diagnostic.Message)
		} else if diagnostic.Unattached {
			log.Printf("symbolication batch %s event %d: %q", batch.ID, *diagnostic.EventIndex, diagnostic.Message)
		}
	}
	return msg.Ack(false)
}
