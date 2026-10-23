package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/application"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"net/http"
)

type Stock struct{ service *application.Stock }

func NewStock(service *application.Stock) *Stock    { return &Stock{service} }
func stockScope(r *http.Request) requestscope.Scope { s, _ := requestscope.From(r.Context()); return s }
func stockResponse(w http.ResponseWriter, status int, out any, err error) {
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	if status == 204 {
		w.WriteHeader(status)
		return
	}
	shared.WriteJSON(w, status, out)
}
func stockFilter(r *http.Request) application.StockFilter {
	page, size := shared.PageParams(r)
	q := r.URL.Query()
	return application.StockFilter{Page: page, Size: size, Query: q.Get("q"), InventoryItemID: q.Get("inventoryItemId"), ProductID: q.Get("productId"), MovementType: q.Get("movementType"), SourceType: q.Get("sourceType"), From: q.Get("from"), To: q.Get("to")}
}
func (h *Stock) ListInventory(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Items(r.Context(), stockScope(r), stockFilter(r))
	stockResponse(w, 200, out, err)
}
func (h *Stock) ListInventoryProducts(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Products(r.Context(), stockScope(r), stockFilter(r))
	stockResponse(w, 200, out, err)
}
func (h *Stock) ListInventoryMovements(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Movements(r.Context(), stockScope(r), stockFilter(r))
	stockResponse(w, 200, out, err)
}
func (h *Stock) CreateInventoryEntry(w http.ResponseWriter, r *http.Request) {
	var in domain.InventoryEntryInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_inventory_entry", "Revisa los datos de la entrada.")
		return
	}
	out, err := h.service.CreateEntry(r.Context(), stockScope(r), in)
	stockResponse(w, 201, out, err)
}
func (h *Stock) CreateInventoryAdjustment(w http.ResponseWriter, r *http.Request) {
	var in domain.InventoryAdjustmentInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_inventory_adjustment", "Revisa los datos del ajuste.")
		return
	}
	out, err := h.service.CreateAdjustment(r.Context(), stockScope(r), in)
	stockResponse(w, 201, out, err)
}
func (h *Stock) CreateInventoryCombination(w http.ResponseWriter, r *http.Request) {
	var in domain.InventoryCombinationInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_inventory_combination", "Revisa los datos de la combinación.")
		return
	}
	out, err := h.service.CreateCombination(r.Context(), stockScope(r), in)
	stockResponse(w, 201, out, err)
}
func (h *Stock) SavePurchasePresentation(w http.ResponseWriter, r *http.Request) {
	var in domain.InventoryPresentationInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_inventory_presentation", "Revisa la presentación.")
		return
	}
	out, err := h.service.SavePresentation(r.Context(), stockScope(r), r.PathValue("id"), in)
	stockResponse(w, 201, out, err)
}
func (h *Stock) UpdateInventoryLocationSettings(w http.ResponseWriter, r *http.Request) {
	var in domain.LocationSettings
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_inventory_settings", "Revisa los niveles de stock.")
		return
	}
	err := h.service.LocationSettings(r.Context(), stockScope(r), r.PathValue("id"), in)
	stockResponse(w, 204, nil, err)
}
func (h *Stock) ListInventoryCombinations(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Combinations(r.Context(), stockScope(r), r.URL.Query().Get("unit"))
	stockResponse(w, 200, map[string]any{"items": out}, err)
}
func (h *Stock) ListLinkableProducts(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.LinkableProducts(r.Context(), stockScope(r), r.URL.Query().Get("q"))
	stockResponse(w, 200, map[string]any{"items": out}, err)
}
func (h *Stock) CreatePurchaseInventoryItem(w http.ResponseWriter, r *http.Request) {
	var in domain.PurchaseInventoryItemInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_purchase_item", "Revisa los datos del artículo.")
		return
	}
	out, err := h.service.CreatePurchaseItem(r.Context(), stockScope(r), in)
	stockResponse(w, 201, out, err)
}

func (h *Stock) CreateInventoryTransfer(w http.ResponseWriter, r *http.Request) {
	var in domain.TransferInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_transfer", "Revisa la transferencia.")
		return
	}
	out, err := h.service.Transfer(r.Context(), stockScope(r), in)
	status := 201
	if out["idempotent"] == true {
		status = 200
	}
	stockResponse(w, status, out, err)
}
func (h *Stock) ListInventoryTransfers(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Transfers(r.Context(), stockScope(r), stockFilter(r))
	stockResponse(w, 200, out, err)
}
