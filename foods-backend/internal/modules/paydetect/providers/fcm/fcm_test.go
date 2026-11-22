package fcm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/foods-platform/foods-backend/internal/modules/paydetect/application"
)

func TestWakeSendsSilentHighPriorityMessage(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Error(err)
		}
		_, _ = w.Write([]byte(`{"name":"projects/p/messages/1"}`))
	}))
	defer server.Close()
	if err := NewWithClient(server.Client(), server.URL).Wake(context.Background(), "token-1"); err != nil {
		t.Fatal(err)
	}
	message := got["message"].(map[string]any)
	android := message["android"].(map[string]any)
	if message["token"] != "token-1" || android["priority"] != "HIGH" || message["notification"] != nil {
		t.Fatalf("mensaje inesperado: %+v", message)
	}
	if _, collapsible := android["collapse_key"]; collapsible {
		t.Fatal("sin collapse_key: FCM limita los plegables a 1 cada 3 min por teléfono")
	}
	if data := message["data"].(map[string]any); len(data) != 1 || data["type"] != "wake" {
		t.Fatalf("el push no debe llevar datos del local: %+v", data)
	}
}

func TestWakeReportsGoneTokens(t *testing.T) {
	cases := []struct {
		status int
		body   string
		gone   bool
	}{
		{404, `{"error":{"code":404,"status":"NOT_FOUND","details":[{"errorCode":"UNREGISTERED"}]}}`, true},
		{400, `{"error":{"code":400,"status":"INVALID_ARGUMENT","details":[{"errorCode":"INVALID_ARGUMENT"}]}}`, true},
		{503, `{"error":{"code":503,"status":"UNAVAILABLE"}}`, false},
		{403, `{"error":{"code":403,"status":"PERMISSION_DENIED"}}`, false},
	}
	for _, tc := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(tc.body))
		}))
		err := NewWithClient(server.Client(), server.URL).Wake(context.Background(), "token")
		server.Close()
		if err == nil || errors.Is(err, application.ErrPushTokenGone) != tc.gone {
			t.Fatalf("estado %d: err=%v, gone esperado %v", tc.status, err, tc.gone)
		}
	}
}
