// Package http exposes the Fudia Pay endpoints: Locales manages the devices
// of each location (pairing QR, list, revoke) and reviews the detected payments; the
// device authenticates with its own revocable credential and the backend
// resolves its tenancy from the pairing, never from the request.
package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/paydetect/application"
	"github.com/foods-platform/foods-backend/internal/modules/paydetect/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

type Handler struct {
	devices       *application.PayDevices
	detector      *application.Detector
	notifications *application.Notifications
	now           func() time.Time
}

func New(devices *application.PayDevices, detector *application.Detector, notifications *application.Notifications) *Handler {
	return &Handler{devices, detector, notifications, func() time.Time { return time.Now() }}
}

type deviceKey struct{}

func deviceFrom(ctx context.Context) domain.DeviceSession {
	session, _ := ctx.Value(deviceKey{}).(domain.DeviceSession)
	return session
}

func bearerToken(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if token, found := strings.CutPrefix(header, "Bearer "); found {
		return strings.TrimSpace(token)
	}
	return ""
}

// DeviceAuth requires the device credential: the token is exchanged for the
// device session resolved by the backend, with its tenancy and currency.
func (h *Handler) DeviceAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		session, err := h.detector.Session(r.Context(), bearerToken(r))
		if err != nil {
			shared.WriteError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), deviceKey{}, session)))
	})
}

func respond(w http.ResponseWriter, status int, out any, err error) {
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, status, out)
}

func decode(w http.ResponseWriter, r *http.Request, in any) bool {
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384)).Decode(in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_request", "Revisa los datos enviados.")
		return false
	}
	return true
}

// locationScope resolves the location chosen in Locales ({locationId}) within
// the session organization.
func (h *Handler) locationScope(w http.ResponseWriter, r *http.Request) (requestscope.Scope, bool) {
	session, ok := shared.Scope(w, r)
	if !ok {
		return session, false
	}
	located, err := h.devices.ForLocation(r.Context(), session, r.PathValue("locationId"))
	if err != nil {
		shared.WriteError(w, err)
		return located, false
	}
	return located, true
}

// CreatePairing generates the temporary code that Locales shows as a QR.
func (h *Handler) CreatePairing(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.locationScope(w, r)
	if !ok {
		return
	}
	draft, err := h.devices.PairingCode(r.Context(), scope, scope.UserID, h.now())
	respond(w, http.StatusOK, draft, err)
}

// PairingStatus tells Locales whether the code it shows is still waiting, was
// scanned by the app, linked a device or expired.
func (h *Handler) PairingStatus(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.locationScope(w, r)
	if !ok {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	status, err := h.devices.PairingStatus(r.Context(), scope, r.PathValue("pairingId"), h.now())
	respond(w, http.StatusOK, status, err)
}

// Devices pages the devices of the location: ?status=active|revoked (empty:
// all), page from 1 and size from 10 to 100 (20 by default).
func (h *Handler) Devices(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.locationScope(w, r)
	if !ok {
		return
	}
	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	size, _ := strconv.Atoi(query.Get("size"))
	page = max(page, 1)
	if size < 10 {
		size = 20
	}
	size = min(size, 100)
	filter := domain.DeviceFilter{Status: strings.TrimSpace(query.Get("status")), Limit: size, Offset: (page - 1) * size}
	result, err := h.devices.Devices(r.Context(), scope, filter)
	respond(w, http.StatusOK, result, err)
}

// RevokeDevice disables a device credential.
func (h *Handler) RevokeDevice(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.locationScope(w, r)
	if !ok {
		return
	}
	device, err := h.devices.Revoke(r.Context(), scope, r.PathValue("deviceId"))
	respond(w, http.StatusOK, device, err)
}

// Pair exchanges the pairing code from the QR for the device credentials.
// The token travels in the response once and never again.
func (h *Handler) Pair(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code     string `json:"code"`
		Platform string `json:"platform"`
		Model    string `json:"model"`
	}
	if !decode(w, r, &in) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	activation, err := h.detector.Pair(r.Context(), in.Code, domain.PairInput{Platform: in.Platform, Model: in.Model}, h.now())
	respond(w, http.StatusOK, activation, err)
}

// ScanPairing is called by the app the moment it reads the QR, before it
// redeems the code, so Cash can show «Vinculando…».
func (h *Handler) ScanPairing(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code  string `json:"code"`
		Model string `json:"model"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := h.detector.Scan(r.Context(), in.Code, in.Model, h.now()); err != nil {
		shared.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeviceSession returns the restaurant and location resolved by the backend
// for the device, with its connection status.
func (h *Handler) DeviceSession(w http.ResponseWriter, r *http.Request) {
	session := deviceFrom(r.Context())
	shared.WriteJSON(w, http.StatusOK, map[string]any{
		"device":     map[string]any{"id": session.DeviceID, "name": session.DeviceName},
		"restaurant": map[string]any{"id": session.OrganizationID, "name": session.OrganizationName},
		"location":   map[string]any{"id": session.LocationID, "name": session.LocationName},
		"currency":   session.Currency,
	})
}

// Report registers the notifications reported by the device.
func (h *Handler) Report(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Events []domain.Event `json:"events"`
	}
	if !decode(w, r, &in) {
		return
	}
	result, err := h.detector.Report(r.Context(), deviceFrom(r.Context()), in.Events, h.now())
	respond(w, http.StatusOK, result, err)
}

// Notifications returns the detected payments for Cash.
func (h *Handler) Notifications(w http.ResponseWriter, r *http.Request) {
	scope, ok := shared.Scope(w, r)
	if !ok {
		return
	}
	query := r.URL.Query()
	filter := domain.NotificationFilter{
		Status:   strings.TrimSpace(query.Get("status")),
		Provider: strings.TrimSpace(query.Get("provider")),
		Query:    domain.NotificationQuery(query.Get("q")),
		Limit:    50,
	}
	// page y size paginan la lista de Caja (size de 10 a 100); sin ellos se
	// conservan los 50 más recientes que lee el aviso flotante.
	if page, err := strconv.Atoi(query.Get("page")); err == nil && page > 0 {
		size, sizeErr := strconv.Atoi(query.Get("size"))
		if sizeErr != nil || size < 10 {
			size = 20
		}
		if size > 100 {
			size = 100
		}
		filter.Limit, filter.Offset = size, (page-1)*size
	}
	result, err := h.notifications.Page(r.Context(), scope, filter)
	respond(w, http.StatusOK, result, err)
}

// AssociateNotification links a detected payment to an order of the same
// local, validated through the contracts of the orders module.
func (h *Handler) AssociateNotification(w http.ResponseWriter, r *http.Request) {
	scope, ok := shared.Scope(w, r)
	if !ok {
		return
	}
	var in struct {
		OrderID string `json:"orderId"`
		Method  string `json:"method"`
	}
	if !decode(w, r, &in) {
		return
	}
	notification, err := h.notifications.Associate(r.Context(), scope, r.PathValue("id"), in.OrderID, in.Method, scope.UserID, h.now())
	respond(w, http.StatusOK, notification, err)
}

// DismissNotification marks the event as reviewed and irrelevant.
func (h *Handler) DismissNotification(w http.ResponseWriter, r *http.Request) {
	scope, ok := shared.Scope(w, r)
	if !ok {
		return
	}
	notification, err := h.notifications.Dismiss(r.Context(), scope, r.PathValue("id"), scope.UserID, h.now())
	respond(w, http.StatusOK, notification, err)
}
