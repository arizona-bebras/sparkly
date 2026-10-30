package db

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// Open connects to an existing ClickHouse database and verifies it is reachable.
func Open(ctx context.Context) (*sql.DB, error) {
	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{net.JoinHostPort(env("CLICKHOUSE_HOST", "localhost"), env("CLICKHOUSE_NATIVE_PORT", "9000"))},
		Auth: clickhouse.Auth{
			Database: env("CLICKHOUSE_DB", "sparkly"),
			Username: env("CLICKHOUSE_USER", "sparkly"),
			Password: env("CLICKHOUSE_PASSWORD", "sparkly"),
		},
		DialTimeout: 5 * time.Second,
		ReadTimeout: 15 * time.Second,
	})
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("connect to ClickHouse: %w", err)
	}
	return conn, nil
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
