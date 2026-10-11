package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/billing/application"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
)

type Payments struct{ service *application.Payments }

func NewPayments(service *application.Payments) *Payments { return &Payments{service} }
func (h *Payments) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var in domain.PaymentInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_payment", "Revisa los datos del cobro.")
		return
	}
	out, err := h.service.Create(r.Context(), billingScope(r), in)
	billingResponse(w, 201, out, err)
}
func (h *Payments) RefundPayment(w http.ResponseWriter, r *http.Request) {
	var in domain.RefundInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_refund", "Ingresa un monto válido para la devolución.")
		return
	}
	err := h.service.Refund(r.Context(), billingScope(r), r.PathValue("id"), in)
	billingResponse(w, 204, nil, err)
}
func (h *Payments) CreatePaymentBatch(w http.ResponseWriter, r *http.Request) {
	var in domain.PaymentBatchInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_payment", "Revisa los datos del cobro.")
		return
	}
	out, err := h.service.CreateBatch(r.Context(), billingScope(r), in)
	billingResponse(w, 201, out, err)
}
func (h *Payments) ListPOSOrders(w http.ResponseWriter, r *http.Request) {
	page, size := shared.PageParams(r)
	q := r.URL.Query()
	out, err := h.service.List(r.Context(), billingScope(r), application.POSFilter{Query: q.Get("q"), From: q.Get("from"), To: q.Get("to"), Status: q.Get("paymentStatus"), Page: page, Size: size})
	billingResponse(w, 200, out, err)
}
func (h *Payments) GetPOSOrder(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Get(r.Context(), billingScope(r), r.PathValue("id"))
	billingResponse(w, 200, out, err)
}
