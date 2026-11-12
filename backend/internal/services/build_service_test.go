package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"sparkly/backend/internal/domain"
	"sparkly/backend/internal/metadata"

	"github.com/google/uuid"
)

type mockBuildsRepo struct {
	parks     map[uuid.UUID]bool
	builds    map[uuid.UUID]domain.Build
	userParks map[uuid.UUID]map[uuid.UUID]bool
}

func newMockBuildsRepo() *mockBuildsRepo {
	return &mockBuildsRepo{
		parks:     make(map[uuid.UUID]bool),
		builds:    make(map[uuid.UUID]domain.Build),
		userParks: make(map[uuid.UUID]map[uuid.UUID]bool),
	}
}

func (m *mockBuildsRepo) addPark(parkID uuid.UUID) {
	m.parks[parkID] = true
}

func (m *mockBuildsRepo) allow(userID, parkID uuid.UUID) {
	m.addPark(parkID)
	if _, ok := m.userParks[userID]; !ok {
		m.userParks[userID] = make(map[uuid.UUID]bool)
	}
	m.userParks[userID][parkID] = true
}

func (m *mockBuildsRepo) hasAccess(userID, parkID uuid.UUID) bool {
	parks, ok := m.userParks[userID]
	if !ok {
		return false
	}
	return parks[parkID]
}

func (m *mockBuildsRepo) CreateBuild(ctx context.Context, userID uuid.UUID, data domain.CreateBuildParams) (domain.Build, error) {
	if !m.parks[data.ParkID] {
		return domain.Build{}, domain.ErrParkNotFound
	}
	if !m.hasAccess(userID, data.ParkID) {
		return domain.Build{}, domain.ErrParkAccessDenied
	}
	id := uuid.New()
	b := domain.Build{
		ID:        id,
		ParkID:    data.ParkID,
		Version:   data.Version,
		CreatedAt: time.Now(),
	}
	if data.Commit != nil {
		b.Commit = *data.Commit
	}
	if data.ElfObjectKey != nil {
		b.ElfObjectKey = *data.ElfObjectKey
	}
	if data.DwarfObjectKey != nil {
		b.DwarfObjectKey = *data.DwarfObjectKey
	}
	m.builds[id] = b
	return b, nil
}

func (m *mockBuildsRepo) GetBuild(ctx context.Context, buildID, userID uuid.UUID) (domain.Build, error) {
	b, ok := m.builds[buildID]
	if !ok || !m.hasAccess(userID, b.ParkID) {
		return domain.Build{}, domain.ErrBuildNotFound
	}
	return b, nil
}

func (m *mockBuildsRepo) GetBuilds(ctx context.Context, filters domain.GetBuildsFilters, userID uuid.UUID) ([]domain.Build, error) {
	if !m.hasAccess(userID, filters.ParkID) {
		return []domain.Build{}, nil
	}
	var result []domain.Build
	for _, b := range m.builds {
		if b.ParkID != filters.ParkID {
			continue
		}
		if filters.Version != nil && b.Version != *filters.Version {
			continue
		}
		if filters.Commit != nil && b.Commit != *filters.Commit {
			continue
		}
		result = append(result, b)
	}
	return result, nil
}

func (m *mockBuildsRepo) UpdateBuild(ctx context.Context, newData domain.UpdateBuildParams, buildID, userID uuid.UUID) (domain.Build, error) {
	b, ok := m.builds[buildID]
	if !ok || !m.hasAccess(userID, b.ParkID) {
		return domain.Build{}, domain.ErrBuildNotFound
	}
	if newData.Version != nil {
		b.Version = *newData.Version
	}
	if newData.Commit != nil {
		b.Commit = *newData.Commit
	}
	if newData.ElfObjectKey != nil {
		b.ElfObjectKey = *newData.ElfObjectKey
	}
	if newData.DwarfObjectKey != nil {
		b.DwarfObjectKey = *newData.DwarfObjectKey
	}
	m.builds[buildID] = b
	return b, nil
}

func (m *mockBuildsRepo) DeleteBuild(ctx context.Context, buildID, userID uuid.UUID) error {
	b, ok := m.builds[buildID]
	if !ok || !m.hasAccess(userID, b.ParkID) {
		return domain.ErrBuildNotFound
	}
	delete(m.builds, buildID)
	return nil
}

func withUserBuildCtx(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, metadata.UserIDContextKey, userID.String())
}

func TestBuildServiceCRUD(t *testing.T) {
	repo := newMockBuildsRepo()
	svc := NewBuildService(repo)

	userID := uuid.New()
	parkID := uuid.New()
	repo.allow(userID, parkID)

	ctx := withUserBuildCtx(context.Background(), userID)

	commit := "abc1234"
	elf := "elf/key"
	dwarf := "dwarf/key"

	created, err := svc.CreateBuild(ctx, parkID.String(), domain.CreateBuildParams{
		Version:        "1.0.0",
		Commit:         &commit,
		ElfObjectKey:   &elf,
		DwarfObjectKey: &dwarf,
	})

	if err != nil {
		t.Fatalf("unexpected create build error: %v", err)
	}
	if created.Version != "1.0.0" || created.Commit != commit || created.ElfObjectKey != elf || created.DwarfObjectKey != dwarf {
		t.Fatalf("unexpected created build data: %+v", created)
	}

	got, err := svc.GetBuild(ctx, created.ID.String())
	if err != nil {
		t.Fatalf("unexpected get build error: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, got.ID)
	}

	builds, err := svc.GetBuilds(ctx, BuildFiltersDTO{ParkID: parkID.String()})
	if err != nil {
		t.Fatalf("unexpected get builds error: %v", err)
	}
	if len(builds) != 1 {
		t.Fatalf("expected 1 build, got %d", len(builds))
	}

	newVersion := "1.0.1"
	updated, err := svc.UpdateBuild(ctx, created.ID.String(), domain.UpdateBuildParams{
		Version:        &newVersion,
		Commit:         nil,
		ElfObjectKey:   nil,
		DwarfObjectKey: nil,
	})
	if err != nil {
		t.Fatalf("unexpected update build error: %v", err)
	}
	if updated.Version != newVersion {
		t.Fatalf("expected version %s, got %s", newVersion, updated.Version)
	}

	err = svc.DeleteBuild(ctx, created.ID.String())
	if err != nil {
		t.Fatalf("unexpected delete build error: %v", err)
	}

	_, err = svc.GetBuild(ctx, created.ID.String())
	if !errors.Is(err, domain.ErrBuildNotFound) {
		t.Fatalf("expected ErrBuildNotFound, got %v", err)
	}
}

func TestBuildServiceValidationAndAuthErrors(t *testing.T) {
	repo := newMockBuildsRepo()
	svc := NewBuildService(repo)

	userID := uuid.New()
	parkID := uuid.New()
	ctxWithoutUser := context.Background()
	ctx := withUserBuildCtx(context.Background(), userID)

	_, err := svc.CreateBuild(ctxWithoutUser, parkID.String(), domain.CreateBuildParams{
		Version:        "1.0.0",
		Commit:         nil,
		ElfObjectKey:   nil,
		DwarfObjectKey: nil,
	})
	if !errors.Is(err, domain.ErrUnknownUser) {
		t.Fatalf("expected ErrUnknownUser, got %v", err)
	}

	_, err = svc.CreateBuild(ctx, "invalid-uuid", domain.CreateBuildParams{
		Version:        "1.0.0",
		Commit:         nil,
		ElfObjectKey:   nil,
		DwarfObjectKey: nil,
	})
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = svc.GetBuild(ctx, "invalid-uuid")
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = svc.GetBuilds(ctx, BuildFiltersDTO{ParkID: "invalid-uuid"})
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = svc.UpdateBuild(ctx, "invalid-uuid", domain.UpdateBuildParams{
		Version:        nil,
		Commit:         nil,
		ElfObjectKey:   nil,
		DwarfObjectKey: nil,
	})
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	err = svc.DeleteBuild(ctx, "invalid-uuid")
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = svc.CreateBuild(ctx, parkID.String(), domain.CreateBuildParams{
		Version:        "1.0.0",
		Commit:         nil,
		ElfObjectKey:   nil,
		DwarfObjectKey: nil,
	})
	if !errors.Is(err, domain.ErrParkNotFound) {
		t.Fatalf("expected ErrParkNotFound, got %v", err)
	}

	repo.addPark(parkID)
	_, err = svc.CreateBuild(ctx, parkID.String(), domain.CreateBuildParams{
		Version:        "1.0.0",
		Commit:         nil,
		ElfObjectKey:   nil,
		DwarfObjectKey: nil,
	})
	if !errors.Is(err, domain.ErrParkAccessDenied) {
		t.Fatalf("expected ErrParkAccessDenied, got %v", err)
	}
}
