package domain

import "github.com/google/uuid"

const (
	GROUP_TOKEN_LENGTH = 32
)

type Group struct {
	ID          uuid.UUID
	Name        string
	Description string
	ParkID      uuid.UUID
}

type CreateGroupParams struct {
	Name        string
	Description string
	ParkID      uuid.UUID
}

type UpdateGroupParams struct {
	Name        string
	Description string
}
