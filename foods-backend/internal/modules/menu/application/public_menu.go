package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"strings"
)

func (a *Catalog) PublicMenu(ctx context.Context, s scope, q, productID string) ([]domain.PublicMenuItem, error) {
	day, err := a.repo.BusinessDate(ctx, s)
	if err != nil {
		return nil, menuFailure(apperror.Unavailable, "concierge_unavailable", "No pudimos determinar la fecha operativa.")
	}
	raw, err := a.repo.PublicMenu(ctx, s, strings.TrimSpace(q), strings.TrimSpace(productID), day)
	if err != nil {
		return nil, menuFailure(apperror.Unavailable, "concierge_unavailable", "No pudimos cargar la carta.")
	}
	items := []domain.PublicMenuItem{}
	for _, v := range raw {
		item := v.Item
		item.Status = domain.ResolveAvailability(v.Availability, day.Format("2006-01-02")).Status
		if item.IsCombo && item.Status == "available" {
			ok, err := a.repo.ComboAvailable(ctx, s, item.ProductID, day)
			if err != nil {
				return nil, menuFailure(apperror.Unavailable, "concierge_unavailable", "No pudimos validar un menú.")
			}
			if !ok {
				item.Status = "sold_out"
			}
		}
		items = append(items, item)
	}
	return items, nil
}
func (a *Catalog) PublicCombo(ctx context.Context, s scope, id string) (domain.PublicComboDetail, error) {
	var result domain.PublicComboDetail
	id = strings.TrimSpace(id)
	day, err := a.repo.BusinessDate(ctx, s)
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "combo_unavailable", "No pudimos determinar la fecha operativa.")
	}

	var out domain.PublicComboDetail
	var available bool
	err = func() error { var err error; out, available, err = a.repo.PublicCombo(ctx, s, id, day); return err }()
	if errors.Is(err, ErrNotFound) {
		return result, menuFailure(apperror.NotFound, "combo_not_found", "El menú o combo no existe.")
	}
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "combo_unavailable", "No pudimos cargar el menú.")
	}
	if !available {
		return result, menuFailure(apperror.Conflict, "combo_unavailable", "Este menú o combo no está disponible en este momento.")
	}
	viable, err := a.repo.ComboAvailable(ctx, s, id, day)
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "combo_unavailable", "No pudimos validar las alternativas del menú.")
	}
	if !viable {
		return result, menuFailure(apperror.Conflict, "combo_unavailable", "Este menú no tiene suficientes alternativas disponibles.")
	}
	out.ID = id
	out.Groups, err = a.repo.PublicComboGroups(ctx, s, id, day)
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "combo_unavailable", "No pudimos cargar los grupos del menú.")
	}
	return out, nil
}
func (a *Catalog) PublicModifiers(ctx context.Context, s scope, productID string) (domain.ModifierConfigView, error) {
	var result domain.ModifierConfigView
	productID = strings.TrimSpace(productID)
	var exists, isCombo bool
	if err := func() error {
		var err error
		exists, isCombo, err = a.repo.PublicModifierProduct(ctx, s, productID)
		return err
	}(); err != nil {
		return result, menuFailure(apperror.Unavailable, "modifiers_unavailable", "No pudimos validar el producto.")
	}
	if !exists {
		return result, menuFailure(apperror.NotFound, "product_not_found", "El producto no existe o está inactivo.")
	}
	if isCombo {
		return result, menuFailure(apperror.Conflict, "modifiers_not_supported_for_combo", "Este producto administra sus opciones como menú o combo.")
	}

	out, err := a.repo.Modifiers(ctx, s, productID)
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "modifiers_unavailable", "No pudimos cargar los modificadores.")
	}
	return out, nil

}
