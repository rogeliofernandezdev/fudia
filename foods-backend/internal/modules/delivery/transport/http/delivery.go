// Package http expone Delivery Fudia: centrales, su orden y la activación por
// empresa (plataforma), enlace de cada local, despacho del pedido (caja) y la
// integración con Concierge.
package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/foods-platform/foods-backend/internal/modules/delivery/application"
	"github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

type Handler struct {
	partners   *application.Partners
	dispatcher *application.Dispatcher
}

func New(partners *application.Partners, dispatcher *application.Dispatcher) *Handler {
	return &Handler{partners: partners, dispatcher: dispatcher}
}

func requestScope(r *http.Request) requestscope.Scope {
	s, _ := requestscope.From(r.Context())
	return s
}

func respond(w http.ResponseWriter, status int, out any, err error) {
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, status, out)
}

// Plataforma: centrales con las que trabaja Fudia.

func (h *Handler) ListPartners(w http.ResponseWriter, r *http.Request) {
	out, err := h.partners.List(r.Context())
	respond(w, http.StatusOK, map[string]any{"items": out}, err)
}

func (h *Handler) CreatePartner(w http.ResponseWriter, r *http.Request) {
	var in domain.PartnerInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_delivery_partner", "Revisa los datos de la central.")
		return
	}
	out, err := h.partners.Create(r.Context(), in)
	respond(w, http.StatusCreated, out, err)
}

func (h *Handler) UpdatePartner(w http.ResponseWriter, r *http.Request) {
	var in domain.PartnerInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_delivery_partner", "Revisa los datos de la central.")
		return
	}
	out, err := h.partners.Update(r.Context(), r.PathValue("id"), in)
	if err == nil {
		_ = h.dispatcher.Sweep(r.Context())
	}
	respond(w, http.StatusOK, out, err)
}

// ReorderPartners fija el orden global de las centrales.
func (h *Handler) ReorderPartners(w http.ResponseWriter, r *http.Request) {
	var in domain.PartnerOrderInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_delivery_partner_order", "Revisa el orden de las centrales.")
		return
	}
	out, err := h.partners.Reorder(r.Context(), in)
	respond(w, http.StatusOK, map[string]any{"items": out}, err)
}

// Plataforma: Delivery Fudia por empresa (todos sus locales).

func (h *Handler) OrganizationDelivery(w http.ResponseWriter, r *http.Request) {
	out, err := h.partners.OrganizationDelivery(r.Context(), r.PathValue("id"))
	respond(w, http.StatusOK, out, err)
}

func (h *Handler) SetOrganizationDelivery(w http.ResponseWriter, r *http.Request) {
	var in domain.OrganizationDeliveryInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_delivery_settings", "Revisa el delivery de la empresa.")
		return
	}
	out, err := h.partners.SetOrganizationDelivery(r.Context(), r.PathValue("id"), in)
	respond(w, http.StatusOK, out, err)
}

// DeliveryLink devuelve el enlace/QR de delivery del local.
func (h *Handler) DeliveryLink(w http.ResponseWriter, r *http.Request) {
	out, err := h.partners.Link(r.Context(), requestScope(r), r.PathValue("id"))
	respond(w, http.StatusOK, out, err)
}

// RotateDeliveryLink genera el enlace o lo reemplaza por uno nuevo.
func (h *Handler) RotateDeliveryLink(w http.ResponseWriter, r *http.Request) {
	out, err := h.partners.RotateLink(r.Context(), requestScope(r), r.PathValue("id"))
	respond(w, http.StatusOK, out, err)
}

// Caja: despacho del pedido.

func (h *Handler) OrderDispatch(w http.ResponseWriter, r *http.Request) {
	out, err := h.dispatcher.Dispatch(r.Context(), requestScope(r), r.PathValue("id"))
	respond(w, http.StatusOK, out, err)
}

func (h *Handler) RetryDispatch(w http.ResponseWriter, r *http.Request) {
	out, err := h.dispatcher.Retry(r.Context(), requestScope(r), r.PathValue("id"))
	respond(w, http.StatusOK, out, err)
}

func (h *Handler) UseOwnDelivery(w http.ResponseWriter, r *http.Request) {
	out, err := h.dispatcher.UseOwn(r.Context(), requestScope(r), r.PathValue("id"))
	respond(w, http.StatusOK, out, err)
}

// Concierge (clave de servicio): bandeja de salida y botones de la central.

func (h *Handler) ClaimMessages(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	out, err := h.dispatcher.Claim(r.Context(), limit)
	respond(w, http.StatusOK, map[string]any{"items": out}, err)
}

func (h *Handler) MarkMessageDispatched(w http.ResponseWriter, r *http.Request) {
	if err := h.dispatcher.MarkDispatched(r.Context(), r.PathValue("id")); err != nil {
		shared.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) PartnerAction(w http.ResponseWriter, r *http.Request) {
	var in domain.ActionInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_delivery_action", "La acción no es válida.")
		return
	}
	out, err := h.dispatcher.Act(r.Context(), in)
	respond(w, http.StatusOK, out, err)
}
