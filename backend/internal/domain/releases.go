package domain

import (
	"time"

	"github.com/google/uuid"
)

type Release struct {
	ID        uuid.UUID
	ParkID    uuid.UUID
	Name      string
	Version   string
	CreatedAt time.Time
}

type CreateReleaseParams struct {
	ParkID  uuid.UUID
	Name    string
	Version string
}

type UpdateReleaseParams struct {
	Name    *string
	Version *string
}

type GetReleasesFilters struct {
	ParkID   uuid.UUID
	Name     *string
	Version  *string
	FromTime *time.Time
	ToTime   *time.Time
}
