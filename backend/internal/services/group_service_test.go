package services

import (
	"context"
	"errors"
	"testing"

	"sparkly/backend/internal/domain"
	"sparkly/backend/internal/metadata"

	"github.com/google/uuid"
)

type mockGroupsRepo struct {
	parks     map[uuid.UUID]bool
	groups    map[uuid.UUID]domain.Group
	userParks map[uuid.UUID]map[uuid.UUID]bool
}

func newMockGroupsRepo() *mockGroupsRepo {
	return &mockGroupsRepo{
		parks:     make(map[uuid.UUID]bool),
		groups:    make(map[uuid.UUID]domain.Group),
		userParks: make(map[uuid.UUID]map[uuid.UUID]bool),
	}
}

func (m *mockGroupsRepo) addPark(parkID uuid.UUID) {
	m.parks[parkID] = true
}

func (m *mockGroupsRepo) allow(userID, parkID uuid.UUID) {
	m.addPark(parkID)
	if _, ok := m.userParks[userID]; !ok {
		m.userParks[userID] = make(map[uuid.UUID]bool)
	}
	m.userParks[userID][parkID] = true
}

func (m *mockGroupsRepo) hasAccess(userID, parkID uuid.UUID) bool {
	parks, ok := m.userParks[userID]
	if !ok {
		return false
	}
	return parks[parkID]
}

func (m *mockGroupsRepo) CreateGroup(ctx context.Context, name string, description string, parkID, userID uuid.UUID) (domain.Group, error) {
	if !m.parks[parkID] {
		return domain.Group{}, domain.ErrParkNotFound
	}
	if !m.hasAccess(userID, parkID) {
		return domain.Group{}, domain.ErrParkAccessDenied
	}
	id := uuid.New()
	g := domain.Group{
		ID:          id,
		ParkID:      parkID,
		Name:        name,
		Description: description,
	}
	m.groups[id] = g
	return g, nil
}

func (m *mockGroupsRepo) GetGroup(ctx context.Context, groupID, userID uuid.UUID) (domain.Group, error) {
	g, ok := m.groups[groupID]
	if !ok || !m.hasAccess(userID, g.ParkID) {
		return domain.Group{}, domain.ErrGroupNotFound
	}
	return g, nil
}

func (m *mockGroupsRepo) GetAllGroups(ctx context.Context, parkID, userID uuid.UUID) ([]domain.Group, error) {
	if !m.hasAccess(userID, parkID) {
		return []domain.Group{}, nil
	}
	var result []domain.Group
	for _, g := range m.groups {
		if g.ParkID == parkID {
			result = append(result, g)
		}
	}
	return result, nil
}

func (m *mockGroupsRepo) UpdateGroup(ctx context.Context, newName string, newDescription string, groupID, userID uuid.UUID) (domain.Group, error) {
	g, ok := m.groups[groupID]
	if !ok || !m.hasAccess(userID, g.ParkID) {
		return domain.Group{}, domain.ErrGroupNotFound
	}
	if newName != "" {
		g.Name = newName
	}
	if newDescription != "" {
		g.Description = newDescription
	}
	m.groups[groupID] = g
	return g, nil
}

func (m *mockGroupsRepo) DeleteGroup(ctx context.Context, groupID, userID uuid.UUID) error {
	g, ok := m.groups[groupID]
	if !ok || !m.hasAccess(userID, g.ParkID) {
		return domain.ErrGroupNotFound
	}
	delete(m.groups, groupID)
	return nil
}

func withUserContext(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, metadata.UserIDContextKey, userID.String())
}

func TestGroupsServiceCRUD(t *testing.T) {
	repo := newMockGroupsRepo()
	svc := NewGroupsService(repo)

	user1ID := uuid.New()
	user2ID := uuid.New()
	parkID := uuid.New()

	repo.allow(user1ID, parkID)

	ctxUser1 := withUserContext(context.Background(), user1ID)
	ctxUser2 := withUserContext(context.Background(), user2ID)

	_, err := svc.CreateGroup(ctxUser2, parkID.String(), "Sensors", "Temperature sensors")
	if !errors.Is(err, domain.ErrParkAccessDenied) {
		t.Fatalf("Expected ErrParkAccessDenied for user2 CreateGroup, got: %v", err)
	}

	nonExistentParkID := uuid.New()
	_, err = svc.CreateGroup(ctxUser1, nonExistentParkID.String(), "Sensors", "Temperature sensors")
	if !errors.Is(err, domain.ErrParkNotFound) {
		t.Fatalf("Expected ErrParkNotFound for non-existent park, got: %v", err)
	}

	created, err := svc.CreateGroup(ctxUser1, parkID.String(), "Sensors", "Temperature sensors")
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	if created.Name != "Sensors" || created.Description != "Temperature sensors" {
		t.Errorf("Unexpected created group: %+v", created)
	}

	fetched, err := svc.GetGroup(ctxUser1, created.ID.String())
	if err != nil {
		t.Fatalf("GetGroup failed for user1: %v", err)
	}
	if fetched.ID != created.ID || fetched.Name != "Sensors" {
		t.Errorf("Unexpected fetched group: %+v", fetched)
	}

	allUser1, err := svc.GetAllGroups(ctxUser1, parkID.String())
	if err != nil {
		t.Fatalf("GetAllGroups failed for user1: %v", err)
	}
	if len(allUser1) != 1 || allUser1[0].ID != created.ID {
		t.Errorf("Unexpected all groups count for user1: %d", len(allUser1))
	}

	updated, err := svc.UpdateGroup(ctxUser1, created.ID.String(), "Updated Sensors", "New desc")
	if err != nil {
		t.Fatalf("UpdateGroup failed for user1: %v", err)
	}
	if updated.Name != "Updated Sensors" || updated.Description != "New desc" {
		t.Errorf("Unexpected updated group: %+v", updated)
	}

	_, err = svc.GetGroup(ctxUser2, created.ID.String())
	if !errors.Is(err, domain.ErrGroupNotFound) {
		t.Fatalf("Expected ErrGroupNotFound for user2 GetGroup, got: %v", err)
	}

	allUser2, err := svc.GetAllGroups(ctxUser2, parkID.String())
	if err != nil {
		t.Fatalf("GetAllGroups failed for user2: %v", err)
	}
	if len(allUser2) != 0 {
		t.Errorf("Expected 0 groups for user2, got: %d", len(allUser2))
	}

	_, err = svc.UpdateGroup(ctxUser2, created.ID.String(), "Hacked", "Hacked")
	if !errors.Is(err, domain.ErrGroupNotFound) {
		t.Fatalf("Expected ErrGroupNotFound for user2 UpdateGroup, got: %v", err)
	}

	err = svc.DeleteGroup(ctxUser2, created.ID.String())
	if !errors.Is(err, domain.ErrGroupNotFound) {
		t.Fatalf("Expected ErrGroupNotFound for user2 DeleteGroup, got: %v", err)
	}

	err = svc.DeleteGroup(ctxUser1, created.ID.String())
	if err != nil {
		t.Fatalf("DeleteGroup failed for user1: %v", err)
	}

	_, err = svc.GetGroup(ctxUser1, created.ID.String())
	if !errors.Is(err, domain.ErrGroupNotFound) {
		t.Fatalf("Expected ErrGroupNotFound after delete, got: %v", err)
	}
}

func TestGroupsServiceValidationAndAuthErrors(t *testing.T) {
	repo := newMockGroupsRepo()
	svc := NewGroupsService(repo)

	emptyCtx := context.Background()

	_, err := svc.CreateGroup(emptyCtx, uuid.NewString(), "name", "desc")
	if !errors.Is(err, domain.ErrUnknownUser) {
		t.Fatalf("Expected ErrUnknownUser on missing user context in CreateGroup, got: %v", err)
	}

	validUserCtx := withUserContext(emptyCtx, uuid.New())

	_, err = svc.CreateGroup(validUserCtx, "not-a-uuid", "name", "desc")
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("Expected ErrInvalidID on invalid park id in CreateGroup, got: %v", err)
	}

	_, err = svc.GetGroup(emptyCtx, uuid.NewString())
	if !errors.Is(err, domain.ErrUnknownUser) {
		t.Fatalf("Expected ErrUnknownUser on missing user context in GetGroup, got: %v", err)
	}

	_, err = svc.GetAllGroups(emptyCtx, uuid.NewString())
	if !errors.Is(err, domain.ErrUnknownUser) {
		t.Fatalf("Expected ErrUnknownUser on missing user context in GetAllGroups, got: %v", err)
	}

	_, err = svc.UpdateGroup(emptyCtx, uuid.NewString(), "name", "desc")
	if !errors.Is(err, domain.ErrUnknownUser) {
		t.Fatalf("Expected ErrUnknownUser on missing user context in UpdateGroup, got: %v", err)
	}

	err = svc.DeleteGroup(emptyCtx, uuid.NewString())
	if !errors.Is(err, domain.ErrUnknownUser) {
		t.Fatalf("Expected ErrUnknownUser on missing user context in DeleteGroup, got: %v", err)
	}

	ctxWithInvalidUser := context.WithValue(emptyCtx, metadata.UserIDContextKey, "invalid-user-uuid")
	_, err = svc.CreateGroup(ctxWithInvalidUser, uuid.NewString(), "name", "desc")
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("Expected ErrInvalidID on malformed user ID in CreateGroup, got: %v", err)
	}

	_, err = svc.GetGroup(ctxWithInvalidUser, uuid.NewString())
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("Expected ErrInvalidID on malformed user ID in GetGroup, got: %v", err)
	}

	_, err = svc.GetGroup(validUserCtx, "invalid-group-uuid")
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("Expected ErrInvalidID on malformed group ID in GetGroup, got: %v", err)
	}

	_, err = svc.GetAllGroups(validUserCtx, "invalid-park-uuid")
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("Expected ErrInvalidID on malformed park ID in GetAllGroups, got: %v", err)
	}

	_, err = svc.UpdateGroup(validUserCtx, "invalid-group-uuid", "name", "desc")
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("Expected ErrInvalidID on malformed group ID in UpdateGroup, got: %v", err)
	}

	err = svc.DeleteGroup(validUserCtx, "invalid-group-uuid")
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("Expected ErrInvalidID on malformed group ID in DeleteGroup, got: %v", err)
	}
}
