package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
)

func (h *Catalog) ListCombos(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Combos(r.Context(), menuScope(r), catalogFilter(r))
	respond(w, 200, out, err)
}
func (h *Catalog) ListOrderCombos(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.OrderCombos(r.Context(), menuScope(r), catalogFilter(r))
	respond(w, 200, out, err)
}
func (h *Catalog) GetCombo(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Combo(r.Context(), menuScope(r), r.PathValue("id"))
	respond(w, 200, out, err)
}
func (h *Catalog) GetOrderCombo(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.OrderCombo(r.Context(), menuScope(r), r.PathValue("id"))
	respond(w, 200, out, err)
}
func (h *Catalog) SaveCombo(w http.ResponseWriter, r *http.Request) {
	var in domain.ComboInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_combo", "La información del menú no es válida.")
		return
	}
	out, err := h.service.SaveCombo(r.Context(), menuScope(r), r.PathValue("id"), in)
	respond(w, 201, out, err)
}
func (h *Catalog) UpdateComboStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Active *bool `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.Active == nil {
		shared.Fail(w, 400, "invalid_combo_status", "El estado del menú no es válido.")
		return
	}
	err := h.service.SetComboActive(r.Context(), menuScope(r), r.PathValue("id"), *in.Active)
	respond(w, 204, nil, err)
}
