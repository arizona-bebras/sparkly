package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"sparkly/analytics/internal/domain"

	"github.com/google/uuid"
)

type mockGroupMembershipRepo struct {
	calls []domain.GroupsToDeviceAssociationParams
	err   error
}

func (m *mockGroupMembershipRepo) WriteAssociatedGroups(ctx context.Context, params domain.GroupsToDeviceAssociationParams) error {
	if m.err != nil {
		return m.err
	}
	m.calls = append(m.calls, params)
	return nil
}

func TestGroupMembershipServiceAssociateGroupsToDevice(t *testing.T) {
	repo := &mockGroupMembershipRepo{}
	svc := NewGroupMembershipService(repo)

	deviceID := uuid.New()
	g1 := uuid.New()
	g2 := uuid.New()
	ts := time.Now().UTC().Truncate(time.Millisecond)

	err := svc.AssociateGroupsToDevice(context.Background(), deviceID.String(), []string{g1.String(), g2.String()}, ts.Format(time.RFC3339Nano))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.calls) != 1 {
		t.Fatalf("expected 1 repo call, got %d", len(repo.calls))
	}
	call := repo.calls[0]
	if call.DeviceID != deviceID {
		t.Fatalf("expected device ID %s, got %s", deviceID, call.DeviceID)
	}
	if len(call.GroupIDs) != 2 || call.GroupIDs[0] != g1 || call.GroupIDs[1] != g2 {
		t.Fatalf("unexpected group IDs: %+v", call.GroupIDs)
	}
	if !call.Version.Equal(ts) {
		t.Fatalf("expected version %s, got %s", ts, call.Version)
	}
}

func TestGroupMembershipServiceInvalidInputs(t *testing.T) {
	repo := &mockGroupMembershipRepo{}
	svc := NewGroupMembershipService(repo)

	validID := uuid.New().String()
	validTime := time.Now().UTC().Format(time.RFC3339Nano)

	if err := svc.AssociateGroupsToDevice(context.Background(), "bad-uuid", []string{validID}, validTime); !errors.Is(err, domain.ErrInvalidBrokerMsg) {
		t.Fatalf("expected ErrInvalidBrokerMsg for bad device id, got %v", err)
	}

	if err := svc.AssociateGroupsToDevice(context.Background(), validID, []string{"bad-uuid"}, validTime); !errors.Is(err, domain.ErrInvalidBrokerMsg) {
		t.Fatalf("expected ErrInvalidBrokerMsg for bad group id, got %v", err)
	}

	if err := svc.AssociateGroupsToDevice(context.Background(), validID, []string{validID}, "bad-time"); !errors.Is(err, domain.ErrInvalidBrokerMsg) {
		t.Fatalf("expected ErrInvalidBrokerMsg for bad version, got %v", err)
	}
}
