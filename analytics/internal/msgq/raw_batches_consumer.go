package msgq

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"sparkly/analytics/internal/domain"
)

type RawBatchPersister interface {
	PersistRawBatch(context.Context, domain.RawBatch) error
}

type RawBatchesConsumer struct {
	ch        *amqp.Channel
	queueName string
	persister RawBatchPersister
	cancel    context.CancelFunc
	done      chan struct{}
}

func NewRawBatchesConsumer(broker *MessageBroker, persister RawBatchPersister) (*RawBatchesConsumer, error) {
	return newRawBatchesConsumer(broker, persister, RawPersistenceQueue)
}

func newRawBatchesConsumer(broker *MessageBroker, persister RawBatchPersister, queue string) (*RawBatchesConsumer, error) {
	if persister == nil {
		return nil, fmt.Errorf("raw batch persister cannot be nil")
	}
	ch, err := broker.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open raw persistence channel: %w", err)
	}
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		ch.Close()
		return nil, fmt.Errorf("declare raw persistence queue: %w", err)
	}
	if err := ch.Qos(1, 0, false); err != nil {
		ch.Close()
		return nil, fmt.Errorf("set raw persistence prefetch: %w", err)
	}
	return &RawBatchesConsumer{ch: ch, queueName: queue, persister: persister, done: make(chan struct{})}, nil
}

// A storage failure leaves the delivery eligible for retry; invalid envelopes are rejected.
func (c *RawBatchesConsumer) handleDelivery(ctx context.Context, msg amqp.Delivery) (retry bool, err error) {
	batch, validationErr := decodeRawBatch(msg)
	if validationErr != nil {
		log.Printf("raw persistence rejected message %s: %v", msg.MessageId, validationErr)
		return false, msg.Nack(false, false)
	}
	if err := c.persister.PersistRawBatch(ctx, batch); err != nil {
		log.Printf("raw persistence batch %s: %v", batch.ID, err)
		requeue := !errors.Is(err, domain.ErrInvalidRawBatch)
		return requeue, msg.Nack(false, requeue)
	}

	return false, msg.Ack(false)
}

const rawPersistenceRetryDelay = time.Second

func (c *RawBatchesConsumer) Listen(ctx context.Context) error {
	if c.cancel != nil {
		return fmt.Errorf("raw persistence consumer already started")
	}
	msgs, err := c.ch.Consume(c.queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("start raw persistence consumer: %w", err)
	}
	ctx, c.cancel = context.WithCancel(ctx)
	go c.consumeLoop(ctx, msgs)
	return nil
}

func (c *RawBatchesConsumer) consumeLoop(ctx context.Context, msgs <-chan amqp.Delivery) {
	defer close(c.done)
	defer c.ch.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgs:
			if !ok {
				log.Printf("raw persistence deliveries closed; restart analytics to reconnect")
				return
			}
			retry, err := c.handleDelivery(ctx, msg)
			if err != nil {
				log.Printf("raw persistence acknowledgement failed: %v", err)
				return
			}
			if retry {
				select {
				case <-ctx.Done():
					return
				case <-time.After(rawPersistenceRetryDelay):
				}
			}
		}
	}
}

// Close cancels pending writes and waits before the database connection is closed.
func (c *RawBatchesConsumer) Close() error {
	if c.cancel != nil {
		c.cancel()
		<-c.done
		return nil
	}
	err := c.ch.Close()
	if errors.Is(err, amqp.ErrClosed) {
		return nil
	}
	return err
}
