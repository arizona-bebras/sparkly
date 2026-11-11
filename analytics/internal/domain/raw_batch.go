package domain

import "time"

// RawBatch contains the parcel's JSON representation and server-assigned metadata.
// Identity comes from the authenticated gateway, never from the parcel payload.
type RawBatch struct {
	Identity   DeviceIdentity
	Payload    []byte
	ReceivedAt time.Time
}
