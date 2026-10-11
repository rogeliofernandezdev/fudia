package httpapi

import (
	"net/http"
)

func (a *API) listOrders(w http.ResponseWriter, r *http.Request) { a.handlers.Orders.ListOrders(w, r) }
func (a *API) getOrdersFloor(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.GetOrdersFloor(w, r)
}
func (a *API) getOrder(w http.ResponseWriter, r *http.Request) { a.handlers.Orders.GetOrder(w, r) }
func (a *API) createOrder(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.CreateOrder(w, r)
}
func (a *API) updateOrder(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.UpdateOrder(w, r)
}
func (a *API) updateOrderStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.UpdateOrderStatus(w, r)
}
