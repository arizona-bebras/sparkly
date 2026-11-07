package services

import (
	"context"
	"encoding/base64"
	"sparkly/backend/internal/domain"

	"crypto/rand"
	"fmt"

	"github.com/google/uuid"
)

type DevicesRepo interface {
	CreateDevice(ctx context.Context, parkToken, deviceToken string) error
	GetDevice(ctx context.Context, deviceID, userID uuid.UUID) (domain.Device, error)
	GetDevices(ctx context.Context, filters domain.GetDevicesFilters, userID uuid.UUID) ([]domain.Device, error)
}

type DeviceService struct {
	repo DevicesRepo
}

func NewDeviceService(repo DevicesRepo) *DeviceService {
	return &DeviceService{
		repo: repo,
	}
}

func (s *DeviceService) genDeviceToken() (string, error) {
	bytes := make([]byte, domain.DEVICE_TOKEN_LENGTH)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to read secure random bytes: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(bytes)
	return token, nil
}

func (s *DeviceService) RegDevice(ctx context.Context, parkToken string) (string, error) {
	deviceToken, err := s.genDeviceToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate device token: %w", err)
	}

	err = s.repo.CreateDevice(ctx,
		parkToken,
		deviceToken,
	)
	if err != nil {
		return "", fmt.Errorf("failed to register device: %w", err)
	}

	return deviceToken, nil
}

func (s *DeviceService) GetDevice(ctx context.Context, devicesIDStr string) (domain.Device, error) {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return domain.Device{}, fmt.Errorf("failed to define user: %w", err)
	}

	deviceID, err := uuid.Parse(devicesIDStr)
	if err != nil {
		return domain.Device{}, fmt.Errorf("failed to parse device id: %w", err)
	}

	device, err := s.repo.GetDevice(ctx, deviceID, userID)
	if err != nil {
		return domain.Device{}, fmt.Errorf("failed to get device: %w", err)
	}

	return device, nil
}

type DeviceFiltersDTO struct {
	ParkID     string
	DevicesIDs []string
	GroupIDs   []string
	Name       *string
	BuildIDs   []string
}

func (f DeviceFiltersDTO) ToDomain() (domain.GetDevicesFilters, error) {
	parkID, err := uuid.Parse(f.ParkID)
	if err != nil {
		return domain.GetDevicesFilters{}, err
	}

	var deviceIDsP *[]uuid.UUID
	if len(f.DevicesIDs) != 0 {
		deviceIDs := make([]uuid.UUID, 0, len(f.DevicesIDs))
		for _, IDStr := range f.DevicesIDs {
			id, err := uuid.Parse(IDStr)
			if err != nil {
				return domain.GetDevicesFilters{}, err
			}
			deviceIDs = append(deviceIDs, id)
		}

		deviceIDsP = &deviceIDs
	}

	var groupIDsP *[]uuid.UUID
	if len(f.GroupIDs) != 0 {
		groupIDs := make([]uuid.UUID, 0, len(f.GroupIDs))
		for _, IDStr := range f.GroupIDs {
			id, err := uuid.Parse(IDStr)
			if err != nil {
				return domain.GetDevicesFilters{}, err
			}
			groupIDs = append(groupIDs, id)
		}

		groupIDsP = &groupIDs
	}

	var buildIDsP *[]uuid.UUID
	if len(f.BuildIDs) != 0 {
		buildIDs := make([]uuid.UUID, 0, len(f.BuildIDs))
		for _, IDStr := range f.BuildIDs {
			id, err := uuid.Parse(IDStr)
			if err != nil {
				return domain.GetDevicesFilters{}, err
			}
			buildIDs = append(buildIDs, id)
		}

		buildIDsP = &buildIDs
	}

	return domain.GetDevicesFilters{
		ParkID:    parkID,
		DeviceIDs: deviceIDsP,
		Name:      f.Name,
		GroupIDs:  groupIDsP,
		BuildIDs:  buildIDsP,
	}, nil
}

func (s *DeviceService) GetDevices(ctx context.Context, filters DeviceFiltersDTO) ([]domain.Device, error) {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return []domain.Device{}, fmt.Errorf("failed to define user: %w", err)
	}

	domainFilters, err := filters.ToDomain()
	if err != nil {
		return []domain.Device{}, fmt.Errorf("failed to parse filter values: %w", err)
	}

	devices, err := s.repo.GetDevices(ctx, domainFilters, userID)
	if err != nil {
		return []domain.Device{}, fmt.Errorf("failed to get devices: %w", err)
	}

	return devices, nil
}
