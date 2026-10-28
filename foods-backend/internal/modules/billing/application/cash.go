package application

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"strings"
)

var ErrConflict = errors.New("billing record conflict")

type CashFilter struct {
	Query, Status string
	// Set by the use case, never from request parameters.
	OwnerID    string
	Page, Size int
}
type CashPage struct {
	Items    []domain.CashShiftView `json:"items"`
	Total    int                    `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
}
type Cash struct {
	repo CashRepository
	work unitofwork.Manager
}

func NewCash(repo CashRepository, work unitofwork.Manager) *Cash { return &Cash{repo, work} }

type CashRepository interface {
	CanReadAllShifts(context.Context, scope) (bool, error)
	OwnsShift(context.Context, scope, string) (bool, error)
	RequireShiftAssignment(context.Context, scope, string) error
	ExpectedCash(context.Context, scope, string) (float64, error)
	CreateRegister(ctx context.Context, s scope, in domain.RegisterInput) (domain.CashRegisterView, error)
	UpdateRegister(ctx context.Context, s scope, id string, in domain.RegisterInput) (domain.CashRegisterView, error)
	RegisterHasOpenShift(ctx context.Context, s scope, id string) (bool, error)
	SetRegisterActive(ctx context.Context, s scope, id string, in domain.RegisterStatusInput) (int64, error)
	LockRegister(ctx context.Context, s scope, in domain.OpenShiftInput) (string, error)
	OpenShift(ctx context.Context, s scope, in domain.OpenShiftInput) (string, error)
	AssignOpeningUser(ctx context.Context, s scope, id string) (int64, error)
	LockMovementShift(ctx context.Context, s scope, shiftID string) (string, error)
	CreateMovement(ctx context.Context, s scope, shiftID string, in domain.CashMovementInput) (domain.CashMovementView, error)
	LockClosingShift(ctx context.Context, s scope, shiftID string) (string, bool, float64, float64, float64, error)
	SaveCountLine(ctx context.Context, s scope, shiftID string, line domain.CountLineInput) (int64, error)
	CloseShift(ctx context.Context, s scope, shiftID string, in domain.CloseShiftInput, expected float64, counted float64, variance float64) (int64, error)
	UnassignClosingUsers(ctx context.Context, s scope, shiftID string) (int64, error)
	SaveReport(ctx context.Context, s scope, shiftID string, data []byte) (int64, error)
	AuditClosedShift(ctx context.Context, s scope, shiftID string) (int64, error)
	LockAssignShift(ctx context.Context, s scope, id string) (bool, error)
	AssignableUser(ctx context.Context, s scope, in domain.ShiftUserInput) (string, error)
	AssignUser(ctx context.Context, s scope, id string, in domain.ShiftUserInput, name string) (domain.CashShiftUserView, error)
	UnassignUser(ctx context.Context, s scope, id string, userID string) (int64, error)
	LockTargetShift(ctx context.Context, s scope, in domain.CashOperationInput) (bool, error)
	CreateOperation(ctx context.Context, s scope, sourceShiftID string, in domain.CashOperationInput) (domain.CashOperationView, error)
	OperationOutflow(ctx context.Context, s scope, sourceShiftID string, sourceType string, operation domain.CashOperationView, in domain.CashOperationInput) (int64, error)
	OperationInflow(ctx context.Context, s scope, operation domain.CashOperationView, in domain.CashOperationInput) (int64, error)
	Shift(ctx context.Context, s scope, id string) (domain.CashShiftView, error)
	OpenShiftForRegister(ctx context.Context, s scope, registerID string) (domain.CashShiftView, error)
	UserShift(ctx context.Context, s scope) (domain.CashShiftView, error)
	Movements(ctx context.Context, s scope, shiftID string) ([]domain.CashMovementView, error)
	Registers(ctx context.Context, s scope, query string) ([]domain.CashRegisterView, error)
	Shifts(ctx context.Context, s scope, f CashFilter) (CashPage, error)
	UserOptions(ctx context.Context, s scope) ([]domain.CashUserOption, error)
	ShiftUsers(ctx context.Context, s scope, id string) ([]domain.CashShiftUserView, error)
	BuildReport(ctx context.Context, s scope, id string) (domain.CashShiftReport, error)
	StoredReport(ctx context.Context, s scope, id string) ([]byte, error)
	IsPlatformAdmin(ctx context.Context, s scope) (bool, error)
	CanSeeExpected(ctx context.Context, s scope) (bool, error)
}

func (a *Cash) CreateRegister(ctx context.Context, s scope, in domain.RegisterInput) (domain.CashRegisterView, error) {
	var result domain.CashRegisterView
	in.Name = strings.TrimSpace(in.Name)
	if len(in.Name) < 1 || len(in.Name) > 80 {
		return result, paymentFailure(apperror.Invalid, "invalid_cash_register", "El nombre de la caja debe tener entre 1 y 80 caracteres.")
	}

	var item domain.CashRegisterView
	err := func() error { var err error; item, err = a.repo.CreateRegister(ctx, s, in); return err }()
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return result, paymentFailure(apperror.Conflict, "cash_register_exists", "Ya existe una caja con ese nombre en este local.")
		}
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos registrar la caja.")
	}
	return item, nil

}
func (a *Cash) UpdateRegister(ctx context.Context, s scope, id string, in domain.RegisterInput) (domain.CashRegisterView, error) {
	var result domain.CashRegisterView
	all, err := a.readAllShifts(ctx, s)
	if err != nil {
		return result, err
	}
	in.Name = strings.TrimSpace(in.Name)
	if len(in.Name) < 1 || len(in.Name) > 80 {
		return result, paymentFailure(apperror.Invalid, "invalid_cash_register", "El nombre de la caja debe tener entre 1 y 80 caracteres.")
	}

	var item domain.CashRegisterView
	err = func() error { var err error; item, err = a.repo.UpdateRegister(ctx, s, id, in); return err }()
	if errors.Is(err, ErrNotFound) {
		return result, paymentFailure(apperror.NotFound, "cash_register_not_found", "La caja no existe en este local.")
	}
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return result, paymentFailure(apperror.Conflict, "cash_register_exists", "Ya existe una caja con ese nombre en este local.")
		}
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos actualizar la caja.")
	}
	openShift, err := a.openShift(ctx, s, item.ID)
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "La caja se actualizó, pero no pudimos cargar su turno.")
	}
	item.OpenShift = cashRegisterShift(openShift, s.UserID, all)
	return item, nil

}
func (a *Cash) SetRegisterActive(ctx context.Context, s scope, id string, in domain.RegisterStatusInput) error {
	if !in.Active {
		var open bool
		if err := func() error { var err error; open, err = a.repo.RegisterHasOpenShift(ctx, s, id); return err }(); err != nil {
			return paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos validar la caja.")
		}
		if open {
			return paymentFailure(apperror.Conflict, "cash_register_has_open_shift", "Cierra el turno activo antes de desactivar la caja.")
		}
	}
	tag, err := a.repo.SetRegisterActive(ctx, s, id, in)
	if err != nil {
		return paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos actualizar el estado de la caja.")
	}
	if tag == 0 {
		return paymentFailure(apperror.NotFound, "cash_register_not_found", "La caja no existe en este local.")
	}
	return nil

}
func (a *Cash) Open(ctx context.Context, s scope, in domain.OpenShiftInput) (domain.CashShiftView, error) {
	var result domain.CashShiftView
	if strings.TrimSpace(in.CashRegisterID) == "" || in.OpeningAmount < 0 {
		return result, paymentFailure(apperror.Invalid, "invalid_cash_shift", "Selecciona una caja e ingresa un fondo inicial válido.")
	}
	in.CashRegisterID = strings.TrimSpace(in.CashRegisterID)
	in.Note = strings.TrimSpace(in.Note)
	if len(in.Note) > 240 {
		return result, paymentFailure(apperror.Invalid, "invalid_cash_shift", "La observación no puede superar 240 caracteres.")
	}

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos iniciar el turno de caja.")
	}
	defer tx.Rollback(ctx)

	var registerName string
	_ = registerName
	if err := func() error { var err error; registerName, err = a.repo.LockRegister(tx.Context(), s, in); return err }(); errors.Is(err, ErrNotFound) {
		return result, paymentFailure(apperror.NotFound, "cash_register_not_found", "La caja seleccionada no existe o está inactiva.")
	} else if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos validar la caja.")
	}

	var id string
	err = func() error { var err error; id, err = a.repo.OpenShift(tx.Context(), s, in); return err }()
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return result, paymentFailure(apperror.Conflict, "cash_shift_already_open", "Esa caja ya tiene un turno abierto o ya tienes otro turno abierto en este local.")
		}
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos iniciar el turno de caja.")
	}

	if _, err := a.repo.AssignOpeningUser(tx.Context(), s, id); err != nil {
		if errors.Is(err, ErrConflict) {
			return result, paymentFailure(apperror.Conflict, "cash_user_already_assigned", "Ya estás asignado a otro turno abierto en este local.")
		}
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos asignarte al turno.")
	}

	if err := tx.Commit(ctx); err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos confirmar la apertura del turno.")
	}

	shift, err := a.shift(ctx, s, id)
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "El turno se inició, pero no pudimos cargar su detalle.")
	}
	return shift, nil

}
func (a *Cash) Movement(ctx context.Context, s scope, id string, in domain.CashMovementInput) (domain.CashMovementView, error) {
	var result domain.CashMovementView
	if (in.MovementType != "income" && in.MovementType != "expense") || in.Amount <= 0 {
		return result, paymentFailure(apperror.Invalid, "invalid_cash_movement", "Revisa el tipo y monto del movimiento.")
	}
	in.Reason = strings.TrimSpace(in.Reason)
	in.Note = strings.TrimSpace(in.Note)
	if in.Reason == "" || len(in.Reason) > 120 || len(in.Note) > 240 {
		return result, paymentFailure(apperror.Invalid, "invalid_cash_movement", "Indica un motivo válido y una observación de hasta 240 caracteres.")
	}

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos iniciar el movimiento de caja.")
	}
	defer tx.Rollback(ctx)

	shiftID := id
	var status string
	err = func() error {
		var err error
		status, err = a.repo.LockMovementShift(tx.Context(), s, shiftID)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		return result, paymentFailure(apperror.NotFound, "cash_shift_not_found", "El turno de caja no existe en este local.")
	}
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos validar el turno de caja.")
	}
	if status != "open" {
		return result, paymentFailure(apperror.Conflict, "cash_shift_closed", "El turno ya fue cerrado y no admite nuevos movimientos.")
	}
	if err := a.repo.RequireShiftAssignment(tx.Context(), s, shiftID); errors.Is(err, ErrNotFound) {
		return result, paymentFailure(apperror.Forbidden, "cash_shift_not_assigned", "Debes estar asignado al turno para operar esta caja.")
	} else if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos validar tu asignación.")
	}
	if in.MovementType == "expense" {
		expected, err := a.repo.ExpectedCash(tx.Context(), s, shiftID)
		if err != nil {
			return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos calcular el saldo esperado.")
		}
		if in.Amount > expected+0.00001 {
			return result, paymentFailure(apperror.Conflict, "cash_expense_exceeds_expected", "El egreso supera el efectivo esperado del turno.")
		}
	}

	var item domain.CashMovementView
	err = func() error {
		var err error
		item, err = a.repo.CreateMovement(tx.Context(), s, shiftID, in)
		return err
	}()
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos registrar el movimiento de caja.")
	}

	if err := tx.Commit(ctx); err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos confirmar el movimiento de caja.")
	}
	return item, nil

}
func (a *Cash) Close(ctx context.Context, s scope, id string, in domain.CloseShiftInput) (domain.CashShiftView, error) {
	var result domain.CashShiftView
	in.Note = strings.TrimSpace(in.Note)
	if len(in.Note) > 240 {
		return result, paymentFailure(apperror.Invalid, "invalid_cash_close", "La observación no puede superar 240 caracteres.")
	}

	counted := 0.0
	if len(in.Counts) > 0 {
		seen := map[float64]bool{}
		for _, line := range in.Counts {
			if line.Denomination <= 0 || line.Quantity < 0 || seen[line.Denomination] {
				return result, paymentFailure(apperror.Invalid, "invalid_cash_count", "Revisa las denominaciones del conteo.")
			}
			seen[line.Denomination] = true
			counted += line.Denomination * float64(line.Quantity)
		}
	} else if in.CountedAmount != nil && *in.CountedAmount >= 0 {
		counted = *in.CountedAmount
	} else {
		return result, paymentFailure(apperror.Invalid, "invalid_cash_close", "Ingresa el efectivo contado o detalla las denominaciones.")
	}

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos iniciar el cierre de caja.")
	}
	defer tx.Rollback(ctx)

	shiftID := id
	var status string
	var owned bool
	var opening, income, expense float64
	err = func() error {
		var err error
		status, owned, opening, income, expense, err = a.repo.LockClosingShift(tx.Context(), s, shiftID)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		return result, paymentFailure(apperror.NotFound, "cash_shift_not_found", "El turno de caja no existe en este local.")
	}
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos validar el turno de caja.")
	}
	if !owned {
		return result, paymentFailure(apperror.Forbidden, "cash_shift_owner_required", "Solo la persona que abrió el turno puede cerrarlo.")
	}
	if status != "open" {
		return result, paymentFailure(apperror.Conflict, "cash_shift_closed", "El turno ya fue cerrado.")
	}

	if len(in.Counts) > 0 {
		for _, line := range in.Counts {
			if line.Quantity == 0 {
				continue
			}
			if _, err := a.repo.SaveCountLine(tx.Context(), s, shiftID, line); err != nil {
				return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos guardar el conteo por denominaciones.")
			}
		}
	}

	expected := opening + income - expense
	variance := counted - expected
	if changed, err := a.repo.CloseShift(tx.Context(), s, shiftID, in, expected, counted, variance); err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos cerrar el turno de caja.")
	} else if changed != 1 {
		return result, paymentFailure(apperror.Conflict, "cash_shift_closed", "El turno ya no está disponible para cierre.")
	}

	if _, err := a.repo.UnassignClosingUsers(tx.Context(), s, shiftID); err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos liberar al equipo del turno.")
	}

	report, err := a.repo.BuildReport(tx.Context(), s, shiftID)
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_report_unavailable", "No pudimos preparar el detalle del cierre. El turno sigue abierto.")
	}
	report.Persisted = true
	data, err := json.Marshal(report)
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_report_unavailable", "No pudimos preparar el informe de cierre.")
	}
	if _, err = a.repo.SaveReport(tx.Context(), s, shiftID, data); err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_report_unavailable", "No pudimos guardar el informe. El turno sigue abierto.")
	}
	if _, err = a.repo.AuditClosedShift(tx.Context(), s, shiftID); err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos registrar la trazabilidad del cierre.")
	}
	if err := tx.Commit(ctx); err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos confirmar el cierre de caja.")
	}
	return report.Shift, nil

}
func (a *Cash) Assign(ctx context.Context, s scope, id string, in domain.ShiftUserInput) (domain.CashShiftUserView, error) {
	var result domain.CashShiftUserView
	if strings.TrimSpace(in.UserID) == "" {
		return result, paymentFailure(apperror.Invalid, "invalid_cash_user", "Selecciona un usuario válido.")
	}
	in.UserID = strings.TrimSpace(in.UserID)
	if in.UserID == s.UserID {
		return result, paymentFailure(apperror.Invalid, "cash_self_assignment", "No puedes asignarte como colaborador de tu propio turno.")
	}

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos asignar el usuario.")
	}
	defer tx.Rollback(ctx)

	var open bool
	if err := func() error { var err error; open, err = a.repo.LockAssignShift(tx.Context(), s, id); return err }(); errors.Is(err, ErrNotFound) {
		return result, paymentFailure(apperror.NotFound, "cash_shift_not_found", "El turno no existe en este local.")
	} else if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos validar el turno.")
	}
	if !open {
		return result, paymentFailure(apperror.Conflict, "cash_shift_closed", "No puedes asignar usuarios a un turno cerrado.")
	}

	var name string
	if err := func() error { var err error; name, err = a.repo.AssignableUser(tx.Context(), s, in); return err }(); errors.Is(err, ErrNotFound) {
		return result, paymentFailure(apperror.NotFound, "cash_user_not_found", "El usuario no está disponible en este local.")
	} else if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos validar el usuario.")
	}

	var item domain.CashShiftUserView
	err = func() error { var err error; item, err = a.repo.AssignUser(tx.Context(), s, id, in, name); return err }()
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return result, paymentFailure(apperror.Conflict, "cash_user_already_assigned", "El usuario ya está asignado a un turno abierto.")
		}
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos asignar el usuario.")
	}
	if err := tx.Commit(ctx); err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos confirmar la asignación.")
	}
	return item, nil

}
func (a *Cash) Unassign(ctx context.Context, s scope, id string, userID string) error {
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos retirar al usuario del turno.")
	}
	defer tx.Rollback(ctx)
	open, err := a.repo.LockAssignShift(tx.Context(), s, id)
	if errors.Is(err, ErrNotFound) {
		return paymentFailure(apperror.NotFound, "cash_shift_not_found", "El turno no existe en este local.")
	}
	if err != nil {
		return paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos validar el turno.")
	}
	if !open {
		return paymentFailure(apperror.Conflict, "cash_shift_closed", "El turno ya está cerrado.")
	}
	targetScope := s
	targetScope.UserID = userID
	responsible, err := a.repo.OwnsShift(tx.Context(), targetScope, id)
	if err != nil {
		return paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos validar al responsable del turno.")
	}
	if responsible {
		return paymentFailure(apperror.Conflict, "cash_shift_owner_not_removable", "El cajero responsable no se puede quitar del turno.")
	}
	tag, err := a.repo.UnassignUser(tx.Context(), s, id, userID)
	if err != nil {
		return paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos retirar al usuario del turno.")
	}
	if tag == 0 {
		return paymentFailure(apperror.NotFound, "cash_user_not_assigned", "El usuario ya no está asignado a este turno.")
	}
	if err := tx.Commit(ctx); err != nil {
		return paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos confirmar el retiro del colaborador.")
	}
	return nil

}
func (a *Cash) Operation(ctx context.Context, s scope, id string, in domain.CashOperationInput) (domain.CashOperationView, error) {
	var result domain.CashOperationView
	if in.Amount <= 0 {
		return result, paymentFailure(apperror.Invalid, "invalid_cash_operation", "Ingresa un monto válido.")
	}
	if in.OperationType != "cash_pull" && in.OperationType != "deposit" && in.OperationType != "transfer" {
		return result, paymentFailure(apperror.Invalid, "invalid_cash_operation", "Selecciona una operación de caja válida.")
	}
	in.Reason = strings.TrimSpace(in.Reason)
	in.Note = strings.TrimSpace(in.Note)
	in.TargetShiftID = strings.TrimSpace(in.TargetShiftID)
	if in.Reason == "" || len(in.Reason) > 120 || len(in.Note) > 240 || (in.OperationType == "transfer" && in.TargetShiftID == "") {
		return result, paymentFailure(apperror.Invalid, "invalid_cash_operation", "Completa el motivo y los datos de la operación.")
	}

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos iniciar la operación de caja.")
	}
	defer tx.Rollback(ctx)

	sourceShiftID := id
	if err := a.repo.RequireShiftAssignment(tx.Context(), s, sourceShiftID); errors.Is(err, ErrNotFound) {
		return result, paymentFailure(apperror.Forbidden, "cash_shift_not_assigned", "Debes estar asignado al turno para operar esta caja.")
	} else if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos validar tu asignación.")
	}

	expected, err := a.repo.ExpectedCash(tx.Context(), s, sourceShiftID)
	if errors.Is(err, ErrNotFound) {
		return result, paymentFailure(apperror.Conflict, "cash_shift_closed", "El turno no está abierto.")
	}
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos calcular el saldo disponible.")
	}
	if in.Amount > expected+0.00001 {
		return result, paymentFailure(apperror.Conflict, "cash_operation_exceeds_expected", "La operación supera el efectivo esperado del turno.")
	}

	if in.OperationType == "transfer" {
		if in.TargetShiftID == sourceShiftID {
			return result, paymentFailure(apperror.Invalid, "invalid_cash_transfer", "La caja de destino debe ser diferente.")
		}
		var targetOpen bool
		if err := func() error { var err error; targetOpen, err = a.repo.LockTargetShift(tx.Context(), s, in); return err }(); errors.Is(err, ErrNotFound) || !targetOpen {
			return result, paymentFailure(apperror.Conflict, "cash_transfer_target_unavailable", "La caja de destino no tiene un turno abierto.")
		} else if err != nil {
			return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos validar la caja de destino.")
		}
	}

	var operation domain.CashOperationView
	err = func() error {
		var err error
		operation, err = a.repo.CreateOperation(tx.Context(), s, sourceShiftID, in)
		return err
	}()
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos registrar la operación.")
	}

	sourceType := in.OperationType
	if in.OperationType == "transfer" {
		sourceType = "transfer_out"
	}
	if _, err := a.repo.OperationOutflow(tx.Context(), s, sourceShiftID, sourceType, operation, in); err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos afectar la caja de origen.")
	}

	if in.OperationType == "transfer" {
		if _, err := a.repo.OperationInflow(tx.Context(), s, operation, in); err != nil {
			return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos afectar la caja de destino.")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos confirmar la operación.")
	}
	return operation, nil

}
func (a *Cash) Report(ctx context.Context, s scope, id string) (domain.CashShiftReport, error) {
	var result domain.CashShiftReport
	tx, err := a.work.Begin(ctx, unitofwork.Options{RepeatableRead: true, ReadOnly: true})
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_report_unavailable", "No pudimos cargar el resumen de caja.")
	}
	defer tx.Rollback(ctx)
	if err := a.requireShiftRead(tx.Context(), s, id); err != nil {
		return result, err
	}
	var stored []byte
	err = func() error { var err error; stored, err = a.repo.StoredReport(tx.Context(), s, id); return err }()
	var out domain.CashShiftReport
	if err == nil {
		err = json.Unmarshal(stored, &out)
	} else if errors.Is(err, ErrNotFound) {
		out, err = a.repo.BuildReport(tx.Context(), s, id)
	}
	if errors.Is(err, ErrNotFound) {
		return result, paymentFailure(apperror.NotFound, "cash_shift_not_found", "El turno de caja no existe en este local.")
	}
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_report_unavailable", "No pudimos cargar el resumen de caja.")
	}
	out.Shift.DetailsVisible = true
	if out.Shift.Status == "open" && out.Shift.BlindClose && !a.canSeeExpected(ctx, s) {
		return result, paymentFailure(apperror.Forbidden, "cash_report_blind_close", "El detalle de este cierre ciego estará disponible después de cerrar el turno.")
	}
	if err = tx.Commit(ctx); err != nil {
		return result, paymentFailure(apperror.Unavailable, "cash_report_unavailable", "No pudimos confirmar el resumen de caja.")
	}

	return out, nil

}

func (a *Cash) canSeeExpected(ctx context.Context, s scope) bool {
	admin, err := a.repo.IsPlatformAdmin(ctx, s)
	if err == nil && admin {
		return true
	}
	allowed, err := a.repo.CanSeeExpected(ctx, s)
	return err == nil && allowed
}
func (a *Cash) visibility(ctx context.Context, s scope, shift *domain.CashShiftView) {
	if shift == nil {
		return
	}
	shift.ExpectedVisible = true
	shift.DetailsVisible = true
	if shift.Status == "open" && shift.BlindClose && !a.canSeeExpected(ctx, s) {
		shift.OpeningAmount = ""
		shift.IncomeAmount = ""
		shift.ExpenseAmount = ""
		shift.ExpectedAmount = ""
		shift.ExpectedVisible = false
	}
}
func (a *Cash) shift(ctx context.Context, s scope, id string) (domain.CashShiftView, error) {
	out, err := a.repo.Shift(ctx, s, id)
	if err != nil {
		return out, err
	}
	out.Movements, err = a.repo.Movements(ctx, s, id)
	if err != nil {
		return out, err
	}
	a.visibility(ctx, s, &out)
	return out, nil
}
func (a *Cash) openShift(ctx context.Context, s scope, id string) (*domain.CashShiftView, error) {
	out, err := a.repo.OpenShiftForRegister(ctx, s, id)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.visibility(ctx, s, &out)
	return &out, nil
}
func (a *Cash) Get(ctx context.Context, s scope, id string) (domain.CashShiftView, error) {
	if err := a.requireShiftRead(ctx, s, id); err != nil {
		return domain.CashShiftView{}, err
	}
	out, err := a.shift(ctx, s, id)
	if errors.Is(err, ErrNotFound) {
		return out, paymentFailure(apperror.NotFound, "cash_shift_not_found", "El turno de caja no existe en este local.")
	}
	if err != nil {
		return out, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos cargar el turno de caja.")
	}
	return out, nil
}
func (a *Cash) Registers(ctx context.Context, s scope, q string) ([]domain.CashRegisterView, error) {
	all, err := a.readAllShifts(ctx, s)
	if err != nil {
		return nil, err
	}
	out, err := a.repo.Registers(ctx, s, q)
	if err != nil {
		return nil, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos cargar las cajas del local.")
	}
	for i := range out {
		shift, err := a.openShift(ctx, s, out[i].ID)
		if err != nil {
			return nil, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos cargar el turno de la caja.")
		}
		out[i].OpenShift = cashRegisterShift(shift, s.UserID, all)
	}
	return out, nil
}
func (a *Cash) List(ctx context.Context, s scope, f CashFilter) (CashPage, error) {
	all, err := a.readAllShifts(ctx, s)
	if err != nil {
		return CashPage{}, err
	}
	f.OwnerID = ""
	if !all {
		f.OwnerID = s.UserID
	}
	f.Query = strings.TrimSpace(f.Query)
	f.Status = strings.TrimSpace(f.Status)
	if f.Status != "" && f.Status != "open" && f.Status != "closed" {
		return CashPage{}, paymentFailure(apperror.Invalid, "invalid_cash_shift_status", "El estado del turno no es válido.")
	}
	out, err := a.repo.Shifts(ctx, s, f)
	if err != nil {
		return out, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos cargar los turnos de caja.")
	}
	for i := range out.Items {
		a.visibility(ctx, s, &out.Items[i])
	}
	return out, nil
}
func (a *Cash) Current(ctx context.Context, s scope) (*domain.CashShiftView, error) {
	out, err := a.repo.UserShift(ctx, s)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos cargar tu turno de caja.")
	}
	all, err := a.readAllShifts(ctx, s)
	if err != nil {
		return nil, err
	}
	if !all && out.OpenedByID != s.UserID {
		return nil, nil
	}
	out.Movements, err = a.repo.Movements(ctx, s, out.ID)
	if err != nil {
		return nil, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos cargar los movimientos del turno.")
	}
	a.visibility(ctx, s, &out)
	return &out, nil
}
func (a *Cash) UserOptions(ctx context.Context, s scope) ([]domain.CashUserOption, error) {
	out, err := a.repo.UserOptions(ctx, s)
	if err != nil {
		return nil, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos cargar los usuarios disponibles.")
	}
	return out, nil
}
func (a *Cash) Users(ctx context.Context, s scope, id string) ([]domain.CashShiftUserView, error) {
	if err := a.requireShiftRead(ctx, s, id); err != nil {
		return nil, err
	}
	out, err := a.repo.ShiftUsers(ctx, s, id)
	if err != nil {
		return nil, paymentFailure(apperror.Unavailable, "cash_unavailable", "No pudimos cargar el equipo del turno.")
	}
	return out, nil
}
