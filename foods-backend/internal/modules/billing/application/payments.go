package application

import (
	"context"
	"errors"
	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	ordercontracts "github.com/foods-platform/foods-backend/internal/modules/orders/contracts"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"math"
	"strconv"
	"strings"
	"time"
)

type scope = requestscope.Scope
type POSFilter struct {
	Query, From, To, Status string
	Page, Size              int
}
type POSPage struct {
	Items    []domain.PosOrderSummary `json:"items"`
	Total    int                      `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"pageSize"`
}
type POSOrderDetail struct {
	ReceiptContext  domain.PaymentReceiptContext `json:"receiptContext"`
	Order           ordercontracts.Order            `json:"order"`
	PaidAmount      string                       `json:"paidAmount"`
	RemainingAmount string                       `json:"remainingAmount"`
	PaymentStatus   string                       `json:"paymentStatus"`
	Payments        []domain.PaymentView         `json:"payments"`
}
type PaymentOrders interface {
	RequireDelivered(context.Context, scope, string, string) error
	CompleteDeliveredAccount(context.Context, scope, string) (bool, error)
	PaymentOrder(context.Context, scope, string) (ordercontracts.Order, error)
}
type Payments struct {
	repo    PaymentRepository
	work    unitofwork.Manager
	audit   audit.Repository
	methods ExpenseMethods
	orders  PaymentOrders
}

func NewPayments(repo PaymentRepository, work unitofwork.Manager, audit audit.Repository, methods ExpenseMethods, orders PaymentOrders) *Payments {
	return &Payments{repo, work, audit, methods, orders}
}
func paymentFailure(k apperror.Kind, c, m string) error { return apperror.New(k, c, m) }
func validPOSDates(from, to string) bool {
	for _, d := range []string{from, to} {
		if d != "" {
			if _, err := time.Parse("2006-01-02", d); err != nil {
				return false
			}
		}
	}
	return true
}

type PaymentRepository interface {
	CurrentShift(ctx context.Context, s scope) (string, error)
	ExpectedCash(ctx context.Context, s scope, shiftID string) (float64, error)
	RequireShiftAssignment(ctx context.Context, s scope, shiftID string) error
	OrderPayments(ctx context.Context, s scope, orderID string) ([]domain.PaymentView, error)
	Payment(ctx context.Context, s scope, id string) (domain.PaymentView, error)
	LockPaymentOrder(ctx context.Context, s scope, in domain.PaymentInput) (string, string, float64, bool, bool, bool, error)
	PaidForOrder(ctx context.Context, s scope, in domain.PaymentInput) (float64, error)
	CreatePayment(ctx context.Context, s scope, in domain.PaymentInput, shiftID string) (string, error)
	SaleMovement(ctx context.Context, s scope, in domain.PaymentInput, shiftID string, id string, orderCode string) (int64, error)
	LockRefundPayment(ctx context.Context, s scope, id string) (bool, string, string, float64, bool, bool, float64, error)
	CreateRefund(ctx context.Context, s scope, id string, in domain.RefundInput, shiftID string) (string, error)
	RefundMovement(ctx context.Context, s scope, in domain.RefundInput, shiftID string, refundID string, orderCode string) (int64, error)
	LockBatchOrder(ctx context.Context, s scope, in domain.PaymentBatchInput) (string, string, float64, bool, bool, bool, error)
	BatchPaid(ctx context.Context, s scope, in domain.PaymentBatchInput) (float64, error)
	CreateBatchPayment(ctx context.Context, s scope, in domain.PaymentBatchInput, shiftID string, p domain.PaymentLine) (string, error)
	BatchSaleMovement(ctx context.Context, s scope, shiftID string, id string, p domain.PaymentLine, orderCode string) (int64, error)
	POSOrders(ctx context.Context, s scope, f POSFilter) (POSPage, error)
	ReceiptContext(ctx context.Context, s scope) (domain.PaymentReceiptContext, error)
}

func (a *Payments) Create(ctx context.Context, s scope, in domain.PaymentInput) (domain.PaymentView, error) {
	var result domain.PaymentView
	if strings.TrimSpace(in.OrderID) == "" || in.Amount <= 0 {
		return result, paymentFailure(apperror.Invalid, "invalid_payment", "Revisa los datos del cobro.")
	}
	in.OrderID = strings.TrimSpace(in.OrderID)
	in.Reference = strings.TrimSpace(in.Reference)
	if len(in.Reference) > 120 {
		return result, paymentFailure(apperror.Invalid, "invalid_payment", "La referencia no puede superar 120 caracteres.")
	}

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos iniciar el cobro.")
	}
	defer tx.Rollback(ctx)

	methodDef, err := a.methods.Find(tx.Context(), s.OrganizationID, in.Method, "sales")
	if errors.Is(err, ErrNotFound) {
		return result, paymentFailure(apperror.Invalid, "invalid_payment_method", "Selecciona un medio de pago activo.")
	}
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "payment_methods_unavailable", "No pudimos validar el medio de pago.")
	}

	shiftID, err := a.repo.CurrentShift(tx.Context(), s)
	if errors.Is(err, ErrNotFound) {
		return result, paymentFailure(apperror.Conflict, "cash_shift_required", "Debes estar asignado a un turno de caja abierto para cobrar.")
	}
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos validar tu turno de caja.")
	}

	var orderCode, status string
	var completed, salon, billClosed bool
	var total, paid float64
	err = func() error {
		var err error
		orderCode, status, total, completed, salon, billClosed, err = a.repo.LockPaymentOrder(tx.Context(), s, in)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		return result, paymentFailure(apperror.NotFound, "order_not_found", "El pedido no existe en este local.")
	}
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos validar el pedido.")
	}
	if err = func() error { var err error; paid, err = a.repo.PaidForOrder(tx.Context(), s, in); return err }(); err != nil {
		return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos validar el saldo pendiente.")
	}
	if completed || status == "cancelado" || (salon && !billClosed) || (!salon && status != "listo" && status != "en_camino" && status != "entregado") {
		return result, paymentFailure(apperror.Conflict, "order_not_ready_for_payment", "Cierra la cuenta de la mesa antes de cobrar. El pedido debe seguir abierto.")
	}
	remaining := math.Max(0, total-paid)
	if salon {
		if err := a.orders.RequireDelivered(tx.Context(), s, in.OrderID, status); err != nil {
			return result, err
		}
	}
	if in.Amount > remaining+0.00001 {
		return result, paymentFailure(apperror.Conflict, "payment_exceeds_remaining", "El cobro supera el saldo pendiente del pedido.")
	}

	var id string
	err = func() error { var err error; id, err = a.repo.CreatePayment(tx.Context(), s, in, shiftID); return err }()
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos registrar el cobro.")
	}

	if methodDef.AffectsCash {
		if _, err := a.repo.SaleMovement(tx.Context(), s, in, shiftID, id, orderCode); err != nil {
			return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos reflejar el cobro en caja.")
		}
	}
	closed, err := a.orders.CompleteDeliveredAccount(tx.Context(), s, in.OrderID)
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos finalizar la cuenta del pedido.")
	}
	// Read the result before committing so a failed projection cannot leave a
	// persisted charge behind while the client receives a retryable error.
	item, err := a.repo.Payment(tx.Context(), s, id)
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos verificar el detalle del cobro.")
	}
	if err := tx.Commit(ctx); err != nil {
		return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos confirmar el cobro.")
	}
	if closed {
		_ = a.audit.Record(ctx, s, "order.completed", "order", in.OrderID)
	}
	return item, nil

}
func (a *Payments) Refund(ctx context.Context, s scope, id string, in domain.RefundInput) error {
	if in.Amount <= 0 {
		return paymentFailure(apperror.Invalid, "invalid_refund", "Ingresa un monto válido para la devolución.")
	}
	in.Reason = strings.TrimSpace(in.Reason)
	in.Note = strings.TrimSpace(in.Note)
	if in.Reason == "" || len(in.Reason) > 120 || len(in.Note) > 240 {
		return paymentFailure(apperror.Invalid, "invalid_refund", "Indica el motivo de la devolución.")
	}

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos iniciar la devolución.")
	}
	defer tx.Rollback(ctx)

	shiftID, err := a.repo.CurrentShift(tx.Context(), s)
	if errors.Is(err, ErrNotFound) {
		return paymentFailure(apperror.Conflict, "cash_shift_required", "Debes estar asignado a un turno de caja abierto para devolver un pago.")
	}
	if err != nil {
		return paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos validar tu turno de caja.")
	}

	var affectsCash bool
	var orderCode, orderStatus string
	var completed, salon bool
	var amount, refunded float64
	err = func() error {
		var err error
		affectsCash, orderCode, orderStatus, amount, completed, salon, refunded, err = a.repo.LockRefundPayment(tx.Context(), s, id)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		return paymentFailure(apperror.NotFound, "payment_not_found", "El pago no existe en este local.")
	}
	if err != nil {
		return paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos validar el pago.")
	}
	if completed || (orderStatus == "entregado" && !salon) || orderStatus == "cancelado" {
		return paymentFailure(apperror.Conflict, "closed_order_refund_requires_void", "El pedido ya está cerrado. Usa un flujo de anulación o devolución posterior al cierre para no reabrir su saldo operativo.")
	}
	if in.Amount > amount-refunded+0.00001 {
		return paymentFailure(apperror.Conflict, "refund_exceeds_payment", "La devolución supera el saldo disponible del pago.")
	}
	if affectsCash {
		expected, err := a.repo.ExpectedCash(tx.Context(), s, shiftID)
		if err != nil {
			return paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos validar el efectivo disponible.")
		}
		if in.Amount > expected+0.00001 {
			return paymentFailure(apperror.Conflict, "refund_exceeds_expected", "La devolución supera el efectivo esperado del turno.")
		}
	}

	var refundID string
	err = func() error {
		var err error
		refundID, err = a.repo.CreateRefund(tx.Context(), s, id, in, shiftID)
		return err
	}()
	if err != nil {
		return paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos registrar la devolución.")
	}

	if affectsCash {
		if _, err := a.repo.RefundMovement(tx.Context(), s, in, shiftID, refundID, orderCode); err != nil {
			return paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos reflejar la devolución en caja.")
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos confirmar la devolución.")
	}
	return nil

}
func (a *Payments) CreateBatch(ctx context.Context, s scope, in domain.PaymentBatchInput) (map[string]any, error) {
	var result map[string]any
	if strings.TrimSpace(in.OrderID) == "" || len(in.Payments) == 0 {
		return result, paymentFailure(apperror.Invalid, "invalid_payment", "Revisa los datos del cobro.")
	}
	in.OrderID = strings.TrimSpace(in.OrderID)
	totalBatch := 0.0
	for i := range in.Payments {
		p := &in.Payments[i]
		p.Method = strings.TrimSpace(p.Method)
		p.Reference = strings.TrimSpace(p.Reference)
		if p.Amount <= 0 || p.Method == "" {
			return result, paymentFailure(apperror.Invalid, "invalid_payment", "Todos los medios de pago deben tener método y monto válidos.")
		}
		if len(p.Reference) > 120 {
			return result, paymentFailure(apperror.Invalid, "invalid_payment", "La referencia no puede superar 120 caracteres.")
		}
		totalBatch += p.Amount
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos iniciar el cobro.")
	}
	defer tx.Rollback(ctx)

	methodDefs := map[string]domain.PaymentMethodView{}
	for _, p := range in.Payments {
		if _, ok := methodDefs[p.Method]; ok {
			continue
		}
		methodDef, lookupErr := a.methods.Find(tx.Context(), s.OrganizationID, p.Method, "sales")
		if errors.Is(lookupErr, ErrNotFound) {
			return result, paymentFailure(apperror.Invalid, "invalid_payment_method", "Uno de los medios de pago no está activo para ventas.")
		}
		if lookupErr != nil {
			return result, paymentFailure(apperror.Unavailable, "payment_methods_unavailable", "No pudimos validar los medios de pago.")
		}
		methodDefs[p.Method] = methodDef
	}

	shiftID, err := a.repo.CurrentShift(tx.Context(), s)
	if errors.Is(err, ErrNotFound) {
		return result, paymentFailure(apperror.Conflict, "cash_shift_required", "Debes estar asignado a un turno de caja abierto para cobrar.")
	}
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos validar tu turno de caja.")
	}

	var orderCode, status string
	var completed, salon, billClosed bool
	var orderTotal, paid float64
	err = func() error {
		var err error
		orderCode, status, orderTotal, completed, salon, billClosed, err = a.repo.LockBatchOrder(tx.Context(), s, in)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		return result, paymentFailure(apperror.NotFound, "order_not_found", "El pedido no existe en este local.")
	}
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos validar el pedido.")
	}
	if err = func() error { var err error; paid, err = a.repo.BatchPaid(tx.Context(), s, in); return err }(); err != nil {
		return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos validar el saldo pendiente.")
	}
	if completed || status == "cancelado" || (salon && !billClosed) || (!salon && status != "listo" && status != "en_camino" && status != "entregado") {
		return result, paymentFailure(apperror.Conflict, "order_not_ready_for_payment", "Cierra la cuenta de la mesa antes de cobrar. El pedido debe seguir abierto.")
	}
	remaining := math.Max(0, orderTotal-paid)
	if salon {
		if err := a.orders.RequireDelivered(tx.Context(), s, in.OrderID, status); err != nil {
			return result, err
		}
	}
	if totalBatch > remaining+0.00001 {
		return result, paymentFailure(apperror.Conflict, "payment_exceeds_remaining", "El cobro supera el saldo pendiente del pedido.")
	}

	ids := make([]string, 0, len(in.Payments))
	for _, p := range in.Payments {
		var id string
		err = func() error {
			var err error
			id, err = a.repo.CreateBatchPayment(tx.Context(), s, in, shiftID, p)
			return err
		}()
		if err != nil {
			return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos registrar uno de los medios de pago.")
		}
		ids = append(ids, id)
		if methodDefs[p.Method].AffectsCash {
			if _, err = a.repo.BatchSaleMovement(tx.Context(), s, shiftID, id, p, orderCode); err != nil {
				return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos reflejar el cobro en caja.")
			}
		}
	}
	closed, err := a.orders.CompleteDeliveredAccount(tx.Context(), s, in.OrderID)
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos finalizar la cuenta del pedido.")
	}
	if err = tx.Commit(ctx); err != nil {
		return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos confirmar el cobro.")
	}
	if closed {
		_ = a.audit.Record(ctx, s, "order.completed", "order", in.OrderID)
	}
	return map[string]any{
		"paymentIds":      ids,
		"paidNow":         strconv.FormatFloat(totalBatch, 'f', 2, 64),
		"remainingAmount": strconv.FormatFloat(math.Max(0, remaining-totalBatch), 'f', 2, 64),
		"paymentStatus":   domain.PaymentStatus(orderTotal, paid+totalBatch),
	}, nil

}
func (a *Payments) List(ctx context.Context, s scope, f POSFilter) (POSPage, error) {
	var result POSPage
	q, from, to := strings.TrimSpace(f.Query), strings.TrimSpace(f.From), strings.TrimSpace(f.To)
	if !validPOSDates(from, to) || strings.HasPrefix(from, "0000-") || strings.HasPrefix(to, "0000-") {
		return result, paymentFailure(apperror.Invalid, "invalid_order_date_filter", "Ingresa fechas válidas con el formato AAAA-MM-DD.")
	}
	if from != "" && to != "" && from > to {
		return result, paymentFailure(apperror.Invalid, "invalid_order_date_filter", "La fecha final no puede ser anterior a la inicial.")
	}
	status := strings.TrimSpace(f.Status)
	if status == "" {
		status = "unpaid"
	}
	if status != "unpaid" && status != "pending" && status != "partial" && status != "paid" && status != "all" {
		return result, paymentFailure(apperror.Invalid, "invalid_payment_status", "El estado de cobro no es válido.")
	}

	f.Query = q
	f.From = from
	f.To = to
	f.Status = status
	out, err := a.repo.POSOrders(ctx, s, f)
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos cargar los pedidos por cobrar.")
	}
	return out, nil
}
func (a *Payments) Get(ctx context.Context, s scope, id string) (POSOrderDetail, error) {
	var result POSOrderDetail
	o, err := a.orders.PaymentOrder(ctx, s, id)
	if err != nil {
		return result, err
	}
	payments, err := a.repo.OrderPayments(ctx, s, o.ID)
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos cargar los pagos del pedido.")
	}
	paid := 0.0
	for _, p := range payments {
		value, _ := strconv.ParseFloat(p.NetAmount, 64)
		paid += value
	}
	total, _ := strconv.ParseFloat(o.Total, 64)
	remaining := math.Max(0, total-paid)
	var identity domain.PaymentReceiptContext
	err = func() error { var err error; identity, err = a.repo.ReceiptContext(ctx, s); return err }()
	if err != nil {
		return result, paymentFailure(apperror.Unavailable, "payments_unavailable", "No pudimos cargar los datos del restaurante.")
	}
	return POSOrderDetail{
		ReceiptContext:  identity,
		Order:           o,
		PaidAmount:      strconv.FormatFloat(paid, 'f', 2, 64),
		RemainingAmount: strconv.FormatFloat(remaining, 'f', 2, 64),
		PaymentStatus:   domain.PaymentStatus(total, paid),
		Payments:        payments,
	}, nil

}
