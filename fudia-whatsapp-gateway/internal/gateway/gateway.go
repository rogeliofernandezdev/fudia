package gateway

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var countryPattern = regexp.MustCompile(`^[A-Z]{2}$`)
var phonePattern = regexp.MustCompile(`^[1-9][0-9]{7,14}$`)
var refPattern = regexp.MustCompile(`^[0-9a-fA-F-]{36}$`)

type Connection struct {
	Provider    string `json:"provider"`
	CountryCode string `json:"countryCode"`
	SessionRef  string `json:"sessionRef"`
	State       string `json:"state"`
	PhoneNumber string `json:"phoneNumber"`
	QRCode      string `json:"qrCode,omitempty"`
	ExpiresAt   string `json:"expiresAt,omitempty"`
}

type connectionClient interface{ ConnectContext(context.Context) error }

type session struct {
	mu                             sync.Mutex
	ref, country, phone, state, qr string
	expires                        time.Time
	client                         *whatsmeow.Client
	connector                      connectionClient
	sender                         messagingClient
	blocked                        atomic.Bool
}

func (s *session) view() Connection {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := Connection{Provider: "WHATSMEOW", CountryCode: s.country, SessionRef: s.ref, State: s.state, PhoneNumber: s.phone}
	if s.state == "pairing" && time.Now().Before(s.expires) {
		out.QRCode = s.qr
		out.ExpiresAt = s.expires.UTC().Format(time.RFC3339)
	}
	return out
}

type Gateway struct {
	config                                    Config
	db                                        *sql.DB
	devices                                   *sqlstore.Container
	owner                                     *Owner
	http                                      *http.Client
	ctx                                       context.Context
	stop                                      context.CancelFunc
	mu                                        sync.RWMutex
	sessions                                  map[string]*session
	delay                                     func(context.Context) error
	mediaSlots                                chan struct{}
	linkMu                                    sync.Mutex
	received, relayFailures, reconnects, sent atomic.Uint64
}

func Open(ctx context.Context, c Config) (*Gateway, error) { return openGateway(ctx, c, true) }
func openGateway(ctx context.Context, c Config, connect bool) (*Gateway, error) {
	db, err := sql.Open("pgx", c.DatabaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(4)
	owner, err := acquireOwner(ctx, db)
	if err != nil {
		db.Close()
		return nil, err
	}
	cleanup := func() { owner.Close(); db.Close() }
	if _, err = db.ExecContext(ctx, schema); err != nil {
		cleanup()
		return nil, err
	}
	devices := sqlstore.NewWithDB(db, "postgres", nil)
	if err = devices.Upgrade(ctx); err != nil {
		cleanup()
		return nil, err
	}
	lifecycle, stop := context.WithCancel(ctx)
	previousCleanup := cleanup
	cleanup = func() { stop(); previousCleanup() }
	g := &Gateway{config: c, db: db, devices: devices, owner: owner, ctx: lifecycle, stop: stop, sessions: map[string]*session{}, delay: waitReplyDelay, mediaSlots: make(chan struct{}, 2), http: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	rows, err := db.QueryContext(ctx, `SELECT id::text,country_code,device_jid FROM gateway_sessions WHERE device_jid IS NOT NULL ORDER BY created_at`)
	if err != nil {
		cleanup()
		return nil, err
	}
	for rows.Next() {
		var id, country, jidText string
		if err = rows.Scan(&id, &country, &jidText); err != nil {
			break
		}
		jid, parseErr := types.ParseJID(jidText)
		if parseErr != nil {
			err = parseErr
			break
		}
		device, deviceErr := devices.GetDevice(ctx, jid)
		if deviceErr != nil {
			err = deviceErr
			break
		}
		if device == nil {
			continue
		}
		s := &session{ref: id, country: country, state: "disconnected", client: whatsmeow.NewClient(device, nil)}
		if jid.Server == types.DefaultUserServer && phonePattern.MatchString(jid.User) {
			s.phone = "+" + jid.User
		}
		g.attach(s)
		g.sessions[id] = s
	}
	if scanErr := rows.Err(); err == nil {
		err = scanErr
	}
	rows.Close()
	if err != nil {
		cleanup()
		return nil, err
	}
	if connect {
		for _, s := range g.sessions {
			go g.connect(s)
		}
	}
	go owner.monitor(lifecycle, g.loseOwnership)
	go g.relay(lifecycle)
	return g, nil
}

func (g *Gateway) attach(s *session) {
	// The library's automatic reconnect stops on logout/stream replacement. A
	// permanent unlink always requires an administrator to pair again.
	s.sender = s.client
	s.connector = s.client
	s.client.EnableAutoReconnect = true
	s.client.InitialAutoReconnect = true
	s.client.AutoReconnectHook = func(error) bool {
		g.reconnects.Add(1)
		return g.ctx.Err() == nil && !g.owner.lost.Load() && !s.blocked.Load()
	}
	s.client.AddEventHandler(func(event any) { g.onEvent(s, event) })
}

func (g *Gateway) connect(s *session) {
	// whatsmeow retains this context for keepalive and reconnection. It must
	// outlive the initial handshake and end only with the gateway's lifecycle.
	if err := s.connector.ConnectContext(g.ctx); err != nil {
		s.mu.Lock()
		s.state = "disconnected"
		s.mu.Unlock()
	}
}

func (g *Gateway) StartLink(ctx context.Context, country string) (Connection, error) {
	if !countryPattern.MatchString(country) || g.owner.lost.Load() {
		return Connection{}, errors.New("gateway unavailable")
	}
	g.linkMu.Lock()
	defer g.linkMu.Unlock()
	g.mu.RLock()
	for _, s := range g.sessions {
		v := s.view()
		if v.CountryCode == country && v.State == "pairing" {
			g.mu.RUnlock()
			return v, nil
		}
	}
	g.mu.RUnlock()
	var id string
	if err := g.db.QueryRowContext(ctx, `INSERT INTO gateway_sessions(country_code) VALUES($1) RETURNING id::text`, country).Scan(&id); err != nil {
		return Connection{}, err
	}
	s := &session{ref: id, country: country, state: "pairing", client: whatsmeow.NewClient(g.devices.NewDevice(), nil)}
	g.attach(s)
	qr, err := s.client.GetQRChannel(g.ctx)
	if err != nil {
		return Connection{}, err
	}
	g.mu.Lock()
	g.sessions[id] = s
	g.mu.Unlock()
	go func() {
		for code := range qr {
			s.mu.Lock()
			if code.Event == "code" {
				s.qr = code.Code
				s.expires = time.Now().Add(code.Timeout)
			} else if code.Event != "success" && s.state == "pairing" {
				s.state = "disconnected"
				s.qr = ""
			}
			s.mu.Unlock()
		}
	}()
	go g.connect(s)
	return s.view(), nil
}

func (g *Gateway) Status(ctx context.Context, id, country string) (Connection, error) {
	s, err := g.getSession(id)
	if err != nil {
		return Connection{}, err
	}
	if country != "" && country != s.country {
		return Connection{}, errors.New("session not found")
	}
	out := s.view()
	if out.State == "connected" && (!s.client.IsConnected() || !s.client.IsLoggedIn() || g.owner.lost.Load()) {
		out.State = "disconnected"
	}
	return out, nil
}

func (g *Gateway) getSession(id string) (*session, error) {
	if !refPattern.MatchString(id) {
		return nil, errors.New("session not found")
	}
	g.mu.RLock()
	s := g.sessions[id]
	g.mu.RUnlock()
	if s == nil {
		return nil, errors.New("session not found")
	}
	return s, nil
}

func (g *Gateway) onEvent(s *session, event any) {
	switch v := event.(type) {
	case *events.PairSuccess:
		if v.ID.Server != types.DefaultUserServer || !phonePattern.MatchString(v.ID.User) {
			s.client.Disconnect()
			return
		}
		if _, err := g.db.ExecContext(g.ctx, `UPDATE gateway_sessions SET device_jid=$2 WHERE id=$1`, s.ref, v.ID.String()); err != nil {
			s.client.Disconnect()
			return
		}
		s.mu.Lock()
		s.phone = "+" + v.ID.User
		s.qr = ""
		s.mu.Unlock()
	case *events.Connected:
		s.mu.Lock()
		if s.phone != "" {
			s.state = "connected"
		}
		s.qr = ""
		s.mu.Unlock()
	case *events.Disconnected:
		s.mu.Lock()
		s.state = "disconnected"
		if s.blocked.Load() {
			s.state = "unlinked"
		} else if s.phone == "" {
			s.state = "pairing"
		}
		s.qr = ""
		s.mu.Unlock()
	case *events.LoggedOut, *events.StreamReplaced:
		s.blocked.Store(true)
		s.client.Disconnect()
		s.mu.Lock()
		s.state = "unlinked"
		s.qr = ""
		s.mu.Unlock()
	case *events.Message:
		g.incoming(s, v)
	case *events.Receipt:
		g.receipt(s, v)
	}
}

func (g *Gateway) disconnectAll() {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, s := range g.sessions {
		s.client.Disconnect()
	}
}
func (g *Gateway) loseOwnership() {
	g.owner.lost.Store(true)
	if g.stop != nil {
		g.stop()
	}
	g.disconnectAll()
}
func (g *Gateway) Close() error {
	if g.stop != nil {
		g.stop()
	}
	g.disconnectAll()
	g.owner.Close()
	return g.db.Close()
}
