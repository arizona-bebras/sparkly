package domain

import "github.com/google/uuid"

type Park struct {
	ID uuid.UUID
	Name string
	Desc string
}