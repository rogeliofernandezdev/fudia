package httpapi

import (
	billingapp "github.com/foods-platform/foods-backend/internal/modules/billing/application"
	billingdomain "github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"net/http"
)

type paymentView = billingdomain.PaymentView
type posOrderSummary = billingdomain.PosOrderSummary
type paymentReceiptContext = billingdomain.PaymentReceiptContext
type posOrderDetail = billingapp.POSOrderDetail

var paymentStatus = billingdomain.PaymentStatus

const netPaidSQL = `
	COALESCE((
	  SELECT sum(
	    p.amount
	    - COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)
	  )
	  FROM payments p
	  WHERE p.order_id=o.id AND p.organization_id=o.organization_id AND p.location_id=o.location_id
	),0)
`

func (a *API) completePaidOrder(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.CompletePaidOrder(w, r)
}
func (a *API) createPayment(w http.ResponseWriter, r *http.Request) {
	a.handlers.Payments.CreatePayment(w, r)
}
func (a *API) refundPayment(w http.ResponseWriter, r *http.Request) {
	a.handlers.Payments.RefundPayment(w, r)
}
func (a *API) createPaymentBatch(w http.ResponseWriter, r *http.Request) {
	a.handlers.Payments.CreatePaymentBatch(w, r)
}
func (a *API) listPOSOrders(w http.ResponseWriter, r *http.Request) {
	a.handlers.Payments.ListPOSOrders(w, r)
}
func (a *API) getPOSOrder(w http.ResponseWriter, r *http.Request) {
	a.handlers.Payments.GetPOSOrder(w, r)
}
