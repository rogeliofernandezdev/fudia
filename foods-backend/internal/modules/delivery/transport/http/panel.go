package http

import (
	"encoding/json"
	"net/http"

	"github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
)

// Panel web del proveedor de Delivery Fudia. Son rutas públicas: el enlace
// privado de la ruta identifica a la central y el caso de uso lo valida.

// RotatePanelLink reemplaza el enlace del panel de una central (solo plataforma).
func (h *Handler) RotatePanelLink(w http.ResponseWriter, r *http.Request) {
	out, err := h.partners.RotatePanel(r.Context(), r.PathValue("id"))
	respond(w, http.StatusOK, out, err)
}

func (h *Handler) Panel(w http.ResponseWriter, r *http.Request) {
	out, err := h.dispatcher.Panel(r.Context(), r.PathValue("token"))
	w.Header().Set("Cache-Control", "no-store")
	respond(w, http.StatusOK, out, err)
}

func (h *Handler) PanelAction(w http.ResponseWriter, r *http.Request) {
	var in domain.PanelActionInput
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_delivery_action", "La acción no es válida.")
		return
	}
	out, err := h.dispatcher.PanelAct(r.Context(), r.PathValue("token"), in)
	respond(w, http.StatusOK, out, err)
}

func (h *Handler) PanelLocation(w http.ResponseWriter, r *http.Request) {
	var in domain.CourierLocation
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_location", "La ubicación no es válida.")
		return
	}
	if err := h.dispatcher.PanelLocation(r.Context(), r.PathValue("token"), in); err != nil {
		shared.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PanelCourier asigna o cambia el motorizado de un reparto y devuelve su enlace.
func (h *Handler) PanelCourier(w http.ResponseWriter, r *http.Request) {
	var in domain.CourierInput
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_courier_name", "Escribe el nombre del motorizado (2 a 60 caracteres).")
		return
	}
	out, err := h.dispatcher.PanelCourier(r.Context(), r.PathValue("token"), in)
	respond(w, http.StatusOK, out, err)
}

// Enlace del motorizado: ruta pública; el enlace de la ruta identifica el reparto.

func (h *Handler) Courier(w http.ResponseWriter, r *http.Request) {
	out, err := h.dispatcher.CourierView(r.Context(), r.PathValue("token"))
	w.Header().Set("Cache-Control", "no-store")
	respond(w, http.StatusOK, out, err)
}

func (h *Handler) CourierAction(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Action string `json:"action"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_delivery_action", "La acción no es válida.")
		return
	}
	out, err := h.dispatcher.CourierAct(r.Context(), r.PathValue("token"), in.Action)
	respond(w, http.StatusOK, out, err)
}

func (h *Handler) CourierLocation(w http.ResponseWriter, r *http.Request) {
	var in domain.CourierLocation
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		shared.Fail(w, http.StatusBadRequest, "invalid_location", "La ubicación no es válida.")
		return
	}
	if err := h.dispatcher.CourierLocation(r.Context(), r.PathValue("token"), in); err != nil {
		shared.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
