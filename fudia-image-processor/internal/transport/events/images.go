package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type ImageProcessor interface {
	Process(context.Context, string, int64) error
}

// The Cloud Run service must require IAM invocation. Eventarc uses its dedicated invoker identity.
func Images(processor ImageProcessor, bucket string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(405)
			return
		}
		if r.Header.Get("Ce-Type") != "google.cloud.storage.object.v1.finalized" || r.Header.Get("Ce-Id") == "" {
			http.Error(w, "invalid event", 400)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
		var event struct {
			Bucket     string `json:"bucket"`
			Name       string `json:"name"`
			Generation string `json:"generation"`
		}
		if json.NewDecoder(r.Body).Decode(&event) != nil || event.Bucket != bucket {
			http.Error(w, "invalid storage event", 400)
			return
		}
		generation, err := strconv.ParseInt(event.Generation, 10, 64)
		if err != nil || generation <= 0 {
			http.Error(w, "invalid generation", 400)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		if err = processor.Process(ctx, event.Name, generation); err != nil {
			slog.Error("image_processing_retry", "eventId", r.Header.Get("Ce-Id"), "error", err)
			http.Error(w, "processing retry required", 503)
			return
		}
		w.WriteHeader(204)
	})
}
