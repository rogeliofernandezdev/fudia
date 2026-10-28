package application

import (
	"context"

	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

func cashRegisterShift(shift *domain.CashShiftView, userID string, all bool) *domain.CashShiftView {
	if shift == nil || all || shift.OpenedByID == userID {
		return shift
	}
	// Keep only occupancy metadata; never offer an occupied register as free.
	return &domain.CashShiftView{
		ID: shift.ID, CashRegisterID: shift.CashRegisterID, CashRegisterName: shift.CashRegisterName,
		Status: shift.Status, OpenedByID: shift.OpenedByID, OpenedByName: shift.OpenedByName,
		OpenedAt: shift.OpenedAt, BusinessDate: shift.BusinessDate, ActiveUserNames: []string{},
	}
}

func (a *Cash) readAllShifts(ctx context.Context, s scope) (bool, error) {
	all, err := a.repo.CanReadAllShifts(ctx, s)
	if err != nil {
		return false, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos validar el acceso a los turnos.")
	}
	return all, nil
}

// Owning the shift means having opened it. Being a temporary collaborator, or
// having cash.read/cash.manage, does not expose another cashier's finances.
func (a *Cash) requireShiftRead(ctx context.Context, s scope, id string) error {
	all, err := a.readAllShifts(ctx, s)
	if err != nil || all {
		return err
	}
	owned, err := a.repo.OwnsShift(ctx, s, id)
	if err != nil {
		return paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos validar el acceso al turno.")
	}
	if !owned {
		return paymentFailure(apperror.NotFound, "cash_shift_not_found", "El turno de caja no está disponible para tu usuario en este local.")
	}
	return nil
}
