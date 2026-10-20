package domain

import (
	"crypto/rsa"
	"time"
)

const (
	KEY_SIZE                   = 2048
	ACCESS_TOKEN_LIFETIME      = time.Hour * 24
	REFRESH_TOKEN_LIFETIME     = time.Hour * 24 * 30
	PRIVATE_KEY_SIGNING_PERIOD = time.Hour * 24 * 60
	PRIVATE_KEY_LIFETIME       = time.Hour * 24 * 90
)

type JWK struct {
	KID       string
	N         string
	E         string
	CreatedAt time.Time
	ExpiresAt time.Time
	SignUntil time.Time
	Private   *rsa.PrivateKey
}
