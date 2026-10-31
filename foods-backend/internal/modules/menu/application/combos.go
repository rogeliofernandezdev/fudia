package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

func (a *Catalog) SetComboActive(ctx context.Context, s scope, id string, active bool) error {
	tag, err := a.repo.SetComboActive(ctx, s, id, struct{ Active *bool }{&active})
	if err != nil {
		return menuFailure(apperror.Unavailable, "combo_status_unavailable", "No pudimos actualizar el estado del menú.")
	}
	if tag == 0 {
		return menuFailure(apperror.NotFound, "combo_not_found", "El menú o combo no existe.")
	}
	action := "combo.deactivated"
	if active {
		action = "combo.activated"
	}
	_ = a.audit.Record(ctx, s, action, "product", id)
	return nil

}
func (a *Catalog) SaveCombo(ctx context.Context, s scope, id string, in domain.ComboInput) (map[string]string, error) {
	var result map[string]string
	if message := domain.ValidateCombo(in); message != "" {
		return result, menuFailure(apperror.Invalid, "invalid_combo", message)
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "combo_unavailable", "No pudimos iniciar el registro.")
	}
	defer tx.Rollback(ctx)

	if id == "" {
		sku, skuErr := a.nextSKU(tx.Context(), s)
		if skuErr != nil {
			return result, menuFailure(apperror.Unavailable, "combo_unavailable", "No pudimos generar el código.")
		}
		err = func() error { var err error; id, err = a.repo.CreateComboProduct(tx.Context(), s, in, sku); return err }()
		if err == nil {
			_, err = a.repo.CreateCombo(tx.Context(), s, id)
		}
	} else {
		tag, updateErr := a.repo.UpdateCombo(tx.Context(), s, id, in)
		err = updateErr
		if err == nil && tag == 0 {
			err = ErrNotFound
		}
		if err == nil {
			_, err = a.repo.DeleteComboGroups(tx.Context(), s, id)
		}
	}
	if errors.Is(err, ErrNotFound) {
		return result, menuFailure(apperror.NotFound, "combo_not_found", "El menú o combo no existe.")
	}
	if err != nil {
		return result, menuFailure(apperror.Conflict, "combo_conflict", "No pudimos guardar el menú o combo.")
	}
	for groupIndex, group := range in.Groups {
		var groupID string
		err = func() error {
			var err error
			groupID, err = a.repo.CreateComboGroup(tx.Context(), s, id, group, groupIndex)
			return err
		}()
		if err != nil {
			return result, menuFailure(apperror.Conflict, "combo_conflict", "No pudimos guardar los grupos.")
		}
		for optionIndex, option := range group.Options {
			tag, optionErr := a.repo.CreateComboOption(tx.Context(), s, id, groupID, option, optionIndex)
			if optionErr != nil {
				return result, menuFailure(apperror.Conflict, "combo_option_conflict", "No pudimos guardar una de las alternativas.")
			}
			if tag == 0 {
				return result, menuFailure(apperror.Conflict, "combo_option_invalid", "Una alternativa no está disponible o su cantidad reservada supera las existencias del local.")
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return result, menuFailure(apperror.Unavailable, "combo_unavailable", "No pudimos confirmar el registro.")
	}
	_ = a.audit.Record(ctx, s, "combo.saved", "product", id)
	return map[string]string{"id": id}, nil

}
func (a *Catalog) Combos(ctx context.Context, s scope, f CatalogFilter) (ComboPage, error) {
	var out ComboPage
	out, err := a.repo.Combos(ctx, s, f)
	if err != nil {
		return out, menuFailure(apperror.Unavailable, "combos_unavailable", "No pudimos cargar los menús y combos.")
	}
	return out, nil
}
func (a *Catalog) OrderCombos(ctx context.Context, s scope, f CatalogFilter) (OrderComboPage, error) {
	var out OrderComboPage
	day, err := a.repo.BusinessDate(ctx, s)
	if err != nil {
		return out, menuFailure(apperror.Unavailable, "combos_unavailable", "No pudimos determinar la fecha operativa del local.")
	}
	out, err = a.repo.OrderCombos(ctx, s, f, day)
	if err != nil {
		return out, menuFailure(apperror.Unavailable, "combos_unavailable", "No pudimos cargar los menús y combos.")
	}
	ids := make([]string, 0, len(out.Items))
	for _, p := range out.Items {
		ids = append(ids, p.ID)
	}
	views, err := a.productImageViews(ctx, s, ids, "thumbnail", false)
	if err != nil {
		return out, err
	}
	for i := range out.Items {
		if v, ok := views[out.Items[i].ID]; ok {
			out.Items[i].ImageView = v.ImageView
			out.Items[i].ImageURL = v.ImageURL
		}
	}
	return out, nil
}
func (a *Catalog) Combo(ctx context.Context, s scope, id string) (domain.Combo, error) {
	var out domain.Combo
	out, err := a.repo.Combo(ctx, s, id)
	if errors.Is(err, ErrNotFound) {
		return out, menuFailure(apperror.NotFound, "combo_not_found", "El menú o combo no existe.")
	}
	if err != nil {
		return out, menuFailure(apperror.Unavailable, "combo_unavailable", "No pudimos cargar el menú.")
	}
	return out, nil
}
func (a *Catalog) OrderCombo(ctx context.Context, s scope, id string) (domain.OrderCombo, error) {
	var out domain.OrderCombo
	day, err := a.repo.BusinessDate(ctx, s)
	if err != nil {
		return out, menuFailure(apperror.Unavailable, "combo_unavailable", "No pudimos determinar la fecha operativa del local.")
	}
	out, available, err := a.repo.OrderCombo(ctx, s, id, day)
	if errors.Is(err, ErrNotFound) {
		return out, menuFailure(apperror.NotFound, "combo_not_found", "El menú o combo no existe.")
	}
	if err != nil {
		return out, menuFailure(apperror.Unavailable, "combo_unavailable", "No pudimos cargar el menú.")
	}
	if !available {
		return out, menuFailure(apperror.Conflict, "combo_unavailable", "Este menú o combo no está disponible en este momento.")
	}
	// Resolve physical availability once per batch, across every selected group.
	ids := []string{}
	seen := map[string]bool{}
	for _, group := range out.Groups {
		for _, option := range group.Options {
			if !seen[option.ProductID] {
				ids = append(ids, option.ProductID)
				seen[option.ProductID] = true
			}
		}
	}
	physical := map[string]domain.AvailabilityItem{}
	for offset := 0; offset < len(ids); offset += 100 {
		end := offset + 100
		if end > len(ids) {
			end = len(ids)
		}
		snapshots, _, readErr := a.repo.Availability(ctx, s, CatalogFilter{Page: 1, Size: 100, ProductIDs: ids[offset:end]}, day, false)
		if readErr != nil {
			return out, menuFailure(apperror.Unavailable, "combo_unavailable", "No pudimos comprobar las existencias de las opciones.")
		}
		for _, snapshot := range snapshots {
			item := domain.ResolveAvailability(snapshot, day.Format("2006-01-02"))
			physical[item.ProductID] = item
		}
	}
	for gi := range out.Groups {
		for oi := range out.Groups[gi].Options {
			option := &out.Groups[gi].Options[oi]
			item, found := physical[option.ProductID]
			option.Remaining = item.Remaining
			option.Available = option.Available && found && (item.Status == "available" || item.Status == "low") && (option.QuotaRemaining == nil || *option.QuotaRemaining >= 1) && (option.Remaining == nil || *option.Remaining >= 1)
		}
	}
	views, err := a.productImageViews(ctx, s, []string{id}, "medium", false)
	if err != nil {
		return out, err
	}
	if v, ok := views[id]; ok {
		out.ImageView = v.ImageView
		out.ImageURL = v.ImageURL
	}
	return out, nil
}

type ComboPage struct {
	Items    []domain.ComboSummary `json:"items"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}
type OrderComboPage struct {
	Items    []domain.OrderComboSummary `json:"items"`
	Total    int                        `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"pageSize"`
}
