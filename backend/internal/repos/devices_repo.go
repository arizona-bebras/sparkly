package repos

import (
	"context"
	"errors"
	"fmt"
	"time"

	"sparkly/backend/internal/db/models"
	"sparkly/backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type DevicesRepo struct {
	db *gorm.DB
}

func NewDevicesRepo(db *gorm.DB) *DevicesRepo {
	return &DevicesRepo{
		db: db,
	}
}

func (r *DevicesRepo) CreateDevice(ctx context.Context, parkTokenHash, deviceTokenHash string) error {
	var park models.Park

	if err := r.db.WithContext(ctx).First(&park, "token_hash = ?", parkTokenHash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrParkNotFound
		}
		return fmt.Errorf("failed to write device to db: %w")
	}

	id, err := uuid.NewV7()
	if err != nil {
		fmt.Errorf("failed to generate id for device: %w", err)
	}
	device := models.Device{
		ID:       id,
		ParkID:   park.ID,
		Name:     "device_" + id.String(),
		JoinedAt: time.Now(),
	}

	if err := r.db.WithContext(ctx).Create(&device).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, gorm.ErrDuplicatedKey) || (errors.As(err, &pgErr) && pgErr.Code == "23505") {
			return domain.ErrDuplicatedDevice
		}
		return fmt.Errorf("failed to write new device to db: %w", err)
	}

	return nil
}
