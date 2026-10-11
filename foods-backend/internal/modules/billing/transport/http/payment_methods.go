package http

import (
	"context"
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/billing/application"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"net/http"
)

type Methods struct{ service *application.Methods }

func NewMethods(service *application.Methods) *Methods { return &Methods{service} }
func billingScope(r *http.Request) requestscope.Scope {
	s, _ := requestscope.From(r.Context())
	return s
}
func billingResponse(w http.ResponseWriter, status int, out any, err error) {
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
func (h *Methods) Options(ctx context.Context, org, usage string) ([]map[string]string, error) {
	return h.service.Options(ctx, org, usage)
}
func (h *Methods) ListOperationalPaymentMethods(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Operational(r.Context(), billingScope(r).OrganizationID, r.URL.Query().Get("usage"))
	billingResponse(w, 200, map[string]any{"items": out}, err)
}
func (h *Methods) ListPaymentMethodsAdmin(w http.ResponseWriter, r *http.Request) {
	page, size := shared.PageParams(r)
	out, err := h.service.List(r.Context(), billingScope(r).OrganizationID, application.MethodFilter{Query: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status"), Page: page, Size: size})
	billingResponse(w, 200, out, err)
}
func (h *Methods) CreatePaymentMethod(w http.ResponseWriter, r *http.Request) { h.save(w, r, false) }
func (h *Methods) UpdatePaymentMethod(w http.ResponseWriter, r *http.Request) { h.save(w, r, true) }
func (h *Methods) save(w http.ResponseWriter, r *http.Request, updating bool) {
	var in domain.PaymentMethodInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_payment_method", "Revisa los datos del medio de pago.")
		return
	}
	out, err := h.service.Save(r.Context(), billingScope(r).OrganizationID, r.PathValue("code"), in, updating)
	status := 201
	if updating {
		status = 200
	}
	billingResponse(w, status, out, err)
}
func (h *Methods) UpdatePaymentMethodStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Active bool `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_payment_method_status", "El estado no es válido.")
		return
	}
	err := h.service.SetActive(r.Context(), billingScope(r).OrganizationID, r.PathValue("code"), in.Active)
	billingResponse(w, 204, nil, err)
}
