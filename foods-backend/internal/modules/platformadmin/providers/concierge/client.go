// Package whatsapp verifies transport configuration through Concierge. Provider
// credentials and whatsmeow sessions never enter the administration API.
package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
)

type Client struct {
	base, key string
	http      *http.Client
}

func New(base, key string) *Client {
	return &Client{strings.TrimRight(base, "/"), key, &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (c *Client) call(ctx context.Context, path string, in, out any) error {
	u, err := url.Parse(c.base)
	if err != nil || c.key == "" || u.Host == "" || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"))) || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("provider service is not configured")
	}
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-Fudia-Concierge-Key", c.key)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return errors.New("provider service unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return errors.New("provider verification rejected")
	}
	return json.NewDecoder(io.LimitReader(res.Body, 262144)).Decode(out)
}
func (c *Client) Verify(ctx context.Context, in domain.PlatformWhatsAppChannelInput) (domain.WhatsAppConnection, error) {
	var out domain.WhatsAppConnection
	err := c.call(ctx, "/internal/whatsapp/verify", map[string]string{"countryCode": in.CountryCode, "provider": in.Provider, "phoneNumber": in.PhoneNumber, "sessionRef": in.SessionRef}, &out)
	return out, err
}
func (c *Client) Link(ctx context.Context, in domain.WhatsAppLinkInput) (domain.WhatsAppConnection, error) {
	var out domain.WhatsAppConnection
	err := c.call(ctx, "/internal/whatsapp/link", in, &out)
	return out, err
}
