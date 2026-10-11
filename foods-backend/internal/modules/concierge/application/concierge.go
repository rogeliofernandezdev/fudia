package application

import (
	"context"
	"errors"
	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	"github.com/foods-platform/foods-backend/internal/modules/concierge/domain"
	menudomain "github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	orderapp "github.com/foods-platform/foods-backend/internal/modules/orders/application"
	orderdomain "github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"strings"
)

var ErrNotFound = errors.New("concierge record not found")

type scope = requestscope.Scope
type QRContext struct {
	Scope                                                              scope
	TableID, TableName, OrganizationName, LocationName, CurrencySymbol string
}
type MenuResult struct {
	Items          []menudomain.PublicMenuItem `json:"items"`
	CurrencySymbol string                      `json:"currencySymbol"`
}
type Menu interface {
	PublicMenu(context.Context, scope, string, string) ([]menudomain.PublicMenuItem, error)
	PublicCombo(context.Context, scope, string) (menudomain.PublicComboDetail, error)
	PublicModifiers(context.Context, scope, string) (menudomain.ModifierConfigView, error)
}
type Orders interface {
	SubmitTable(context.Context, scope, string, string, orderapp.QROrderInput) (orderapp.TableSubmission, error)
	TableBill(context.Context, scope, string) (orderdomain.Order, error)
}
type Concierge struct {
	repo   Repository
	menu   Menu
	orders Orders
	audit  audit.Repository
}

func New(repo Repository, menu Menu, orders Orders, audit audit.Repository) *Concierge {
	return &Concierge{repo, menu, orders, audit}
}
func conciergeFailure(k apperror.Kind, c, m string) error { return apperror.New(k, c, m) }

type Bill struct {
	OrderID         string                  `json:"orderId"`
	Code            string                  `json:"code"`
	TableName       string                  `json:"tableName"`
	Status          string                  `json:"status"`
	CurrencySymbol  string                  `json:"currencySymbol"`
	Items           []orderdomain.OrderItem `json:"items"`
	Total           string                  `json:"total"`
	PaidAmount      string                  `json:"paidAmount"`
	RemainingAmount string                  `json:"remainingAmount"`
	PaymentStatus   string                  `json:"paymentStatus"`
}
type Repository interface {
	ResolveQR(context.Context, string) (QRContext, error)
	RecordBillRequest(ctx context.Context, s scope, qr QRContext, out orderdomain.Order, in domain.BillRequestInput) (int64, error)
	Settings(ctx context.Context, s scope) (domain.SettingsView, error)
	Handoff(ctx context.Context, s scope, id string, qr QRContext) (domain.HandoffView, error)
	HandoffStatus(ctx context.Context, s scope, conversationID string, qr QRContext) (domain.HandoffView, error)
	RequestHandoff(ctx context.Context, s scope, qr QRContext, in domain.HandoffInput) (string, error)
	Handoffs(context.Context, scope, string) ([]domain.HandoffView, error)
	ResolveHandoff(ctx context.Context, s scope, id string) (string, error)
}

func (a *Concierge) resolve(ctx context.Context, token string) (QRContext, error) {
	qr, err := a.repo.ResolveQR(ctx, token)
	if errors.Is(err, ErrNotFound) {
		return qr, conciergeFailure(apperror.NotFound, "table_not_found", "La mesa no existe o el QR no está activo.")
	}
	if err != nil {
		return qr, conciergeFailure(apperror.Unavailable, "concierge_unavailable", "No pudimos resolver el QR.")
	}
	return qr, nil
}
func (a *Concierge) Menu(ctx context.Context, token, q, productID string) (MenuResult, error) {
	var out MenuResult
	qr, err := a.resolve(ctx, token)
	if err != nil {
		return out, err
	}
	out.CurrencySymbol = qr.CurrencySymbol
	out.Items, err = a.menu.PublicMenu(ctx, qr.Scope, q, productID)
	return out, err
}
func (a *Concierge) Combo(ctx context.Context, token, id string) (menudomain.PublicComboDetail, error) {
	qr, err := a.resolve(ctx, token)
	if err != nil {
		return menudomain.PublicComboDetail{}, err
	}
	return a.menu.PublicCombo(ctx, qr.Scope, id)
}
func (a *Concierge) Modifiers(ctx context.Context, token, id string) (menudomain.ModifierConfigView, error) {
	qr, err := a.resolve(ctx, token)
	if err != nil {
		return menudomain.ModifierConfigView{}, err
	}
	return a.menu.PublicModifiers(ctx, qr.Scope, id)
}
func (a *Concierge) Submit(ctx context.Context, token string, in orderapp.QROrderInput) (orderapp.TableSubmission, error) {
	qr, err := a.resolve(ctx, token)
	if err != nil {
		return orderapp.TableSubmission{}, err
	}
	return a.orders.SubmitTable(ctx, qr.Scope, qr.TableID, token, in)
}
func (a *Concierge) Bill(ctx context.Context, token string, in domain.BillRequestInput) (Bill, error) {
	var result Bill
	qr, err := a.resolve(ctx, token)
	if err != nil {
		return result, err
	}
	s := qr.Scope
	in.ConversationID = strings.TrimSpace(in.ConversationID)
	in.CustomerPhone = strings.TrimSpace(in.CustomerPhone)
	if in.ConversationID == "" || len(in.ConversationID) > 160 || len(in.CustomerPhone) > 32 {
		return result, conciergeFailure(apperror.Invalid, "invalid_bill_request", "Revisa los datos de la solicitud de cuenta.")
	}

	out, err := a.orders.TableBill(ctx, qr.Scope, qr.TableID)
	if err != nil {
		return result, err
	}

	if _, err = a.repo.RecordBillRequest(ctx, s, qr, out, in); err != nil {
		return result, conciergeFailure(apperror.Unavailable, "bill_unavailable", "No pudimos registrar la solicitud de cuenta.")
	}

	return Bill{
		OrderID:         out.ID,
		Code:            out.Code,
		TableName:       qr.TableName,
		Status:          out.Status,
		CurrencySymbol:  qr.CurrencySymbol,
		Items:           out.Items,
		Total:           out.Total,
		PaidAmount:      out.PaidAmount,
		RemainingAmount: out.RemainingAmount,
		PaymentStatus:   out.PaymentStatus,
	}, nil

}
func (a *Concierge) Settings(ctx context.Context, s scope) (domain.SettingsView, error) {
	var result domain.SettingsView
	var out domain.SettingsView
	err := func() error { var err error; out, err = a.repo.Settings(ctx, s); return err }()
	if err != nil {
		return result, conciergeFailure(apperror.Unavailable, "concierge_settings_unavailable", "No pudimos cargar el estado de Fudia Concierge.")
	}
	out.ManagedByPlatform = true
	return out, nil

}
func (a *Concierge) Handoff(ctx context.Context, token string, in domain.HandoffInput) (domain.HandoffView, error) {
	var result domain.HandoffView
	qr, err := a.resolve(ctx, token)
	if err != nil {
		return result, err
	}
	s := qr.Scope
	in.ConversationID = strings.TrimSpace(in.ConversationID)
	in.CustomerPhone = strings.TrimSpace(in.CustomerPhone)
	in.Reason = strings.TrimSpace(in.Reason)
	if in.ConversationID == "" || len(in.ConversationID) > 160 || len(in.CustomerPhone) > 32 || len([]rune(in.Reason)) > 240 {
		return result, conciergeFailure(apperror.Invalid, "invalid_handoff", "Revisa la solicitud de atención.")
	}
	if in.Reason == "" {
		in.Reason = "El comensal solicitó atención del personal."
	}

	var id string
	err = func() error { var err error; id, err = a.repo.RequestHandoff(ctx, s, qr, in); return err }()
	if err != nil {
		return result, conciergeFailure(apperror.Unavailable, "handoff_unavailable", "No pudimos avisar al personal.")
	}

	out, err := a.repo.Handoff(ctx, s, id, qr)
	if err != nil {
		return result, conciergeFailure(apperror.Unavailable, "handoff_unavailable", "No pudimos recuperar la solicitud de atención.")
	}
	return out, nil

}
func (a *Concierge) HandoffStatus(ctx context.Context, token, conversationID string) (*domain.HandoffView, error) {
	qr, err := a.resolve(ctx, token)
	if err != nil {
		return nil, err
	}
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" || len(conversationID) > 160 {
		return nil, conciergeFailure(apperror.Invalid, "invalid_handoff", "La conversación no es válida.")
	}
	out, err := a.repo.HandoffStatus(ctx, qr.Scope, conversationID, qr)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, conciergeFailure(apperror.Unavailable, "handoff_unavailable", "No pudimos consultar la solicitud de atención.")
	}
	return &out, nil
}
func (a *Concierge) Handoffs(ctx context.Context, s scope, status string) ([]domain.HandoffView, error) {
	status = strings.TrimSpace(status)
	if status == "" {
		status = "pending"
	}
	if status != "pending" && status != "resolved" {
		return nil, conciergeFailure(apperror.Invalid, "invalid_status", "El estado de atención no es válido.")
	}
	out, err := a.repo.Handoffs(ctx, s, status)
	if err != nil {
		return nil, conciergeFailure(apperror.Unavailable, "handoffs_unavailable", "No pudimos cargar las solicitudes de atención.")
	}
	return out, nil
}
func (a *Concierge) ResolveHandoff(ctx context.Context, s scope, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return conciergeFailure(apperror.Invalid, "invalid_handoff", "La solicitud de atención no es válida.")
	}

	var resolvedID string
	err := func() error { var err error; resolvedID, err = a.repo.ResolveHandoff(ctx, s, id); return err }()
	if errors.Is(err, ErrNotFound) {
		return conciergeFailure(apperror.NotFound, "handoff_not_found", "La solicitud ya fue atendida o no existe.")
	}
	if err != nil {
		return conciergeFailure(apperror.Unavailable, "handoff_unavailable", "No pudimos cerrar la solicitud de atención.")
	}
	_ = a.audit.Record(ctx, s, "concierge.handoff.resolved", "concierge_handoff", resolvedID)
	return nil

}
