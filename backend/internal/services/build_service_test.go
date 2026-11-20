package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"sparkly/backend/internal/domain"
	"sparkly/backend/internal/metadata"

	"github.com/google/uuid"
)

type mockBuildsRepo struct {
	parks     map[uuid.UUID]bool
	releases  map[uuid.UUID]bool
	builds    map[uuid.UUID]domain.Build
	userParks map[uuid.UUID]map[uuid.UUID]bool
}

func newMockBuildsRepo() *mockBuildsRepo {
	return &mockBuildsRepo{
		parks:     make(map[uuid.UUID]bool),
		releases:  make(map[uuid.UUID]bool),
		builds:    make(map[uuid.UUID]domain.Build),
		userParks: make(map[uuid.UUID]map[uuid.UUID]bool),
	}
}

func (m *mockBuildsRepo) addPark(parkID uuid.UUID) {
	m.parks[parkID] = true
}

func (m *mockBuildsRepo) addRelease(releaseID uuid.UUID) {
	m.releases[releaseID] = true
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
	if data.ReleaseID != nil && !m.releases[*data.ReleaseID] {
		return domain.Build{}, domain.ErrReleaseNotFound
	}
	id := uuid.New()
	b := domain.Build{
		ID:           id,
		ParkID:       data.ParkID,
		ElfObjectKey: "elf-" + id.String(),
		CreatedAt:    time.Now(),
	}
	if data.ReleaseID != nil {
		b.ReleaseID = *data.ReleaseID
	}
	if data.Commit != nil {
		b.Commit = *data.Commit
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
		if filters.ReleaseID != nil && b.ReleaseID != *filters.ReleaseID {
			continue
		}
		if filters.Commit != nil && !strings.Contains(b.Commit, *filters.Commit) {
			continue
		}
		if filters.FromTime != nil && b.CreatedAt.Before(*filters.FromTime) {
			continue
		}
		if filters.ToTime != nil && b.CreatedAt.After(*filters.ToTime) {
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
	if newData.ReleaseID != nil {
		if !m.releases[*newData.ReleaseID] {
			return domain.Build{}, domain.ErrReleaseNotFound
		}
		b.ReleaseID = *newData.ReleaseID
	}
	if newData.Commit != nil {
		b.Commit = *newData.Commit
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
	releaseID := uuid.New()
	repo.allow(userID, parkID)
	repo.addRelease(releaseID)

	ctx := withUserBuildCtx(context.Background(), userID)

	commit := "abc1234"
	releaseIDStr := releaseID.String()
	elf := []byte("elf binary payload")

	created, err := svc.CreateBuild(ctx, CreateBuildDTO{
		ParkID:    parkID.String(),
		ReleaseID: &releaseIDStr,
		Commit:    &commit,
		ElfObject: elf,
	})
	if err != nil {
		t.Fatalf("unexpected create build error: %v", err)
	}
	if created.ParkID != parkID || created.ReleaseID != releaseID || created.Commit != commit {
		t.Fatalf("unexpected created build data: %+v", created)
	}
	if created.ElfObjectKey == "" {
		t.Fatal("expected non-empty ElfObjectKey")
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

	buildsByRelease, err := svc.GetBuilds(ctx, BuildFiltersDTO{
		ParkID:    parkID.String(),
		ReleaseID: &releaseIDStr,
	})
	if err != nil {
		t.Fatalf("unexpected get builds by release error: %v", err)
	}
	if len(buildsByRelease) != 1 {
		t.Fatalf("expected 1 build, got %d", len(buildsByRelease))
	}

	buildsByCommit, err := svc.GetBuilds(ctx, BuildFiltersDTO{
		ParkID: parkID.String(),
		Commit: &commit,
	})
	if err != nil {
		t.Fatalf("unexpected get builds by commit error: %v", err)
	}
	if len(buildsByCommit) != 1 {
		t.Fatalf("expected 1 build, got %d", len(buildsByCommit))
	}

	newReleaseID := uuid.New()
	repo.addRelease(newReleaseID)
	newReleaseIDStr := newReleaseID.String()
	newCommit := "def5678"

	updated, err := svc.UpdateBuild(ctx, UpdateBuildDTO{
		BuildID:   created.ID.String(),
		ReleaseID: &newReleaseIDStr,
		Commit:    &newCommit,
	})
	if err != nil {
		t.Fatalf("unexpected update build error: %v", err)
	}
	if updated.ReleaseID != newReleaseID || updated.Commit != newCommit {
		t.Fatalf("expected release %s commit %s, got %+v", newReleaseIDStr, newCommit, updated)
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
	releaseID := uuid.New()
	ctxWithoutUser := context.Background()
	ctx := withUserBuildCtx(context.Background(), userID)

	releaseIDStr := releaseID.String()
	commit := "abc"
	invalidUUID := "invalid-uuid"

	_, err := svc.CreateBuild(ctxWithoutUser, CreateBuildDTO{
		ParkID: parkID.String(),
	})
	if !errors.Is(err, domain.ErrUnknownUser) {
		t.Fatalf("expected ErrUnknownUser, got %v", err)
	}

	_, err = svc.GetBuild(ctxWithoutUser, uuid.New().String())
	if !errors.Is(err, domain.ErrUnknownUser) {
		t.Fatalf("expected ErrUnknownUser, got %v", err)
	}

	_, err = svc.GetBuilds(ctxWithoutUser, BuildFiltersDTO{ParkID: parkID.String()})
	if !errors.Is(err, domain.ErrUnknownUser) {
		t.Fatalf("expected ErrUnknownUser, got %v", err)
	}

	_, err = svc.UpdateBuild(ctxWithoutUser, UpdateBuildDTO{BuildID: uuid.New().String()})
	if !errors.Is(err, domain.ErrUnknownUser) {
		t.Fatalf("expected ErrUnknownUser, got %v", err)
	}

	err = svc.DeleteBuild(ctxWithoutUser, uuid.New().String())
	if !errors.Is(err, domain.ErrUnknownUser) {
		t.Fatalf("expected ErrUnknownUser, got %v", err)
	}

	_, err = svc.CreateBuild(ctx, CreateBuildDTO{
		ParkID: invalidUUID,
	})
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = svc.CreateBuild(ctx, CreateBuildDTO{
		ParkID:    parkID.String(),
		ReleaseID: &invalidUUID,
	})
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = svc.GetBuild(ctx, invalidUUID)
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = svc.GetBuilds(ctx, BuildFiltersDTO{
		ParkID: invalidUUID,
	})
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = svc.GetBuilds(ctx, BuildFiltersDTO{
		ParkID:    parkID.String(),
		ReleaseID: &invalidUUID,
	})
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = svc.UpdateBuild(ctx, UpdateBuildDTO{
		BuildID: invalidUUID,
	})
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = svc.UpdateBuild(ctx, UpdateBuildDTO{
		BuildID:   uuid.New().String(),
		ReleaseID: &invalidUUID,
	})
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	err = svc.DeleteBuild(ctx, invalidUUID)
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = svc.CreateBuild(ctx, CreateBuildDTO{
		ParkID: parkID.String(),
	})
	if !errors.Is(err, domain.ErrParkNotFound) {
		t.Fatalf("expected ErrParkNotFound, got %v", err)
	}

	repo.addPark(parkID)
	_, err = svc.CreateBuild(ctx, CreateBuildDTO{
		ParkID: parkID.String(),
	})
	if !errors.Is(err, domain.ErrParkAccessDenied) {
		t.Fatalf("expected ErrParkAccessDenied, got %v", err)
	}

	repo.allow(userID, parkID)
	_, err = svc.CreateBuild(ctx, CreateBuildDTO{
		ParkID:    parkID.String(),
		ReleaseID: &releaseIDStr,
		Commit:    &commit,
	})
	if !errors.Is(err, domain.ErrReleaseNotFound) {
		t.Fatalf("expected ErrReleaseNotFound, got %v", err)
	}

	_, err = svc.GetBuild(ctx, uuid.New().String())
	if !errors.Is(err, domain.ErrBuildNotFound) {
		t.Fatalf("expected ErrBuildNotFound, got %v", err)
	}

	_, err = svc.UpdateBuild(ctx, UpdateBuildDTO{
		BuildID: uuid.New().String(),
	})
	if !errors.Is(err, domain.ErrBuildNotFound) {
		t.Fatalf("expected ErrBuildNotFound, got %v", err)
	}

	err = svc.DeleteBuild(ctx, uuid.New().String())
	if !errors.Is(err, domain.ErrBuildNotFound) {
		t.Fatalf("expected ErrBuildNotFound, got %v", err)
	}
}
