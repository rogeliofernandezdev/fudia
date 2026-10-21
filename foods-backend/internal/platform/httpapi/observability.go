package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

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

func ensureCorrelationID(w http.ResponseWriter) string {
	if value := w.Header().Get("X-Request-ID"); value != "" {
		return value
	}
	value := NewCorrelationID()
	w.Header().Set("X-Request-ID", value)
	return value
}

func flowLogAttrs(w http.ResponseWriter, r *http.Request, flow string, extra ...any) []any {
	attrs := []any{
		"correlation_id", ensureCorrelationID(w),
		"flow", flow,
		"method", r.Method,
		"path", r.URL.Path,
	}
	return append(attrs, extra...)
}

func logFlowStarted(w http.ResponseWriter, r *http.Request, flow string, extra ...any) {
	slog.Info("application_flow_started", flowLogAttrs(w, r, flow, extra...)...)
}

func logFlowStep(w http.ResponseWriter, r *http.Request, flow, step string, extra ...any) {
	attrs := []any{"step", step}
	attrs = append(attrs, extra...)
	slog.Info("application_flow_step_completed", flowLogAttrs(w, r, flow, attrs...)...)
}

func logFlowUserError(w http.ResponseWriter, r *http.Request, flow, step, code string) {
	slog.Info("application_flow_rejected", flowLogAttrs(w, r, flow, "step", step, "error_code", code, "error_kind", "user")...)
}

func logFlowFailure(w http.ResponseWriter, r *http.Request, flow, step string, err error) {
	attrs := flowLogAttrs(w, r, flow, "step", step, "error_kind", "system")
	attrs = append(attrs, safeErrorLogAttrs(err)...)
	slog.Error("application_flow_failed", attrs...)
}

func logFlowCompleted(w http.ResponseWriter, r *http.Request, flow string, extra ...any) {
	slog.Info("application_flow_completed", flowLogAttrs(w, r, flow, extra...)...)
}

func safeErrorLogAttrs(err error) []any {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return []any{
			"error_type", "postgres",
			"sql_state", pgErr.Code,
			"severity", pgErr.Severity,
			"constraint", pgErr.ConstraintName,
			"table", pgErr.TableName,
			"column", pgErr.ColumnName,
			"database_message", pgErr.Message,
		}
	}
	if err == nil {
		return []any{"error_type", "unknown"}
	}
	return []any{
		"error_type", fmt.Sprintf("%T", err),
		"error", err.Error(),
	}
}
