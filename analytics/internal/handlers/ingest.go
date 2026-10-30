package handlers

import (
	"errors"
	"io"
	"net/http"
	"sync"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	agentv1 "sparkly/analytics/proto/agent/v1"
)

const maxParcelBytes = 1 << 20

// NewIngestHandler decodes binary protobuf parcels and prints one JSON line per
// request. This development endpoint does not persist or deduplicate parcels.
func NewIngestHandler(output io.Writer) http.Handler {
	var outputMu sync.Mutex
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		outputMu.Lock()
		_, err = output.Write(append(data, '\n'))
		outputMu.Unlock()
		if err != nil {
			http.Error(w, "could not print parcel", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
