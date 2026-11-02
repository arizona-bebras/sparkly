package services

import (
	"context"
	"encoding/base64"
	"sparkly/backend/internal/domain"

	"crypto/rand"
	"fmt"
)

type DevicesRepo interface {
	CreateDevice(ctx context.Context, parkToken, deviceToken string) error
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
