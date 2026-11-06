package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"sparkly/backend/internal/domain"
)

type DevicesRepo interface {
	CreateDevice(ctx context.Context, connectToken string, enroll func(domain.Device) error) error
	GetDevice(ctx context.Context, deviceID, userID uuid.UUID) (domain.Device, error)
	GetDevices(ctx context.Context, filters domain.GetDevicesFilters, userID uuid.UUID) ([]domain.Device, error)
}

type DeviceCertificateIssuer interface {
	Issue(context.Context, domain.DeviceIdentity, domain.DeviceCertificateRequest) (domain.DeviceCertificate, error)
}

type DeviceEnrollment struct {
	Identity    domain.DeviceIdentity
	Certificate domain.DeviceCertificate
}

type DeviceService struct {
	repo   DevicesRepo
	issuer DeviceCertificateIssuer
}

func NewDeviceService(repo DevicesRepo, issuer DeviceCertificateIssuer) *DeviceService {
	return &DeviceService{repo: repo, issuer: issuer}
}

func (s *DeviceService) RegDevice(ctx context.Context, parkToken, csrPEM string) (DeviceEnrollment, error) {
	if parkToken == "" {
		return DeviceEnrollment{}, domain.ErrInvalidEnrollment
	}
	request, err := domain.NewDeviceCertificateRequest(csrPEM)
	if err != nil {
		return DeviceEnrollment{}, fmt.Errorf("%w: %w", domain.ErrInvalidEnrollment, err)
	}

	var enrollment DeviceEnrollment
	err = s.repo.CreateDevice(ctx, parkToken, func(device domain.Device) error {
		identity, err := device.Identity()
		if err != nil {
			return fmt.Errorf("resolve registered device identity: %w", err)
		}
		certificate, err := s.issuer.Issue(ctx, identity, request)
		if err != nil {
			return fmt.Errorf("%w: %w", domain.ErrCertificateIssuance, err)
		}
		enrollment = DeviceEnrollment{Identity: identity, Certificate: certificate}
		return nil
	})
	if err != nil {
		return DeviceEnrollment{}, fmt.Errorf("register device: %w", err)
	}
	return enrollment, nil
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
