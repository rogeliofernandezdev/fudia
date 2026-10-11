package httpapi

import (
	"context"

	"net/http"
)

type PaymentMethodsHandler interface {
	ListOperationalPaymentMethods(http.ResponseWriter, *http.Request)
	ListPaymentMethodsAdmin(http.ResponseWriter, *http.Request)
	CreatePaymentMethod(http.ResponseWriter, *http.Request)
	UpdatePaymentMethod(http.ResponseWriter, *http.Request)
	UpdatePaymentMethodStatus(http.ResponseWriter, *http.Request)
	Options(context.Context, string, string) ([]map[string]string, error)
}

func (a *API) listOperationalPaymentMethods(w http.ResponseWriter, r *http.Request) {
	a.handlers.PaymentMethods.ListOperationalPaymentMethods(w, r)
}
func (a *API) listPaymentMethodsAdmin(w http.ResponseWriter, r *http.Request) {
	a.handlers.PaymentMethods.ListPaymentMethodsAdmin(w, r)
}
func (a *API) createPaymentMethod(w http.ResponseWriter, r *http.Request) {
	a.handlers.PaymentMethods.CreatePaymentMethod(w, r)
}
func (a *API) updatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	a.handlers.PaymentMethods.UpdatePaymentMethod(w, r)
}
func (a *API) updatePaymentMethodStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.PaymentMethods.UpdatePaymentMethodStatus(w, r)
}
