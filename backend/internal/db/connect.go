package db

import (
	"fmt"
	"sparkly/backend/internal/db/models"

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

func NewPostgresConn(dsnParams DSN) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		dsnParams.Host,
		dsnParams.User,
		dsnParams.Password,
		dsnParams.DBName,
		dsnParams.Port,
		dsnParams.SSLMode,
		dsnParams.TimeZone,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.Exec(`DO $$ BEGIN
    	CREATE TYPE issue_severity AS ENUM ('info', 'low', 'medium', 'high', 'critical');
	EXCEPTION
    	WHEN duplicate_object THEN null;
	END $$;`)

	db.AutoMigrate(
		&models.User{},
		&models.Park{},
		&models.Group{},
		&models.Device{},
		&models.Issue{},
		&models.Build{},
		&models.Release{},
	)

	return db, nil
}
