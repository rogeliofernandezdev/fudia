package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"strings"
	"unicode/utf8"
)

func (a *Stock) updateNamedItem(ctx context.Context, s scope, id string, in domain.LocationSettings) error {
	name := strings.TrimSpace(*in.Name)
	if name == "" || utf8.RuneCountInString(name) > 160 {
		return stockFailure(apperror.Invalid, "invalid_inventory_item_name", "Ingresa un nombre de hasta 160 caracteres.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos iniciar la edición del artículo.")
	}
	defer tx.Rollback(ctx)
	target, err := a.repo.LockItemName(tx.Context(), s, id, name)
	if errors.Is(err, ErrNotFound) {
		return stockFailure(apperror.NotFound, "inventory_item_not_found", "El artículo no existe en esta empresa o local.")
	}
	if errors.Is(err, ErrNameTaken) {
		return stockFailure(apperror.Conflict, "inventory_item_name_taken", "Ya existe un artículo o producto con ese nombre. Elige otro nombre.")
	}
	if err != nil {
		return stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos validar el nombre del artículo.")
	}
	if target.ProductID != nil && target.Name != name {
		if a.productNames == nil {
			return stockFailure(apperror.Unavailable, "product_unavailable", "No pudimos actualizar el nombre del producto.")
		}
		if err = a.productNames.RenameInventoryProduct(tx.Context(), s, *target.ProductID, name); err != nil {
			return err
		}
	}
	count, err := a.repo.RenameItem(tx.Context(), s, id, name)
	if err != nil || count != 1 {
		return stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos actualizar el nombre del artículo.")
	}
	if _, err = a.repo.LocationSettings(tx.Context(), s, id, in); err != nil {
		return stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos guardar los niveles de stock del artículo.")
	}
	if err = a.audit.Record(tx.Context(), s, "inventory.settings_updated", "inventory_item", id); err != nil {
		return stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos registrar la edición del artículo.")
	}
	if err = tx.Commit(ctx); err != nil {
		return stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos guardar el artículo.")
	}
	return nil
}
