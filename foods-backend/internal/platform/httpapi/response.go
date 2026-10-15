// Package httpapi contains HTTP mechanics only, with no business persistence.
package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/observability"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
)

type ErrorResponse struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlationId"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func Fail(w http.ResponseWriter, status int, code, message string) {
	id := w.Header().Get("X-Request-ID")
	if id == "" {
		id = NewCorrelationID()
		w.Header().Set("X-Request-ID", id)
	}
	WriteJSON(w, status, ErrorResponse{code, message, id})
}
func WriteError(w http.ResponseWriter, err error) {
	var failure *apperror.Error
	if !errors.As(err, &failure) {
		Fail(w, 500, "internal_error", "No pudimos completar la operación.")
		return
	}
	status := 500
	switch failure.Kind {
	case apperror.Invalid:
		status = 400
	case apperror.Unauthenticated:
		status = 401
	case apperror.Forbidden:
		status = 403
	case apperror.NotFound:
		status = 404
	case apperror.Conflict:
		status = 409
	case apperror.Unavailable:
		status = 503
	}
	Fail(w, status, failure.Code, failure.Message)
	if status >= 500 && failure.Cause != nil {
		// The response stays generic; the log keeps the cause under the same correlationId.
		attrs := append([]any{"correlation_id", w.Header().Get("X-Request-ID"), "code", failure.Code, "status", status}, observability.ErrorAttrs(failure.Cause)...)
		slog.Error("api_error", attrs...)
	}
}
func Scope(w http.ResponseWriter, r *http.Request) (requestscope.Scope, bool) {
	s, ok := requestscope.From(r.Context())
	if !ok {
		Fail(w, 401, "unauthenticated", "Inicia sesión para continuar.")
	}
	return s, ok
}
func PageParams(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
func NewCorrelationID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		fallback := sha256.Sum256([]byte(fmt.Sprintf("%d:%d", time.Now().UnixNano(), os.Getpid())))
		copy(value[:], fallback[:16])
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:16])
}
