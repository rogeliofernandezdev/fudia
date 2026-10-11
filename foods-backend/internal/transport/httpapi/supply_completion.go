package httpapi

import (
	"net/http"
)

func (a *API) updateInventoryLocationSettings(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.UpdateInventoryLocationSettings(w, r)
}
func (a *API) createInventoryTransfer(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.CreateInventoryTransfer(w, r)
}
func (a *API) listInventoryTransfers(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.ListInventoryTransfers(w, r)
}
