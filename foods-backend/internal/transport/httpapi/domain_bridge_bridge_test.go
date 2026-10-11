package httpapi

import (
	delivery "github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	inventory "github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
)

type inventoryUnitView = inventory.Unit
type inventoryUnitInput = inventory.UnitInput

var normalizeInventoryUnit = inventory.NormalizeUnit
var validateDeliveryContact = delivery.ValidateContact
