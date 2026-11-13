package msgq

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"sparkly/analytics/internal/domain"
)

const publishConfirmationTimeout = 10 * time.Second

type IngestPublisher struct {
	conn     *amqp.Connection
	exchange string
}

func NewIngestPublisher(broker *MessageBroker) (*IngestPublisher, error) {
	return newIngestPublisher(broker, ParcelExchange, RawPersistenceQueue, SymbolicationRequestsQueue, SymbolicationResultsQueue)
}

func newIngestPublisher(broker *MessageBroker, exchange, rawQueue, requestsQueue, resultsQueue string) (*IngestPublisher, error) {
	ch, err := broker.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open ingest topology channel: %w", err)
	}
	defer ch.Close()
	if err := declareParcelTopology(ch, exchange, rawQueue, requestsQueue, resultsQueue); err != nil {
		return nil, err
	}

	return &IngestPublisher{conn: broker.conn, exchange: exchange}, nil
}

func (p *IngestPublisher) PublishRawBatch(ctx context.Context, batch domain.RawBatch) error {
	body, err := encodeParcelMessage(batch)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, publishConfirmationTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	// Isolate concurrent requests and uncertain confirmations on their own channels.
	// The existing group-membership consumer owns a separate channel.
	ch, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("open ingest publish channel: %w", err)
	}
	defer ch.Close()
	if err := ch.Confirm(false); err != nil {
		return fmt.Errorf("enable publisher confirms: %w", err)
	}
	returns := ch.NotifyReturn(make(chan amqp.Return, 1))
	confirmation, err := ch.PublishWithDeferredConfirmWithContext(ctx, p.exchange, "", true, false, amqp.Publishing{
		ContentType: "application/json", DeliveryMode: amqp.Persistent,
		MessageId: batch.ID.String(), Timestamp: batch.ReceivedAt, Body: body,
	})
	if err != nil {
		return fmt.Errorf("publish parcel: %w", err)
	}
	acked, err := confirmation.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("wait for parcel confirmation: %w", err)
	}
	if !acked {
		return fmt.Errorf("broker rejected parcel")
	}
	// RabbitMQ sends mandatory returns before the corresponding confirmation.
	select {
	case returned := <-returns:
		return fmt.Errorf("parcel returned by broker: %s", returned.ReplyText)
	default:
		return nil
	}
}
