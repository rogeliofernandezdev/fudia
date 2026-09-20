package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	recorder := httptest.NewRecorder()
	health(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusOK { t.Fatalf("expected 200, got %d", recorder.Code) }
	if !strings.Contains(recorder.Body.String(), `"status":"ok"`) { t.Fatalf("unexpected body: %s", recorder.Body.String()) }
}

func TestEnvOrDefault(t *testing.T) {
	t.Setenv("FOODS_TEST_VALUE", "configured")
	if got := envOrDefault("FOODS_TEST_VALUE", "fallback"); got != "configured" { t.Fatalf("got %q", got) }
}
