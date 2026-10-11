package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"strings"
)

func (a *Catalog) SaveModifiers(ctx context.Context, s scope, productID string, in domain.ModifierConfigInput) (domain.ModifierConfigView, error) {
	var result domain.ModifierConfigView
	if message := domain.ValidateModifierConfig(in); message != "" {
		return result, menuFailure(apperror.Invalid, "invalid_modifiers", message)
	}

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "modifiers_unavailable", "No pudimos iniciar la actualización.")
	}
	defer tx.Rollback(ctx)

	supported, err := a.repo.SupportsModifiers(tx.Context(), s, productID)
	if errors.Is(err, ErrNotFound) {
		return result, menuFailure(apperror.NotFound, "product_not_found", "El producto no existe.")
	}
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "modifiers_unavailable", "No pudimos validar el producto.")
	}
	if !supported {
		return result, menuFailure(apperror.Conflict, "modifiers_not_supported_for_combo", "Los combos administran sus opciones desde Menús y combos.")
	}

	if _, err = a.repo.DeleteModifiers(tx.Context(), s, productID); err != nil {
		return result, menuFailure(apperror.Unavailable, "modifiers_unavailable", "No pudimos reemplazar los modificadores.")
	}

	for groupIndex, group := range in.Groups {
		var groupID string
		err = func() error {
			var err error
			groupID, err = a.repo.CreateModifierGroup(tx.Context(), s, productID, group, groupIndex)
			return err
		}()
		if err != nil {
			return result, menuFailure(apperror.Conflict, "modifier_group_conflict", "No pudimos guardar un grupo de modificadores.")
		}
		for optionIndex, option := range group.Options {
			surcharge := strings.TrimSpace(option.Surcharge)
			if surcharge == "" {
				surcharge = "0"
			}
			if _, err = a.repo.CreateModifierOption(tx.Context(), s, groupID, option, surcharge, optionIndex); err != nil {
				return result, menuFailure(apperror.Conflict, "modifier_option_conflict", "No pudimos guardar una opción de modificador.")
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return result, menuFailure(apperror.Unavailable, "modifiers_unavailable", "No pudimos confirmar los modificadores.")
	}

	_ = a.audit.Record(ctx, s, "product.modifiers_saved", "product", productID)
	out, err := a.repo.Modifiers(ctx, s, productID)
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "modifiers_unavailable", "Los modificadores se guardaron, pero no pudimos recargarlos.")
	}
	return out, nil

}
