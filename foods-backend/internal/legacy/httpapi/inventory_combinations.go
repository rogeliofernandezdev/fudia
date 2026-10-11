package httpapi

import (
	inventorydomain "github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"net/http"
)

type inventoryCombinationView = inventorydomain.InventoryCombinationView
type inventoryCombinationInput = inventorydomain.InventoryCombinationInput
type inventoryPresentationInput = inventorydomain.InventoryPresentationInput

func (a *API) listInventoryCombinations(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.ListInventoryCombinations(w, r)
}

func (a *API) createInventoryCombination(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.CreateInventoryCombination(w, r)
}

var normalizeInventoryPresentation = inventorydomain.NormalizeInventoryPresentation

func (a *API) savePurchasePresentation(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.SavePurchasePresentation(w, r)
}
