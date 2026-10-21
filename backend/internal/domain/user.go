package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	ACCESS_TOKEN_LIFETIME  = time.Minute * 15
	REFRESH_TOKEN_LIFETIME = time.Hour * 24 * 30
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
