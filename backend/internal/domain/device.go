package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Device struct {
	ID       uuid.UUID
	Name     string
	ParkID   uuid.UUID
	GroupIDs []uuid.UUID
}

type GetDevicesFilters struct {
	ParkID    uuid.UUID
	DeviceIDs *[]uuid.UUID
	GroupIDs  *[]uuid.UUID
	Name      *string
	BuildIDs  *[]uuid.UUID
}

func (d Device) Identity() (DeviceIdentity, error) {
	return NewDeviceIdentity(d.ID, d.ParkID)
}

// DeviceIdentity binds a device to its park. Its fields cannot be changed independently.
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

func (i DeviceIdentity) URI() string {
	return fmt.Sprintf("spiffe://sparkly/parks/%s/devices/%s", i.parkID, i.deviceID)
}

// DeviceCertificate contains issued credentials, independent of their issuer or transport.
type DeviceCertificate struct {
	ChainPEM  string
	ExpiresAt time.Time
}
