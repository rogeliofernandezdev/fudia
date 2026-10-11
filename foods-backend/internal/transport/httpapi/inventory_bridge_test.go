package httpapi

import (
	"context"
	"errors"
	inventoryapp "github.com/foods-platform/foods-backend/internal/modules/inventory/application"
	inventorydomain "github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	inventorypg "github.com/foods-platform/foods-backend/internal/modules/inventory/infrastructure/postgres"
	orderdomain "github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/database"

	"github.com/jackc/pgx/v5"
)

type inventoryProductOption = inventorydomain.InventoryProductOption
type inventoryNewProductInput = inventorydomain.InventoryNewProductInput
type inventoryEntryInput = inventorydomain.InventoryEntryInput
type inventoryMovementView = inventorydomain.InventoryMovementView
type inventoryAdjustmentInput = inventorydomain.InventoryAdjustmentInput

var normalizeInventoryEntry = inventorydomain.NormalizeInventoryEntry
var normalizeInventoryAdjustment = inventorydomain.NormalizeInventoryAdjustment

var preparedQuantityUsage = orderdomain.PreparedQuantityUsage
var quantityUsageDelta = orderdomain.QuantityUsageDelta

// Exercise the production inventory use case inside the fixture's transaction.
func (a *integrationAPI) applyOrderQuantityDelta(ctx context.Context, tx pgx.Tx, s scope, orderID string, delta map[string]float64, movementType string) *orderPreparationError {
	service := inventoryapp.NewStock(inventorypg.NewStock(a.db), database.NewUnitOfWork(a.db), nil)
	err := service.ApplyOrderQuantityDelta(database.WithTransaction(ctx, a.db, tx), s, orderID, delta, movementType)
	if err == nil {
		return nil
	}
	var failure *apperror.Error
	if errors.As(err, &failure) {
		status := 503
		if failure.Kind == apperror.Invalid {
			status = 400
		} else if failure.Kind == apperror.Conflict {
			status = 409
		} else if failure.Kind == apperror.NotFound {
			status = 404
		}
		return &orderPreparationError{Status: status, Code: failure.Code, Message: failure.Message}
	}
	return &orderPreparationError{Status: 500, Code: "unexpected_error", Message: err.Error()}
}
