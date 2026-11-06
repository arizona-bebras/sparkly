package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"sparkly/analytics/internal/domain"
)

// The gateway replaces these headers with identity from a verified certificate.
// Analytics must only be reachable through that trusted gateway.
func deviceIdentityFromHeaders(headers http.Header) (domain.DeviceIdentity, error) {
	deviceID, err := identityIDFromHeader(headers, "X-Device-Id")
	if err != nil {
		return domain.DeviceIdentity{}, err
	}
	parkID, err := identityIDFromHeader(headers, "X-Park-Id")
	if err != nil {
		return domain.DeviceIdentity{}, err
	}
	return domain.NewDeviceIdentity(deviceID, parkID)
}

func identityIDFromHeader(headers http.Header, name string) (uuid.UUID, error) {
	values := headers.Values(name)
	if len(values) != 1 {
		return uuid.Nil, domain.ErrInvalidDeviceIdentity
	}
	id, err := uuid.Parse(values[0])
	if err != nil || id.String() != values[0] {
		return uuid.Nil, domain.ErrInvalidDeviceIdentity
	}
	return id, nil
}
