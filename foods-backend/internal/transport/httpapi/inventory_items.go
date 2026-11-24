package httpapi

import "net/http"

func (a *API) getInventoryItem(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.GetInventoryItem(w, r)
}
func (a *API) previewInventoryUnitChange(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.PreviewInventoryUnitChange(w, r)
}
func (a *API) updateInventoryItemStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.UpdateInventoryItemStatus(w, r)
}
func (a *API) listAssignableItems(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.ListAssignableItems(w, r)
}
func (a *API) assignInventoryItem(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.AssignInventoryItem(w, r)
}
func (a *API) listPurchaseInventoryItems(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.ListPurchaseInventoryItems(w, r)
}
