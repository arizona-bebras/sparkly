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

type mockReleaseRepo struct {
	parks     map[uuid.UUID]bool
	releases  map[uuid.UUID]domain.Release
	userParks map[uuid.UUID]map[uuid.UUID]bool
}

func newMockReleaseRepo() *mockReleaseRepo {
	return &mockReleaseRepo{
		parks:     make(map[uuid.UUID]bool),
		releases:  make(map[uuid.UUID]domain.Release),
		userParks: make(map[uuid.UUID]map[uuid.UUID]bool),
	}
}

func (m *mockReleaseRepo) addPark(parkID uuid.UUID) {
	m.parks[parkID] = true
}

func (m *mockReleaseRepo) allow(userID, parkID uuid.UUID) {
	m.addPark(parkID)
	if _, ok := m.userParks[userID]; !ok {
		m.userParks[userID] = make(map[uuid.UUID]bool)
	}
	m.userParks[userID][parkID] = true
}

func (m *mockReleaseRepo) hasAccess(userID, parkID uuid.UUID) bool {
	parks, ok := m.userParks[userID]
	if !ok {
		return false
	}
	return parks[parkID]
}

func (m *mockReleaseRepo) CreateRelease(ctx context.Context, userID uuid.UUID, data domain.CreateReleaseParams) (domain.Release, error) {
	if !m.parks[data.ParkID] {
		return domain.Release{}, domain.ErrParkNotFound
	}
	if !m.hasAccess(userID, data.ParkID) {
		return domain.Release{}, domain.ErrParkAccessDenied
	}
	id := uuid.New()
	r := domain.Release{
		ID:        id,
		ParkID:    data.ParkID,
		Name:      data.Name,
		Version:   data.Version,
		CreatedAt: time.Now(),
	}
	m.releases[id] = r
	return r, nil
}

func (m *mockReleaseRepo) GetRelease(ctx context.Context, releaseID, userID uuid.UUID) (domain.Release, error) {
	r, ok := m.releases[releaseID]
	if !ok || !m.hasAccess(userID, r.ParkID) {
		return domain.Release{}, domain.ErrReleaseNotFound
	}
	return r, nil
}

func (m *mockReleaseRepo) GetReleases(ctx context.Context, filters domain.GetReleasesFilters, userID uuid.UUID) ([]domain.Release, error) {
	if !m.hasAccess(userID, filters.ParkID) {
		return []domain.Release{}, nil
	}
	var result []domain.Release
	for _, r := range m.releases {
		if r.ParkID != filters.ParkID {
			continue
		}
		if filters.Version != nil && !strings.Contains(r.Version, *filters.Version) {
			continue
		}
		if filters.Name != nil && !strings.Contains(r.Name, *filters.Name) {
			continue
		}
		if filters.FromTime != nil && r.CreatedAt.Before(*filters.FromTime) {
			continue
		}
		if filters.ToTime != nil && r.CreatedAt.After(*filters.ToTime) {
			continue
		}
		result = append(result, r)
	}
	return result, nil
}

func (m *mockReleaseRepo) UpdateRelease(ctx context.Context, newData domain.UpdateReleaseParams, releaseID, userID uuid.UUID) (domain.Release, error) {
	r, ok := m.releases[releaseID]
	if !ok || !m.hasAccess(userID, r.ParkID) {
		return domain.Release{}, domain.ErrReleaseNotFound
	}
	if newData.Name != nil {
		r.Name = *newData.Name
	}
	if newData.Version != nil {
		r.Version = *newData.Version
	}
	m.releases[releaseID] = r
	return r, nil
}

func (m *mockReleaseRepo) DeleteRelease(ctx context.Context, releaseID, userID uuid.UUID) error {
	r, ok := m.releases[releaseID]
	if !ok || !m.hasAccess(userID, r.ParkID) {
		return domain.ErrReleaseNotFound
	}
	delete(m.releases, releaseID)
	return nil
}

func withUserReleaseCtx(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, metadata.UserIDContextKey, userID.String())
}

func TestReleaseServiceCRUD(t *testing.T) {
	repo := newMockReleaseRepo()
	svc := NewReleaseService(repo)

	userID := uuid.New()
	parkID := uuid.New()
	repo.allow(userID, parkID)

	ctx := withUserReleaseCtx(context.Background(), userID)

	created, err := svc.CreateRelease(ctx, parkID.String(), domain.CreateReleaseParams{
		Name:    "v1.0-main",
		Version: "1.0.0",
	})
	if err != nil {
		t.Fatalf("unexpected create release error: %v", err)
	}
	if created.Name != "v1.0-main" || created.Version != "1.0.0" || created.ParkID != parkID {
		t.Fatalf("unexpected created release data: %+v", created)
	}

	got, err := svc.GetRelease(ctx, created.ID.String())
	if err != nil {
		t.Fatalf("unexpected get release error: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, got.ID)
	}

	releases, err := svc.GetReleases(ctx, parkID.String(), domain.GetReleasesFilters{})
	if err != nil {
		t.Fatalf("unexpected get releases error: %v", err)
	}
	if len(releases) != 1 {
		t.Fatalf("expected 1 release, got %d", len(releases))
	}

	newName := "v1.0.1-main"
	newVersion := "1.0.1"
	updated, err := svc.UpdateRelease(ctx, created.ID.String(), domain.UpdateReleaseParams{
		Name:    &newName,
		Version: &newVersion,
	})
	if err != nil {
		t.Fatalf("unexpected update release error: %v", err)
	}
	if updated.Name != newName || updated.Version != newVersion {
		t.Fatalf("expected name %s, version %s, got %+v", newName, newVersion, updated)
	}

	err = svc.DeleteRelease(ctx, created.ID.String())
	if err != nil {
		t.Fatalf("unexpected delete release error: %v", err)
	}

	_, err = svc.GetRelease(ctx, created.ID.String())
	if !errors.Is(err, domain.ErrReleaseNotFound) {
		t.Fatalf("expected ErrReleaseNotFound, got %v", err)
	}
}

func TestReleaseServiceValidationAndAuthErrors(t *testing.T) {
	repo := newMockReleaseRepo()
	svc := NewReleaseService(repo)

	userID := uuid.New()
	parkID := uuid.New()
	ctxWithoutUser := context.Background()
	ctx := withUserReleaseCtx(context.Background(), userID)

	_, err := svc.CreateRelease(ctxWithoutUser, parkID.String(), domain.CreateReleaseParams{
		Name:    "rel",
		Version: "1.0.0",
	})
	if !errors.Is(err, domain.ErrUnknownUser) {
		t.Fatalf("expected ErrUnknownUser, got %v", err)
	}

	_, err = svc.CreateRelease(ctx, "invalid-uuid", domain.CreateReleaseParams{
		Name:    "rel",
		Version: "1.0.0",
	})
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = svc.GetRelease(ctx, "invalid-uuid")
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = svc.GetReleases(ctx, "invalid-uuid", domain.GetReleasesFilters{})
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = svc.UpdateRelease(ctx, "invalid-uuid", domain.UpdateReleaseParams{})
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	err = svc.DeleteRelease(ctx, "invalid-uuid")
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}

	_, err = svc.CreateRelease(ctx, parkID.String(), domain.CreateReleaseParams{
		Name:    "rel",
		Version: "1.0.0",
	})
	if !errors.Is(err, domain.ErrParkNotFound) {
		t.Fatalf("expected ErrParkNotFound, got %v", err)
	}

	repo.addPark(parkID)
	_, err = svc.CreateRelease(ctx, parkID.String(), domain.CreateReleaseParams{
		Name:    "rel",
		Version: "1.0.0",
	})
	if !errors.Is(err, domain.ErrParkAccessDenied) {
		t.Fatalf("expected ErrParkAccessDenied, got %v", err)
	}
}
