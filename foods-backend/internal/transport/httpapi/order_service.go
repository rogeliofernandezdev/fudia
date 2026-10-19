package httpapi

import (
	"net/http"

	menucontracts "github.com/foods-platform/foods-backend/internal/modules/menu/contracts"
)

// listServiceDestinations entrega el catálogo único del módulo de carta; no se duplica aquí.
func (a *API) listServiceDestinations(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"items": menucontracts.ServiceDestinations})
}
func (a *API) closeOrderBill(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.CloseOrderBill(w, r)
}
func (a *API) deliverOrderServiceItem(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.DeliverOrderServiceItem(w, r)
}
func (a *API) appendOrderItems(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.AppendOrderItems(w, r)
}
