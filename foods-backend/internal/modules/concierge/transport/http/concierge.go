package http

import (
	"crypto/subtle"
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/concierge/application"
	"github.com/foods-platform/foods-backend/internal/modules/concierge/domain"
	ordercontracts "github.com/foods-platform/foods-backend/internal/modules/orders/contracts"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"net/http"
	"strings"
)

type Handler struct {
	service *application.Concierge
	key     func() string
}

func New(service *application.Concierge, key func() string) *Handler { return &Handler{service, key} }
func conciergeScope(r *http.Request) requestscope.Scope {
	s, _ := requestscope.From(r.Context())
	return s
}
func respond(w http.ResponseWriter, status int, out any, err error) {
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	if status == 204 {
		w.WriteHeader(status)
		return
	}
	shared.WriteJSON(w, status, out)
}
func (h *Handler) ServiceAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := strings.TrimSpace(h.key())
		provided := strings.TrimSpace(r.Header.Get("X-Fudia-Concierge-Key"))
		if expected == "" {
			shared.Fail(w, 503, "concierge_not_configured", "Fudia Concierge no está configurado.")
			return
		}
		if len(expected) != len(provided) || subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
			shared.Fail(w, 401, "concierge_unauthorized", "Credencial de Concierge inválida.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (h *Handler) CreateConciergeOrder(w http.ResponseWriter, r *http.Request) {
	var in ordercontracts.QROrderInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_order", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.Submit(r.Context(), r.PathValue("token"), in)
	status := 200
	if out.Created {
		status = 201
	}
	respond(w, status, out.Order, err)
}
func (h *Handler) RequestConciergeBill(w http.ResponseWriter, r *http.Request) {
	var in domain.BillRequestInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_bill_request", "No pudimos procesar la solicitud de cuenta.")
		return
	}
	out, err := h.service.Bill(r.Context(), r.PathValue("token"), in)
	respond(w, 200, out, err)
}
func (h *Handler) RequestConciergeHandoff(w http.ResponseWriter, r *http.Request) {
	var in domain.HandoffInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_handoff", "Revisa la solicitud de atención.")
		return
	}
	out, err := h.service.Handoff(r.Context(), r.PathValue("token"), in)
	respond(w, 200, out, err)
}
func (h *Handler) GetConciergeHandoffStatus(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.HandoffStatus(r.Context(), r.PathValue("token"), r.PathValue("conversationId"))
	if err == nil && out == nil {
		shared.WriteJSON(w, 200, map[string]string{"status": "none"})
		return
	}
	respond(w, 200, out, err)
}
func (h *Handler) ListOperationalConciergeHandoffs(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Handoffs(r.Context(), conciergeScope(r), r.URL.Query().Get("status"))
	respond(w, 200, map[string]any{"items": out, "total": len(out)}, err)
}
func (h *Handler) ResolveOperationalConciergeHandoff(w http.ResponseWriter, r *http.Request) {
	err := h.service.ResolveHandoff(r.Context(), conciergeScope(r), r.PathValue("id"))
	respond(w, 204, nil, err)
}
func (h *Handler) GetConciergeSettings(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Settings(r.Context(), conciergeScope(r))
	respond(w, 200, out, err)
}
func (h *Handler) ListConciergeMenu(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	out, err := h.service.Menu(r.Context(), r.PathValue("token"), q.Get("q"), q.Get("productId"))
	respond(w, 200, out, err)
}
func (h *Handler) GetConciergeCombo(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Combo(r.Context(), r.PathValue("token"), r.PathValue("id"))
	respond(w, 200, out, err)
}
func (h *Handler) GetConciergeProductModifiers(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Modifiers(r.Context(), r.PathValue("token"), r.PathValue("id"))
	respond(w, 200, out, err)
}
func (h *Handler) GetConciergeDeliveryOptions(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.DeliveryOptions(r.Context(), r.PathValue("token"))
	respond(w, 200, out, err)
}

// QuoteConciergeDelivery recibe la ubicación del cliente en el cuerpo, nunca en la URL.
func (h *Handler) QuoteConciergeDelivery(w http.ResponseWriter, r *http.Request) {
	var in domain.DeliveryQuoteInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_location", "Comparte una ubicación válida para calcular el delivery.")
		return
	}
	out, err := h.service.QuoteDelivery(r.Context(), r.PathValue("token"), in)
	respond(w, 200, out, err)
}
func (h *Handler) CreateConciergeDeliveryOrder(w http.ResponseWriter, r *http.Request) {
	var in application.DeliveryOrderInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_order", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.SubmitDelivery(r.Context(), r.PathValue("token"), in)
	status := 200
	if out.Created {
		status = 201
	}
	respond(w, status, out.Order, err)
}
func (h *Handler) GetConciergeOrderStatus(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.OrderStatus(r.Context(), r.PathValue("token"), r.PathValue("orderId"), r.URL.Query().Get("conversationId"))
	respond(w, 200, out, err)
}
// QuoteAdminDelivery recibe el punto del mapa en el cuerpo, nunca en la URL.
func (h *Handler) QuoteAdminDelivery(w http.ResponseWriter, r *http.Request) {
	var in domain.DeliveryQuoteInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_location", "Marca un punto de entrega válido en el mapa.")
		return
	}
	out, err := h.service.QuoteAdminDelivery(r.Context(), conciergeScope(r), in)
	respond(w, 200, out, err)
}
func (h *Handler) GetConciergeDeliveryFees(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.DeliveryFees(r.Context(), conciergeScope(r))
	respond(w, 200, out, err)
}
func (h *Handler) SaveConciergeDeliveryFees(w http.ResponseWriter, r *http.Request) {
	var in domain.DeliveryFeesInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_delivery_fees", "Revisa los rangos de distancia y costos.")
		return
	}
	out, err := h.service.SaveDeliveryFees(r.Context(), conciergeScope(r), in)
	respond(w, 200, out, err)
}
