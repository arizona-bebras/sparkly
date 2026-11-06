package stepca

import (
	"crypto/ecdsa"
	"encoding/base64"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"sparkly/backend/internal/domain"
)

const (
	authorizationTokenLifetime = time.Minute
	authorizationClockSkew     = 30 * time.Second
)

type provisioner struct {
	name  string
	keyID string
	key   *ecdsa.PrivateKey
}

type authorizationClaims struct {
	jwt.RegisteredClaims
	SANs         []string             `json:"sans"`
	Confirmation csrConfirmation      `json:"cnf"`
	Identity     deviceIdentityClaims `json:"sparkly"`
}

type csrConfirmation struct {
	Fingerprint string `json:"x5rt#S256"`
}

type deviceIdentityClaims struct {
	DeviceID string `json:"device_id"`
	ParkID   string `json:"park_id"`
	URI      string `json:"uri"`
}

func (p provisioner) authorizationToken(audience string, identity domain.DeviceIdentity, request domain.DeviceCertificateRequest) (string, error) {
	now := time.Now()
	fingerprint := request.Fingerprint()
	claims := authorizationClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    p.name,
			Subject:   identity.DeviceID().String(),
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-authorizationClockSkew)),
			ExpiresAt: jwt.NewNumericDate(now.Add(authorizationTokenLifetime)),
			ID:        uuid.NewString(),
		},
		SANs:         []string{identity.URI()},
		Confirmation: csrConfirmation{Fingerprint: base64.RawURLEncoding.EncodeToString(fingerprint[:])},
		Identity: deviceIdentityClaims{
			DeviceID: identity.DeviceID().String(),
			ParkID:   identity.ParkID().String(),
			URI:      identity.URI(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = p.keyID
	return token.SignedString(p.key)
}
