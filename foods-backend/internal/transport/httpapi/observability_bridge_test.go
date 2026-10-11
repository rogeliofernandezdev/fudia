package httpapi

import (
	"net/http"
)

func ensureCorrelationID(w http.ResponseWriter) string {
	if value := w.Header().Get("X-Request-ID"); value != "" {
		return value
	}
	value := NewCorrelationID()
	w.Header().Set("X-Request-ID", value)
	return value
}
