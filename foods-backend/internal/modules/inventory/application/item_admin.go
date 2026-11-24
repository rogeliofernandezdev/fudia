package application

import (
	"context"
	"errors"
	"strings"

	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

// ItemAdminRepository reads and changes an article's identity (company) and its
// assignment to the active location. Rules live in domain.ItemFacts helpers.
type ItemAdminRepository interface {
	// ItemFacts reads the article as seen from the active location; lock takes
	// the article row so concurrent rule checks serialize until commit.
	ItemFacts(ctx context.Context, s scope, id string, lock bool) (domain.ItemFacts, error)
	ItemLevels(ctx context.Context, s scope, id string) (string, string, error)
	ItemPresentations(ctx context.Context, s scope, id string) ([]domain.ItemPresentation, error)
	UnitTarget(ctx context.Context, s scope, code string) (domain.UnitChangeTarget, error)
	IncompatiblePresentations(ctx context.Context, s scope, id, unit string) ([]string, error)
	LocationsWithoutPermission(ctx context.Context, s scope, id, permission string) ([]string, error)
	ChangeUnit(ctx context.Context, s scope, id, from, to string) error
	SetLocationActive(ctx context.Context, s scope, id string, active bool) error
	DeleteItem(ctx context.Context, s scope, id string, productID *string) error
	RemovePresentation(ctx context.Context, s scope, id, presentationID string) (string, error)
	AssignableItems(ctx context.Context, s scope, query string, limit int) ([]domain.AssignableItem, error)
	AuditItem(ctx context.Context, s scope, action, id string, metadata map[string]any) error
}

// manageInventory is the permission a company-wide change requires in every
// other location that keeps the article.
const manageInventory = "inventory.manage"

func itemUnavailable(message string) error {
	return stockFailure(apperror.Unavailable, "inventory_unavailable", message)
}

func itemNotFound() error {
	return stockFailure(apperror.NotFound, "inventory_item_not_found", "El artículo no existe en este local.")
}

// facts loads the article and requires it to belong to the active location's list.
func (a *Stock) facts(ctx context.Context, s scope, id string, lock bool) (domain.ItemFacts, error) {
	f, err := a.repo.ItemFacts(ctx, s, id, lock)
	if errors.Is(err, ErrNotFound) {
		return f, itemNotFound()
	}
	if err != nil {
		return f, itemUnavailable("No pudimos cargar el artículo.")
	}
	if !f.AssignedHere {
		return f, itemNotFound()
	}
	return f, nil
}

// ItemDetail returns the article with the rules already evaluated for the screen.
func (a *Stock) ItemDetail(ctx context.Context, s scope, id string) (domain.ItemDetail, error) {
	var out domain.ItemDetail
	f, err := a.facts(ctx, s, id, false)
	if err != nil {
		return out, err
	}
	minimum, optimal, err := a.repo.ItemLevels(ctx, s, id)
	if err != nil {
		return out, itemUnavailable("No pudimos cargar los niveles de stock.")
	}
	presentations, err := a.repo.ItemPresentations(ctx, s, id)
	if err != nil {
		return out, itemUnavailable("No pudimos cargar las presentaciones.")
	}
	blockers := domain.DeactivateBlockers(f)
	deleteBlocker := domain.DeleteBlocker(f)
	return domain.ItemDetail{
		InventoryItemID: f.InventoryItemID, ProductID: f.ProductID, ProductName: f.ProductName, Name: f.Name,
		Unit: f.Unit, UnitName: f.UnitName, UnitCountable: f.UnitCountable, Active: f.ActiveHere,
		Quantity: domain.FormatQuantity(f.QuantityHere), MinimumStock: minimum, OptimalStock: optimal,
		HasHistory: f.HasHistory, Recipes: f.Recipes, OtherLocations: f.OtherLocations, Presentations: presentations,
		CanDelete: deleteBlocker == "", DeleteBlocker: deleteBlocker,
		CanDeactivate: f.ActiveHere && len(blockers) == 0, DeactivateBlockers: blockers,
	}, nil
}

// unitChange evaluates a base-unit change. mode is "same", "free" (no history)
// or "correction" (history keeps its numbers, read in the new unit).
func (a *Stock) unitChange(ctx context.Context, s scope, f domain.ItemFacts, unit string) (domain.UnitChangePreview, error) {
	out := domain.UnitChangePreview{Unit: unit, FromUnit: f.UnitName, Quantity: domain.FormatQuantity(f.QuantityHere), Blockers: []string{}}
	target, err := a.repo.UnitTarget(ctx, s, unit)
	if err != nil {
		return out, itemUnavailable("No pudimos validar la unidad.")
	}
	out.UnitName, out.ToUnit = target.Name, target.Name
	if unit == f.Unit {
		out.Mode, out.Allowed = "same", true
		return out, nil
	}
	out.Mode = "correction"
	if !f.HasHistory {
		out.Mode = "free"
	}
	incompatible, err := a.repo.IncompatiblePresentations(ctx, s, f.InventoryItemID, unit)
	if err != nil {
		return out, itemUnavailable("No pudimos validar las presentaciones.")
	}
	out.Blockers = domain.UnitChangeBlockers(f, target, incompatible)
	if denied, err := a.deniedLocations(ctx, s, f.InventoryItemID); err != nil {
		return out, err
	} else if denied != "" {
		out.Blockers = append(out.Blockers, denied)
	}
	out.Allowed = len(out.Blockers) == 0
	return out, nil
}

func (a *Stock) deniedLocations(ctx context.Context, s scope, id string) (string, error) {
	names, err := a.repo.LocationsWithoutPermission(ctx, s, id, manageInventory)
	if err != nil {
		return "", itemUnavailable("No pudimos validar tus permisos en los otros locales.")
	}
	if len(names) == 0 {
		return "", nil
	}
	return "También lo usa " + strings.Join(names, ", ") + ", donde no tienes permiso para gestionar inventario.", nil
}

// PreviewUnitChange lets the screen explain a unit change before saving it.
func (a *Stock) PreviewUnitChange(ctx context.Context, s scope, id, unit string) (domain.UnitChangePreview, error) {
	unit = strings.TrimSpace(unit)
	f, err := a.facts(ctx, s, id, false)
	if err != nil {
		return domain.UnitChangePreview{}, err
	}
	return a.unitChange(ctx, s, f, unit)
}

// applyIdentityChanges runs inside updateNamedItem's transaction once the
// article row is locked: it rejects company-wide changes that reach other
// locations the user cannot manage and applies the unit correction.
func (a *Stock) applyIdentityChanges(ctx context.Context, s scope, id string, renamed bool, unit *string) error {
	if !renamed && unit == nil {
		return nil
	}
	f, err := a.facts(ctx, s, id, true)
	if err != nil {
		return err
	}
	changesUnit := unit != nil && strings.TrimSpace(*unit) != f.Unit
	if !renamed && !changesUnit {
		return nil
	}
	if !changesUnit {
		denied, err := a.deniedLocations(ctx, s, id)
		if err != nil {
			return err
		}
		if denied != "" {
			return stockFailure(apperror.Forbidden, "inventory_item_other_locations", denied)
		}
		return nil
	}
	to := strings.TrimSpace(*unit)
	preview, err := a.unitChange(ctx, s, f, to)
	if err != nil {
		return err
	}
	if !preview.Allowed {
		return stockFailure(apperror.Conflict, "inventory_unit_change_blocked", strings.Join(preview.Blockers, " "))
	}
	if err = a.repo.ChangeUnit(ctx, s, id, f.Unit, to); err != nil {
		return itemUnavailable("No pudimos cambiar la unidad del artículo.")
	}
	if err = a.repo.AuditItem(ctx, s, "inventory.unit_changed", id, map[string]any{"from": f.Unit, "to": to, "mode": preview.Mode}); err != nil {
		return itemUnavailable("No pudimos registrar el cambio de unidad.")
	}
	return nil
}

// SetItemStatus deactivates the article only in the active location, or
// reactivates it there. History stays; other locations are not affected.
func (a *Stock) SetItemStatus(ctx context.Context, s scope, id string, in domain.ItemStatusInput) error {
	if in.Active == nil {
		return stockFailure(apperror.Invalid, "invalid_inventory_status", "Indica si el artículo queda activo.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return itemUnavailable("No pudimos cambiar el estado del artículo.")
	}
	defer tx.Rollback(ctx)
	f, err := a.facts(tx.Context(), s, id, true)
	if err != nil {
		return err
	}
	if *in.Active == f.ActiveHere {
		return nil
	}
	action := "inventory.item_reactivated"
	if !*in.Active {
		if blockers := domain.DeactivateBlockers(f); len(blockers) > 0 {
			return stockFailure(apperror.Conflict, "inventory_item_deactivation_blocked", strings.Join(blockers, " "))
		}
		action = "inventory.item_deactivated"
	}
	if err = a.repo.SetLocationActive(tx.Context(), s, id, *in.Active); err != nil {
		return itemUnavailable("No pudimos cambiar el estado del artículo.")
	}
	if err = a.repo.AuditItem(tx.Context(), s, action, id, map[string]any{"active": *in.Active}); err != nil {
		return itemUnavailable("No pudimos registrar el cambio de estado.")
	}
	if err = tx.Commit(ctx); err != nil {
		return itemUnavailable("No pudimos cambiar el estado del artículo.")
	}
	return nil
}

// AssignItem adds an existing company article to the active location (or
// reactivates it there) instead of creating a duplicate.
func (a *Stock) AssignItem(ctx context.Context, s scope, id string) error {
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return itemUnavailable("No pudimos agregar el artículo al local.")
	}
	defer tx.Rollback(ctx)
	f, err := a.repo.ItemFacts(tx.Context(), s, id, true)
	if errors.Is(err, ErrNotFound) {
		return stockFailure(apperror.NotFound, "inventory_item_not_found", "El artículo no existe en esta empresa.")
	}
	if err != nil {
		return itemUnavailable("No pudimos cargar el artículo.")
	}
	if f.ActiveHere {
		return nil
	}
	if err = a.repo.SetLocationActive(tx.Context(), s, id, true); err != nil {
		return itemUnavailable("No pudimos agregar el artículo al local.")
	}
	if err = a.repo.AuditItem(tx.Context(), s, "inventory.item_assigned", id, map[string]any{"reactivated": f.AssignedHere}); err != nil {
		return itemUnavailable("No pudimos registrar el alta en el local.")
	}
	if err = tx.Commit(ctx); err != nil {
		return itemUnavailable("No pudimos agregar el artículo al local.")
	}
	return nil
}

// DeleteItem removes an article permanently when no location ever used it and
// no other location keeps it.
func (a *Stock) DeleteItem(ctx context.Context, s scope, id string) error {
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return itemUnavailable("No pudimos eliminar el artículo.")
	}
	defer tx.Rollback(ctx)
	f, err := a.facts(tx.Context(), s, id, true)
	if err != nil {
		return err
	}
	if blocker := domain.DeleteBlocker(f); blocker != "" {
		return stockFailure(apperror.Conflict, "inventory_item_delete_blocked", blocker)
	}
	if err = a.repo.AuditItem(tx.Context(), s, "inventory.item_deleted", id, map[string]any{"name": f.Name, "unit": f.Unit, "productId": f.ProductID}); err != nil {
		return itemUnavailable("No pudimos registrar la eliminación.")
	}
	if err = a.repo.DeleteItem(tx.Context(), s, id, f.ProductID); err != nil {
		return itemUnavailable("No pudimos eliminar el artículo.")
	}
	if err = tx.Commit(ctx); err != nil {
		return itemUnavailable("No pudimos eliminar el artículo.")
	}
	return nil
}

// RemovePresentation quits a purchase presentation; documents that used it keep it.
func (a *Stock) RemovePresentation(ctx context.Context, s scope, id, presentationID string) (string, error) {
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return "", itemUnavailable("No pudimos quitar la presentación.")
	}
	defer tx.Rollback(ctx)
	if _, err = a.facts(tx.Context(), s, id, true); err != nil {
		return "", err
	}
	if denied, err := a.deniedLocations(tx.Context(), s, id); err != nil {
		return "", err
	} else if denied != "" {
		return "", stockFailure(apperror.Forbidden, "inventory_item_other_locations", denied)
	}
	mode, err := a.repo.RemovePresentation(tx.Context(), s, id, presentationID)
	if errors.Is(err, ErrNotFound) {
		return "", stockFailure(apperror.NotFound, "inventory_presentation_not_found", "La presentación no existe.")
	}
	if err != nil {
		return "", itemUnavailable("No pudimos quitar la presentación.")
	}
	if mode == "base" {
		return "", stockFailure(apperror.Conflict, "inventory_base_presentation", "La presentación base no se puede quitar.")
	}
	if err = a.repo.AuditItem(tx.Context(), s, "inventory.presentation_removed", id, map[string]any{"presentationId": presentationID, "mode": mode}); err != nil {
		return "", itemUnavailable("No pudimos registrar el cambio.")
	}
	if err = tx.Commit(ctx); err != nil {
		return "", itemUnavailable("No pudimos quitar la presentación.")
	}
	return mode, nil
}

// AssignableItems searches company articles the active location does not use yet.
func (a *Stock) AssignableItems(ctx context.Context, s scope, query string) ([]domain.AssignableItem, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []domain.AssignableItem{}, nil
	}
	items, err := a.repo.AssignableItems(ctx, s, query, 20)
	if err != nil {
		return nil, itemUnavailable("No pudimos buscar en los otros locales.")
	}
	return items, nil
}
