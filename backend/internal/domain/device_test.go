package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestDeviceIdentity(t *testing.T) {
	deviceID := uuid.MustParse("01900000-0000-7000-8000-000000000001")
	parkID := uuid.MustParse("01900000-0000-7000-8000-000000000002")
	identity, err := (Device{ID: deviceID, ParkID: parkID}).Identity()
	if err != nil {
		t.Fatal(err)
	}
	if identity.DeviceID() != deviceID || identity.ParkID() != parkID {
		t.Fatal("identity must retain both device and park")
	}
	expectedURI := "spiffe://sparkly/parks/01900000-0000-7000-8000-000000000002/devices/01900000-0000-7000-8000-000000000001"
	if identity.URI() != expectedURI {
		t.Fatalf("identity URI = %s, want %s", identity.URI(), expectedURI)
	}
	for _, test := range []struct {
		name             string
		deviceID, parkID uuid.UUID
	}{
		{name: "missing device", parkID: parkID},
		{name: "missing park", deviceID: deviceID},
		{name: "missing both"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewDeviceIdentity(test.deviceID, test.parkID)
			if !errors.Is(err, ErrInvalidDeviceIdentity) {
				t.Fatalf("expected invalid identity, got %v", err)
			}
		})
	}
}
