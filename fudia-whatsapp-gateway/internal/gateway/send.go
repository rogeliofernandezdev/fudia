package gateway

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	_ "golang.org/x/image/webp"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

type messagingClient interface {
	IsConnected() bool
	IsLoggedIn() bool
	Upload(context.Context, []byte, whatsmeow.MediaType) (whatsmeow.UploadResponse, error)
	GenerateMessageID() types.MessageID
	SendMessage(context.Context, types.JID, *waE2E.Message, ...whatsmeow.SendRequestExtra) (whatsmeow.SendResponse, error)
}

type Button struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}
type SendRequest struct {
	SessionRef  string   `json:"sessionRef"`
	Recipient   string   `json:"recipient"`
	CallbackRef string   `json:"callbackRef"`
	Kind        string   `json:"kind"`
	Text        string   `json:"text,omitempty"`
	ImageURL    string   `json:"imageUrl,omitempty"`
	Caption     string   `json:"caption,omitempty"`
	Buttons     []Button `json:"buttons,omitempty"`
}
type SendResult struct {
	State      string `json:"state"`
	ExternalID string `json:"externalId"`
}

var callbackPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var ErrNotConnected = errors.New("session is not connected")
var ErrSendConflict = errors.New("idempotency key already has another payload")

func parseRecipient(value string) (types.JID, error) {
	if phonePattern.MatchString(value) {
		return types.NewJID(value, types.DefaultUserServer), nil
	}
	jid, err := types.ParseJID(value)
	if err != nil || jid.Device != 0 || (jid.Server != types.DefaultUserServer && jid.Server != types.HiddenUserServer) || jid.User == "" {
		return types.JID{}, errors.New("invalid recipient")
	}
	if !regexp.MustCompile(`^[0-9]{1,64}$`).MatchString(jid.User) || (jid.Server == types.DefaultUserServer && !phonePattern.MatchString(jid.User)) {
		return types.JID{}, errors.New("invalid recipient")
	}
	return jid, nil
}

func (g *Gateway) Send(ctx context.Context, in SendRequest) (SendResult, error) {
	ctx, cancelSend := context.WithTimeout(ctx, 50*time.Second)
	defer cancelSend()
	var out SendResult
	if !callbackPattern.MatchString(in.CallbackRef) || utf8.RuneCountInString(in.Text) > 4096 || utf8.RuneCountInString(in.Caption) > 1024 || len(in.Buttons) > 3 || (in.Kind != "text" && in.Kind != "image") {
		return out, errors.New("invalid message")
	}
	to, err := parseRecipient(in.Recipient)
	if err != nil {
		return out, err
	}
	s, err := g.getSession(in.SessionRef)
	if err != nil {
		return out, err
	}
	if !g.owner.alive(ctx) || s.blocked.Load() || !s.sender.IsConnected() || !s.sender.IsLoggedIn() {
		return out, ErrNotConnected
	}
	payload, _ := json.Marshal(in)
	hash := sha256.Sum256(payload)
	digest := hex.EncodeToString(hash[:])
	var oldHash string
	err = g.db.QueryRowContext(ctx, `SELECT payload_hash,state,external_id FROM gateway_outgoing WHERE session_id=$1 AND callback_ref=$2`, in.SessionRef, in.CallbackRef).Scan(&oldHash, &out.State, &out.ExternalID)
	if err == nil {
		if oldHash != digest {
			return out, ErrSendConflict
		}
		if out.State == "sending" {
			out.State = "uncertain"
		}
		return out, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return out, err
	}
	if g.delay != nil {
		if err := g.delay(ctx); err != nil {
			return out, err
		}
	}
	if !g.owner.alive(ctx) || s.blocked.Load() {
		return out, ErrNotConnected
	}
	message := &waE2E.Message{}
	if in.Kind == "image" {
		data, mime, err := g.image(ctx, in.ImageURL)
		if err != nil {
			return out, err
		}
		upload, err := s.sender.Upload(ctx, data, whatsmeow.MediaImage)
		if err != nil {
			return out, errors.New("image upload unavailable")
		}
		message.ImageMessage = &waE2E.ImageMessage{Caption: proto.String(in.Caption), Mimetype: proto.String(mime), URL: &upload.URL, DirectPath: &upload.DirectPath, MediaKey: upload.MediaKey, FileEncSHA256: upload.FileEncSHA256, FileSHA256: upload.FileSHA256, FileLength: &upload.FileLength}
	} else {
		text := in.Text
		if text == "" {
			return out, errors.New("empty message")
		}
		for index, button := range in.Buttons {
			if button.ID == "" || len(button.ID) > 256 || utf8.RuneCountInString(button.Title) > 80 {
				return out, errors.New("invalid option")
			}
			text += fmt.Sprintf("\n%d. %s", index+1, button.Title)
		}
		if len(in.Buttons) > 0 {
			text += "\nResponde a este mensaje con el número de tu elección."
		}
		if utf8.RuneCountInString(text) > 4096 {
			return out, errors.New("message exceeds size limit")
		}
		message.Conversation = proto.String(text)
	}
	buttons, _ := json.Marshal(in.Buttons)
	if in.Buttons == nil {
		buttons = []byte("[]")
	}
	out.ExternalID = string(s.sender.GenerateMessageID())
	out.State = "sending"
	tag, err := g.db.ExecContext(ctx, `INSERT INTO gateway_outgoing(session_id,callback_ref,external_id,payload_hash,buttons,state) VALUES($1,$2,$3,$4,$5::jsonb,'sending') ON CONFLICT(session_id,callback_ref) DO NOTHING`, in.SessionRef, in.CallbackRef, out.ExternalID, digest, buttons)
	if err != nil {
		return out, err
	}
	count, _ := tag.RowsAffected()
	if count == 0 {
		return g.Send(ctx, in)
	}
	// A crash after this checkpoint is uncertain, never a reason to send again
	// with another WhatsApp id. Receipts reconcile the persisted external id.
	_, err = s.sender.SendMessage(ctx, to, message, whatsmeow.SendRequestExtra{ID: types.MessageID(out.ExternalID), Timeout: 12 * time.Second})
	out.State = "sent"
	if err != nil {
		out.State = "uncertain"
	} else {
		g.sent.Add(1)
	}
	persist, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_, writeErr := g.db.ExecContext(persist, `UPDATE gateway_outgoing SET state=$3 WHERE session_id=$1 AND callback_ref=$2 AND state='sending'`, in.SessionRef, in.CallbackRef, out.State)
	if writeErr != nil {
		out.State = "uncertain"
	}
	return out, nil
}

func (g *Gateway) image(ctx context.Context, raw string) ([]byte, string, error) {
	if g.mediaSlots != nil {
		select {
		case g.mediaSlots <- struct{}{}:
			defer func() { <-g.mediaSlots }()
		case <-ctx.Done():
			return nil, "", ctx.Err()
		}
	}

	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Fragment != "" || (u.Scheme != "https" && !(g.config.Development && u.Scheme == "http")) {
		return nil, "", errors.New("unauthorized image URL")
	}
	allowed := false
	for _, host := range g.config.MediaHosts {
		if strings.EqualFold(u.Host, host) {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, "", errors.New("unauthorized image host")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	res, err := g.http.Do(req)
	if err != nil {
		return nil, "", errors.New("image unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, "", errors.New("image unavailable")
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, 8*1024*1024+1))
	if err != nil || len(data) > 8*1024*1024 {
		return nil, "", errors.New("image exceeds size limit")
	}
	info, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || int64(info.Width)*int64(info.Height) > 40_000_000 {
		return nil, "", errors.New("invalid image")
	}
	if format == "webp" {
		decoded, _, decodeErr := image.Decode(bytes.NewReader(data))
		if decodeErr != nil {
			return nil, "", errors.New("invalid WebP image")
		}
		var converted bytes.Buffer
		if alpha, ok := decoded.(interface{ Opaque() bool }); ok && !alpha.Opaque() {
			err = png.Encode(&converted, decoded)
			format = "png"
		} else {
			err = jpeg.Encode(&converted, decoded, &jpeg.Options{Quality: 85})
			format = "jpeg"
		}
		if err != nil || converted.Len() > 8*1024*1024 {
			return nil, "", errors.New("converted image exceeds size limit")
		}
		data = converted.Bytes()
	}
	mime := "image/" + format
	if format != "jpeg" && format != "png" {
		return nil, "", errors.New("unsupported image")
	}
	return data, mime, nil
}
