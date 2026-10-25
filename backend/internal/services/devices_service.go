package services

import (
	"context"
	"encoding/base64"
	"sparkly/backend/internal/domain"

	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

type DevicesRepo interface {
	CreateDevice(ctx context.Context, parkTokenHash, deviceTokenHash string) error
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

	deviceTokenHash := sha256.Sum256([]byte(deviceToken))
	parkTokenHash := sha256.Sum256([]byte(parkToken))

	err = s.repo.CreateDevice(ctx,
		base64.RawURLEncoding.EncodeToString(parkTokenHash[:]),
		base64.RawURLEncoding.EncodeToString(deviceTokenHash[:]),
	)
	if err != nil {
		return "", fmt.Errorf("failed to register device: %w", err)
	}

	return deviceToken, nil
}
