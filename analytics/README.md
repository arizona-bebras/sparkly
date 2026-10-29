# Analytics

Run `go run ./cmd/analytics` from this directory. The server loads `.env` when
present and listens on `PORT` (default `8081`). The existing AnalyticsService is
mounted using ConnectRPC; all RPC methods currently return unimplemented.
