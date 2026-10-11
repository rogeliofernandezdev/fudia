package httpapi

import (
	inventorydomain "github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
)

type inventoryCombinationView = inventorydomain.InventoryCombinationView
type inventoryPresentationInput = inventorydomain.InventoryPresentationInput

var normalizeInventoryPresentation = inventorydomain.NormalizeInventoryPresentation
