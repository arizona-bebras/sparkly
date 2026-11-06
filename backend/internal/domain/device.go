package domain

import (
	"github.com/google/uuid"
)

const (
	DEVICE_TOKEN_LENGTH = 32
)

type Device struct {
	ID   uuid.UUID
	Name string
	ParkID uuid.UUID
}

type GetDevicesFilters struct {
	ParkID    *uuid.UUID
	DeviceIDs *[]uuid.UUID
	GroupIDs  *[]uuid.UUID
	Name      *string
	BuildIDs  *[]uuid.UUID
}
