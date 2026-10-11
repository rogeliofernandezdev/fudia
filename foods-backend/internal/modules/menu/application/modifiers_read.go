package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

func (a *Catalog) Modifiers(ctx context.Context, s scope, productID string) (domain.ModifierConfigView, error) {
	var result domain.ModifierConfigView
	supported, err := a.repo.SupportsModifiers(ctx, s, productID)
	if errors.Is(err, ErrNotFound) {
		return result, menuFailure(apperror.NotFound, "product_not_found", "El producto no existe.")
	}
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "modifiers_unavailable", "No pudimos validar el producto.")
	}
	if !supported {
		return result, menuFailure(apperror.Conflict, "modifiers_not_supported_for_combo", "Los combos administran sus opciones desde Menús y combos.")
	}
	out, err := a.repo.Modifiers(ctx, s, productID)
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "modifiers_unavailable", "No pudimos cargar los modificadores.")
	}
	return out, nil

}
