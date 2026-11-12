package gateway

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Transport interface {
	StartLink(context.Context, string) (Connection, error)
	Status(context.Context, string, string) (Connection, error)
	Send(context.Context, SendRequest) (SendResult, error)
}

func Handler(transport Transport, key string, ready func(context.Context) bool, metrics func() string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /live", func(w http.ResponseWriter, r *http.Request) { reply(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		if !ready(r.Context()) {
			reply(w, 503, map[string]string{"status": "unavailable"})
			return
		}
		reply(w, 200, map[string]string{"status": "ok"})
	})
	private := http.NewServeMux()
	private.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprint(w, metrics())
	})
	private.HandleFunc("POST /v1/link", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			CountryCode string `json:"countryCode"`
		}
		if !decode(w, r, &in) || !countryPattern.MatchString(in.CountryCode) {
			reply(w, 400, map[string]string{"error": "invalid_country"})
			return
		}
		out, err := transport.StartLink(r.Context(), in.CountryCode)
		if err != nil {
			reply(w, 503, map[string]string{"error": "link_unavailable"})
			return
		}
		reply(w, 200, out)
	})
	private.HandleFunc("GET /v1/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		out, err := transport.Status(r.Context(), r.PathValue("id"), r.URL.Query().Get("countryCode"))
		if err != nil {
			reply(w, 404, map[string]string{"error": "session_not_found"})
			return
		}
		reply(w, 200, out)
	})
	private.HandleFunc("POST /v1/messages", func(w http.ResponseWriter, r *http.Request) {
		var in SendRequest
		if !decode(w, r, &in) {
			reply(w, 400, map[string]string{"error": "invalid_message"})
			return
		}
		out, err := transport.Send(r.Context(), in)
		if err != nil {
			status := 400
			code := "message_rejected"
			if errors.Is(err, ErrNotConnected) {
				status = 503
				code = "session_disconnected"
			}
			if errors.Is(err, ErrSendConflict) {
				status = 409
				code = "idempotency_conflict"
			}
			reply(w, status, map[string]string{"error": code})
			return
		}
		status := 200
		if out.State == "uncertain" || out.State == "sending" {
			status = 202
		}
		reply(w, status, out)
	})
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided := r.Header.Get("X-Fudia-Gateway-Key")
		if len(key) < 32 || len(key) != len(provided) || subtle.ConstantTimeCompare([]byte(key), []byte(provided)) != 1 {
			reply(w, 401, map[string]string{"error": "unauthorized"})
			return
		}
		private.ServeHTTP(w, r)
	}))
	return mux
}

func decode(w http.ResponseWriter, r *http.Request, out any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536))
	decoder.DisallowUnknownFields()
	return decoder.Decode(out) == nil && decoder.Decode(new(any)) == io.EOF
}
func reply(w http.ResponseWriter, status int, out any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(out)
}

func (g *Gateway) Handler() http.Handler {
	return Handler(g, g.config.APIKey, func(ctx context.Context) bool { return g.owner.alive(ctx) && g.db.PingContext(ctx) == nil }, g.metrics)
}
func (g *Gateway) metrics() string {
	var connected uint64
	g.mu.RLock()
	for _, s := range g.sessions {
		if s.view().State == "connected" {
			connected++
		}
	}
	g.mu.RUnlock()
	var pending int
	_ = g.db.QueryRowContext(g.ctx, `SELECT count(*) FROM gateway_inbound_outbox WHERE delivered_at IS NULL`).Scan(&pending)
	var out strings.Builder
	fmt.Fprintf(&out, "fudia_whatsapp_gateway_connected %d\nfudia_whatsapp_gateway_received_total %d\nfudia_whatsapp_gateway_sent_total %d\nfudia_whatsapp_gateway_reconnects_total %d\nfudia_whatsapp_gateway_relay_failures_total %d\nfudia_whatsapp_gateway_pending %d\n", connected, g.received.Load(), g.sent.Load(), g.reconnects.Load(), g.relayFailures.Load(), pending)
	return out.String()
}
