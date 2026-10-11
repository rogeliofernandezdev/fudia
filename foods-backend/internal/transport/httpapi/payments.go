package httpapi

import (
	"net/http"
)

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
