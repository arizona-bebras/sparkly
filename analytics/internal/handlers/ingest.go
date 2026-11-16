package handlers

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"sparkly/analytics/internal/domain"
	"sparkly/analytics/internal/services"
)

const maxParcelBytes = 1 << 20

type ParcelIngester interface {
	IngestParcel(context.Context, domain.DeviceIdentity, []byte, time.Time) error
}

// NewIngestHandler authenticates gateway identity, bounds the request, and invokes ingest.
func NewIngestHandler(ingester ParcelIngester) http.Handler {
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
		if err := ingester.IngestParcel(r.Context(), identity, body, receivedAt); err != nil {
			switch {
			case errors.Is(err, services.ErrInvalidParcel):
				http.Error(w, "invalid protobuf parcel", http.StatusBadRequest)
			case errors.Is(err, services.ErrIngestUnavailable):
				log.Printf("ingest parcel: %v", err)
				http.Error(w, "could not queue parcel", http.StatusServiceUnavailable)
			default:
				log.Printf("ingest parcel: %v", err)
				http.Error(w, "could not prepare parcel", http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
