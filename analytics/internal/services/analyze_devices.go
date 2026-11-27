package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"sparkly/analytics/internal/domain"
)

type AnalyzeDeviceRepo interface {
	// GetDevicesSummaryForPark(ctx context.Context, parkID uuid.UUID) (domain.DevicesSummary, error)
	// GetDevicesSummaryForGroups(ctx context.Context, parkID uuid.UUID) (domain.DevicesSummary, error)
	// GetDevicesSummaryForBuilds(ctx context.Context, parkID uuid.UUID) (domain.DevicesSummary, error)
	GetDeviceDetails(ctx context.Context, deviceID uuid.UUID, filters domain.GetDeviceDetailsFilters) (domain.DeviceDetails, error)
}

type AnalyzeDeviceService struct {
	repo AnalyzeDeviceRepo
}

func NewAnalyzeDeviceService(repo AnalyzeDeviceRepo) *AnalyzeDeviceService {
	return &AnalyzeDeviceService{
		repo: repo,
	}
}

type GetDeviceDetailsFilters struct {
	FromTime time.Time
	ToTime   time.Time
}

func (s *AnalyzeDeviceService) GetDeviceDetails(ctx context.Context, deviceIDStr string, filters *GetDeviceDetailsFilters) (domain.DeviceDetails, error) {
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		return domain.DeviceDetails{}, fmt.Errorf("failed to parse device id: %w", err)
	}

	var domainFilters domain.GetDeviceDetailsFilters
	if filters != nil {
		domainFilters.FromTime = &filters.FromTime
		domainFilters.ToTime = &filters.ToTime
	}

	details, err := s.repo.GetDeviceDetails(ctx, deviceID, domainFilters)
	if err != nil {
		return domain.DeviceDetails{}, fmt.Errorf("failed to get detailed device data: %w", err)
	}

	return details, nil
}

type GetDeviceSummaryFilters struct {
	FromTime time.Time
	ToTime   time.Time
}

func (s *AnalyzeDeviceService) GetDeviceSummaryForPark(ctx context.Context, parkIDStr string, filters *GetDeviceSummaryFilters) (domain.DevicesSummary, error) {
	return domain.DevicesSummary{}, nil
}
