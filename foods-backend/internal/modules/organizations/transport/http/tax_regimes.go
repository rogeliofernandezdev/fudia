package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/application"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
)

type TaxRegimes struct{ service *application.TaxRegimes }

func NewTaxRegimes(service *application.TaxRegimes) *TaxRegimes { return &TaxRegimes{service} }

func (h *TaxRegimes) List(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.List(r.Context(), orgScope(r), r.PathValue("id"), r.URL.Query().Get("date"))
	respond(w, 200, out, err)
}
func (h *TaxRegimes) Create(w http.ResponseWriter, r *http.Request) {
	var in domain.TaxRegimeInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.Create(r.Context(), orgScope(r), r.PathValue("id"), in)
	respond(w, 201, out, err)
}
func (h *TaxRegimes) Update(w http.ResponseWriter, r *http.Request) {
	var in domain.TaxRegimeInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.Update(r.Context(), orgScope(r), r.PathValue("id"), r.PathValue("regimeId"), in)
	respond(w, 200, out, err)
}
func (h *TaxRegimes) Delete(w http.ResponseWriter, r *http.Request) {
	err := h.service.Delete(r.Context(), orgScope(r), r.PathValue("id"), r.PathValue("regimeId"))
	respond(w, 204, nil, err)
}
