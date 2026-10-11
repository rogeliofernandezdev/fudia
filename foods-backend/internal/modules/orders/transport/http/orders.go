package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/orders/application"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"net/http"
)

type Handler struct{ service *application.Orders }

func New(service *application.Orders) *Handler    { return &Handler{service} }
func scopeFor(r *http.Request) requestscope.Scope { s, _ := requestscope.From(r.Context()); return s }
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
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	page, size := shared.PageParams(r)
	q := r.URL.Query()
	out, err := h.service.List(r.Context(), scopeFor(r), application.Filter{Page: page, Size: size, Query: q.Get("q"), Channel: q.Get("channel"), Status: q.Get("status")})
	respond(w, 200, out, err)
}
func (h *Handler) GetOrdersFloor(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Floor(r.Context(), scopeFor(r))
	respond(w, 200, map[string]any{"items": out}, err)
}
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Get(r.Context(), scopeFor(r), r.PathValue("id"))
	respond(w, 200, out, err)
}
func (h *Handler) CloseOrderBill(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.CloseBill(r.Context(), scopeFor(r), r.PathValue("id"))
	respond(w, 200, out, err)
}
func (h *Handler) DeliverOrderServiceItem(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.DeliverItem(r.Context(), scopeFor(r), r.PathValue("id"), r.PathValue("itemId"))
	respond(w, 200, out, err)
}
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var in domain.OrderInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_order", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.Create(r.Context(), scopeFor(r), in)
	respond(w, 201, out, err)
}
func (h *Handler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	var in domain.OrderUpdateInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_order", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.Update(r.Context(), scopeFor(r), r.PathValue("id"), in)
	respond(w, 200, out, err)
}
func (h *Handler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	var in application.OrderStatusInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_status", "El estado no es válido.")
		return
	}
	out, err := h.service.SetStatus(r.Context(), scopeFor(r), r.PathValue("id"), in)
	respond(w, 200, out, err)
}
func (h *Handler) AppendOrderItems(w http.ResponseWriter, r *http.Request) {
	var in application.AppendInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_order", "Agrega productos y un identificador de envío válido.")
		return
	}
	out, err := h.service.AppendItems(r.Context(), scopeFor(r), r.PathValue("id"), in)
	respond(w, 200, out, err)
}

func (h *Handler) ListKitchenTickets(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Kitchen(r.Context(), scopeFor(r), r.URL.Query().Get("channel"), r.URL.Query().Get("destination"))
	respond(w, 200, out, err)
}
func (h *Handler) UpdateKitchenTicketStatus(w http.ResponseWriter, r *http.Request) {
	var in application.OrderStatusInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_status", "El estado solicitado no es válido.")
		return
	}
	err := h.service.UpdateKitchen(r.Context(), scopeFor(r), r.PathValue("id"), in)
	respond(w, 204, nil, err)
}
