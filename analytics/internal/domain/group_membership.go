package domain

import (
	"time"

	"github.com/google/uuid"
)

type GroupsToDeviceAssociationParams struct {
	DeviceID uuid.UUID
	GroupIDs []uuid.UUID
	Version  time.Time
}
