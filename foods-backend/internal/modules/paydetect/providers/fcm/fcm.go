// Package fcm envía el aviso silencioso de Fudia Pay por Firebase Cloud
// Messaging (API HTTP v1) con las credenciales por defecto del entorno: en
// Cloud Run, la cuenta de servicio de la API (rol «Firebase Cloud Messaging
// API Admin»). No guarda llaves ni las versiona.
package fcm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/foods-platform/foods-backend/internal/modules/paydetect/application"
	"golang.org/x/oauth2/google"
)

const scope = "https://www.googleapis.com/auth/firebase.messaging"

// Client envía mensajes a un proyecto de Firebase.
type Client struct {
	http     *http.Client
	endpoint string
}

var _ application.Pusher = (*Client)(nil)

// New autentica con las credenciales por defecto (ADC) para el proyecto dado.
func New(ctx context.Context, project string) (*Client, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		return nil, fmt.Errorf("fcm: falta el proyecto")
	}
	client, err := google.DefaultClient(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("fcm: credenciales: %w", err)
	}
	return NewWithClient(client, "https://fcm.googleapis.com/v1/projects/"+project+"/messages:send"), nil
}

// NewWithClient permite probar el adaptador contra un servidor local.
func NewWithClient(client *http.Client, endpoint string) *Client {
	return &Client{http: client, endpoint: endpoint}
}

// wakeMessage es un mensaje solo de datos y prioridad alta: Android lo entrega
// aunque el teléfono esté en reposo, la app no muestra nada y el mensaje vence
// al minuto (un aviso viejo ya no sirve, y así tampoco se acumulan).
//
// Sin collapse_key a propósito: FCM limita los mensajes plegables a 20 seguidos
// por teléfono y luego 1 cada 3 minutos. Caja pide uno cada 10–30 s, así que con
// la clave el teléfono congelado solo despertaba cada 3 min y el Yape llegaba
// hasta 3 min tarde (medido 2026-11-22). El límite de los no plegables (240 por
// minuto por teléfono) queda muy por encima de ese ritmo.
func wakeMessage(token string) map[string]any {
	return map[string]any{"message": map[string]any{
		"token": token,
		"data":  map[string]string{"type": "wake"},
		"android": map[string]any{
			"priority": "HIGH",
			"ttl":      "60s",
		},
	}}
}

// Wake envía el aviso; ErrPushTokenGone si FCM ya no reconoce el token.
func (c *Client) Wake(ctx context.Context, token string) error {
	body, err := json.Marshal(wakeMessage(token))
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		_, _ = io.Copy(io.Discard, res.Body)
		return nil
	}
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 8192))
	if tokenGone(res.StatusCode, raw) {
		return application.ErrPushTokenGone
	}
	return fmt.Errorf("fcm: estado %d: %s", res.StatusCode, fcmErrorStatus(raw))
}

// fcmErrorStatus extrae el estado de error de FCM (QUOTA_EXCEEDED,
// PERMISSION_DENIED…) para el log, sin copiar el cuerpo completo.
func fcmErrorStatus(raw []byte) string {
	var payload struct {
		Error struct {
			Status string `json:"status"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &payload)
	if payload.Error.Status == "" {
		return "sin detalle"
	}
	return payload.Error.Status
}

// tokenGone reconoce los errores de FCM que significan «este token ya no sirve»:
// UNREGISTERED (404) o un token mal formado (INVALID_ARGUMENT); el mensaje es
// fijo, así que INVALID_ARGUMENT solo puede venir del token.
func tokenGone(status int, raw []byte) bool {
	var payload struct {
		Error struct {
			Status  string `json:"status"`
			Details []struct {
				ErrorCode string `json:"errorCode"`
			} `json:"details"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &payload)
	for _, detail := range payload.Error.Details {
		if detail.ErrorCode == "UNREGISTERED" || detail.ErrorCode == "INVALID_ARGUMENT" {
			return true
		}
	}
	return status == http.StatusNotFound || payload.Error.Status == "NOT_FOUND"
}
