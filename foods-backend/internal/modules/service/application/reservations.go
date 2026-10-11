package application

import (
	"context"
	"errors"
	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	"github.com/foods-platform/foods-backend/internal/modules/service/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"strings"
	"time"
)

type ReservationPage struct {
	Items    []domain.ReservationView `json:"items"`
	Total    int                      `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"pageSize"`
}
type ReservationRepository interface {
	List(context.Context, scope, FloorFilter) (ReservationPage, error)
	CustomerAvailable(context.Context, scope, string) (bool, error)
	TableSeats(context.Context, scope, string) (int, error)
	Overlaps(context.Context, scope, domain.ReservationInput, string) (bool, error)
	Create(context.Context, scope, domain.ReservationInput, time.Time) (domain.ReservationView, error)
	Update(context.Context, scope, string, domain.ReservationInput, time.Time) (domain.ReservationView, error)
	Status(context.Context, scope, string, bool) (string, error)
	SetStatus(context.Context, scope, string, string) (int64, error)
}
type Reservations struct {
	repo  ReservationRepository
	work  unitofwork.Manager
	audit audit.Repository
}

func NewReservations(repo ReservationRepository, work unitofwork.Manager, audit audit.Repository) *Reservations {
	return &Reservations{repo, work, audit}
}
func (a *Reservations) List(ctx context.Context, s scope, f FloorFilter) (ReservationPage, error) {
	f.Query = strings.TrimSpace(f.Query)
	f.Status = strings.TrimSpace(f.Status)
	if f.Status != "" && !domain.ValidReservationStatus(f.Status) {
		return ReservationPage{}, serviceFailure(apperror.Invalid, "invalid_reservation_status", "El estado de reserva no es válido.")
	}
	out, err := a.repo.List(ctx, s, f)
	if err != nil {
		return out, serviceFailure(apperror.Unavailable, "reservations_unavailable", "No pudimos cargar las reservas.")
	}
	return out, nil
}
func (a *Reservations) validateReferences(ctx context.Context, s scope, in domain.ReservationInput, exclude string) error {
	if in.CustomerID != nil {
		ok, err := a.repo.CustomerAvailable(ctx, s, *in.CustomerID)
		if err != nil || !ok {
			return serviceFailure(apperror.Invalid, "invalid_customer", "El cliente seleccionado no está disponible.")
		}
	}
	if in.TableID != nil {
		seats, err := a.repo.TableSeats(ctx, s, *in.TableID)
		if errors.Is(err, ErrNotFound) {
			return serviceFailure(apperror.Invalid, "invalid_table", "La mesa seleccionada no está disponible en este local.")
		}
		if err != nil {
			return serviceFailure(apperror.Invalid, "reservation_unavailable", "No pudimos validar la mesa.")
		}
		if in.Guests > seats {
			return serviceFailure(apperror.Invalid, "table_capacity", "La mesa seleccionada no tiene capacidad suficiente.")
		}
		conflict, err := a.repo.Overlaps(ctx, s, in, exclude)
		if err != nil {
			return serviceFailure(apperror.Invalid, "reservation_unavailable", "No pudimos validar la disponibilidad de la mesa.")
		}
		if conflict {
			return serviceFailure(apperror.Invalid, "reservation_conflict", "La mesa ya tiene una reserva en ese horario.")
		}
	}
	return nil
}
func (a *Reservations) Create(ctx context.Context, s scope, in domain.ReservationInput) (domain.ReservationView, error) {
	var result domain.ReservationView
	startsAt, ok := domain.ParseReservationInput(&in)
	if !ok {
		return result, serviceFailure(apperror.Invalid, "invalid_reservation", "Completa cliente, fecha, hora y número de personas.")
	}
	in.StartsAt = startsAt.UTC().Format(time.RFC3339)
	if err := a.validateReferences(ctx, s, in, ""); err != nil {
		return result, err
	}
	item, err := a.repo.Create(ctx, s, in, startsAt)
	if err != nil {
		return result, serviceFailure(apperror.Unavailable, "reservations_unavailable", "No pudimos registrar la reserva.")
	}
	_ = a.audit.Record(ctx, s, "reservation.created", "reservation", item.ID)
	return item, nil

}
func (a *Reservations) Update(ctx context.Context, s scope, id string, in domain.ReservationInput) (domain.ReservationView, error) {
	var result domain.ReservationView
	startsAt, ok := domain.ParseReservationInput(&in)
	if !ok {
		return result, serviceFailure(apperror.Invalid, "invalid_reservation", "Completa cliente, fecha, hora y número de personas.")
	}
	in.StartsAt = startsAt.UTC().Format(time.RFC3339)
	if err := a.validateReferences(ctx, s, in, id); err != nil {
		return result, err
	}
	var current string
	if err := func() error { v, e := a.repo.Status(ctx, s, id, false); current = v; return e }(); errors.Is(err, ErrNotFound) {
		return result, serviceFailure(apperror.NotFound, "reservation_not_found", "La reserva no existe.")
	} else if err != nil {
		return result, serviceFailure(apperror.Unavailable, "reservations_unavailable", "No pudimos validar la reserva.")
	}
	if current != "pending" && current != "confirmed" {
		return result, serviceFailure(apperror.Conflict, "reservation_not_editable", "Solo una reserva pendiente o confirmada puede editarse.")
	}
	item, err := a.repo.Update(ctx, s, id, in, startsAt)
	if errors.Is(err, ErrNotFound) {
		return result, serviceFailure(apperror.NotFound, "reservation_not_found", "La reserva no existe.")
	}
	if err != nil {
		return result, serviceFailure(apperror.Unavailable, "reservations_unavailable", "No pudimos actualizar la reserva.")
	}
	_ = a.audit.Record(ctx, s, "reservation.updated", "reservation", item.ID)
	return item, nil

}
func (a *Reservations) SetStatus(ctx context.Context, s scope, id, status string) error {
	if !domain.ValidReservationStatus(status) {
		return serviceFailure(apperror.Invalid, "invalid_reservation_status", "El estado de reserva no es válido.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return serviceFailure(apperror.Unavailable, "reservations_unavailable", "No pudimos iniciar la actualización.")
	}
	defer tx.Rollback(ctx)
	var current string
	err = func() error { v, e := a.repo.Status(tx.Context(), s, id, true); current = v; return e }()
	if errors.Is(err, ErrNotFound) {
		return serviceFailure(apperror.NotFound, "reservation_not_found", "La reserva no existe.")
	}
	if err != nil {
		return serviceFailure(apperror.Unavailable, "reservations_unavailable", "No pudimos validar la reserva.")
	}
	if !domain.ValidReservationTransition(current, status) {
		return serviceFailure(apperror.Conflict, "invalid_reservation_transition", "La reserva ya no admite ese cambio de estado.")
	}
	if _, err = a.repo.SetStatus(tx.Context(), s, id, status); err != nil {
		return serviceFailure(apperror.Unavailable, "reservations_unavailable", "No pudimos actualizar la reserva.")
	}
	if err = tx.Commit(ctx); err != nil {
		return serviceFailure(apperror.Unavailable, "reservations_unavailable", "No pudimos confirmar la actualización.")
	}
	_ = a.audit.Record(ctx, s, "reservation.status_updated", "reservation", id)
	return nil

}
