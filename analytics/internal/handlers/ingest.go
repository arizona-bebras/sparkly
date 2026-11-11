package handlers

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"sparkly/analytics/internal/domain"
	agentv1 "sparkly/analytics/proto/agent/v1"
)

const maxParcelBytes = 1 << 20

type RawBatchStore interface {
	StoreRawBatch(context.Context, domain.RawBatch) error
}

// NewIngestHandler requires gateway identity and persists validated parcels as JSON.
func NewIngestHandler(store RawBatchStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAt := time.Now().UTC()
		identity, err := deviceIdentityFromHeaders(r.Header)
		if err != nil {
			http.Error(w, "missing or invalid device identity", http.StatusUnauthorized)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxParcelBytes)
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				http.Error(w, "parcel exceeds 1 MiB", http.StatusRequestEntityTooLarge)
			} else {
				http.Error(w, "could not read parcel", http.StatusBadRequest)
			}
			return
		}
		if len(body) == 0 {
			http.Error(w, "empty parcel", http.StatusBadRequest)
			return
		}
		var parcel agentv1.Parcel
		if err := proto.Unmarshal(body, &parcel); err != nil {
			http.Error(w, "invalid protobuf parcel", http.StatusBadRequest)
			return
		}
		data, err := protojson.Marshal(&parcel)
		if err != nil {
			http.Error(w, "could not format parcel", http.StatusInternalServerError)
			return
		}
		batch := domain.RawBatch{Identity: identity, Payload: data, ReceivedAt: receivedAt}
		if err := store.StoreRawBatch(r.Context(), batch); err != nil {
			log.Printf("store raw batch: %v", err)
			http.Error(w, "could not store parcel", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
