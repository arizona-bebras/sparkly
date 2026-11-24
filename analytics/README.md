# Analytics

Run `go run ./cmd/analytics` from this directory. The server loads `.env` when
present and listens on `PORT` (default `8081`). The existing AnalyticsService is
mounted using ConnectRPC; all RPC methods currently return unimplemented.

ClickHouse must be available before startup. Start the local database with
`docker compose up -d clickhouse` from the repository root. Connection settings:

| Variable | Default |
| --- | --- |
| `CLICKHOUSE_HOST` | `localhost` |
| `CLICKHOUSE_NATIVE_PORT` | `9000` |
| `CLICKHOUSE_DB` | `sparkly` |
| `CLICKHOUSE_USER` | `sparkly` |
| `CLICKHOUSE_PASSWORD` | `sparkly` |

The database must already exist (Compose creates it). Startup checks the
connection and applies embedded SQL migrations using Goose, recording versions
in `goose_db_version`. Add numbered SQL files under `internal/db/migrations`
with Goose Up/Down annotations and `NO TRANSACTION`; ClickHouse DDL is not
transactional. Only one service instance should run migrations at a time.

`POST /ingest` accepts binary protobuf device parcels (up to 1 MiB), validates
identity and protobuf encoding, and converts them to protobuf JSON. Analytics
assigns one server UUIDv7 `batch_id` before publishing a persistent JSON envelope
to the durable `analytics.parcels` fanout exchange. The durable queues
`raw_persistance` (spelling intentional) and `symbolication_requests` each receive
an identical copy with the same batch ID. `symbolication_results` is also declared,
but is not bound to the parcel exchange and receives no ingest messages.

The version-1 envelope contains `schema_version`, `batch_id`, authenticated
`device_id` and `park_id`, UTC `received_at`, and `payload` (a nested protobuf JSON
object). The producer's batch ID remains inside `payload.batchId`. AMQP
`message_id` also contains the server batch ID. Protobuf JSON preserves base64
bytes and decimal strings for 64-bit integers.

Requests return 204 after a positive publisher confirmation, 401 for invalid
identity, 400/413 for invalid input, or 503 on broker failure, an unroutable
message, or confirmation timeout. Waiting for confirmation is limited to 10 seconds
and respects request cancellation.
A confirmation does not mean a consumer has processed the parcel. Delivery is at
least once: uncertain publication outcomes and HTTP retries can create duplicates;
each HTTP retry currently gets a new server batch ID.

Analytics consumes `raw_persistance` with four concurrent workers by default.
Set `RAW_PERSISTENCE_WORKERS` to a positive integer to change the per-instance
worker count (also passed through by Compose). Each worker owns a dedicated
RabbitMQ channel with prefetch 1, limiting in-flight parcels to the worker count.
Completion order can differ from arrival order.
Each envelope is validated and written into `raw_batches`, preserving the ingest
batch ID, authenticated identity, original receipt time, and protobuf JSON payload.
`group_ids` remains empty. ClickHouse asynchronous inserts wait for the flush to
complete before RabbitMQ acknowledgement. Writes have a 30-second timeout;
transient storage failures are requeued with a one-second pause before this
worker's next delivery. Invalid envelopes are logged and rejected without
requeue (discarded unless a broker dead-letter policy is configured).

Delivery remains at least once. A crash after a successful insert but before
acknowledgement can produce duplicate rows with the same batch ID; the current
MergeTree table does not deduplicate them. All raw workers cancel pending writes
and wait for shutdown before the database is closed.

The [symbolicator worker](../symbolicator/README.md) consumes `symbolication_requests`
and publishes results to `symbolication_results`. Analytics consumes results
in versions 1 through 5 and writes one row per event to `logs`, `metrics`,
`boots`, or `crashes`. Metric counter deltas, frame addresses, and registers
retain their unsigned 64-bit values.
Event diagnostics are stored with their rows. Diagnostics without a matching event
row are emitted in the analytics service log.

A log's `module` column holds the raw target or the decoded defmt module.
The `frames` array contains the captured backtrace, including file and line
when resolved. Analytics extracts the module from older result formats before
insertion.

Query `logs_deduplicated`, `metrics_deduplicated`, `boots_deduplicated`, and
`crashes_deduplicated` for one logical row per park, server batch ID, and event
index. Inserts commit per
table, so rows can become visible before the other tables finish. The consumer
acknowledges RabbitMQ after all applicable inserts succeed. It retries failed
writes with backoff and shuts analytics down on an unexpected consumer exit.
To retain rejected results, configure a RabbitMQ dead-letter policy on
`symbolication_results`.

The existing group-membership consumer and ClickHouse startup checks/migrations
still run. Set `RABBITMQ_URL` to the AMQP connection URL (default
`amqp://localhost:5672/`); Compose supplies its configured credentials and vhost.
Publisher channels are isolated per request. Broker connection recovery currently
requires restarting analytics.

The ingest endpoint requires exactly one `X-Device-Id` and one `X-Park-Id` header.
Both must be canonical lowercase, nonzero UUIDs. Missing, malformed, zero,
noncanonical, or duplicate values return 401 before parcel parsing or publication.
The publisher receives the validated identity as part of a typed raw batch;
identity is never taken from the parcel or query parameters. Existing rows with
zero UUIDs are not rewritten.

Devices send requests through Envoy's mTLS listener at
`https://localhost:10001/ingest`; Envoy verifies the certificate, strips supplied
identity headers, and sets these headers from its URI SAN. Analytics trusts the
gateway's headers and does not terminate device TLS itself. Restrict access to
analytics to Envoy using private Services and NetworkPolicy in Kubernetes.
Published Compose ports are for local development and do not enforce this
network boundary. See [gateway authentication](../envoy/README.md).

Run `go test ./...` and `go vet ./...` to check the service. Set
`CLICKHOUSE_TEST=1` to also run the database integration test against the configured
ClickHouse instance; it uses and removes an isolated temporary database.

Set `RABBITMQ_TEST_URL` to run the broker integration test. It uses uniquely named
queues and an exchange, verifies concurrent ingest copies and unroutable-message
handling, and removes its topology afterward. It does not consume application queues.

Set both `CLICKHOUSE_TEST=1` and `RABBITMQ_TEST_URL` to run the persistence
end-to-end tests against temporary queues and isolated databases. The raw test
checks the stored ID/identity/timestamp/payload and acknowledgement, and verifies
that the corresponding symbolication request stays queued. The result test forces
a partial write and verifies retries, deduplication before merges, and exact
unsigned values.

## Code boundaries

- `domain`: batch and device identity invariants, independent of transports and storage.
- `services`: ingest orchestration and raw persistence use cases; dependency interfaces
  live alongside the code that consumes them.
- `parcel`: agent protobuf decoding and protobuf JSON validation.
- `handlers`: HTTP identity, body limits, and application-error status mapping.
- `msgq`: versioned envelope mapping, RabbitMQ topology, confirmed publication,
  delivery acknowledgements/retries, and worker lifecycle.
- `repos`: ClickHouse insert statements and flush guarantees.
- `config`: environment settings and validation; `cmd/analytics` wires the adapters
  to use cases and closes resources on startup errors and shutdown.
