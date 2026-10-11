package httpapi

import (
	"net/http"
)

func (a *API) listInventoryCombinations(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.ListInventoryCombinations(w, r)
}
func (a *API) createInventoryCombination(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.CreateInventoryCombination(w, r)
}
func (a *API) savePurchasePresentation(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.SavePurchasePresentation(w, r)
}
