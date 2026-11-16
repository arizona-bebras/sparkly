package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"sparkly/analytics/internal/domain"
)

type parcelDecoderFunc func([]byte) ([]byte, error)

func (f parcelDecoderFunc) DecodeParcel(body []byte) ([]byte, error) { return f(body) }

type rawBatchPublisherFunc func(context.Context, domain.RawBatch) error

func (f rawBatchPublisherFunc) PublishRawBatch(ctx context.Context, batch domain.RawBatch) error {
	return f(ctx, batch)
}

func TestIngestServiceFailureBoundaries(t *testing.T) {
	identity, err := domain.NewDeviceIdentity(uuid.New(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	brokerFailure := errors.New("broker unavailable")
	for _, tc := range []struct {
		name                    string
		identity                domain.DeviceIdentity
		receivedAt              time.Time
		decodeErr               error
		publishErr              error
		wantErr                 error
		wantDecode, wantPublish bool
	}{
		{name: "invalid identity", wantErr: domain.ErrInvalidDeviceIdentity},
		{name: "invalid protobuf", identity: identity, receivedAt: time.Now(), decodeErr: ErrInvalidParcel, wantErr: ErrInvalidParcel, wantDecode: true},
		{name: "missing receipt time", identity: identity, wantErr: domain.ErrInvalidRawBatch, wantDecode: true},
		{name: "broker failure", identity: identity, receivedAt: time.Now(), publishErr: brokerFailure, wantErr: ErrIngestUnavailable, wantDecode: true, wantPublish: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decoded, published := false, false
			service := NewIngestService(parcelDecoderFunc(func([]byte) ([]byte, error) { decoded = true; return []byte(`{}`), tc.decodeErr }), rawBatchPublisherFunc(func(context.Context, domain.RawBatch) error { published = true; return tc.publishErr }))
			err := service.IngestParcel(context.Background(), tc.identity, []byte{1}, tc.receivedAt)
			if !errors.Is(err, tc.wantErr) || decoded != tc.wantDecode || published != tc.wantPublish {
				t.Fatalf("err=%v decoded=%v published=%v", err, decoded, published)
			}
			if tc.publishErr != nil && !errors.Is(err, tc.publishErr) {
				t.Fatal("lost underlying publication error")
			}
		})
	}
}
