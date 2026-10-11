package application

import (
	"context"
	"errors"
	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	"github.com/foods-platform/foods-backend/internal/modules/service/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"strings"
)

var ErrNotFound = errors.New("service record not found")
var ErrConflict = errors.New("service record conflict")

type scope = requestscope.Scope
type zone = domain.Zone
type table = domain.Table
type FloorFilter struct {
	Query, Status string
	Page, Size    int
}
type ZonePage struct {
	Items    []domain.Zone `json:"items"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}
type TablePage struct {
	Items    []domain.Table `json:"items"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}
type FloorRepository interface {
	Zones(context.Context, scope, FloorFilter) (ZonePage, error)
	CreateZone(context.Context, scope, string, int) (domain.Zone, error)
	LockZone(context.Context, scope, string) (domain.Zone, error)
	ZoneInUse(context.Context, scope, string) (bool, error)
	UpdateZone(context.Context, scope, string, string, int, bool) (domain.Zone, error)
	RenameTableZone(context.Context, scope, string, string) (int64, error)
	DeactivateZone(context.Context, scope, string) (int64, error)
	Tables(context.Context, scope, FloorFilter) (TablePage, error)
	LockTable(context.Context, scope, string) (bool, error)
	ActiveZone(context.Context, scope, string) error
	TableOccupied(context.Context, scope, string) (bool, error)
	CreateTable(context.Context, scope, string, int, string, bool) (domain.Table, error)
	UpdateTable(context.Context, scope, string, string, int, string, bool, bool) (domain.Table, error)
	DeactivateTable(context.Context, scope, string) (int64, error)
	RegenerateQR(context.Context, scope, string) (domain.Table, error)
	TableQR(context.Context, string) (domain.TableQR, error)
	Batch(context.Context, scope, []domain.TableBatchItem) (domain.TableBatchResult, bool, error)
}
type Floor struct {
	repo  FloorRepository
	work  unitofwork.Manager
	audit audit.Repository
}

func NewFloor(repo FloorRepository, work unitofwork.Manager, audit audit.Repository) *Floor {
	return &Floor{repo, work, audit}
}
func serviceFailure(kind apperror.Kind, code, message string) error {
	return apperror.New(kind, code, message)
}
func (a *Floor) Zones(ctx context.Context, s scope, f FloorFilter) (ZonePage, error) {
	f.Status = strings.TrimSpace(f.Status)
	out, err := a.repo.Zones(ctx, s, f)
	if err != nil {
		return out, serviceFailure(apperror.Unavailable, "zones_unavailable", "No pudimos cargar las zonas.")
	}
	return out, nil
}
func (a *Floor) Tables(ctx context.Context, s scope, f FloorFilter) (TablePage, error) {
	f.Query = strings.TrimSpace(f.Query)
	out, err := a.repo.Tables(ctx, s, f)
	if err != nil {
		return out, serviceFailure(apperror.Unavailable, "tables_unavailable", "No pudimos cargar las mesas.")
	}
	return out, nil
}
func (a *Floor) CreateZone(ctx context.Context, s scope, in domain.ZoneInput) (domain.Zone, error) {
	var result domain.Zone
	if strings.TrimSpace(in.Name) == "" {
		return result, serviceFailure(apperror.Invalid, "invalid_request", "El nombre de la zona es obligatorio.")
	}
	sortOrder := 0
	if in.SortOrder != nil {
		sortOrder = *in.SortOrder
	}
	var z zone
	err := func() error {
		v, e := a.repo.CreateZone(ctx, s, strings.TrimSpace(in.Name), sortOrder)
		z = v
		return e
	}()
	if err != nil {
		return result, serviceFailure(apperror.Conflict, "zone_conflict", "Ya existe una zona con ese nombre.")
	}
	_ = a.audit.Record(ctx, s, "zone.created", "zone", z.ID)
	return z, nil

}
func (a *Floor) UpdateZone(ctx context.Context, s scope, id string, in domain.ZoneInput) (domain.Zone, error) {
	var result domain.Zone
	if strings.TrimSpace(in.Name) == "" {
		return result, serviceFailure(apperror.Invalid, "invalid_request", "El nombre de la zona es obligatorio.")
	}
	sortOrder := 0
	if in.SortOrder != nil {
		sortOrder = *in.SortOrder
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, serviceFailure(apperror.Unavailable, "zone_unavailable", "No pudimos actualizar la zona.")
	}
	defer tx.Rollback(ctx)

	var oldName string
	var oldActive bool
	err = func() error {
		v, e := a.repo.LockZone(tx.Context(), s, id)
		oldName, oldActive = v.Name, v.Active
		return e
	}()
	if err == ErrNotFound {
		return result, serviceFailure(apperror.NotFound, "zone_not_found", "La zona no existe.")
	}
	if err != nil {
		return result, serviceFailure(apperror.Unavailable, "zone_unavailable", "No pudimos bloquear la zona.")
	}
	if oldActive && !active {
		var used bool
		if err = func() error { v, e := a.repo.ZoneInUse(tx.Context(), s, oldName); used = v; return e }(); err != nil {
			return result, serviceFailure(apperror.Unavailable, "zone_unavailable", "No pudimos validar las mesas de la zona.")
		}
		if used {
			return result, serviceFailure(apperror.Conflict, "zone_in_use", "Mueve o desactiva las mesas activas de esta zona antes de desactivarla.")
		}
	}

	newName := strings.TrimSpace(in.Name)
	var z zone
	err = func() error {
		v, e := a.repo.UpdateZone(tx.Context(), s, id, newName, sortOrder, active)
		z = v
		return e
	}()
	if err != nil {
		return result, serviceFailure(apperror.Conflict, "zone_conflict", "Ya existe una zona con ese nombre.")
	}
	if oldName != newName {
		if _, err = a.repo.RenameTableZone(tx.Context(), s, oldName, newName); err != nil {
			return result, serviceFailure(apperror.Unavailable, "zone_unavailable", "No pudimos actualizar las mesas de la zona.")
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return result, serviceFailure(apperror.Unavailable, "zone_unavailable", "No pudimos confirmar la actualización.")
	}
	_ = a.audit.Record(ctx, s, "zone.updated", "zone", z.ID)
	return z, nil

}
func (a *Floor) DeactivateZone(ctx context.Context, s scope, id string) error {
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return serviceFailure(apperror.Unavailable, "zone_unavailable", "No pudimos desactivar la zona.")
	}
	defer tx.Rollback(ctx)

	var name string
	var active bool
	err = func() error { v, e := a.repo.LockZone(tx.Context(), s, id); name, active = v.Name, v.Active; return e }()
	if err == ErrNotFound || !active {
		return serviceFailure(apperror.NotFound, "zone_not_found", "La zona no existe o ya está inactiva.")
	}
	if err != nil {
		return serviceFailure(apperror.Unavailable, "zone_unavailable", "No pudimos validar la zona.")
	}
	var used bool
	if err = func() error { v, e := a.repo.ZoneInUse(tx.Context(), s, name); used = v; return e }(); err != nil {
		return serviceFailure(apperror.Unavailable, "zone_unavailable", "No pudimos validar las mesas de la zona.")
	}
	if used {
		return serviceFailure(apperror.Conflict, "zone_in_use", "Mueve o desactiva las mesas activas de esta zona antes de desactivarla.")
	}
	if _, err = a.repo.DeactivateZone(tx.Context(), s, id); err != nil {
		return serviceFailure(apperror.Unavailable, "zone_unavailable", "No pudimos desactivar la zona.")
	}
	if err = tx.Commit(ctx); err != nil {
		return serviceFailure(apperror.Unavailable, "zone_unavailable", "No pudimos confirmar la desactivación.")
	}
	_ = a.audit.Record(ctx, s, "zone.deactivated", "zone", id)
	return nil

}
func (a *Floor) CreateTable(ctx context.Context, s scope, in domain.TableInput) (domain.Table, error) {
	var result domain.Table
	if strings.TrimSpace(in.Name) == "" {
		return result, serviceFailure(apperror.Invalid, "invalid_request", "El nombre de la mesa es obligatorio.")
	}
	seats := 2
	if in.Seats != nil && *in.Seats > 0 {
		seats = *in.Seats
	}
	qrEnabled := true
	if in.QrEnabled != nil {
		qrEnabled = *in.QrEnabled
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, serviceFailure(apperror.Unavailable, "tables_unavailable", "No pudimos registrar la mesa.")
	}
	defer tx.Rollback(ctx)
	if err = a.repo.ActiveZone(tx.Context(), s, strings.TrimSpace(in.Zone)); errors.Is(err, ErrNotFound) {
		return result, serviceFailure(apperror.Invalid, "invalid_zone", "La zona seleccionada no existe o está inactiva en este local.")
	} else if err != nil {
		return result, serviceFailure(apperror.Unavailable, "zones_unavailable", "No pudimos validar la zona de la mesa.")
	}
	var t table
	err = func() error {
		v, e := a.repo.CreateTable(tx.Context(), s, strings.TrimSpace(in.Name), seats, strings.TrimSpace(in.Zone), qrEnabled)
		t = v
		return e
	}()
	if err != nil {
		return result, serviceFailure(apperror.Conflict, "table_conflict", "Ya existe una mesa con ese nombre.")
	}
	if err = tx.Commit(ctx); err != nil {
		return result, serviceFailure(apperror.Unavailable, "tables_unavailable", "No pudimos confirmar la mesa.")
	}
	_ = a.audit.Record(ctx, s, "table.created", "table", t.ID)
	return t, nil

}
func (a *Floor) UpdateTable(ctx context.Context, s scope, id string, in domain.TableInput) (domain.Table, error) {
	var result domain.Table
	if strings.TrimSpace(in.Name) == "" {
		return result, serviceFailure(apperror.Invalid, "invalid_request", "El nombre de la mesa es obligatorio.")
	}
	seats := 2
	if in.Seats != nil && *in.Seats > 0 {
		seats = *in.Seats
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	qrEnabled := true
	if in.QrEnabled != nil {
		qrEnabled = *in.QrEnabled
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, serviceFailure(apperror.Unavailable, "table_unavailable", "No pudimos actualizar la mesa.")
	}
	defer tx.Rollback(ctx)

	var currentActive bool
	err = func() error { v, e := a.repo.LockTable(tx.Context(), s, id); currentActive = v; return e }()
	if errors.Is(err, ErrNotFound) {
		return result, serviceFailure(apperror.NotFound, "table_not_found", "La mesa no existe.")
	}
	if err != nil {
		return result, serviceFailure(apperror.Unavailable, "table_unavailable", "No pudimos bloquear la mesa.")
	}
	if err = a.repo.ActiveZone(tx.Context(), s, strings.TrimSpace(in.Zone)); errors.Is(err, ErrNotFound) {
		return result, serviceFailure(apperror.Invalid, "invalid_zone", "La zona seleccionada no existe o está inactiva en este local.")
	} else if err != nil {
		return result, serviceFailure(apperror.Unavailable, "zones_unavailable", "No pudimos validar la zona de la mesa.")
	}
	if currentActive && !active {
		var occupied bool
		if err = func() error { v, e := a.repo.TableOccupied(tx.Context(), s, id); occupied = v; return e }(); err != nil {
			return result, serviceFailure(apperror.Unavailable, "table_unavailable", "No pudimos validar el estado de la mesa.")
		}
		if occupied {
			return result, serviceFailure(apperror.Conflict, "table_in_use", "No puedes desactivar una mesa con un pedido abierto.")
		}
	}

	var t table
	err = func() error {
		v, e := a.repo.UpdateTable(tx.Context(), s, id, strings.TrimSpace(in.Name), seats, strings.TrimSpace(in.Zone), active, qrEnabled)
		t = v
		return e
	}()
	if err != nil {
		return result, serviceFailure(apperror.Conflict, "table_conflict", "Ya existe una mesa con ese nombre.")
	}
	if err = tx.Commit(ctx); err != nil {
		return result, serviceFailure(apperror.Unavailable, "table_unavailable", "No pudimos confirmar la actualización.")
	}
	_ = a.audit.Record(ctx, s, "table.updated", "table", t.ID)
	return t, nil

}
func (a *Floor) DeactivateTable(ctx context.Context, s scope, id string) error {
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return serviceFailure(apperror.Unavailable, "table_unavailable", "No pudimos desactivar la mesa.")
	}
	defer tx.Rollback(ctx)
	var active bool
	err = func() error { v, e := a.repo.LockTable(tx.Context(), s, id); active = v; return e }()
	if errors.Is(err, ErrNotFound) || !active {
		return serviceFailure(apperror.NotFound, "table_not_found", "La mesa no existe o ya está inactiva.")
	}
	if err != nil {
		return serviceFailure(apperror.Unavailable, "table_unavailable", "No pudimos validar la mesa.")
	}
	var occupied bool
	if err = func() error { v, e := a.repo.TableOccupied(tx.Context(), s, id); occupied = v; return e }(); err != nil {
		return serviceFailure(apperror.Unavailable, "table_unavailable", "No pudimos validar el estado de la mesa.")
	}
	if occupied {
		return serviceFailure(apperror.Conflict, "table_in_use", "No puedes desactivar una mesa con un pedido abierto.")
	}
	if _, err = a.repo.DeactivateTable(tx.Context(), s, id); err != nil {
		return serviceFailure(apperror.Unavailable, "table_unavailable", "No pudimos desactivar la mesa.")
	}
	if err = tx.Commit(ctx); err != nil {
		return serviceFailure(apperror.Unavailable, "table_unavailable", "No pudimos confirmar la desactivación.")
	}
	_ = a.audit.Record(ctx, s, "table.deactivated", "table", id)
	return nil

}
func (a *Floor) RegenerateTableQR(ctx context.Context, s scope, id string) (domain.Table, error) {
	var result domain.Table
	var t table
	err := func() error { v, e := a.repo.RegenerateQR(ctx, s, id); t = v; return e }()
	if err == ErrNotFound {
		return result, serviceFailure(apperror.NotFound, "table_not_found", "La mesa no existe.")
	} else if err != nil {
		return result, serviceFailure(apperror.Unavailable, "table_unavailable", "No pudimos regenerar el QR.")
	}
	_ = a.audit.Record(ctx, s, "table.qr_regenerated", "table", t.ID)
	return t, nil

}
func (a *Floor) TableQR(ctx context.Context, token string) (domain.TableQR, error) {
	var out domain.TableQR
	if token == "" {
		return out, serviceFailure(apperror.Invalid, "invalid_request", "Token QR requerido.")
	}
	out, err := a.repo.TableQR(ctx, token)
	if err != nil {
		return out, serviceFailure(apperror.NotFound, "table_not_found", "La mesa no existe o el QR no está activo.")
	}
	return out, nil
}
func (a *Floor) Batch(ctx context.Context, s scope, input []domain.TableInput) (domain.TableBatchResult, error) {
	var out domain.TableBatchResult
	if len(input) == 0 || len(input) > 500 {
		return out, serviceFailure(apperror.Invalid, "invalid_request", "Agrega entre 1 y 500 mesas.")
	}
	items := make([]domain.TableBatchItem, 0, len(input))
	names := map[string]bool{}
	for _, item := range input {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		if names[name] {
			return out, serviceFailure(apperror.Conflict, "table_conflict", "No repitas el nombre de una mesa en el mismo lote.")
		}
		names[name] = true
		seats := 2
		if item.Seats != nil && *item.Seats > 0 {
			seats = *item.Seats
		}
		enabled := true
		if item.QrEnabled != nil {
			enabled = *item.QrEnabled
		}
		items = append(items, domain.TableBatchItem{Name: name, Seats: seats, Zone: strings.TrimSpace(item.Zone), QrEnabled: enabled})
	}
	if len(items) == 0 {
		return out, serviceFailure(apperror.Invalid, "invalid_request", "Agrega al menos una mesa con nombre.")
	}
	out, invalid, err := a.repo.Batch(ctx, s, items)
	if errors.Is(err, ErrConflict) {
		return out, serviceFailure(apperror.Conflict, "table_conflict", "Ya existe una mesa con uno de los nombres enviados.")
	}
	if err != nil {
		return out, serviceFailure(apperror.Unavailable, "tables_unavailable", "No pudimos confirmar el registro de las mesas. Actualiza la lista antes de intentarlo nuevamente.")
	}
	if invalid {
		return out, serviceFailure(apperror.Invalid, "invalid_zone", "Una de las mesas usa una zona inexistente o inactiva.")
	}
	return out, nil
}
