package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/application"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
)

// SpecialTaxes serves the platform calendar of legal special tax rates.
type SpecialTaxes struct{ service *application.SpecialTaxes }

func NewSpecialTaxes(service *application.SpecialTaxes) *SpecialTaxes { return &SpecialTaxes{service} }

func (h *SpecialTaxes) List(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.List(r.Context(), orgScope(r))
	respond(w, 200, out, err)
}
func (h *SpecialTaxes) CreateRate(w http.ResponseWriter, r *http.Request) {
	var in domain.SpecialTaxRateInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.CreateRate(r.Context(), orgScope(r), r.PathValue("code"), in)
	respond(w, 201, out, err)
}
func (h *SpecialTaxes) UpdateRate(w http.ResponseWriter, r *http.Request) {
	var in domain.SpecialTaxRateInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	err := h.service.UpdateRate(r.Context(), orgScope(r), r.PathValue("id"), in)
	respond(w, 204, nil, err)
}
func (h *SpecialTaxes) DeleteRate(w http.ResponseWriter, r *http.Request) {
	err := h.service.DeleteRate(r.Context(), orgScope(r), r.PathValue("id"))
	respond(w, 204, nil, err)
}
