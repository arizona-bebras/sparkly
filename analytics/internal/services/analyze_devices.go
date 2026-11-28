package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"sparkly/analytics/internal/domain"
)

type AnalyzeDeviceRepo interface {
	GetDevicesSummary(ctx context.Context, parkID uuid.UUID, filters domain.GetDeviceSummaryFilters) (domain.DevicesSummary, error)
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
	GroupIDsStr *[]string
}

func (s *AnalyzeDeviceService) GetDeviceSummary(ctx context.Context, parkIDStr string, filters GetDeviceSummaryFilters) (domain.DevicesSummary, error) {
	parkID, err := uuid.Parse(parkIDStr)
	if err != nil {
		return domain.DevicesSummary{}, fmt.Errorf("failed to parse park id: %w", err)
	}

	var domainFilters domain.GetDeviceSummaryFilters
	
	if filters.GroupIDsStr != nil {
		groupIDs := make([]uuid.UUID, 0, len(*filters.GroupIDsStr))
		for _, IDStr := range *filters.GroupIDsStr {
			id, err := uuid.Parse(IDStr)
			if err != nil {
				return domain.DevicesSummary{}, fmt.Errorf("failed to parse group id: %w", err)
			}
			groupIDs = append(groupIDs, id)
		}

		domainFilters.GroupIDs = &groupIDs
	}
	
	summary, err := s.repo.GetDevicesSummary(ctx, parkID, domainFilters)
	if err != nil {
		return domain.DevicesSummary{}, fmt.Errorf("failed to get devices summary: %w", err)
	}
	
	return summary, nil
}
