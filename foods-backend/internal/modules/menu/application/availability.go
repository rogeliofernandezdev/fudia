package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"strings"
)

func (a *Catalog) Availability(ctx context.Context, s scope, f CatalogFilter, excludeCombos bool) (AvailabilityPage, error) {
	out := AvailabilityPage{Items: []domain.AvailabilityItem{}, Page: f.Page, PageSize: f.Size}
	day, err := a.repo.BusinessDate(ctx, s)
	if err != nil {
		return out, menuFailure(apperror.Unavailable, "availability_unavailable", "No pudimos determinar la fecha operativa del local.")
	}
	out.BusinessDate = day.Format("2006-01-02")
	raw, total, err := a.repo.Availability(ctx, s, f, day, excludeCombos)
	if err != nil {
		return out, menuFailure(apperror.Unavailable, "availability_unavailable", "No pudimos cargar la disponibilidad de la carta.")
	}
	out.Total = total
	for _, snapshot := range raw {
		item := domain.ResolveAvailability(snapshot, out.BusinessDate)
		if snapshot.IsCombo && item.Status == "available" {
			available, err := a.repo.ComboAvailable(ctx, s, item.ProductID, day)
			if err != nil {
				return out, menuFailure(apperror.Unavailable, "availability_unavailable", "No pudimos validar las opciones de un menú.")
			}
			if !available {
				item.Status, item.Source = "sold_out", "combo_components"
			}
		}
		out.Items = append(out.Items, item)
	}
	ids := make([]string, 0, len(out.Items))
	for _, p := range out.Items {
		ids = append(ids, p.ProductID)
	}
	views, err := a.productImageViews(ctx, s, ids, "thumbnail", false)
	if err != nil {
		return out, err
	}
	for i := range out.Items {
		if v, ok := views[out.Items[i].ProductID]; ok {
			out.Items[i].ImageView = v.ImageView
			out.Items[i].ImageURL = v.ImageURL
		}
	}
	return out, nil
}
func (a *Catalog) UpdateAvailability(ctx context.Context, s scope, productID string, in domain.AvailabilityInput) error {
	day, err := a.repo.BusinessDate(ctx, s)
	if err != nil {
		return menuFailure(apperror.Unavailable, "availability_unavailable", "No pudimos determinar la fecha operativa del local.")
	}
	if in.Status != "available" && in.Status != "sold_out" {
		return menuFailure(apperror.Invalid, "invalid_availability", "Selecciona disponible o agotado.")
	}
	if in.PortionQuantity != nil && *in.PortionQuantity < 0 {
		return menuFailure(apperror.Invalid, "invalid_availability", "La cantidad de porciones no puede ser negativa.")
	}
	in.Note = strings.TrimSpace(in.Note)
	in.Reason = strings.TrimSpace(in.Reason)
	if in.Reason == "" || len([]rune(in.Reason)) > 240 {
		return menuFailure(apperror.Invalid, "availability_reason_required", "Indica el motivo del cambio, de hasta 240 caracteres.")
	}
	if len([]rune(in.Note)) > 240 {
		return menuFailure(apperror.Invalid, "invalid_availability", "La nota no puede superar 240 caracteres.")
	}

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return menuFailure(apperror.Unavailable, "availability_unavailable", "No pudimos iniciar la actualización de disponibilidad.")
	}
	defer tx.Rollback(ctx)

	var quantityControl string
	err = func() error {
		var err error
		quantityControl, err = a.repo.LockAvailabilityProduct(tx.Context(), s, productID)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		return menuFailure(apperror.NotFound, "product_not_found", "El producto no existe o está inactivo.")
	}
	if err != nil {
		return menuFailure(apperror.Unavailable, "availability_unavailable", "No pudimos validar el producto.")
	}

	var currentPortionQuantity *int
	var soldQuantity int
	currentStatus := "available"
	currentNote := ""
	err = func() error {
		var err error
		currentPortionQuantity, soldQuantity, currentStatus, currentNote, err = a.repo.LockDailyAvailability(tx.Context(), s, productID, day)
		return err
	}()
	hasDailyAvailability := true
	if errors.Is(err, ErrNotFound) {
		hasDailyAvailability = false
		currentStatus = "available"
		currentNote = ""
		soldQuantity = 0
		err = nil
	}
	if err != nil {
		return menuFailure(apperror.Unavailable, "availability_unavailable", "No pudimos validar la disponibilidad actual.")
	}

	if quantityControl != "portions" {
		in.PortionQuantity = nil
	} else {
		if in.PortionQuantity == nil && hasDailyAvailability {
			in.PortionQuantity = currentPortionQuantity
		}
		if in.PortionQuantity != nil && *in.PortionQuantity < soldQuantity {
			return menuFailure(apperror.Conflict, "portion_quantity_below_sold", "El cupo de hoy no puede ser menor que las porciones ya vendidas.")
		}
		if in.Status != "sold_out" && (in.PortionQuantity == nil || *in.PortionQuantity < 1) {
			return menuFailure(apperror.Invalid, "invalid_availability", "Ingresa cuántas porciones están disponibles hoy.")
		}
	}
	if currentStatus == "low" {
		currentStatus = "available"
	}
	quantityChanged := (currentPortionQuantity == nil) != (in.PortionQuantity == nil)
	if currentPortionQuantity != nil && in.PortionQuantity != nil {
		quantityChanged = *currentPortionQuantity != *in.PortionQuantity
	}
	if hasDailyAvailability && !quantityChanged && currentStatus == in.Status && currentNote == in.Note {
		return nil
	}

	_, err = a.repo.UpdateAvailability(tx.Context(), s, productID, day, in)
	if err != nil {
		return menuFailure(apperror.Unavailable, "availability_unavailable", "No pudimos actualizar la disponibilidad.")
	}
	_, err = a.repo.AuditAvailability(tx.Context(), s, productID, day, in, currentPortionQuantity, soldQuantity, currentStatus)
	if err != nil {
		return menuFailure(apperror.Unavailable, "availability_unavailable", "No pudimos guardar el historial del cambio. La disponibilidad no se modificó.")
	}
	if err = tx.Commit(ctx); err != nil {
		return menuFailure(apperror.Unavailable, "availability_unavailable", "No pudimos confirmar la actualización de disponibilidad.")
	}
	return nil

}
func (a *Catalog) AvailabilityHistory(ctx context.Context, s scope, productID string, f CatalogFilter) (AvailabilityHistoryPage, error) {
	var out AvailabilityHistoryPage
	tx, err := a.work.Begin(ctx, unitofwork.Options{ReadOnly: true, RepeatableRead: true})
	if err != nil {
		return out, menuFailure(apperror.Unavailable, "availability_history_unavailable", "No pudimos consultar el historial.")
	}
	defer tx.Rollback(ctx)
	exists, err := a.repo.ProductExists(tx.Context(), s, productID)
	if err != nil {
		return out, menuFailure(apperror.Unavailable, "availability_history_unavailable", "No pudimos validar el producto.")
	}
	if !exists {
		return out, menuFailure(apperror.NotFound, "product_not_found", "El producto no existe.")
	}
	out, err = a.repo.AvailabilityHistory(tx.Context(), s, productID, f)
	if err != nil {
		return out, menuFailure(apperror.Unavailable, "availability_history_unavailable", "No pudimos cargar los cambios.")
	}
	if err = tx.Commit(ctx); err != nil {
		return out, menuFailure(apperror.Unavailable, "availability_history_unavailable", "No pudimos confirmar la lectura del historial.")
	}
	return out, nil
}

type AvailabilityPage struct {
	Items        []domain.AvailabilityItem `json:"items"`
	BusinessDate string                    `json:"businessDate"`
	Total        int                       `json:"total"`
	Page         int                       `json:"page"`
	PageSize     int                       `json:"pageSize"`
}
type AvailabilityHistoryPage struct {
	Items    []domain.AvailabilityHistoryItem `json:"items"`
	Total    int                              `json:"total"`
	Page     int                              `json:"page"`
	PageSize int                              `json:"pageSize"`
}
