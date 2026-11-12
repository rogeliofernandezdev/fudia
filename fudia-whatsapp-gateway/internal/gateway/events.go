package gateway

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type Point struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
type Event struct {
	Type          string `json:"type"`
	Provider      string `json:"provider"`
	SessionRef    string `json:"sessionRef"`
	CountryCode   string `json:"countryCode"`
	ExternalID    string `json:"externalId,omitempty"`
	Sender        string `json:"sender,omitempty"`
	VerifiedPhone string `json:"verifiedPhone,omitempty"`
	Text          string `json:"text,omitempty"`
	Location      *Point `json:"location,omitempty"`
	ButtonPayload string `json:"buttonPayload,omitempty"`
	Timestamp     int64  `json:"timestamp"`
	CallbackRef   string `json:"callbackRef,omitempty"`
	State         string `json:"state,omitempty"`
}

// Only authenticated WhatsApp sender metadata can prove a telephone. A LID's
// numeric user part is never a phone, and quoted text cannot prove an identity.
func normalizeMessage(v *events.Message) (Event, bool) {
	var out Event
	if v.Info.IsFromMe || v.Info.IsGroup || v.Info.Sender.IsEmpty() || v.IsEdit || v.Message == nil {
		return out, false
	}
	sender := v.Info.Sender.ToNonAD()
	if sender.Server != types.DefaultUserServer && sender.Server != types.HiddenUserServer {
		return out, false
	}
	out = Event{Type: "message", Provider: "WHATSMEOW", ExternalID: v.Info.ID, Sender: sender.String(), Timestamp: v.Info.Timestamp.Unix()}
	for _, jid := range []types.JID{sender, v.Info.SenderAlt.ToNonAD()} {
		if jid.Server == types.DefaultUserServer && phonePattern.MatchString(jid.User) {
			out.VerifiedPhone = jid.User
			break
		}
	}
	out.Text = v.Message.GetConversation()
	if out.Text == "" {
		out.Text = v.Message.GetExtendedTextMessage().GetText()
	}
	if pin := v.Message.GetLocationMessage(); pin != nil {
		lat, lon := pin.GetDegreesLatitude(), pin.GetDegreesLongitude()
		if math.IsNaN(lat) || math.IsNaN(lon) || math.IsInf(lat, 0) || math.IsInf(lon, 0) || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			return Event{}, false
		}
		out.Location = &Point{lat, lon}
	}
	if len(out.Text) > 8000 || len(out.ExternalID) == 0 || len(out.ExternalID) > 200 {
		return Event{}, false
	}
	return out, out.Text != "" || out.Location != nil
}

func (g *Gateway) incoming(s *session, v *events.Message) {
	out, ok := normalizeMessage(v)
	if !ok || !g.owner.alive(g.ctx) || s.blocked.Load() {
		return
	}
	out.SessionRef = s.ref
	out.CountryCode = s.country
	if out.VerifiedPhone == "" && v.Info.Sender.Server == types.HiddenUserServer {
		if pn, err := g.devices.LIDMap.GetPNForLID(g.ctx, v.Info.Sender.ToNonAD()); err == nil && pn.Server == types.DefaultUserServer && phonePattern.MatchString(pn.User) {
			out.VerifiedPhone = pn.User
		}
	}
	// Text fallback for providers without reliable buttons: the customer replies
	// to the original message with its option number. Never infer the offer.
	quoted := v.Message.GetExtendedTextMessage().GetContextInfo().GetStanzaID()
	if option, err := strconv.Atoi(strings.TrimSpace(out.Text)); err == nil && option > 0 && quoted != "" {
		var raw []byte
		if err = g.db.QueryRowContext(g.ctx, `SELECT buttons FROM gateway_outgoing WHERE session_id=$1 AND external_id=$2`, s.ref, quoted).Scan(&raw); err == nil {
			var buttons []Button
			if json.Unmarshal(raw, &buttons) == nil && option <= len(buttons) {
				out.ButtonPayload = buttons[option-1].ID
			}
		}
	}
	ctx, cancel := context.WithTimeout(g.ctx, 10*time.Second)
	defer cancel()
	// Persist before delivery to Concierge. A duplicate event is a no-op.
	if err := g.enqueue(ctx, s.ref, "message:"+out.ExternalID, out); err != nil {
		g.loseOwnership()
	}
}

func (g *Gateway) receipt(s *session, v *events.Receipt) {
	state := ""
	switch v.Type {
	case types.ReceiptTypeDelivered:
		state = "delivered"
	case types.ReceiptTypeRead:
		state = "read"
	default:
		return
	}
	for _, id := range v.MessageIDs {
		var callback string
		err := g.db.QueryRowContext(g.ctx, `UPDATE gateway_outgoing SET state=CASE WHEN state='read' THEN state ELSE $3 END WHERE session_id=$1 AND external_id=$2 RETURNING callback_ref`, s.ref, id, state).Scan(&callback)
		if err != nil {
			continue
		}
		out := Event{Type: "status", Provider: "WHATSMEOW", SessionRef: s.ref, CountryCode: s.country, CallbackRef: callback, State: state, Timestamp: v.Timestamp.Unix()}
		_ = g.enqueue(g.ctx, s.ref, "status:"+string(id)+":"+state, out)
	}
}
