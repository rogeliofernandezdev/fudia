package gateway

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

//go:embed schema.sql
var schema string

// A dedicated PostgreSQL connection owns the transport. Losing it closes every
// client; another replica can never keep using the same sessions concurrently.
type Owner struct {
	conn *sql.Conn
	lost atomic.Bool
}

func acquireOwner(ctx context.Context, db *sql.DB) (*Owner, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	var ok bool
	if err = conn.QueryRowContext(ctx, `SELECT pg_try_advisory_lock(6743300182840)`).Scan(&ok); err != nil || !ok {
		conn.Close()
		return nil, errors.New("another gateway owns the session database")
	}
	return &Owner{conn: conn}, nil
}

func (o *Owner) alive(ctx context.Context) bool {
	if o.lost.Load() {
		return false
	}
	if ctx.Err() != nil {
		return false
	}
	check, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	var held bool
	err := o.conn.QueryRowContext(check, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE pid=pg_backend_pid() AND locktype='advisory' AND granted AND classid=(6743300182840::bigint >> 32)::oid AND objid=(6743300182840::bigint & 4294967295)::oid AND objsubid=1)`).Scan(&held)
	if err != nil || !held {
		o.lost.Store(true)
		return false
	}
	return true
}
func (o *Owner) monitor(ctx context.Context, onLost func()) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check, cancel := context.WithTimeout(ctx, 3*time.Second)
			held := o.alive(check)
			cancel()
			if !held {
				o.lost.Store(true)
				onLost()
				return
			}
		}
	}
}

func (o *Owner) Close() { o.lost.Store(true); _ = o.conn.Close() }

func (g *Gateway) enqueue(ctx context.Context, sessionID, key string, event any) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = g.db.ExecContext(ctx, `INSERT INTO gateway_inbound_outbox(session_id,event_key,payload) VALUES($1,$2,$3::jsonb) ON CONFLICT(session_id,event_key) DO NOTHING`, sessionID, key, data)
	if err == nil {
		g.received.Add(1)
	}
	return err
}

func (g *Gateway) relay(ctx context.Context) {
	for ctx.Err() == nil && !g.owner.lost.Load() {
		var id string
		var payload []byte
		err := g.db.QueryRowContext(ctx, `SELECT id::text,payload FROM gateway_inbound_outbox WHERE delivered_at IS NULL ORDER BY sequence LIMIT 1`).Scan(&id, &payload)
		if errors.Is(err, sql.ErrNoRows) {
			if !wait(ctx, time.Second) {
				return
			}
			continue
		}
		if err != nil {
			if !wait(ctx, 3*time.Second) {
				return
			}
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.config.ConciergeURL+"/internal/whatsapp/gateway/events", bytes.NewReader(payload))
		if err == nil {
			req.Header.Set("X-Fudia-Gateway-Key", g.config.IngressKey)
			req.Header.Set("Content-Type", "application/json")
			var response *http.Response
			response, err = g.http.Do(req)
			if response != nil {
				_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
				response.Body.Close()
				if response.StatusCode < 200 || response.StatusCode >= 300 {
					err = errors.New("ingress rejected event")
				}
			}
		}
		if err != nil {
			g.relayFailures.Add(1)
			_, _ = g.db.ExecContext(ctx, `UPDATE gateway_inbound_outbox SET attempts=attempts+1 WHERE id=$1`, id)
			if !wait(ctx, 3*time.Second) {
				return
			}
			continue
		}
		_, err = g.db.ExecContext(ctx, `UPDATE gateway_inbound_outbox SET delivered_at=now(),payload='{}'::jsonb WHERE id=$1`, id)
		if err != nil && !wait(ctx, time.Second) {
			return
		}
	}
}

func wait(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
