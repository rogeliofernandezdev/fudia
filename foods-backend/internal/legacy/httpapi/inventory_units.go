package httpapi

import (
	"context"
	"github.com/jackc/pgx/v5"
)

func validateInventoryUnit(ctx context.Context, tx pgx.Tx, organizationID, code string) *inventoryCatalogCreateError {
	var valid bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM inventory_units WHERE organization_id=$1 AND code=$2)`, organizationID, code).Scan(&valid); err != nil {
		return &inventoryCatalogCreateError{Status: 503, Code: "inventory_units_unavailable", Message: "No pudimos validar la unidad base."}
	}
	if !valid {
		return &inventoryCatalogCreateError{Status: 400, Code: "invalid_inventory_unit", Message: "Selecciona una unidad base registrada en tu empresa."}
	}
	return nil
}
