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
}
