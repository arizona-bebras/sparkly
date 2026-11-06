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
them, and stores their protobuf JSON representation in `raw_batches.payload_raw`
using ClickHouse's `JSON` type. Each row has a server-generated UUIDv7 `batch_id`,
authenticated UUIDs for `device_id` and `park_id`, an empty `group_ids` array,
and the UTC request arrival time with millisecond precision. The parcel's original batch ID
is retained inside the payload. Raw batch inserts use ClickHouse asynchronous
buffering (`async_insert=1`, `wait_for_async_insert=0`) to batch writes on the
server. Requests return 204 once ClickHouse accepts the batch into its in-memory
buffer, 401 for missing or invalid device identity, 400/413 for invalid input, or
503 if submission fails. Flush errors after acknowledgment are not returned to
devices; buffered data can be lost before
persistence. Retries currently create new rows; ingestion does not deduplicate
batches.

The ingest endpoint requires exactly one `X-Device-Id` and one `X-Park-Id` header.
Both must be canonical lowercase, nonzero UUIDs. Missing, malformed, zero,
noncanonical, or duplicate values return 401 before parcel parsing or storage.
The repository receives the validated identity as part of a typed raw batch;
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
