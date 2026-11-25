package domain

import (
	"time"

	"github.com/google/uuid"
)

type Boot struct {
	ID              string
	DeviceID        uuid.UUID
	ReceivedAt      time.Time
	BuildID         *uuid.UUID
	Diagnostics     []string
	ResetReason     string
	FirmwareVersion string
	AgentVersion    string
}