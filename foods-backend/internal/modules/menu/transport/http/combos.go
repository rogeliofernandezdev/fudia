package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
	"strings"
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

// MaxOrderComboDetails acota GET /order-combo-details: una comanda real no lleva
// más menús distintos que este tope.
const MaxOrderComboDetails = 20

// GetOrderComboDetails devuelve varios menús de la comanda en una petición. La
// comanda refresca cada 10 s los menús elegidos; antes era una petición por menú.
func (h *Catalog) GetOrderComboDetails(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query()["ids"]
	if len(raw) != 1 || raw[0] == "" {
		shared.Fail(w, 400, "invalid_combo_ids", "Selecciona entre 1 y 20 menús válidos.")
		return
	}
	ids := strings.Split(raw[0], ",")
	seen := map[string]bool{}
	if len(ids) > MaxOrderComboDetails {
		shared.Fail(w, 400, "invalid_combo_ids", "Selecciona entre 1 y 20 menús válidos.")
		return
	}
	for _, id := range ids {
		if !availabilityProductID.MatchString(id) || seen[strings.ToLower(id)] {
			shared.Fail(w, 400, "invalid_combo_ids", "Selecciona entre 1 y 20 menús válidos.")
			return
		}
		seen[strings.ToLower(id)] = true
	}
	out, err := h.service.OrderComboDetails(r.Context(), menuScope(r), ids)
	respond(w, 200, map[string]any{"items": out}, err)
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
