package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Migrate applies pending embedded migrations before the service accepts traffic.
// Run only one migration process at a time; ClickHouse has no advisory locks.
func Migrate(ctx context.Context, conn *sql.DB) error {
	files, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectClickHouse, conn, files)
	if err != nil {
		return fmt.Errorf("initialize ClickHouse migrations: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("migrate ClickHouse: %w", err)
	}
	return nil
}
