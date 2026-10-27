package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"strings"
)

func (a *Stock) CreateCombination(ctx context.Context, s scope, in domain.InventoryCombinationInput) (domain.InventoryCombinationView, error) {
	var result domain.InventoryCombinationView
	normalized, invalid := domain.NormalizeUnit(domain.UnitInput{Code: in.Code, Name: in.Name})
	if invalid != "" {
		return result, stockFailure(apperror.Invalid, "invalid_inventory_combination", invalid)
	}
	in.Unit = strings.TrimSpace(in.Unit)
	in.Code = normalized.Code
	in.Name = normalized.Name
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_combinations_unavailable", "No pudimos guardar la combinación.")
	}
	defer tx.Rollback(ctx)
	// Serialize catalog/default changes on the base-unit row.
	var unitName string
	var unitCountable bool
	err = func() error {
		var err error
		unitName, unitCountable, err = a.repo.LockUnit(tx.Context(), s, in)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		return result, stockFailure(apperror.Invalid, "invalid_inventory_unit", "Selecciona una unidad base de tu empresa.")
	}
	if err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_combinations_unavailable", "No pudimos validar la unidad.")
	}
	if _, err = a.repo.EnsurePresentationType(tx.Context(), s, in); err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_combinations_unavailable", "No pudimos guardar la presentación.")
	}
	var name string
	if err = func() error { var err error; name, err = a.repo.PresentationTypeName(tx.Context(), s, in); return err }(); err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_combinations_unavailable", "No pudimos validar la presentación.")
	}
	if name != in.Name {
		return result, stockFailure(apperror.Conflict, "inventory_presentation_conflict", "Ese código ya corresponde a otra presentación.")
	}
	if in.IsDefault {
		if _, err = a.repo.ClearDefaultCombination(tx.Context(), s, in); err != nil {
			return result, stockFailure(apperror.Unavailable, "inventory_combinations_unavailable", "No pudimos guardar la preferencia.")
		}
	}
	var id string
	err = func() error { var err error; id, err = a.repo.SaveCombination(tx.Context(), s, in); return err }()
	if err == nil {
		_, err = a.repo.AuditCombination(tx.Context(), s, id, in)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_combinations_unavailable", "No pudimos guardar la combinación.")
	}
	return domain.InventoryCombinationView{Code: in.Code, Name: name, Unit: in.Unit, UnitName: unitName, UnitCountable: unitCountable, IsDefault: in.IsDefault}, nil

}
func (a *Stock) SavePresentation(ctx context.Context, s scope, itemID string, in domain.InventoryPresentationInput) (domain.InventoryPresentationOption, error) {
	var result domain.InventoryPresentationOption
	in, invalid := domain.NormalizeInventoryPresentation(in)
	if invalid != "" {
		return result, stockFailure(apperror.Invalid, "invalid_inventory_presentation", invalid)
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_presentations_unavailable", "No pudimos guardar la presentación.")
	}
	defer tx.Rollback(ctx)
	_, err = a.repo.LockPresentationItem(tx.Context(), s, itemID)
	if errors.Is(err, ErrNotFound) {
		return result, stockFailure(apperror.NotFound, "inventory_item_not_found", "El artículo no existe.")
	}
	if err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_presentations_unavailable", "No pudimos validar el artículo.")
	}
	id, err := a.ensurePresentation(tx.Context(), s, itemID, in.PresentationType, in.UnitsPerPresentation)
	var invalidCombo *inventoryCombinationError
	if errors.As(err, &invalidCombo) {
		return result, stockFailure(apperror.Invalid, invalidCombo.Code(), invalidCombo.Error())
	}
	if err == nil && in.IsDefault {
		err = a.repo.DefaultPresentation(tx.Context(), s, itemID, id)
	}
	if err == nil {
		_, err = a.repo.AuditPresentation(tx.Context(), s, id, itemID, in)
	}
	var out domain.InventoryPresentationOption
	if err == nil {
		err = func() error { var err error; out, err = a.repo.Presentation(tx.Context(), s, id); return err }()
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_presentations_unavailable", "No pudimos guardar la presentación.")
	}
	return out, nil

}
func (a *Stock) Combinations(ctx context.Context, s scope, unit string) ([]domain.InventoryCombinationView, error) {
	items, err := a.repo.Combinations(ctx, s, unit)
	if err != nil {
		return nil, stockFailure(apperror.Unavailable, "inventory_combinations_unavailable", "No pudimos cargar las combinaciones.")
	}
	return items, nil
}
func (a *Stock) LocationSettings(ctx context.Context, s scope, id string, in domain.LocationSettings) error {
	if !domain.ValidLocationSettings(in) {
		return stockFailure(apperror.Invalid, "invalid_inventory_settings", "Revisa los niveles de stock.")
	}
	var exists bool
	if err := func() error { var err error; exists, err = a.repo.ItemExists(ctx, s, id); return err }(); err != nil {
		return stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos validar el artículo de inventario.")
	}
	if !exists {
		return stockFailure(apperror.NotFound, "inventory_item_not_found", "El artículo de inventario no existe.")
	}
	rule, err := a.repo.PresentationRule(ctx, s, id, "unit")
	if err != nil {
		return stockFailure(apperror.Unavailable, "inventory_units_unavailable", "No pudimos validar la unidad del artículo.")
	}
	if rule.Countable && !domain.WholeLocationSettings(in) {
		return stockFailure(apperror.Invalid, "fractional_inventory_settings", "Usa cantidades enteras para los niveles de stock de esta unidad.")
	}
	if in.ReorderPoint < in.MinimumStock {
		in.ReorderPoint = in.MinimumStock
	}
	if in.OptimalStock < in.ReorderPoint {
		in.OptimalStock = in.ReorderPoint
	}
	_, err = a.repo.LocationSettings(ctx, s, id, in)
	if err != nil {
		return stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos actualizar los niveles del artículo.")
	}
	_ = a.audit.Record(ctx, s, "inventory.settings_updated", "inventory_item", id)
	return nil

}
