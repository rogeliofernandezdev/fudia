package application

import (
	"context"
	"errors"
	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	billingcontracts "github.com/foods-platform/foods-backend/internal/modules/billing/contracts"
	noticecontracts "github.com/foods-platform/foods-backend/internal/modules/notifications/contracts"
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
	// DefaultToleranceMinutes: tolerancia del local que toma una reserva nueva.
	DefaultToleranceMinutes int `json:"defaultToleranceMinutes"`
	// DefaultDepositAmount: garantía del local para una reserva nueva («0.00» = sin costo).
	DefaultDepositAmount string `json:"defaultDepositAmount"`
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
	// Seat pasa la reserva a «En mesa» con la mesa indicada o la que ya tenía;
	// devuelve false si quedaría sin mesa.
	Seat(context.Context, scope, string, string) (bool, error)
	// Defaults devuelve la tolerancia de llegada y la garantía configuradas en el local.
	Defaults(context.Context, scope) (ReservationDefaults, error)
	// ExpireOverdue pasa a «No asistió» las reservas pendientes o confirmadas
	// cuya hora más la tolerancia ya pasó, en todas las empresas y locales.
	ExpireOverdue(context.Context, int) ([]domain.ExpiredReservation, error)
}
// ReservationDefaults son los valores del local que toma una reserva nueva.
type ReservationDefaults struct {
	ToleranceMinutes int
	DepositAmount    string
}

type Reservations struct {
	repo    ReservationRepository
	work    unitofwork.Manager
	audit   audit.Repository
	notices noticecontracts.Publisher
	// deposits devuelve o retiene en Caja la garantía cobrada al cancelar.
	deposits billingcontracts.ReservationDeposits
}

// WithDeposits conecta la garantía cobrada en Caja.
func (a *Reservations) WithDeposits(deposits billingcontracts.ReservationDeposits) *Reservations {
	a.deposits = deposits
	return a
}

// WithNotices avisa en la campana de caja las reservas vencidas por el barrido.
func (a *Reservations) WithNotices(notices noticecontracts.Publisher) *Reservations {
	a.notices = notices
	return a
}

func NewReservations(repo ReservationRepository, work unitofwork.Manager, audit audit.Repository) *Reservations {
	return &Reservations{repo: repo, work: work, audit: audit}
}
func (a *Reservations) List(ctx context.Context, s scope, f FloorFilter) (ReservationPage, error) {
	f.Query = strings.TrimSpace(f.Query)
	f.Status = strings.TrimSpace(f.Status)
	if f.Status != "" && !domain.ValidReservationStatus(f.Status) && f.Status != domain.ReservationAttended {
		return ReservationPage{}, serviceFailure(apperror.Invalid, "invalid_reservation_status", "El estado de reserva no es válido.")
	}
	out, err := a.repo.List(ctx, s, f)
	if err != nil {
		return out, serviceFailure(apperror.Unavailable, "reservations_unavailable", "No pudimos cargar las reservas.")
	}
	defaults, err := a.repo.Defaults(ctx, s)
	if err != nil {
		return out, serviceFailure(apperror.Unavailable, "reservations_unavailable", "No pudimos cargar las reservas.")
	}
	out.DefaultToleranceMinutes, out.DefaultDepositAmount = defaults.ToleranceMinutes, defaults.DepositAmount
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
		return result, serviceFailure(apperror.Invalid, "invalid_reservation", "Completa cliente, fecha, hora y personas; la tolerancia va de 0 a 240 minutos y la garantía admite hasta dos decimales.")
	}
	in.StartsAt = startsAt.UTC().Format(time.RFC3339)
	if err := a.validateReferences(ctx, s, in, ""); err != nil {
		return result, err
	}
	if in.ToleranceMinutes == nil || in.DepositAmount == nil {
		defaults, err := a.repo.Defaults(ctx, s)
		if err != nil {
			return result, serviceFailure(apperror.Unavailable, "reservations_unavailable", "No pudimos registrar la reserva.")
		}
		if in.ToleranceMinutes == nil {
			in.ToleranceMinutes = &defaults.ToleranceMinutes
		}
		if in.DepositAmount == nil {
			in.DepositAmount = &defaults.DepositAmount
		}
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
		return result, serviceFailure(apperror.Invalid, "invalid_reservation", "Completa cliente, fecha, hora y personas; la tolerancia va de 0 a 240 minutos y la garantía admite hasta dos decimales.")
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
// SetStatus cambia el estado. Cancelar una reserva con la garantía cobrada
// exige decidir qué pasa con ella: deposit "refund" la devuelve desde el turno
// de caja del usuario y "retain" la deja en el local con su boleta.
func (a *Reservations) SetStatus(ctx context.Context, s scope, id, status, deposit, tableID string) error {
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
	retainedOrder := ""
	if status == "cancelled" && current != status && a.deposits != nil {
		held, err := a.deposits.HeldDeposit(tx.Context(), s, id)
		if err != nil {
			return err
		}
		switch {
		case !held:
		case deposit == domain.DepositRefund:
			if err := a.deposits.RefundHeldDeposit(tx.Context(), s, id); err != nil {
				return err
			}
		case deposit == domain.DepositRetain:
			if retainedOrder, err = a.deposits.RetainHeldDeposit(tx.Context(), s, id); err != nil {
				return err
			}
		default:
			return serviceFailure(apperror.Conflict, "deposit_decision_required", "Elige si devuelves la garantía o el local la retiene.")
		}
	}
	if status == "seated" && current != status {
		// «En mesa» exige saber qué mesa ocupa el cliente: la elegida al
		// ubicarlo o la que la reserva ya tenía asignada.
		tableID = strings.TrimSpace(tableID)
		if tableID != "" {
			if _, err := a.repo.TableSeats(tx.Context(), s, tableID); errors.Is(err, ErrNotFound) {
				return serviceFailure(apperror.Invalid, "invalid_table", "La mesa no existe o está inactiva en este local.")
			} else if err != nil {
				return serviceFailure(apperror.Unavailable, "reservations_unavailable", "No pudimos validar la mesa.")
			}
		}
		seated, err := a.repo.Seat(tx.Context(), s, id, tableID)
		if err != nil {
			return serviceFailure(apperror.Unavailable, "reservations_unavailable", "No pudimos actualizar la reserva.")
		}
		if !seated {
			return serviceFailure(apperror.Invalid, "seat_table_required", "Elige la mesa donde se ubicó el cliente.")
		}
	} else if _, err = a.repo.SetStatus(tx.Context(), s, id, status); err != nil {
		return serviceFailure(apperror.Unavailable, "reservations_unavailable", "No pudimos actualizar la reserva.")
	}
	if err = tx.Commit(ctx); err != nil {
		return serviceFailure(apperror.Unavailable, "reservations_unavailable", "No pudimos confirmar la actualización.")
	}
	_ = a.audit.Record(ctx, s, "reservation.status_updated", "reservation", id)
	if retainedOrder != "" {
		a.deposits.IssueRetainedReceipt(ctx, s, retainedOrder)
	}
	return nil

}

// ExpireBatch acota cuántas reservas vence cada barrido; el resto espera al siguiente.
const ExpireBatch = 200

// ExpireOverdue cancela como «No asistió» las reservas cuya tolerancia venció y
// avisa a caja en la campana. El aviso es de mejor esfuerzo: la reserva ya quedó
// vencida aunque falle la publicación.
func (a *Reservations) ExpireOverdue(ctx context.Context) (int, error) {
	expired, err := a.repo.ExpireOverdue(ctx, ExpireBatch)
	for _, item := range expired {
		s := scope{OrganizationID: item.OrganizationID, LocationID: item.LocationID}
		_ = a.audit.Record(ctx, s, "reservation.no_show_expired", "reservation", item.ID)
		if a.notices != nil {
			_ = a.notices.Publish(ctx, s, noticecontracts.ReservationNoShow(item.ID, item.CustomerName, item.Guests, item.LocalTime, item.TableName, item.Deposit))
		}
	}
	return len(expired), err
}
