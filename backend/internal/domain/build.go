package domain

import (
	"time"

	"github.com/google/uuid"
)

const ELF_OBJECTS_BUCKET_NAME = "elf-objects"

type Build struct {
	ID             uuid.UUID
	ParkID         uuid.UUID
	ReleaseID      uuid.UUID
	Commit         string
	ElfObjectKey   string
	DwarfObjectKey string
	CreatedAt      time.Time
}

type CreateBuildParams struct {
	ParkID    uuid.UUID
	ReleaseID *uuid.UUID
	Commit    *string
	ElfObject []byte
}

type UpdateBuildParams struct {
	ReleaseID    *uuid.UUID
	Commit       *string
}

type GetBuildsFilters struct {
	ParkID    uuid.UUID
	ReleaseID *uuid.UUID
	Commit    *string
	FromTime  *time.Time
	ToTime    *time.Time
}
