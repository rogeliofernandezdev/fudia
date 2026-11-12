package application

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"strings"
	"unicode/utf8"
)

// RenameInventoryProduct uses the caller's transaction, without changing prices,
// recipes, availability or the product identity referenced by historical documents.
func (a *Catalog) RenameInventoryProduct(ctx context.Context, s scope, id, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 160 {
		return menuFailure(apperror.Invalid, "invalid_inventory_item_name", "Ingresa un nombre de hasta 160 caracteres.")
	}
	count, err := a.repo.RenameInventoryProduct(ctx, s, id, name)
	if err != nil {
		return menuFailure(apperror.Unavailable, "product_unavailable", "No pudimos actualizar el nombre del producto.")
	}
	if count != 1 {
		return menuFailure(apperror.NotFound, "product_not_found", "El producto no existe en esta empresa.")
	}
	return a.audit.Record(ctx, s, "product.renamed_from_inventory", "product", id)
}
