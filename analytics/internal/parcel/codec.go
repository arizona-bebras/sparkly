// Package parcel adapts the agent protobuf wire format to protobuf JSON.
package parcel

import (
	"bytes"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"sparkly/analytics/internal/services"
	agentv1 "sparkly/analytics/proto/agent/v1"
)

type Codec struct{}

func (Codec) DecodeParcel(body []byte) ([]byte, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("%w: empty parcel", services.ErrInvalidParcel)
	}
	var parcel agentv1.Parcel
	if err := proto.Unmarshal(body, &parcel); err != nil {
		return nil, fmt.Errorf("%w: %w", services.ErrInvalidParcel, err)
	}
	payload, err := protojson.Marshal(&parcel)
	if err != nil {
		return nil, fmt.Errorf("format parcel: %w", err)
	}
	return payload, nil
}

func ValidateJSON(payload []byte) error {
	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 || payload[0] != '{' {
		return fmt.Errorf("payload must be a parcel object")
	}
	var parcel agentv1.Parcel
	// Preserve unknown fields in the stored payload during rolling upgrades.
	return (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(payload, &parcel)
}
