package domain

import (
	"time"

	"github.com/google/uuid"
)

type Build struct {
	ID             uuid.UUID
	ParkID         uuid.UUID
	Version        string
	Commit         string
	ElfObjectKey   string
	DwarfObjectKey string
	CreatedAt      time.Time
}

type CreateBuildParams struct {
	ParkID         uuid.UUID
	Version        string
	Commit         *string
	ElfObjectKey   *string
	DwarfObjectKey *string
}

type UpdateBuildParams struct {
	Version        *string
	Commit         *string
	ElfObjectKey   *string
	DwarfObjectKey *string
}

type GetBuildsFilters struct {
	ParkID   uuid.UUID
	Version  *string
	Commit   *string
	FromTime *time.Time
	ToTime   *time.Time
}
