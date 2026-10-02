package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	recorder := httptest.NewRecorder()
	health(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestEnvOrDefault(t *testing.T) {
	t.Setenv("FOODS_TEST_VALUE", "configured")
	if got := envOrDefault("FOODS_TEST_VALUE", "fallback"); got != "configured" {
		t.Fatalf("got %q", got)
	}
}

func TestRequestLoggerAddsRequestID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := requestLogger(logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
	requestID := recorder.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Fatal("expected X-Request-ID header")
	}
	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !pattern.MatchString(requestID) {
		t.Fatalf("expected UUID v4 request id, got %q", requestID)
	}
}
