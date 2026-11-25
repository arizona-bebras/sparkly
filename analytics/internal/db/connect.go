package db

import (
	"database/sql"
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Open connects to an existing ClickHouse database and verifies it is reachable.
func Open(ctx context.Context) (driver.Conn, *sql.DB, error) {
	opt := &clickhouse.Options{
		Addr: []string{net.JoinHostPort(env("CLICKHOUSE_HOST", "localhost"), env("CLICKHOUSE_NATIVE_PORT", "9000"))},
		Auth: clickhouse.Auth{
			Database: env("CLICKHOUSE_DB", "sparkly"),
			Username: env("CLICKHOUSE_USER", "sparkly"),
			Password: env("CLICKHOUSE_PASSWORD", "sparkly"),
		},
		DialTimeout: 5 * time.Second,
		ReadTimeout: 15 * time.Second,
	}
	
	conn, err := clickhouse.Open(opt)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("connect to ClickHouse: %w", err)
	}

	migrationDB := clickhouse.OpenDB(opt)
	
	return conn, migrationDB, nil
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
