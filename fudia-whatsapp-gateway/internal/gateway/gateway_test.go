package gateway

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waAdv"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestReplyDelayAlwaysExceedsTwoSeconds(t *testing.T) {
	seen := map[time.Duration]bool{}
	for range 100 {
		d, err := randomReplyDelay()
		if err != nil || d <= 2*time.Second || d > 5*time.Second {
			t.Fatalf("delay %s: %v", d, err)
		}
		seen[d] = true
	}
	if len(seen) < 2 {
		t.Fatal("delay is not randomized")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	if !errors.Is(waitReplyDelay(ctx), context.Canceled) || time.Since(start) > time.Second {
		t.Fatal("delay must stop on cancellation")
	}
}

func TestNormalizeVerifiedIdentityAndLocation(t *testing.T) {
	input := &events.Message{Info: types.MessageInfo{MessageSource: types.MessageSource{Sender: types.NewJID("123456789012345", types.HiddenUserServer)}, ID: "event-1", Timestamp: time.Now()}, Message: &waE2E.Message{Conversation: proto.String("hola")}}
	out, ok := normalizeMessage(input)
	if !ok || out.VerifiedPhone != "" || out.Sender != "123456789012345@lid" {
		t.Fatalf("LID must remain opaque: %+v", out)
	}
	input.Info.SenderAlt = types.NewJID("51912345678", types.DefaultUserServer)
	out, ok = normalizeMessage(input)
	if !ok || out.VerifiedPhone != "51912345678" {
		t.Fatal("authenticated PN alternative missing")
	}
	input.Message = &waE2E.Message{LocationMessage: &waE2E.LocationMessage{DegreesLatitude: proto.Float64(-12.1), DegreesLongitude: proto.Float64(-77.1)}}
	out, ok = normalizeMessage(input)
	if !ok || out.Location == nil || out.Location.Latitude != -12.1 {
		t.Fatal("location missing")
	}
	input.Message.LocationMessage.DegreesLatitude = proto.Float64(91)
	if _, ok = normalizeMessage(input); ok {
		t.Fatal("invalid location accepted")
	}
	input.Info.IsGroup = true
	if _, ok = normalizeMessage(input); ok {
		t.Fatal("group accepted")
	}
	for _, bad := range []string{"1@g.us", "1@broadcast", "51912345678:2@s.whatsapp.net", ""} {
		if _, err := parseRecipient(bad); err == nil {
			t.Fatalf("recipient accepted: %q", bad)
		}
	}
}

type httpTransport struct{}

func (httpTransport) StartLink(context.Context, string) (Connection, error) {
	return Connection{State: "pairing", QRCode: "private-device-code"}, nil
}
func (httpTransport) Status(_ context.Context, _ string, country string) (Connection, error) {
	if country != "PE" {
		return Connection{}, errors.New("not found")
	}
	return Connection{State: "connected"}, nil
}
func (httpTransport) Send(context.Context, SendRequest) (SendResult, error) {
	return SendResult{State: "uncertain"}, nil
}

func TestPrivateHTTPAuthenticationAndNoCache(t *testing.T) {
	key := strings.Repeat("k", 32)
	handler := Handler(httpTransport{}, key, func(context.Context) bool { return true }, func() string { return "metric 1" })
	call := func(method, path, body, secret string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("X-Fudia-Gateway-Key", secret)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	if call("GET", "/live", "", "").Code != 200 {
		t.Fatal("health unavailable")
	}
	for _, path := range []string{"/metrics", "/v1/sessions/x?countryCode=PE"} {
		if call("GET", path, "", "").Code != 401 {
			t.Fatal("private endpoint exposed")
		}
	}
	link := call("POST", "/v1/link", `{"countryCode":"PE"}`, key)
	if link.Code != 200 || link.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("link response must stay private")
	}
	if call("POST", "/v1/link", `{"countryCode":"PE"} {}`, key).Code != 400 {
		t.Fatal("trailing JSON accepted")
	}
	if call("GET", "/v1/sessions/x?countryCode=CL", "", key).Code != 404 {
		t.Fatal("cross-country session exposed")
	}
	if call("POST", "/v1/messages", `{}`, key).Code != 202 {
		t.Fatal("uncertain send must not be confirmed")
	}
}

type fakeMessaging struct {
	calls int
	fail  bool
	last  *waE2E.Message
}

func (*fakeMessaging) IsConnected() bool { return true }
func (*fakeMessaging) IsLoggedIn() bool  { return true }
func (*fakeMessaging) GenerateMessageID() types.MessageID {
	return types.MessageID("offline-" + time.Now().Format("150405.000000000"))
}
func (*fakeMessaging) Upload(context.Context, []byte, whatsmeow.MediaType) (whatsmeow.UploadResponse, error) {
	return whatsmeow.UploadResponse{URL: "https://media.example.test/image"}, nil
}
func (f *fakeMessaging) SendMessage(_ context.Context, _ types.JID, message *waE2E.Message, _ ...whatsmeow.SendRequestExtra) (whatsmeow.SendResponse, error) {
	f.calls++
	f.last = message
	if f.fail {
		return whatsmeow.SendResponse{}, errors.New("uncertain")
	}
	return whatsmeow.SendResponse{}, nil
}

func TestPostgresOwnershipDurableSessionAndIdempotentTransport(t *testing.T) {
	databaseURL := os.Getenv("GATEWAY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("GATEWAY_TEST_DATABASE_URL required; use an isolated database")
	}
	var pngData bytes.Buffer
	_ = png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	media := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(pngData.Bytes()) }))
	defer media.Close()
	parsed, _ := url.Parse(media.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := Config{DatabaseURL: databaseURL, ConciergeURL: "http://127.0.0.1:1", Development: true, MediaHosts: []string{parsed.Host}}
	g, err := openGateway(ctx, cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { g.Close() }()
	if second, err := openGateway(ctx, cfg, false); err == nil {
		second.Close()
		t.Fatal("second session owner accepted")
	}
	device := g.devices.NewDevice()
	jid := types.NewJID("51987654321", types.DefaultUserServer)
	device.ID = &jid
	device.Account = &waAdv.ADVSignedDeviceIdentity{Details: []byte{1}, AccountSignature: make([]byte, 64), AccountSignatureKey: make([]byte, 32), DeviceSignature: make([]byte, 64)}
	if err = device.Save(ctx); err != nil {
		t.Fatal(err)
	}
	var ref string
	if err = g.db.QueryRowContext(ctx, `INSERT INTO gateway_sessions(country_code,device_jid) VALUES('PE',$1) RETURNING id::text`, jid.String()).Scan(&ref); err != nil {
		t.Fatal(err)
	}
	fake := &fakeMessaging{}
	g.sessions[ref] = &session{ref: ref, country: "PE", sender: fake}
	delays := 0
	g.delay = func(context.Context) error { delays++; return nil }
	request := SendRequest{SessionRef: ref, Recipient: "123456789012345@lid", CallbackRef: strings.Repeat("a", 64), Kind: "text", Text: "Hola"}
	first, err := g.Send(ctx, request)
	if err != nil || first.State != "sent" {
		t.Fatalf("send: %+v %v", first, err)
	}
	again, err := g.Send(ctx, request)
	if err != nil || again.ExternalID != first.ExternalID || fake.calls != 1 || delays != 1 {
		t.Fatal("duplicate send or delay repeated")
	}
	request.Text = "Changed"
	if _, err = g.Send(ctx, request); !errors.Is(err, ErrSendConflict) {
		t.Fatal("changed payload accepted")
	}
	request.CallbackRef = strings.Repeat("b", 64)
	fake.fail = true
	uncertain, err := g.Send(ctx, request)
	if err != nil || uncertain.State != "uncertain" {
		t.Fatal("send uncertainty lost")
	}
	_, _ = g.Send(ctx, request)
	if fake.calls != 2 {
		t.Fatal("uncertain message resent")
	}
	fake.fail = false
	request.Kind = "image"
	request.ImageURL = media.URL
	request.Caption = "Carta"
	request.CallbackRef = strings.Repeat("c", 64)
	if _, err = g.Send(ctx, request); err != nil || fake.last.ImageMessage == nil {
		t.Fatalf("image: %v", err)
	}
	if _, _, err = g.image(ctx, "http://127.0.0.1:2/private"); err == nil {
		t.Fatal("unauthorized media host accepted")
	}
	event := Event{Type: "message", Text: "Hola"}
	if err = g.enqueue(ctx, ref, "unique-event", event); err != nil {
		t.Fatal(err)
	}
	if err = g.enqueue(ctx, ref, "unique-event", event); err != nil {
		t.Fatal(err)
	}
	var count int
	_ = g.db.QueryRowContext(ctx, `SELECT count(*) FROM gateway_inbound_outbox WHERE session_id=$1 AND event_key='unique-event'`, ref).Scan(&count)
	if count != 1 {
		t.Fatal("duplicate ingress persisted")
	}
	delete(g.sessions, ref) // The fake has no socket; the persisted real device is recovered below.
	originalKey := *device.IdentityKey.Priv
	g.Close()
	g, err = openGateway(ctx, cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	restored := g.sessions[ref]
	if restored == nil || *restored.client.Store.IdentityKey.Priv != originalKey || restored.phone != "+51987654321" {
		t.Fatal("SQL device credentials were not recovered")
	}
	// Test data is confined to the explicitly selected disposable gateway DB.
	_, _ = g.db.ExecContext(ctx, `DELETE FROM gateway_inbound_outbox WHERE session_id=$1`, ref)
	_, _ = g.db.ExecContext(ctx, `DELETE FROM gateway_outgoing WHERE session_id=$1`, ref)
	_, _ = g.db.ExecContext(ctx, `DELETE FROM gateway_sessions WHERE id=$1`, ref)
	_ = restored.client.Store.Delete(ctx)
}

func TestReplyDelayActuallyWaits(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	start := time.Now()
	if err := waitReplyDelay(ctx); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	if elapsed <= 2*time.Second || elapsed > 6*time.Second {
		t.Fatalf("actual response delay: %s", elapsed)
	}
}

func TestWebPCartaIsConvertedForWhatsApp(t *testing.T) {
	data, err := os.ReadFile("testdata/gradient.webp")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer server.Close()
	parsed, _ := url.Parse(server.URL)
	g := &Gateway{config: Config{Development: true, MediaHosts: []string{parsed.Host}}, http: server.Client(), mediaSlots: make(chan struct{}, 2)}
	converted, mime, err := g.image(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(converted))
	if err != nil || mime != "image/jpeg" || format != "jpeg" {
		t.Fatalf("WebP conversion: %s %s %v", mime, format, err)
	}
}

type contextConnection struct{ ctx context.Context }

func (c *contextConnection) ConnectContext(ctx context.Context) error { c.ctx = ctx; return nil }
func TestConnectionContextLivesUntilGatewayStops(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	probe := &contextConnection{}
	g := &Gateway{ctx: ctx}
	g.connect(&session{connector: probe})
	if probe.ctx == nil || probe.ctx.Err() != nil {
		t.Fatal("initial connection canceled keepalive/reconnect context")
	}
	stop()
	if !errors.Is(probe.ctx.Err(), context.Canceled) {
		t.Fatal("connection did not follow gateway shutdown")
	}
}

func TestPairingReconnectPreservesPendingLink(t *testing.T) {
	g := &Gateway{}
	s := &session{state: "pairing"}
	g.onEvent(s, &events.Disconnected{})
	if s.view().State != "pairing" {
		t.Fatal("transient pairing disconnect abandoned pending device")
	}
	s.phone = "+51912345678"
	g.onEvent(s, &events.Disconnected{})
	if s.view().State != "disconnected" {
		t.Fatal("linked disconnect was not reported")
	}
	s.blocked.Store(true)
	g.onEvent(s, &events.Disconnected{})
	if s.view().State != "unlinked" {
		t.Fatal("permanent unlink was overwritten")
	}
}
