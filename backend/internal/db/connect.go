package db

import (
	"fmt"
	"net"
	"net/url"
	"sparkly/backend/internal/db/models"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DSN struct {
	Host     string
	User     string
	Password string
	DBName   string
	Port     string
	SSLMode  string
	TimeZone string
}

func (d DSN) FormatDSN() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.User, d.Password),
		Host:   net.JoinHostPort(d.Host, d.Port),
		Path:   d.DBName,
	}
	q := u.Query()
	if d.SSLMode != "" {
		q.Set("sslmode", d.SSLMode)
	}
	if d.TimeZone != "" {
		q.Set("timezone", d.TimeZone)
	}

	u.RawQuery = strings.ReplaceAll(q.Encode(), "%2F", "/")
	return u.String()
}

func NewPostgresConn(dsnParams DSN) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsnParams.FormatDSN()), &gorm.Config{
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.Exec(`DO $$ BEGIN
    	CREATE TYPE issue_severity AS ENUM ('info', 'low', 'medium', 'high', 'critical');
	EXCEPTION
    	WHEN duplicate_object THEN null;
	END $$;`)

	err = db.AutoMigrate(
		&models.User{},
		&models.Park{},
		&models.Group{},
		&models.Device{},
		&models.Issue{},
		&models.Build{},
		&models.Release{},
	)
	if err != nil {
		return nil, fmt.Errorf("migration error: %w", err)
	}

	return db, nil
}
