package application

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	einvoicing "github.com/foods-platform/foods-backend/internal/modules/einvoicing/contracts"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

// DepositRepository guarda la garantía de reserva como cobro del turno, sin
// pedido hasta que se aplica a la cuenta o se retiene.
type DepositRepository interface {
	LockDepositReservation(ctx context.Context, s scope, reservationID string) (domain.DepositReservation, error)
	CreateDepositPayment(ctx context.Context, s scope, reservationID, shiftID string, in domain.DepositInput, amount float64) (string, error)
	DepositMovement(ctx context.Context, s scope, shiftID, sourceID, movement string, amount float64, reason, note string) error
	DepositMethodName(ctx context.Context, s scope, method string) (string, error)
	LockHeldDeposit(ctx context.Context, s scope, reservationID string) (domain.HeldDeposit, error)
	LockHeldDepositPayment(ctx context.Context, s scope, paymentID string) (domain.HeldDeposit, error)
	AttachDeposit(ctx context.Context, s scope, paymentID, orderID, outcome string) error
	SeatDepositReservation(ctx context.Context, s scope, reservationID string) error
	CreateRetainedDepositOrder(ctx context.Context, s scope, d domain.HeldDeposit) (string, error)
	DepositOptions(ctx context.Context, s scope, orderID string) ([]domain.DepositOption, error)
	ForfeitedDeposits(ctx context.Context, limit int) ([]domain.ForfeitedDeposit, error)
}

// WithDeposits habilita el cobro de garantías de reserva.
func (a *Payments) WithDeposits(repo DepositRepository) *Payments { a.deposits = repo; return a }

func depositsUnavailable() error {
	return paymentFailure(apperror.Unavailable, "deposits_unavailable", "No pudimos registrar la garantía.")
}

func depositReason(customer string) string {
	if customer = strings.TrimSpace(customer); customer != "" {
		return "Garantía de reserva · " + customer
	}
	return "Garantía de reserva"
}

// ChargeDeposit cobra la garantía de una reserva pendiente o confirmada en el
// turno de caja del usuario: entra al turno con su medio de pago y, si es
// efectivo, al cuadre.
func (a *Payments) ChargeDeposit(ctx context.Context, s scope, reservationID string, in domain.DepositInput) (domain.DepositReceipt, error) {
	var out domain.DepositReceipt
	if a.deposits == nil {
		return out, depositsUnavailable()
	}
	in.Reference = strings.TrimSpace(in.Reference)
	if strings.TrimSpace(reservationID) == "" || len(in.Reference) > 120 {
		return out, paymentFailure(apperror.Invalid, "invalid_deposit", "Revisa los datos de la garantía.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return out, depositsUnavailable()
	}
	defer tx.Rollback(ctx)
	method, err := a.methods.Find(tx.Context(), s.OrganizationID, in.Method, "sales")
	if errors.Is(err, ErrNotFound) {
		return out, paymentFailure(apperror.Invalid, "invalid_payment_method", "Selecciona un medio de pago activo.")
	}
	if err != nil {
		return out, paymentFailure(apperror.Unavailable, "payment_methods_unavailable", "No pudimos validar el medio de pago.")
	}
	shiftID, err := a.collectionPaymentShift(tx.Context(), s, in.ShiftID)
	if err != nil {
		return out, err
	}
	reservation, err := a.deposits.LockDepositReservation(tx.Context(), s, reservationID)
	if errors.Is(err, ErrNotFound) {
		return out, paymentFailure(apperror.NotFound, "reservation_not_found", "La reserva no existe en este local.")
	}
	if err != nil {
		return out, depositsUnavailable()
	}
	switch {
	case reservation.Amount <= 0:
		return out, paymentFailure(apperror.Conflict, "reservation_without_deposit", "La reserva no tiene garantía.")
	case reservation.Charged:
		return out, paymentFailure(apperror.Conflict, "deposit_already_charged", "La garantía de esta reserva ya se cobró.")
	case reservation.Status != "pending" && reservation.Status != "confirmed":
		return out, paymentFailure(apperror.Conflict, "reservation_not_open", "Solo se cobra la garantía de una reserva pendiente o confirmada.")
	}
	id, err := a.deposits.CreateDepositPayment(tx.Context(), s, reservationID, shiftID, in, reservation.Amount)
	if errors.Is(err, ErrConflict) {
		return out, paymentFailure(apperror.Conflict, "deposit_already_charged", "La garantía de esta reserva ya se cobró.")
	}
	if err != nil {
		return out, depositsUnavailable()
	}
	if method.AffectsCash {
		if err := a.deposits.DepositMovement(tx.Context(), s, shiftID, id, "income", reservation.Amount, depositReason(reservation.CustomerName), in.Reference); err != nil {
			return out, paymentFailure(apperror.Unavailable, "deposits_unavailable", "No pudimos reflejar la garantía en caja.")
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return out, depositsUnavailable()
	}
	_ = a.audit.Record(ctx, s, "reservation_deposit.charged", "payment", id)
	return domain.DepositReceipt{PaymentID: id, Amount: strconv.FormatFloat(reservation.Amount, 'f', 2, 64), MethodName: method.Name}, nil
}

// applyDeposit descuenta una garantía cobrada de la cuenta del pedido dentro
// de la transacción del cobro: el cobro de la garantía pasa al pedido y la
// reserva queda sentada. Devuelve el importe descontado.
func (a *Payments) applyDeposit(ctx context.Context, s scope, orderID, paymentID string, remaining float64) (float64, error) {
	if a.deposits == nil {
		return 0, depositsUnavailable()
	}
	deposit, err := a.deposits.LockHeldDepositPayment(ctx, s, paymentID)
	if errors.Is(err, ErrNotFound) {
		return 0, paymentFailure(apperror.Conflict, "deposit_not_available", "La garantía ya no está disponible para descontar.")
	}
	if err != nil {
		return 0, depositsUnavailable()
	}
	if deposit.ReservationStatus != "pending" && deposit.ReservationStatus != "confirmed" && deposit.ReservationStatus != "seated" {
		return 0, paymentFailure(apperror.Conflict, "deposit_not_available", "La garantía ya no está disponible para descontar.")
	}
	if deposit.Amount > remaining+0.00001 {
		return 0, paymentFailure(apperror.Conflict, "deposit_exceeds_remaining", "La garantía supera el saldo de la cuenta. Cobra la cuenta completa y devuelve la garantía desde Reservas.")
	}
	if err := a.deposits.AttachDeposit(ctx, s, deposit.PaymentID, orderID, domain.DepositApplied); err != nil {
		return 0, depositsUnavailable()
	}
	if err := a.deposits.SeatDepositReservation(ctx, s, deposit.ReservationID); err != nil {
		return 0, depositsUnavailable()
	}
	return deposit.Amount, nil
}

// DepositOptions lista las garantías que la cuenta del pedido puede descontar.
func (a *Payments) depositOptions(ctx context.Context, s scope, orderID string) []domain.DepositOption {
	if a.deposits == nil {
		return []domain.DepositOption{}
	}
	options, err := a.deposits.DepositOptions(ctx, s, orderID)
	if err != nil {
		// El cobro sigue sin garantías, pero el fallo debe quedar visible en el log.
		slog.Default().Warn("deposit_options_failed", "order", orderID, "error", err)
		return []domain.DepositOption{}
	}
	return options
}

// HeldDeposit indica si la reserva tiene una garantía cobrada que aún espera al
// cliente (ni descontada, ni retenida, ni devuelta).
func (a *Payments) HeldDeposit(ctx context.Context, s scope, reservationID string) (bool, error) {
	if a.deposits == nil {
		return false, nil
	}
	_, err := a.deposits.LockHeldDeposit(ctx, s, reservationID)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, depositsUnavailable()
	}
	return true, nil
}

// RefundHeldDeposit devuelve la garantía al cliente desde el turno de caja del
// usuario. Corre dentro de la transacción de quien llama.
func (a *Payments) RefundHeldDeposit(ctx context.Context, s scope, reservationID string) error {
	if a.deposits == nil {
		return depositsUnavailable()
	}
	deposit, err := a.deposits.LockHeldDeposit(ctx, s, reservationID)
	if errors.Is(err, ErrNotFound) {
		return paymentFailure(apperror.Conflict, "deposit_not_available", "La reserva no tiene una garantía por devolver.")
	}
	if err != nil {
		return depositsUnavailable()
	}
	shiftID, err := a.repo.CurrentShift(ctx, s)
	if errors.Is(err, ErrNotFound) {
		return paymentFailure(apperror.Conflict, "cash_shift_required", "Debes estar asignado a un turno de caja abierto para devolver la garantía.")
	}
	if err != nil {
		return paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos validar tu turno de caja.")
	}
	refund := domain.RefundInput{Amount: deposit.Amount, Reason: "Devolución de garantía de reserva"}
	refundID, err := a.repo.CreateRefund(ctx, s, deposit.PaymentID, refund, shiftID)
	if err != nil {
		return depositsUnavailable()
	}
	if deposit.AffectsCash {
		if err := a.deposits.DepositMovement(ctx, s, shiftID, refundID, "expense", deposit.Amount, "Devolución de garantía · "+strings.TrimSpace(deposit.CustomerName), ""); err != nil {
			return paymentFailure(apperror.Unavailable, "deposits_unavailable", "No pudimos reflejar la devolución en caja.")
		}
	}
	_ = a.audit.Record(ctx, s, "reservation_deposit.refunded", "payment", deposit.PaymentID)
	return nil
}

// RefundDeposit devuelve la garantía en su propia transacción (Reservas, con la
// reserva todavía abierta o ya sentada).
func (a *Payments) RefundDeposit(ctx context.Context, s scope, reservationID string) error {
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return depositsUnavailable()
	}
	defer tx.Rollback(ctx)
	if err := a.RefundHeldDeposit(tx.Context(), s, reservationID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return depositsUnavailable()
	}
	return nil
}

// RetainHeldDeposit deja la garantía en el local: la registra como venta de
// mostrador «Garantía de reserva no utilizada» y le pasa el cobro. Corre dentro
// de la transacción de quien llama y devuelve el pedido creado; la boleta se
// pide después del commit con IssueRetainedReceipt.
func (a *Payments) RetainHeldDeposit(ctx context.Context, s scope, reservationID string) (string, error) {
	if a.deposits == nil {
		return "", depositsUnavailable()
	}
	deposit, err := a.deposits.LockHeldDeposit(ctx, s, reservationID)
	if errors.Is(err, ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", depositsUnavailable()
	}
	orderID, err := a.deposits.CreateRetainedDepositOrder(ctx, s, deposit)
	if err != nil {
		return "", depositsUnavailable()
	}
	if err := a.deposits.AttachDeposit(ctx, s, deposit.PaymentID, orderID, domain.DepositRetained); err != nil {
		return "", depositsUnavailable()
	}
	_ = a.audit.Record(ctx, s, "reservation_deposit.retained", "order", orderID)
	return orderID, nil
}

// IssueRetainedReceipt pide la boleta de una garantía retenida, ya confirmada.
// Es de mejor esfuerzo: sin facturación electrónica activa y configurada, o
// por encima del monto que exige documento del cliente, la garantía queda
// retenida sin boleta.
func (a *Payments) IssueRetainedReceipt(ctx context.Context, s scope, orderID string) {
	if a.fiscal == nil || orderID == "" {
		return
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)
	order, err := a.orders.PaymentOrder(tx.Context(), s, orderID)
	if err != nil {
		return
	}
	ref, err := a.fiscal.Request(tx.Context(), s, orderID, einvoicing.FiscalRequest{
		DocumentType: einvoicing.DocumentReceipt,
		Customer:     einvoicing.Customer{Name: strings.TrimSpace(order.CustomerName)},
	})
	if err != nil || ref.ID == "" {
		slog.Default().Info("retained_deposit_receipt_skipped", "order", orderID, "reason", errorText(err))
		return
	}
	if err := tx.Commit(ctx); err != nil {
		return
	}
	a.fiscal.Dispatch(ref.ID)
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// RetainBatch acota cuántas garantías retiene cada barrido.
const RetainBatch = 50

// RetainForfeitedDeposits retiene las garantías de reservas que terminaron sin
// el cliente («No asistió» o canceladas sin devolverla) y pide su boleta.
func (a *Payments) RetainForfeitedDeposits(ctx context.Context) (int, error) {
	if a.deposits == nil {
		return 0, nil
	}
	pending, err := a.deposits.ForfeitedDeposits(ctx, RetainBatch)
	if err != nil {
		return 0, err
	}
	retained := 0
	for _, item := range pending {
		s := scope{OrganizationID: item.OrganizationID, LocationID: item.LocationID}
		tx, err := a.work.Begin(ctx, unitofwork.Options{})
		if err != nil {
			return retained, err
		}
		orderID, err := a.RetainHeldDeposit(tx.Context(), s, item.ReservationID)
		if err != nil || orderID == "" {
			_ = tx.Rollback(ctx)
			continue
		}
		if err := tx.Commit(ctx); err != nil {
			_ = tx.Rollback(ctx)
			continue
		}
		retained++
		a.IssueRetainedReceipt(ctx, s, orderID)
	}
	return retained, nil
}
