package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/application"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"net/http"
)

type Handler struct{ service *application.Organizations }

func New(service *application.Organizations) *Handler { return &Handler{service} }
func orgScope(r *http.Request) requestscope.Scope     { s, _ := requestscope.From(r.Context()); return s }
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
func (h *Handler) UpdateOrganization(w http.ResponseWriter, r *http.Request) {
	var in domain.OrganizationInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.Update(r.Context(), orgScope(r), in)
	respond(w, 200, out, err)
}
func (h *Handler) CreateLocation(w http.ResponseWriter, r *http.Request) {
	var in domain.LocationInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.CreateLocation(r.Context(), orgScope(r), in)
	respond(w, 201, out, err)
}
func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	var in domain.LocationInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	err := h.service.UpdateLocation(r.Context(), orgScope(r), r.PathValue("id"), in)
	respond(w, 204, nil, err)
}
func (h *Handler) CreateExchangeRate(w http.ResponseWriter, r *http.Request) {
	var in domain.ExchangeRateInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.CreateRate(r.Context(), orgScope(r), in)
	respond(w, 201, out, err)
}
func (h *Handler) UpdateOrgSettings(w http.ResponseWriter, r *http.Request) {
	var in domain.OrgSettingsInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.UpdateSettings(r.Context(), orgScope(r), in)
	respond(w, 200, out, err)
}
func (h *Handler) UpdateRestaurantSetup(w http.ResponseWriter, r *http.Request) {
	var in application.ServiceModeInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.SetServiceMode(r.Context(), orgScope(r), in)
	respond(w, 200, out, err)
}
func (h *Handler) CreateFiscalProfile(w http.ResponseWriter, r *http.Request) {
	var in domain.FiscalProfileInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.SaveProfile(r.Context(), orgScope(r), r.PathValue("id"), in, true)
	respond(w, 201, out, err)
}
func (h *Handler) UpdateFiscalProfile(w http.ResponseWriter, r *http.Request) {
	var in domain.FiscalProfileInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	_, err := h.service.SaveProfile(r.Context(), orgScope(r), r.PathValue("id"), in, false)
	respond(w, 204, nil, err)
}
func (h *Handler) GetOrganization(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Get(r.Context(), orgScope(r))
	respond(w, 200, out, err)
}
func (h *Handler) GetOrgSettings(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Settings(r.Context(), orgScope(r))
	respond(w, 200, out, err)
}
func (h *Handler) GetRestaurantSetup(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Setup(r.Context(), orgScope(r))
	respond(w, 200, out, err)
}
func (h *Handler) CompleteRestaurantSetup(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.CompleteSetup(r.Context(), orgScope(r))
	respond(w, 200, out, err)
}
func (h *Handler) DeactivateLocation(w http.ResponseWriter, r *http.Request) {
	err := h.service.DeactivateLocation(r.Context(), orgScope(r), r.PathValue("id"))
	respond(w, 204, nil, err)
}
func (h *Handler) DeactivateFiscalProfile(w http.ResponseWriter, r *http.Request) {
	err := h.service.DeactivateProfile(r.Context(), orgScope(r), r.PathValue("id"))
	respond(w, 204, nil, err)
}
func (h *Handler) DeleteExchangeRate(w http.ResponseWriter, r *http.Request) {
	err := h.service.DeleteRate(r.Context(), orgScope(r), r.PathValue("id"))
	respond(w, 204, nil, err)
}
func (h *Handler) GetPlatformOnboardingCatalogs(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.OnboardingCatalogs(r.Context(), r.URL.Query().Get("scope") == "all")
	respond(w, 200, out, err)
}
func (h *Handler) ListLocations(w http.ResponseWriter, r *http.Request) {
	page, size := shared.PageParams(r)
	out, err := h.service.Locations(r.Context(), orgScope(r), page, size)
	respond(w, 200, out, err)
}
func (h *Handler) ListFiscalProfiles(w http.ResponseWriter, r *http.Request) {
	page, size := shared.PageParams(r)
	out, err := h.service.Profiles(r.Context(), orgScope(r), page, size)
	respond(w, 200, out, err)
}
func (h *Handler) ListExchangeRates(w http.ResponseWriter, r *http.Request) {
	page, size := shared.PageParams(r)
	out, err := h.service.Rates(r.Context(), orgScope(r), page, size)
	respond(w, 200, out, err)
}
