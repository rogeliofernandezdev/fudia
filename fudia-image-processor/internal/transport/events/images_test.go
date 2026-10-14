package events

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

type processorFake struct {
	calls int
	err   error
}

func (p *processorFake) Process(context.Context, string, int64) error { p.calls++; return p.err }
func TestEventValidationAndRetries(t *testing.T) {
	for _, scenario := range []struct {
		name, bucket, event, generation string
		failure                         bool
		code                            int
	}{{"valid", "images", "google.cloud.storage.object.v1.finalized", "1", false, 204}, {"wrong-bucket", "other", "google.cloud.storage.object.v1.finalized", "1", false, 400}, {"wrong-type", "images", "wrong", "1", false, 400}, {"wrong-generation", "images", "google.cloud.storage.object.v1.finalized", "0", false, 400}, {"transient", "images", "google.cloud.storage.object.v1.finalized", "1", true, 503}} {
		t.Run(scenario.name, func(t *testing.T) {
			processor := &processorFake{}
			if scenario.failure {
				processor.err = errors.New("transient")
			}
			request := httptest.NewRequest("POST", "/", strings.NewReader(`{"bucket":"`+scenario.bucket+`","name":"object","generation":"`+scenario.generation+`"}`))
			request.Header.Set("Ce-Type", scenario.event)
			request.Header.Set("Ce-Id", "event-id")
			response := httptest.NewRecorder()
			Images(processor, "images").ServeHTTP(response, request)
			if response.Code != scenario.code {
				t.Fatalf("status %d", response.Code)
			}
			if scenario.code == 400 && processor.calls != 0 {
				t.Fatal("processed an invalid event")
			}
		})
	}
}
