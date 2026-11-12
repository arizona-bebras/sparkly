package services

import (
	"context"
	"testing"
	"time"

	"sparkly/backend/internal/domain"
	"sparkly/backend/internal/msgq"

	"github.com/google/uuid"
)

type mockDevicesRepo struct {
	device domain.Device
	groups []domain.Group
	err    error
}

func (m *mockDevicesRepo) CreateDevice(ctx context.Context, parkToken, deviceToken string) (domain.Device, error) {
	if m.err != nil {
		return domain.Device{}, m.err
	}
	return m.device, nil
}

func (m *mockDevicesRepo) GetDevice(ctx context.Context, deviceID, userID uuid.UUID) (domain.Device, error) {
	return m.device, nil
}

func (m *mockDevicesRepo) GetDevices(ctx context.Context, filters domain.GetDevicesFilters, userID uuid.UUID) ([]domain.Device, error) {
	return []domain.Device{m.device}, nil
}

func (m *mockDevicesRepo) GetDeviceGroups(ctx context.Context, deviceID uuid.UUID) ([]domain.Group, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.groups, nil
}

type mockDeviceGroupsQueue struct {
	msgs []msgq.GroupMembershipMsg
	err  error
}

func (m *mockDeviceGroupsQueue) SendMsg(ctx context.Context, msg msgq.GroupMembershipMsg) error {
	if m.err != nil {
		return m.err
	}
	m.msgs = append(m.msgs, msg)
	return nil
}

func TestDeviceServiceRegDeviceSendsGroupMembershipMsg(t *testing.T) {
	deviceID := uuid.New()
	parkID := uuid.New()
	group1 := uuid.New()
	group2 := uuid.New()

	repo := &mockDevicesRepo{
		device: domain.Device{
			ID:     deviceID,
			ParkID: parkID,
			Name:   "device_" + deviceID.String(),
		},
		groups: []domain.Group{
			{ID: group1, ParkID: parkID, Name: "g1"},
			{ID: group2, ParkID: parkID, Name: "g2"},
		},
	}
	queue := &mockDeviceGroupsQueue{}
	svc := NewDeviceService(repo, queue)

	token, err := svc.RegDevice(context.Background(), "park-token")
	if err != nil {
		t.Fatalf("RegDevice failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty device token")
	}

	if len(queue.msgs) != 1 {
		t.Fatalf("expected 1 queued message, got %d", len(queue.msgs))
	}

	msg := queue.msgs[0]
	if msg.DeviceID != deviceID.String() {
		t.Fatalf("expected DeviceID %s, got %s", deviceID, msg.DeviceID)
	}
	if len(msg.GroupIDs) != 2 || msg.GroupIDs[0] != group1.String() || msg.GroupIDs[1] != group2.String() {
		t.Fatalf("unexpected GroupIDs: %+v", msg.GroupIDs)
	}
	if _, err := time.Parse(time.RFC3339Nano, msg.Version); err != nil {
		t.Fatalf("expected RFC3339Nano version, got %q: %v", msg.Version, err)
	}
}
