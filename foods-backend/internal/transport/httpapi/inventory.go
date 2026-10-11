package httpapi

import (
	"net/http"
)

func (a *API) listInventory(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.ListInventory(w, r)
}
func (a *API) listInventoryProducts(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.ListInventoryProducts(w, r)
}
func (a *API) createInventoryEntry(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.CreateInventoryEntry(w, r)
}
func (a *API) createInventoryAdjustment(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.CreateInventoryAdjustment(w, r)
}
func (a *API) listInventoryMovements(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.ListInventoryMovements(w, r)
}
