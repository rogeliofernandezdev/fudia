package httpapi

import (
	"net/http"
)

func (a *API) listKitchenTickets(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.ListKitchenTickets(w, r)
}
func (a *API) updateKitchenTicketStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.UpdateKitchenTicketStatus(w, r)
}
