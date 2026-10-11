package httpapi

import (
	"net/http"
)

var serviceDestinations = []map[string]string{{"value": "kitchen", "label": "Cocina"}, {"value": "bar", "label": "Barra"}, {"value": "direct", "label": "Entrega directa"}}

func (a *API) listServiceDestinations(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"items": serviceDestinations})
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
