package domain

import (
	"errors"

	"github.com/google/uuid"
)

var ErrInvalidDeviceIdentity = errors.New("device and park IDs must be nonzero UUIDs")

// DeviceIdentity binds an authenticated device to its park.
type DeviceIdentity struct {
	deviceID uuid.UUID
	parkID   uuid.UUID
}

func NewDeviceIdentity(deviceID, parkID uuid.UUID) (DeviceIdentity, error) {
	identity := DeviceIdentity{deviceID: deviceID, parkID: parkID}
	if err := identity.Validate(); err != nil {
		return DeviceIdentity{}, err
	}
	return identity, nil
}

func (i DeviceIdentity) Validate() error {
	if i.deviceID == uuid.Nil || i.parkID == uuid.Nil {
		return ErrInvalidDeviceIdentity
	}
	return nil
}

func (i DeviceIdentity) DeviceID() uuid.UUID { return i.deviceID }
func (i DeviceIdentity) ParkID() uuid.UUID   { return i.parkID }
