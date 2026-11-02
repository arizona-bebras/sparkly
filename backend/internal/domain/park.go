package domain

import "github.com/google/uuid"

const (
	PARK_TOKEN_LENGTH = 32
)

type Park struct {
	ID    uuid.UUID
	Name  string
	Desc  string
	Token string
}

type CreateParkParams struct {
	Name        string
	Description string
	Token       string
}

type UpdateParkParams struct {
	Name        string
	Description string
}
