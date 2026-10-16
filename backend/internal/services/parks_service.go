package services

import (
	"context"
	"fmt"
	"sparkly/backend/internal/domain"
	"sparkly/backend/internal/metadata"

	"github.com/google/uuid"
)

type CreateParkParams struct {
	Name        string
	Description string
}

type UpdateParkParams struct {
	Name        string
	Description string
}

type ParkRepo interface {
	CreatePark(ctx context.Context, userID uuid.UUID, data CreateParkParams) (domain.Park, error)
	GetParkByID(ctx context.Context, userID, parkID uuid.UUID) (domain.Park, error)
	GetAllParks(ctx context.Context, userID uuid.UUID) ([]domain.Park, error)
	UpdatePark(ctx context.Context, userID uuid.UUID, data UpdateParkParams) (domain.Park, error)
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
	userID, err := metadata.UserIDFromContext(ctx)
	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to define user: %w", err)
	}
	park, err := s.repo.CreatePark(ctx, userID, CreateParkParams{
		Name: name,
		Description: desc,
	})
	
	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to create park: %w", err)
	}

	return park, nil
}

func (s *ParkService) GetParkByID(ctx context.Context, id uuid.UUID) (domain.Park, error) {
	userID, err := metadata.UserIDFromContext(ctx)
	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to define user: %w", err)
	}
	
	park, err := s.repo.GetParkByID(ctx, userID, id)
	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to get park: %w", err)
	}

	return park, nil
}

func (s *ParkService) GetAllParks(ctx context.Context, id uuid.UUID) ([]domain.Park, error) {
	userID, err := metadata.UserIDFromContext(ctx)
	if err != nil {
		return []domain.Park{}, fmt.Errorf("failed to define user: %w", err)
	}
	parks, err := s.repo.GetAllParks(ctx, userID)
	if err != nil {
		return []domain.Park{}, fmt.Errorf("failed to get park: %w", err)
	}

	return parks, nil
}

func (s *ParkService) UpdatePark(ctx context.Context, newName, newDesc string) (domain.Park, error) {
	userID, err := metadata.UserIDFromContext(ctx)
	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to define user: %w", err)
	}
	park, err := s.repo.UpdatePark(ctx, userID, UpdateParkParams{
		Name: newName,
		Description: newDesc,
	})
	
	if err != nil {
		return domain.Park{}, fmt.Errorf("failed to create park: %w", err)
	}

	return park, nil
}

func (s *ParkService) DeletePark(ctx context.Context, id uuid.UUID) error {
	userID, err := metadata.UserIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to define user: %w", err)
	}
	err = s.repo.DeletePark(ctx, userID, id)
	
	if err != nil {
		return fmt.Errorf("failed to create park: %w", err)
	}

	return nil
}