package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/jackc/pgx/v5/pgconn"
)

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &logs
}

func TestWriteErrorLogsServerCauseWithResponseCorrelation(t *testing.T) {
	logs := captureLogs(t)
	rec := httptest.NewRecorder()
	rec.Header().Set("X-Request-ID", "request-1")
	cause := &pgconn.PgError{Code: "42883", Message: "function gen_random_bytes(integer) does not exist"}
	WriteError(rec, apperror.Wrap(apperror.Unavailable, "tables_not_saved", "No se registraron las mesas.", cause))
	var body ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || rec.Code != 503 || body.Code != "tables_not_saved" || body.CorrelationID != "request-1" {
		t.Fatalf("response: %d %s err=%v", rec.Code, rec.Body.String(), err)
	}
	if strings.Contains(rec.Body.String(), "gen_random_bytes") {
		t.Fatalf("technical cause leaked to the client: %s", rec.Body.String())
	}
	var entry map[string]any
	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatalf("expected one structured log: %q err=%v", logs.String(), err)
	}
	if entry["msg"] != "api_error" || entry["correlation_id"] != "request-1" || entry["code"] != "tables_not_saved" || entry["sql_state"] != "42883" || entry["database_message"] != cause.Message {
		t.Fatalf("unexpected log: %v", entry)
	}
}

func TestWriteErrorDoesNotLogClientFailures(t *testing.T) {
	logs := captureLogs(t)
	rec := httptest.NewRecorder()
	WriteError(rec, apperror.New(apperror.Conflict, "table_conflict", "Ya existe una mesa con ese nombre."))
	WriteError(rec, apperror.Wrap(apperror.Invalid, "invalid_zone", "Zona inválida.", &pgconn.PgError{Code: "23503"}))
	if logs.Len() != 0 {
		t.Fatalf("4xx responses must not be logged as server errors: %s", logs.String())
	}
}
