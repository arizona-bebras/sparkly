package msgq

import (
	"os"
	"testing"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type parcelBrokerFixture struct {
	broker                 *MessageBroker
	publisher              *IngestPublisher
	channel                *amqp.Channel
	raw, requests, results string
}

// Each integration test owns a topology; no application messages are consumed.
func newParcelBrokerFixture(t *testing.T) parcelBrokerFixture {
	t.Helper()
	url := os.Getenv("RABBITMQ_TEST_URL")
	if url == "" {
		t.Skip("set RABBITMQ_TEST_URL for broker integration tests")
	}
	broker, err := NewMessageBroker(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { broker.Close() })
	exchange := "analytics-test-" + uuid.NewString()
	f := parcelBrokerFixture{broker: broker, raw: exchange + "-raw", requests: exchange + "-requests", results: exchange + "-results"}
	f.channel, err = broker.conn.Channel()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.channel.Close() })
	f.publisher, err = newIngestPublisher(broker, exchange, f.raw, f.requests, f.results)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, queue := range []string{f.raw, f.requests, f.results} {
			if _, err := f.channel.QueueDelete(queue, false, false, false); err != nil {
				t.Error(err)
			}
		}
		if err := f.channel.ExchangeDelete(exchange, false, false); err != nil {
			t.Error(err)
		}
	})
	return f
}
