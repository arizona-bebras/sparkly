package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID       uuid.UUID
	Email    string
	Password string
}

type RefreshToken struct {
	ID        uuid.UUID
	Revoked   bool
	ExpiresAt time.Time
}
