package msgq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ParcelExchange             = "analytics.parcels"
	RawPersistenceQueue        = "raw_persistance"
	SymbolicationRequestsQueue = "symbolication_requests"
	SymbolicationResultsQueue  = "symbolication_results"
)

func declareParcelTopology(ch *amqp.Channel, exchange, rawQueue, requestsQueue, resultsQueue string) error {
	if err := ch.ExchangeDeclare(exchange, "fanout", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare parcel exchange: %w", err)
	}
	for _, name := range []string{rawQueue, requestsQueue, resultsQueue} {
		if _, err := ch.QueueDeclare(name, true, false, false, false, nil); err != nil {
			return fmt.Errorf("declare %s: %w", name, err)
		}
	}
	// Only input queues receive parcels; results are produced by the symbolicator.
	for _, name := range []string{rawQueue, requestsQueue} {
		if err := ch.QueueBind(name, "", exchange, false, nil); err != nil {
			return fmt.Errorf("bind %s: %w", name, err)
		}
	}

	return nil
}
