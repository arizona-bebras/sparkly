package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"sparkly/backend/internal/domain"
)

type GroupsRepo interface {
	CreateGroup(ctx context.Context, userID uuid.UUID, data domain.CreateGroupParams) (domain.Group, error)
	GetGroup(ctx context.Context, groupID, userID uuid.UUID) (domain.Group, error)
	GetAllGroups(ctx context.Context, parkID, userID uuid.UUID) ([]domain.Group, error)
	UpdateGroup(ctx context.Context, newData domain.UpdateGroupParams, groupID, userID uuid.UUID) (domain.Group, error)
	DeleteGroup(ctx context.Context, groupID, userID uuid.UUID) error
}

type GroupsService struct {
	repo GroupsRepo
}

func NewGroupsService(repo GroupsRepo) *GroupsService {
	return &GroupsService{
		repo: repo,
	}
}

func (s *GroupsService) CreateGroup(ctx context.Context, parkIDStr, name, description string) (domain.Group, error) {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return domain.Group{}, fmt.Errorf("failed to define user: %w", err)
	}

	parkID, err := uuid.Parse(parkIDStr)
	if err != nil {
		return domain.Group{}, fmt.Errorf("failed to identify park id: %w", domain.ErrInvalidID)
	}

	group, err := s.repo.CreateGroup(ctx, userID, domain.CreateGroupParams{
		Name:        name,
		Description: description,
		ParkID:      parkID,
	})
	if err != nil {
		return domain.Group{}, fmt.Errorf("failed to create group: %w", err)
	}

	return group, nil
}

func (s *GroupsService) GetGroup(ctx context.Context, groupIDStr string) (domain.Group, error) {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return domain.Group{}, fmt.Errorf("failed to define user: %w", err)
	}

	groupID, err := uuid.Parse(groupIDStr)
	if err != nil {
		return domain.Group{}, fmt.Errorf("failed to identify group id: %w", domain.ErrInvalidID)
	}

	group, err := s.repo.GetGroup(ctx, groupID, userID)
	if err != nil {
		return domain.Group{}, fmt.Errorf("failed to get group by id: %w", err)
	}

	return group, nil
}

func (s *GroupsService) GetAllGroups(ctx context.Context, parkIDStr string) ([]domain.Group, error) {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return []domain.Group{}, fmt.Errorf("failed to define user: %w", err)
	}

	parkID, err := uuid.Parse(parkIDStr)
	if err != nil {
		return []domain.Group{}, fmt.Errorf("failed to identify park id: %w", domain.ErrInvalidID)
	}

	groups, err := s.repo.GetAllGroups(ctx, parkID, userID)
	if err != nil {
		return []domain.Group{}, fmt.Errorf("failed to get groups: %w", err)
	}

	return groups, nil
}

func (s *GroupsService) UpdateGroup(ctx context.Context, groupIDStr, newName, newDescription string) (domain.Group, error) {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return domain.Group{}, fmt.Errorf("failed to define user: %w", err)
	}

	groupID, err := uuid.Parse(groupIDStr)
	if err != nil {
		return domain.Group{}, fmt.Errorf("failed to identify group id: %w", domain.ErrInvalidID)
	}

	group, err := s.repo.UpdateGroup(ctx, domain.UpdateGroupParams{
		Name:        newName,
		Description: newDescription,
	}, groupID, userID)
	if err != nil {
		return domain.Group{}, fmt.Errorf("failed to update group: %w", err)
	}

	return group, nil
}

func (s *GroupsService) DeleteGroup(ctx context.Context, groupIDStr string) error {
	userID, err := parseCtxUserIDValue(ctx)
	if err != nil {
		return fmt.Errorf("failed to define user: %w", err)
	}

	groupID, err := uuid.Parse(groupIDStr)
	if err != nil {
		return fmt.Errorf("failed to identify group id: %w", domain.ErrInvalidID)
	}

	err = s.repo.DeleteGroup(ctx, groupID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	return nil
}
