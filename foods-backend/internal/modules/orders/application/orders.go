package application

import (
	"context"
	"errors"
	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

type scope = requestscope.Scope

var ErrNotFound = errors.New("order record not found")

type Orders struct {
	stock OrderStock
	repo  OrderRepository
	work  unitofwork.Manager
	audit audit.Repository
}

func New(repo OrderRepository, work unitofwork.Manager, audit audit.Repository, stock OrderStock) *Orders {
	return &Orders{repo: repo, work: work, audit: audit, stock: stock}
}
func orderFailure(kind apperror.Kind, code, message string) error {
	return apperror.New(kind, code, message)
}

type OrderRepository interface {
	CreateQROrder(ctx context.Context, s scope, tableID string, in QROrderInput, subtotal float64, customerID string) (domain.Order, error)
	// RegisterConciergeCustomer prerregistra al comensal de WhatsApp por su número en la empresa del pedido.
	RegisterConciergeCustomer(ctx context.Context, s scope, phone string, name string) (string, error)
	LockQRTable(ctx context.Context, s scope, tableID string, token string) (string, error)
	QRRequestOrder(ctx context.Context, s scope, tableID string, in QROrderInput) (string, error)
	LockQRTableOrder(ctx context.Context, s scope, tableID string) (string, string, string, float64, float64, *string, bool, bool, error)
	QROrderTotal(ctx context.Context, s scope, openOrderID string) (float64, error)
	UpdateQRSubtotal(ctx context.Context, s scope, openOrderID string, nextSubtotal float64, nextTotal float64) (int64, error)
	SaveQRAppendRequest(ctx context.Context, s scope, tableID string, in QROrderInput, openOrderID string) (int64, error)
	AuditQRAppend(ctx context.Context, s scope, tableID string, in QROrderInput, openOrderID string, openChannel string) (int64, error)
	SaveQRCreateRequest(ctx context.Context, s scope, tableID string, in QROrderInput, out domain.Order) (int64, error)
	AuditQRCreated(ctx context.Context, s scope, tableID string, in QROrderInput, out domain.Order) (int64, error)
	OpenTableOrder(context.Context, scope, string) (domain.Order, error)
	LockConciergeDeliveryRequest(ctx context.Context, s scope, requestID string) error
	ConciergeDeliveryRequestOrder(ctx context.Context, s scope, requestID string) (string, error)
	CreateConciergeDeliveryOrder(ctx context.Context, s scope, in ConciergeDeliveryInput, subtotal float64, total float64, customerID string) (domain.Order, error)
	SaveConciergeDeliveryRequest(ctx context.Context, s scope, in ConciergeDeliveryInput, orderID string) error
	AuditConciergeDeliveryCreated(ctx context.Context, s scope, in ConciergeDeliveryInput, orderID string) error
	ConciergeOrderActive(ctx context.Context, s scope, conversationID string, orderID string) (bool, error)
	LockPaidCompletion(ctx context.Context, s scope, id string) (float64, string, bool, error)
	CompletionPaid(ctx context.Context, s scope, id string) (float64, error)
	CompletionPending(ctx context.Context, s scope, id string) (bool, error)
	DeliverPaidItems(ctx context.Context, s scope, id string) (int64, error)
	CompletePaidOrder(ctx context.Context, s scope, id string) (int64, error)
	LegacyKitchenOrders(context.Context, scope, string, string) ([]domain.Order, error)
	KitchenRounds(context.Context, scope, string, string) ([]domain.KitchenTicket, error)
	ServiceTickets(context.Context, scope, string, string) ([]domain.KitchenTicket, error)
	TargetMinutes(context.Context, scope, string, string) (*int, error)
	LockStationOrder(ctx context.Context, s scope, id string, destination string) (string, error)
	StationStatus(ctx context.Context, s scope, orderID string, id string, destination string) (string, error)
	UpdateStationItems(ctx context.Context, s scope, orderID string, id string, destination string, status string, current string) (int64, error)
	UpdateStationRound(ctx context.Context, s scope, id string) (int64, error)
	LockHistoricalRound(ctx context.Context, s scope, id string) (string, string, error)
	UpdateHistoricalRound(ctx context.Context, s scope, id string, in OrderStatusInput) (int64, error)
	LockHistoricalOrder(ctx context.Context, s scope, id string) (string, error)
	UpdateHistoricalOrder(ctx context.Context, s scope, id string, in OrderStatusInput) (int64, error)
	KitchenRoundCounts(ctx context.Context, s scope, orderID string) (int, int, int, error)
	SyncHistoricalKitchenStatus(ctx context.Context, s scope, orderID string, next string) (int64, error)
	LockServiceOrder(context.Context, scope, string) (domain.Order, error)
	CreateKitchenRound(context.Context, scope, string, string, string) (string, int, error)
	AssignOrphanItems(context.Context, scope, string, string) (int64, error)
	UndeliveredItems(context.Context, scope, string) (bool, error)
	CloseBill(ctx context.Context, s scope, o domain.Order) (int64, error)
	DeliverItem(ctx context.Context, s scope, o domain.Order, itemID string) (int64, error)
	AppendAlreadyExists(ctx context.Context, s scope, o domain.Order, in AppendInput) (bool, error)
	AccountPaid(ctx context.Context, s scope, o domain.Order) (bool, error)
	IdentifyRound(ctx context.Context, s scope, roundID string, in AppendInput) (int64, error)
	ReopenForAppend(ctx context.Context, s scope, o domain.Order) (int64, error)
	AddOrderTotal(ctx context.Context, s scope, o domain.Order, added float64) (int64, error)
	List(context.Context, scope, Filter) (Page, error)
	Floor(context.Context, scope) ([]domain.FloorTable, error)
	CreateOrder(ctx context.Context, s scope, in domain.OrderInput, customerID *string, tableID *string, subtotal float64, total float64, initialStatus string) (domain.Order, error)
	CustomerActive(ctx context.Context, s scope, in domain.OrderInput) (bool, error)
	LocationPoint(ctx context.Context, s scope) (latitude *float64, longitude *float64, err error)
	LockTable(ctx context.Context, s scope, in domain.OrderInput) (string, error)
	TableOccupied(ctx context.Context, s scope, in domain.OrderInput) (bool, error)
	LockEditableOrder(ctx context.Context, s scope, id string) (string, string, string, error)
	HasServiceItems(ctx context.Context, s scope, id string) (bool, error)
	HasKitchenRounds(ctx context.Context, s scope, id string) (bool, error)
	UpdateOrder(ctx context.Context, s scope, id string, in domain.OrderUpdateInput, subtotal float64, total float64) (int64, error)
	DeleteItems(ctx context.Context, s scope, id string) (int64, error)
	LockOrderStatus(ctx context.Context, s scope, id string) (string, string, string, float64, bool, string, error)
	ServiceStarted(ctx context.Context, s scope, id string) (bool, bool, error)
	SubmitServiceItems(ctx context.Context, s scope, id string) (int64, error)
	PendingServiceItems(ctx context.Context, s scope, id string) (bool, error)
	DeliverServiceItems(ctx context.Context, s scope, id string) (int64, error)
	SetStatus(ctx context.Context, s scope, id string, in OrderStatusInput) (int64, error)
	Items(ctx context.Context, s scope, orderID, roundID string) ([]domain.OrderItem, error)
	ServiceItems(ctx context.Context, s scope, orderID string) ([]domain.OrderServiceItem, error)
	NetPaid(ctx context.Context, s scope, orderID string) (float64, error)
	QuantityUsage(ctx context.Context, s scope, orderID string) (map[string]float64, error)
	InsertItems(ctx context.Context, s scope, orderID, roundID string, items []domain.PreparedOrderItem) error
	SyncServiceStatus(ctx context.Context, s scope, id string) error
	CompleteDeliveredAccount(ctx context.Context, s scope, orderID string) (bool, error)
	Record(context.Context, scope, string) (domain.Order, error)
	ExistingSelections(context.Context, scope, string) ([]domain.PreparedOrderSelection, error)
	ComboGroups(context.Context, scope, string) (map[string]*domain.ComboGroup, []string, error)
	ModifierGroups(context.Context, scope, string) (map[string]*domain.ModifierGroup, []string, error)
	// OrderProducts precarga en una consulta los productos activos del local; los
	// que no devuelve se validan uno por uno con ProductIsCombo/ProductForOrder.
	OrderProducts(ctx context.Context, s scope, productIDs []string) (map[string]domain.OrderProductSnapshot, error)
	ProductIsCombo(ctx context.Context, s scope, productID *string) (bool, error)
	ProductForOrder(ctx context.Context, s scope, productID *string) (string, float64, bool, error)
	ExistingComboItem(ctx context.Context, s scope, in domain.OrderItemInput, existingOrderID string) (string, float64, string, string, error)
	ComboForOrder(ctx context.Context, s scope, productID *string) (string, float64, bool, error)
	ExistingOptionUsage(ctx context.Context, s scope, existingOrderID string, key domain.ComboOptionUsageKey) (float64, error)
	LockOptionQuota(ctx context.Context, s scope, key domain.ComboOptionUsageKey) (*int, string, error)
	ExternalOptionUsage(ctx context.Context, s scope, key domain.ComboOptionUsageKey, existingOrderID string) (float64, error)
}
