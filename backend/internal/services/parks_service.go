package services

import (
	"context"
	"fmt"
	"sparkly/backend/internal/domain"
	"sparkly/backend/internal/metadata"

	"github.com/google/uuid"
)

type ParkRepo interface {
	CreatePark(ctx context.Context, userID uuid.UUID, data domain.CreateParkParams) (domain.Park, error)
	GetParkByID(ctx context.Context, userID, parkID uuid.UUID) (domain.Park, error)
	GetAllParks(ctx context.Context, userID uuid.UUID) ([]domain.Park, error)
	UpdatePark(ctx context.Context, userID uuid.UUID, parkID uuid.UUID, data domain.UpdateParkParams) (domain.Park, error)
	DeletePark(ctx context.Context, userID, parkID uuid.UUID) error
}

type ParkService struct {
	repo ParkRepo
}

func NewParkService(repo ParkRepo) *ParkService {
	return &ParkService{
		repo: repo,
	}
}

func (s *ParkService) CreatePark(ctx context.Context, name, desc string) (domain.Park, error) {
	userIDStr, err := metadata.UserIDFromContext(ctx)
	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to define user: %w", err)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to define user: %w", domain.ErrInvalidID)
	}

	park, err := s.repo.CreatePark(ctx, userID, domain.CreateParkParams{
		Name:        name,
		Description: desc,
	})

	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to create park: %w", err)
	}

	return park, nil
}

func (s *ParkService) GetParkByID(ctx context.Context, idStr string) (domain.Park, error) {
	userIDStr, err := metadata.UserIDFromContext(ctx)
	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to define user: %w", err)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to define user: %w", domain.ErrInvalidID)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to define park: %w", domain.ErrInvalidID)
	}

	park, err := s.repo.GetParkByID(ctx, userID, id)
	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to get park: %w", err)
	}

	return park, nil
}

func (s *ParkService) GetAllParks(ctx context.Context) ([]domain.Park, error) {
	userIDStr, err := metadata.UserIDFromContext(ctx)
	if err != nil {
		return []domain.Park{}, fmt.Errorf("failed to define user: %w", err)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return []domain.Park{}, fmt.Errorf("failed to define user: %w", domain.ErrInvalidID)
	}

	parks, err := s.repo.GetAllParks(ctx, userID)
	if err != nil {
		return []domain.Park{}, fmt.Errorf("failed to get parks: %w", err)
	}

	return parks, nil
}

func (s *ParkService) UpdatePark(ctx context.Context, idStr string, newData domain.UpdateParkParams) (domain.Park, error) {
	userIDStr, err := metadata.UserIDFromContext(ctx)
	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to define user: %w", err)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to define user: %w", domain.ErrInvalidID)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to define park: %w", domain.ErrInvalidID)
	}
	park, err := s.repo.UpdatePark(ctx, userID, id, domain.UpdateParkParams{
		Name:        newData.Name,
		Description: newData.Description,
	})

	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to update park: %w", err)
	}

	return park, nil
}

func (s *ParkService) DeletePark(ctx context.Context, idStr string) error {
	userIDStr, err := metadata.UserIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to define user: %w", err)
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("failed to define user: %w", domain.ErrInvalidID)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("failed to define park: %w", domain.ErrInvalidID)
	}
	err = s.repo.DeletePark(ctx, userID, id)

	if err != nil {
		return fmt.Errorf("failed to delete park: %w", err)
	}

	return nil
}
