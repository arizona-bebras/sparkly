package repos

import (
	"context"
	"errors"
	"fmt"
	"log"
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

func (r *DevicesRepo) CreateDevice(ctx context.Context, connectToken, deviceToken string) error {
	var token models.Token
	if err := r.db.WithContext(ctx).Preload("Groups").First(&token, "token = ?", connectToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrTokenNotFound
		}
		return fmt.Errorf("failed to write device to db: %w", err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("failed to generate id for device: %w", err)
	}
	device := models.Device{
		ID:       id,
		ParkID:   token.ParkID,
		Name:     "device_" + id.String(),
		Token:    deviceToken,
		JoinedAt: time.Now(),
	}
	if err := r.db.WithContext(ctx).Create(&device).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, gorm.ErrDuplicatedKey) || (errors.As(err, &pgErr) && pgErr.Code == "23505") {
			return domain.ErrDuplicatedDevice
		}
		return fmt.Errorf("failed to write new device to db: %w", err)
	}
	log.Println(token.Groups)

	if len(token.Groups) > 0 {
		if err := r.db.WithContext(ctx).Model(&device).Association("Groups").Append(token.Groups); err != nil {
			return fmt.Errorf("failed to assign device to groups: %w", err)
		}
	}
	return nil
}
