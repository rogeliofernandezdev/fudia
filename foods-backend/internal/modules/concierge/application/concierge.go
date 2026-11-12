package application

import (
	"context"
	"errors"
	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	"github.com/foods-platform/foods-backend/internal/modules/concierge/domain"
	customercontracts "github.com/foods-platform/foods-backend/internal/modules/customers/contracts"
	deliverycontracts "github.com/foods-platform/foods-backend/internal/modules/delivery/contracts"
	menucontracts "github.com/foods-platform/foods-backend/internal/modules/menu/contracts"
	noticecontracts "github.com/foods-platform/foods-backend/internal/modules/notifications/contracts"
	ordercontracts "github.com/foods-platform/foods-backend/internal/modules/orders/contracts"
	orgcontracts "github.com/foods-platform/foods-backend/internal/modules/organizations/contracts"
	platformcontracts "github.com/foods-platform/foods-backend/internal/modules/platformadmin/contracts"
	servicecontracts "github.com/foods-platform/foods-backend/internal/modules/service/contracts"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"math"
	"strings"
)

var ErrNotFound = errors.New("concierge record not found")

type scope = requestscope.Scope
type QRContext struct {
	Scope                                                                           scope
	TableID, TableName, OrganizationName, LocationName, CurrencySymbol, CountryCode string
	// ServiceType decide el flujo del QR: mesa con cuenta abierta, mostrador
	// o enlace de delivery del local (sin mesa: solo delivery o recojo).
	ServiceType servicecontracts.ServiceType
}
type MenuResult struct {
	orgcontracts.RestaurantBranding
	Items          []menucontracts.PublicMenuItem `json:"items"`
	CurrencySymbol string                         `json:"currencySymbol"`
}
type Menu interface {
	PublicMenu(context.Context, scope, string, string) ([]menucontracts.PublicMenuItem, error)
	PublicCombo(context.Context, scope, string) (menucontracts.PublicComboDetail, error)
	PublicModifiers(context.Context, scope, string) (menucontracts.ModifierConfigView, error)
}
type Orders interface {
	SubmitTable(context.Context, scope, string, string, ordercontracts.QROrderInput) (ordercontracts.TableSubmission, error)
	SubmitCounter(ctx context.Context, s scope, counterTableID, token string, in ordercontracts.QROrderInput) (ordercontracts.TableSubmission, error)
	TableBill(context.Context, scope, string) (ordercontracts.Order, error)
	ConciergeDeliveryReplay(ctx context.Context, s scope, requestID string) (ordercontracts.Order, bool, error)
	SubmitConciergeDelivery(ctx context.Context, s scope, in ordercontracts.ConciergeDeliveryInput) (ordercontracts.TableSubmission, error)
	ConciergeOrder(ctx context.Context, s scope, conversationID, orderID string) (ordercontracts.ConciergeOrderState, error)
	StaleTableAccount(ctx context.Context, s scope, tableID string) (ordercontracts.StaleTableAccount, error)
}
type Concierge struct {
	channels  platformcontracts.WhatsAppRoutes
	directory RestaurantRepository
	work      unitofwork.Manager
	repo      Repository
	menu      Menu
	orders    Orders
	audit     audit.Repository
	branding  orgcontracts.BrandingReader
	customers Customers
	notices   noticecontracts.Publisher
}

func New(repo Repository, menu Menu, orders Orders, audit audit.Repository) *Concierge {
	return &Concierge{repo: repo, menu: menu, orders: orders, audit: audit}
}
func (a *Concierge) WithBranding(reader orgcontracts.BrandingReader) *Concierge {
	a.branding = reader
	return a
}

// Customers reconoce al comensal por su número y guarda su consentimiento promocional.
type Customers interface {
	WhatsAppProfile(ctx context.Context, s scope, phone string) (customercontracts.WhatsAppProfile, error)
	RecordWhatsAppConsent(ctx context.Context, s scope, in customercontracts.ConsentEvidence) (customercontracts.WhatsAppProfile, error)
}

func (a *Concierge) WithCustomers(customers Customers) *Concierge {
	a.customers = customers
	return a
}

// WithNotices avisa al personal de pedidos, cuentas y llamados que llegan por WhatsApp.
func (a *Concierge) WithNotices(notices noticecontracts.Publisher) *Concierge {
	a.notices = notices
	return a
}

// notify publica un aviso después de registrar el evento; un fallo no afecta al comensal.
func (a *Concierge) notify(ctx context.Context, s scope, notice noticecontracts.Notice) {
	if a.notices != nil {
		_ = a.notices.Publish(ctx, s, notice)
	}
}

// productCount suma las cantidades pedidas para el aviso al personal.
func productCount(items []ordercontracts.OrderItemInput) int {
	total := 0.0
	for _, item := range items {
		total += item.Qty
	}
	return int(math.Round(total))
}
func conciergeFailure(k apperror.Kind, c, m string) error { return apperror.New(k, c, m) }

type Bill struct {
	OrderID         string                     `json:"orderId"`
	Code            string                     `json:"code"`
	TableName       string                     `json:"tableName"`
	Status          string                     `json:"status"`
	CurrencySymbol  string                     `json:"currencySymbol"`
	Items           []ordercontracts.OrderItem `json:"items"`
	Total           string                     `json:"total"`
	PaidAmount      string                     `json:"paidAmount"`
	RemainingAmount string                     `json:"remainingAmount"`
	PaymentStatus   string                     `json:"paymentStatus"`
}
type Repository interface {
	ResolveQR(context.Context, string) (QRContext, error)
	RecordBillRequest(ctx context.Context, s scope, qr QRContext, out ordercontracts.Order, in domain.BillRequestInput) (int64, error)
	Settings(ctx context.Context, s scope) (domain.SettingsView, error)
	Handoff(ctx context.Context, s scope, id string, qr QRContext) (domain.HandoffView, error)
	HandoffStatus(ctx context.Context, s scope, conversationID string, qr QRContext) (domain.HandoffView, error)
	RequestHandoff(ctx context.Context, s scope, qr QRContext, in domain.HandoffInput) (string, error)
	Handoffs(context.Context, scope, string) ([]domain.HandoffView, error)
	ResolveHandoff(ctx context.Context, s scope, id string) (string, error)
	DeliverySetup(ctx context.Context, s scope) (DeliverySetup, error)
	DeliveryFees(ctx context.Context, s scope) (domain.DeliveryFeesView, []deliverycontracts.FeeTier, error)
	ReplaceDeliveryFees(ctx context.Context, s scope, tiers []deliverycontracts.FeeTier) error
}

// errDeliveryLinkOnly: el enlace de delivery del local solo toma pedidos de
// delivery o recojo (delivery-orders); no hay mesa, cuenta ni llamado al mozo.
var errDeliveryLinkOnly = conciergeFailure(apperror.Conflict, "delivery_link_only",
	"Este enlace es para pedidos de delivery o recojo en el local.")

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
	if err == nil && a.branding != nil && q == "" && productID == "" {
		branding, brandErr := a.branding.Branding(ctx, qr.Scope)
		if brandErr == nil {
			out.RestaurantBranding = branding
		}
	}
	return out, err
}
func (a *Concierge) Combo(ctx context.Context, token, id string) (menucontracts.PublicComboDetail, error) {
	qr, err := a.resolve(ctx, token)
	if err != nil {
		return menucontracts.PublicComboDetail{}, err
	}
	return a.menu.PublicCombo(ctx, qr.Scope, id)
}
func (a *Concierge) Modifiers(ctx context.Context, token, id string) (menucontracts.ModifierConfigView, error) {
	qr, err := a.resolve(ctx, token)
	if err != nil {
		return menucontracts.ModifierConfigView{}, err
	}
	return a.menu.PublicModifiers(ctx, qr.Scope, id)
}

// Submit es el único punto que decide el flujo de un pedido por QR según el
// tipo de atención: la mesa conserva su cuenta abierta, ocupación y rondas; el
// mostrador crea un pedido independiente por cliente que va a preparación y a
// la bandeja de caja, sin ocupar la mesa ni exigir mozo.
func (a *Concierge) Submit(ctx context.Context, token string, in ordercontracts.QROrderInput) (ordercontracts.TableSubmission, error) {
	qr, err := a.resolve(ctx, token)
	if err != nil {
		return ordercontracts.TableSubmission{}, err
	}
	if qr.ServiceType == servicecontracts.ServiceTypeDelivery {
		return ordercontracts.TableSubmission{}, errDeliveryLinkOnly
	}
	if qr.ServiceType == servicecontracts.ServiceTypeCounter {
		out, err := a.orders.SubmitCounter(ctx, qr.Scope, qr.TableID, token, in)
		if err == nil {
			o := out.Order
			a.notify(ctx, qr.Scope, noticecontracts.CounterOrder(qr.TableName, o.ID, o.Code, o.CustomerName, productCount(in.Items), noticecontracts.Money(qr.CurrencySymbol, o.Total)))
		}
		return out, err
	}
	out, err := a.orders.SubmitTable(ctx, qr.Scope, qr.TableID, token, in)
	var failure *apperror.Error
	if errors.As(err, &failure) && failure.Code == ordercontracts.StaleTableAccountCode {
		a.warnStaleAccount(ctx, qr)
	}
	if err == nil {
		// Mismo criterio de idempotencia que Orders: sin requestId cuenta la conversación.
		requestID := strings.TrimSpace(in.RequestID)
		if requestID == "" {
			requestID = strings.TrimSpace(in.ConversationID)
		}
		a.notify(ctx, qr.Scope, noticecontracts.TableOrder(qr.TableID, qr.TableName, out.Order.ID, strings.TrimSpace(in.CustomerName), productCount(in.Items), requestID))
	}
	return out, err
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
	in.RequestID = strings.TrimSpace(in.RequestID)
	if in.ConversationID == "" || len(in.ConversationID) > 160 || len(in.CustomerPhone) > 32 || len(in.RequestID) > 160 {
		return result, conciergeFailure(apperror.Invalid, "invalid_bill_request", "Revisa los datos de la solicitud de cuenta.")
	}
	if qr.ServiceType == servicecontracts.ServiceTypeCounter {
		return result, conciergeFailure(apperror.Conflict, "counter_has_no_bill", "En el mostrador no hay cuenta abierta: cada pedido se paga en caja.")
	}
	if qr.ServiceType == servicecontracts.ServiceTypeDelivery {
		return result, errDeliveryLinkOnly
	}

	out, err := a.orders.TableBill(ctx, qr.Scope, qr.TableID)
	if err != nil {
		return result, err
	}

	recorded, err := a.repo.RecordBillRequest(ctx, s, qr, out, in)
	if err != nil {
		return result, conciergeFailure(apperror.Unavailable, "bill_unavailable", "No pudimos registrar la solicitud de cuenta.")
	}
	// Un reintento con el mismo requestId no se registra de nuevo ni vuelve a avisar.
	if recorded > 0 {
		a.notify(ctx, s, noticecontracts.BillRequested(qr.TableID, qr.TableName, out.ID, noticecontracts.Money(qr.CurrencySymbol, out.RemainingAmount), in.RequestID))
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
	if qr.ServiceType == servicecontracts.ServiceTypeDelivery {
		return result, errDeliveryLinkOnly
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
	a.notify(ctx, s, noticecontracts.AssistanceRequested(qr.TableID, qr.TableName, out.Reason, id))
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
	if qr.ServiceType == servicecontracts.ServiceTypeDelivery {
		return nil, nil
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
