package repos

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"sparkly/backend/internal/db/models"
	"sparkly/backend/internal/db/scope"
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

func (r *DevicesRepo) CreateDevice(ctx context.Context, connectToken, deviceToken string) (domain.Device, error) {
	var token models.Token
	if err := r.db.WithContext(ctx).Preload("Groups").First(&token, "token = ?", connectToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Device{}, domain.ErrTokenNotFound
		}
		return domain.Device{}, fmt.Errorf("failed to write device to db: %w", err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return domain.Device{}, fmt.Errorf("failed to generate id for device: %w", err)
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
			return domain.Device{}, domain.ErrDuplicatedDevice
		}
		return domain.Device{}, fmt.Errorf("failed to write new device to db: %w", err)
	}
	log.Println(token.Groups)

	if len(token.Groups) > 0 {
		if err := r.db.WithContext(ctx).Model(&device).Association("Groups").Append(token.Groups); err != nil {
			return domain.Device{}, fmt.Errorf("failed to assign device to groups: %w", err)
		}
	}
	return device.ToDomain(), nil
}

func (r *DevicesRepo) GetDevice(ctx context.Context, deviceID, userID uuid.UUID) (domain.Device, error) {
	device := models.Device{
		ID: deviceID,
	}

	if err := r.db.WithContext(ctx).Scopes(scope.UserDeviceScope(userID)).Preload("Groups").First(&device).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Device{}, domain.ErrDeviceNotFound
		}
		return domain.Device{}, fmt.Errorf("failed to get device from db: %w", err)
	}

	return device.ToDomain(), nil
}

func (r *DevicesRepo) GetDevices(ctx context.Context, filters domain.GetDevicesFilters, userID uuid.UUID) ([]domain.Device, error) {
	query := r.db.WithContext(ctx).Scopes(scope.UserDeviceScope(userID)).Preload("Groups").Where("devices.park_id = ?", filters.ParkID)

	if filters.DeviceIDs != nil {
		query = query.Where("devices.id IN ?", *filters.DeviceIDs)
	}

	if filters.GroupIDs != nil {
		groupsSubQuery := r.db.WithContext(ctx).Table("group_devices").Select("device_id").Where("group_id IN ?", *filters.GroupIDs)
		query = query.Where("devices.id IN (?)", groupsSubQuery)
	}

	if filters.BuildIDs != nil {
		query = query.Where("devices.build_id IN ?", *filters.BuildIDs)
	}

	if filters.Name != nil {
		query = query.Select("*, similarity(name, ?) as sim", *filters.Name).
			Where("similarity(name, ?) >= ?", *filters.Name, SIMILARITY_THRESHOLD).
			Order("sim DESC")
	}

	var devices []models.Device
	if err := query.Find(&devices).Error; err != nil {
		return []domain.Device{}, fmt.Errorf("failed to get devices from db by filters: %w", err)
	}

	domainDevices := make([]domain.Device, 0, len(devices))
	for _, device := range devices {
		domainDevices = append(domainDevices, device.ToDomain())
	}

	return domainDevices, nil
}

func (r *DevicesRepo) GetDeviceGroups(ctx context.Context, deviceID uuid.UUID) ([]domain.Group, error) {
	var groups []models.Group
	if err := r.db.WithContext(ctx).
		Model(&models.Device{ID: deviceID}).
		Association("Groups").
		Find(&groups); err != nil {
		return []domain.Group{}, fmt.Errorf("failed to get groups groups from db: %w", err)
	}

	domainGroups := make([]domain.Group, 0, len(groups))
	for _, group := range groups {
		domainGroups = append(domainGroups, group.ToDomain())
	}
	
	return domainGroups, nil
}