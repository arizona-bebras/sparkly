package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"sparkly/backend/internal/domain"
)

type BuildsRepo interface {
	CreateBuild(ctx context.Context, userID uuid.UUID, data domain.CreateBuildParams) (domain.Build, error)
	GetBuild(ctx context.Context, buildID, userID uuid.UUID) (domain.Build, error)
	GetBuilds(ctx context.Context, filters domain.GetBuildsFilters, userID uuid.UUID) ([]domain.Build, error)
	UpdateBuild(ctx context.Context, newData domain.UpdateBuildParams, buildID, userID uuid.UUID) (domain.Build, error)
	DeleteBuild(ctx context.Context, buildID, userID uuid.UUID) error
}

type BuildService struct {
	repo BuildsRepo
}

func NewBuildService(repo BuildsRepo) *BuildService {
	return &BuildService{
		repo: repo,
	}
}

type CreateBuildDTO struct {
	ParkID    string
	ReleaseID *string
	Commit    *string
	ElfObject []byte
}

func (s *BuildService) CreateBuild(ctx context.Context, params CreateBuildDTO) (domain.Build, error) {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return domain.Build{}, fmt.Errorf("failed to define user: %w", err)
	}

	parkID, err := uuid.Parse(params.ParkID)
	if err != nil {
		return domain.Build{}, fmt.Errorf("failed to identify park id: %w", domain.ErrInvalidID)
	}

	var releaseID *uuid.UUID
	if params.ReleaseID != nil {
		inMemReleaseID, err := uuid.Parse(*params.ReleaseID)
		if err != nil {
			return domain.Build{}, fmt.Errorf("failed to identify release id: %w", domain.ErrInvalidID)
		}

		releaseID = &inMemReleaseID
	}

	build, err := s.repo.CreateBuild(ctx, userID, domain.CreateBuildParams{
		ParkID:    parkID,
		ReleaseID: releaseID,
		Commit:    params.Commit,
		ElfObject: params.ElfObject,
	})
	if err != nil {
		return domain.Build{}, fmt.Errorf("failed to create build: %w", err)
	}

	return build, nil
}

func (s *BuildService) GetBuild(ctx context.Context, buildIDStr string) (domain.Build, error) {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return domain.Build{}, fmt.Errorf("failed to define user: %w", err)
	}

	buildID, err := uuid.Parse(buildIDStr)
	if err != nil {
		return domain.Build{}, fmt.Errorf("failed to identify build id: %w", domain.ErrInvalidID)
	}

	build, err := s.repo.GetBuild(ctx, buildID, userID)
	if err != nil {
		return domain.Build{}, fmt.Errorf("failed to get build by id: %w", err)
	}

	return build, nil
}

type BuildFiltersDTO struct {
	ParkID    string
	ReleaseID *string
	Commit    *string
	FromTime  *time.Time
	ToTime    *time.Time
}

func (s *BuildService) GetBuilds(ctx context.Context, filters BuildFiltersDTO) ([]domain.Build, error) {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return []domain.Build{}, fmt.Errorf("failed to define user: %w", err)
	}

	parkID, err := uuid.Parse(filters.ParkID)
	if err != nil {
		return []domain.Build{}, fmt.Errorf("failed to identify park id: %w", domain.ErrInvalidID)
	}

	var releaseID *uuid.UUID
	if filters.ReleaseID != nil {
		inMemReleaseID, err := uuid.Parse(*filters.ReleaseID)
		if err != nil {
			return []domain.Build{}, fmt.Errorf("failed to identify release id: %w", domain.ErrInvalidID)
		}

		releaseID = &inMemReleaseID
	}

	domainFilters := domain.GetBuildsFilters{
		ParkID:    parkID,
		ReleaseID: releaseID,
		Commit:    filters.Commit,
		FromTime:  filters.FromTime,
		ToTime:    filters.ToTime,
	}

	builds, err := s.repo.GetBuilds(ctx, domainFilters, userID)
	if err != nil {
		return []domain.Build{}, fmt.Errorf("failed to get builds: %w", err)
	}

	return builds, nil
}

type UpdateBuildDTO struct {
	BuildID   string
	ReleaseID *string
	Commit    *string
}

func (s *BuildService) UpdateBuild(ctx context.Context, params UpdateBuildDTO) (domain.Build, error) {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return domain.Build{}, fmt.Errorf("failed to define user: %w", err)
	}

	buildID, err := uuid.Parse(params.BuildID)
	if err != nil {
		return domain.Build{}, fmt.Errorf("failed to identify build id: %w", domain.ErrInvalidID)
	}

	var releaseID *uuid.UUID
	if params.ReleaseID != nil {
		inMemReleaseID, err := uuid.Parse(*params.ReleaseID)
		if err != nil {
			return domain.Build{}, fmt.Errorf("failed to identify release id: %w", domain.ErrInvalidID)
		}

		releaseID = &inMemReleaseID
	}

	build, err := s.repo.UpdateBuild(ctx, domain.UpdateBuildParams{
		ReleaseID: releaseID,
		Commit:    params.Commit,
	}, buildID, userID)
	if err != nil {
		return domain.Build{}, fmt.Errorf("failed to update build: %w", err)
	}

	return build, nil
}

func (s *BuildService) DeleteBuild(ctx context.Context, buildIDStr string) error {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return fmt.Errorf("failed to define user: %w", err)
	}

	buildID, err := uuid.Parse(buildIDStr)
	if err != nil {
		return fmt.Errorf("failed to identify build id: %w", domain.ErrInvalidID)
	}

	err = s.repo.DeleteBuild(ctx, buildID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete build: %w", err)
	}

	return nil
}
