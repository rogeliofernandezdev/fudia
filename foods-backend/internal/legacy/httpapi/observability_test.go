package httpapi

import (
	"net/http/httptest"
	"regexp"
	"testing"
)

var correlationIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestNewCorrelationIDUsesUUIDv4Format(t *testing.T) {
	first := NewCorrelationID()
	second := NewCorrelationID()
	if !correlationIDPattern.MatchString(first) {
		t.Fatalf("expected UUID v4 correlation id, got %q", first)
	}
	if first == second {
		t.Fatalf("expected unique correlation ids, got %q twice", first)
	}
}

func TestEnsureCorrelationIDAddsHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	id := ensureCorrelationID(recorder)
	if !correlationIDPattern.MatchString(id) {
		t.Fatalf("expected UUID v4 correlation id, got %q", id)
	}
	if got := recorder.Header().Get("X-Request-ID"); got != id {
		t.Fatalf("expected X-Request-ID %q, got %q", id, got)
	}
}
