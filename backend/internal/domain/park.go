package domain

import "github.com/google/uuid"

type Park struct {
	ID   uuid.UUID
	Name string
	Desc string
}

type CreateParkParams struct {
	Name        string
	Description string
}

type UpdateParkParams struct {
	Name        string
	Description string
}
