package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"math"
	"strconv"
	"strings"
)

type TipRepository interface {
	TipPolicy(context.Context, scope, bool) (domain.TipPolicy, error)
	Shift(context.Context, scope, string) (domain.CashShiftView, error)
	SaveTipSettings(context.Context, scope, bool, domain.TipSettingsInput) error
	TipForOrder(context.Context, scope, string) (*domain.TipView, error)
	TipOrderContext(context.Context, scope, string) (string, float64, error)
	InsertTip(context.Context, scope, string, string, string, float64, domain.TipInput, float64, bool) error
	TipRecipients(context.Context, scope) ([]domain.TipRecipient, error)
	CanReadAllShifts(context.Context, scope) (bool, error)
	OwnsShift(context.Context, scope, string) (bool, error)
	RequireShiftAssignment(context.Context, scope, string) error
	LockMovementShift(context.Context, scope, string) (string, error)
	ExpectedCash(context.Context, scope, string) (float64, error)
	TipReport(context.Context, scope, string) (domain.TipReport, error)
	TipPending(context.Context, scope, bool) (string, error)
	LockTipBalances(context.Context, scope, bool) ([]domain.TipBalance, error)
	FindTipPayout(context.Context, scope, string) (domain.TipPayoutView, error)
	InsertTipPayout(context.Context, scope, string, domain.TipPayoutInput, []domain.TipBalance) (domain.TipPayoutView, error)
}
type Tips struct {
	repo TipRepository
	work unitofwork.Manager
}

func NewTips(repo TipRepository, work unitofwork.Manager) *Tips { return &Tips{repo, work} }
func (a *Payments) WithTips(tips *Tips) *Payments               { a.tips = tips; return a }

func (a *Tips) Policy(ctx context.Context, s scope, global bool) (domain.TipPolicy, error) {
	p, err := a.repo.TipPolicy(ctx, s, global)
	if err != nil {
		return p, paymentFailure(apperror.Unavailable, "tips_unavailable", "No pudimos cargar la configuración de propinas.")
	}
	p.ChannelOptions = []domain.TipChannelOption{{Value: "salon", Label: "Salón"}, {Value: "recojo", Label: "Recojo"}, {Value: "delivery", Label: "Delivery"}, {Value: "whatsapp", Label: "WhatsApp"}}
	return p, nil
}
func (a *Tips) SavePolicy(ctx context.Context, s scope, global bool, in domain.TipSettingsInput) (domain.TipPolicy, error) {
	var out domain.TipPolicy
	admin, err := a.repo.CanReadAllShifts(ctx, s)
	if err != nil {
		return out, err
	}
	if !admin {
		return out, paymentFailure(apperror.Forbidden, "tips_settings_forbidden", "Solo el administrador puede configurar propinas.")
	}
	if _, err := a.Policy(ctx, s, global); err != nil {
		return out, err
	}
	valid := in.Mode == "disabled" || in.Mode == "suggest" || in.Mode == "collect"
	valid = valid && !math.IsNaN(in.Percentage) && !math.IsInf(in.Percentage, 0) && in.Percentage >= 0 && in.Percentage <= 100 && len(in.Channels) > 0
	valid = valid && math.Abs(in.Percentage*100-math.Round(in.Percentage*100)) < .00001
	for _, channel := range in.Channels {
		valid = valid && (channel == "salon" || channel == "recojo" || channel == "delivery" || channel == "whatsapp")
	}
	if (!in.Inherit || global) && !valid {
		return out, paymentFailure(apperror.Invalid, "invalid_tip_settings", "Revisa el porcentaje, el modo y los tipos de atención de las propinas.")
	}
	if global && in.Inherit {
		return out, paymentFailure(apperror.Invalid, "invalid_tip_settings", "La configuración del restaurante no puede heredar de un local.")
	}
	if err = a.repo.SaveTipSettings(ctx, s, global, in); err != nil {
		return out, paymentFailure(apperror.Unavailable, "tips_unavailable", "No pudimos guardar la configuración de propinas.")
	}
	return a.Policy(ctx, s, global)
}

// Called inside the payment transaction, after the order has been locked and authorized.
// A unique order key prevents a second tip when consumption is paid in instalments.
func (a *Tips) RecordPayment(ctx context.Context, s scope, orderID, paymentID, shiftID string, in *domain.TipInput, affectsCash bool) error {
	if in == nil {
		return nil
	}
	p, err := a.Policy(ctx, s, false)
	if err != nil {
		return err
	}
	channel, base, err := a.repo.TipOrderContext(ctx, s, orderID)
	if err != nil {
		return err
	}
	enabled := false
	for _, c := range p.Channels {
		enabled = enabled || c == channel
	}
	if p.Mode != "collect" || !enabled {
		return paymentFailure(apperror.Invalid, "tips_disabled", "La gestión de propinas no está habilitada para esta atención.")
	}
	if in.Kind == "fixed" && !p.AllowFixed {
		return paymentFailure(apperror.Invalid, "fixed_tip_disabled", "El local no permite propinas por monto fijo.")
	}
	amount, valid := domain.TipAmount(*in, base, p.Decimals)
	if !valid {
		return paymentFailure(apperror.Invalid, "invalid_tip", "Ingresa una propina válida en la moneda del local.")
	}
	existing, err := a.repo.TipForOrder(ctx, s, orderID)
	if err != nil {
		return err
	}
	if existing != nil {
		return paymentFailure(apperror.Conflict, "tip_already_recorded", "La propina de esta cuenta ya fue registrada.")
	}
	if err = a.repo.InsertTip(ctx, s, orderID, paymentID, shiftID, base, *in, amount, affectsCash); err != nil {
		return paymentFailure(apperror.Unavailable, "tips_unavailable", "No pudimos registrar la propina. El cobro no se confirmó.")
	}
	return nil
}

type TipWorkspace struct {
	Report     domain.TipReport      `json:"report"`
	Available  string                `json:"available"`
	Recipients []domain.TipRecipient `json:"recipients"`
}

func (a *Tips) Workspace(ctx context.Context, s scope, shiftID string) (TipWorkspace, error) {
	var out TipWorkspace
	all, err := a.repo.CanReadAllShifts(ctx, s)
	if err != nil {
		return out, err
	}
	if !all {
		owned, e := a.repo.OwnsShift(ctx, s, shiftID)
		if e != nil {
			return out, e
		}
		if !owned {
			return out, paymentFailure(apperror.NotFound, "cash_shift_not_found", "El turno no está disponible para tu usuario.")
		}
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{RepeatableRead: true, ReadOnly: true})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if _, err = a.repo.Shift(tx.Context(), s, shiftID); errors.Is(err, ErrNotFound) {
		return out, paymentFailure(apperror.NotFound, "cash_shift_not_found", "El turno no existe en este local.")
	} else if err != nil {
		return out, err
	}
	out.Report, err = a.repo.TipReport(tx.Context(), s, shiftID)
	if err != nil {
		return out, err
	}
	out.Available, err = a.repo.TipPending(tx.Context(), s, all)
	if err != nil {
		return out, err
	}
	out.Recipients, err = a.repo.TipRecipients(tx.Context(), s)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}
func (a *Tips) Payout(ctx context.Context, s scope, shiftID string, in domain.TipPayoutInput) (domain.TipPayoutView, error) {
	var out domain.TipPayoutView
	in.Note = strings.TrimSpace(in.Note)
	in.RequestKey = strings.TrimSpace(in.RequestKey)
	if !domain.ValidMoney(in.Amount, 2) || in.RecipientID == "" || (in.Method != "cash" && in.Method != "transfer") || len(in.Note) > 240 || len(in.RequestKey) < 8 || len(in.RequestKey) > 100 {
		return out, paymentFailure(apperror.Invalid, "invalid_tip_payout", "Revisa el destinatario, monto y medio de entrega de la propina.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	status, err := a.repo.LockMovementShift(tx.Context(), s, shiftID)
	if errors.Is(err, ErrNotFound) {
		return out, paymentFailure(apperror.NotFound, "cash_shift_not_found", "El turno no existe en este local.")
	}
	if err != nil {
		return out, err
	}
	if err = a.repo.RequireShiftAssignment(tx.Context(), s, shiftID); err != nil {
		return out, paymentFailure(apperror.Forbidden, "cash_shift_not_assigned", "Debes estar asignado al turno para entregar propinas.")
	}
	all, err := a.repo.CanReadAllShifts(tx.Context(), s)
	if err != nil {
		return out, err
	}
	if !all {
		owned, e := a.repo.OwnsShift(tx.Context(), s, shiftID)
		if e != nil {
			return out, e
		}
		if !owned {
			return out, paymentFailure(apperror.Forbidden, "tip_payout_forbidden", "Solo puedes entregar propinas desde tu propio turno.")
		}
	}
	previous, err := a.repo.FindTipPayout(tx.Context(), s, in.RequestKey)
	if err == nil {
		amount, _ := strconv.ParseFloat(previous.Amount, 64)
		if previous.ShiftID != shiftID || previous.CreatedByID != s.UserID || previous.RecipientID != in.RecipientID || previous.Method != in.Method || math.Abs(amount-in.Amount) > .00001 || previous.Note != in.Note {
			return out, paymentFailure(apperror.Conflict, "tip_payout_request_conflict", "La solicitud ya se utilizó para otra entrega.")
		}
		return previous, tx.Commit(ctx)
	}
	if !errors.Is(err, ErrNotFound) {
		return out, err
	}
	if status != "open" {
		return out, paymentFailure(apperror.Conflict, "cash_shift_closed", "El turno está cerrado.")
	}
	p, err := a.Policy(tx.Context(), s, false)
	if err != nil {
		return out, err
	}
	if !domain.ValidMoney(in.Amount, p.Decimals) {
		return out, paymentFailure(apperror.Invalid, "invalid_tip_payout", "El monto no respeta los decimales de la moneda del local.")
	}
	recipients, err := a.repo.TipRecipients(tx.Context(), s)
	if err != nil {
		return out, err
	}
	valid := false
	for _, recipient := range recipients {
		valid = valid || recipient.ID == in.RecipientID
	}
	if !valid {
		return out, paymentFailure(apperror.Invalid, "invalid_tip_recipient", "Selecciona un trabajador del local.")
	}
	balances, err := a.repo.LockTipBalances(tx.Context(), s, all)
	if err != nil {
		return out, err
	}
	available := 0.0
	for _, b := range balances {
		available += b.Available
	}
	if in.Amount > available+.00001 {
		return out, paymentFailure(apperror.Conflict, "tip_payout_exceeds_pending", "La entrega supera las propinas pendientes disponibles.")
	}
	if in.Method == "cash" {
		expected, e := a.repo.ExpectedCash(tx.Context(), s, shiftID)
		if e != nil {
			return out, e
		}
		if in.Amount > expected+.00001 {
			return out, paymentFailure(apperror.Conflict, "tip_payout_exceeds_cash", "No hay suficiente efectivo esperado en este turno.")
		}
	}
	out, err = a.repo.InsertTipPayout(tx.Context(), s, shiftID, in, balances)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}
