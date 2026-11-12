package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"sparkly/backend/internal/domain"
)

type ReleaseRepo interface {
	CreateRelease(ctx context.Context, userID uuid.UUID, data domain.CreateReleaseParams) (domain.Release, error)
	GetRelease(ctx context.Context, releaseID, userID uuid.UUID) (domain.Release, error)
	GetReleases(ctx context.Context, filters domain.GetReleasesFilters, userID uuid.UUID) ([]domain.Release, error)
	UpdateRelease(ctx context.Context, newData domain.UpdateReleaseParams, releaseID, userID uuid.UUID) (domain.Release, error)
	DeleteRelease(ctx context.Context, releaseID, userID uuid.UUID) error
}

type ReleaseService struct {
	repo ReleaseRepo
}

func NewReleaseService(repo ReleaseRepo) *ReleaseService {
	return &ReleaseService{
		repo: repo,
	}
}

func (s *ReleaseService) CreateRelease(ctx context.Context, parkIDStr string, params domain.CreateReleaseParams) (domain.Release, error) {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return domain.Release{}, fmt.Errorf("failed to define user: %w", err)
	}

	parkID, err := uuid.Parse(parkIDStr)
	if err != nil {
		return domain.Release{}, fmt.Errorf("failed to identify park id: %w", domain.ErrInvalidID)
	}

	params.ParkID = parkID
	release, err := s.repo.CreateRelease(ctx, userID, params)
	if err != nil {
		return domain.Release{}, fmt.Errorf("failed to create release: %w", err)
	}

	return release, nil
}

func (s *ReleaseService) GetRelease(ctx context.Context, releaseIDStr string) (domain.Release, error) {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return domain.Release{}, fmt.Errorf("failed to define user: %w", err)
	}

	releaseID, err := uuid.Parse(releaseIDStr)
	if err != nil {
		return domain.Release{}, fmt.Errorf("failed to identify release id: %w", domain.ErrInvalidID)
	}

	release, err := s.repo.GetRelease(ctx, releaseID, userID)
	if err != nil {
		return domain.Release{}, fmt.Errorf("failed to get release by id: %w", err)
	}

	return release, nil
}

func (s *ReleaseService) GetReleases(ctx context.Context, parkIDStr string, filters domain.GetReleasesFilters) ([]domain.Release, error) {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return []domain.Release{}, fmt.Errorf("failed to define user: %w", err)
	}

	parkID, err := uuid.Parse(parkIDStr)
	if err != nil {
		return []domain.Release{}, fmt.Errorf("failed to identify park id: %w", domain.ErrInvalidID)
	}

	filters.ParkID = parkID
	releases, err := s.repo.GetReleases(ctx, filters, userID)
	if err != nil {
		return []domain.Release{}, fmt.Errorf("failed to get releases: %w", err)
	}

	return releases, nil
}

func (s *ReleaseService) UpdateRelease(ctx context.Context, releaseIDStr string, params domain.UpdateReleaseParams) (domain.Release, error) {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return domain.Release{}, fmt.Errorf("failed to define user: %w", err)
	}

	releaseID, err := uuid.Parse(releaseIDStr)
	if err != nil {
		return domain.Release{}, fmt.Errorf("failed to identify release id: %w", domain.ErrInvalidID)
	}

	release, err := s.repo.UpdateRelease(ctx, params, releaseID, userID)
	if err != nil {
		return domain.Release{}, fmt.Errorf("failed to update release: %w", err)
	}

	return release, nil
}

func (s *ReleaseService) DeleteRelease(ctx context.Context, releaseIDStr string) error {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return fmt.Errorf("failed to define user: %w", err)
	}

	releaseID, err := uuid.Parse(releaseIDStr)
	if err != nil {
		return fmt.Errorf("failed to identify release id: %w", domain.ErrInvalidID)
	}

	err = s.repo.DeleteRelease(ctx, releaseID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete release: %w", err)
	}

	return nil
}
