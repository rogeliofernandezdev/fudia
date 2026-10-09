package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
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

func TestProcessingDeadlinePrecedesResponseTimeout(t *testing.T) {
	server := newServer(nil)
	if server.WriteTimeout != responseTimeout || server.WriteTimeout <= requestTimeout {
		t.Fatalf("write timeout %s must exceed processing timeout %s", server.WriteTimeout, requestTimeout)
	}
	handler := requestDeadline(10*time.Millisecond, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Deadline(); !ok {
			t.Error("request must have a deadline")
		}
		<-r.Context().Done()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"code":"tables_unavailable"}`)
	}))
	api := httptest.NewServer(handler)
	defer api.Close()
	response, err := api.Client().Get(api.URL)
	if err != nil {
		t.Fatalf("deadline must return HTTP response, not drop connection: %v", err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != 503 || !strings.Contains(string(body), "tables_unavailable") {
		t.Fatalf("unexpected deadline response: %d %s", response.StatusCode, body)
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
