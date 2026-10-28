package httpapi

import (
	"context"
	"net/http"
)

// Handlers injects the extracted verticals. Only cmd/api composes production adapters.
type Handlers struct {
	Tips interface {
		GetSettings(http.ResponseWriter, *http.Request)
		UpdateSettings(http.ResponseWriter, *http.Request)
		Workspace(http.ResponseWriter, *http.Request)
		Payout(http.ResponseWriter, *http.Request)
	}
	Concierge              ConciergeHandler
	ConciergeUsage         ConciergeUsageHandler
	Platform               PlatformHandler
	Subscriptions          SubscriptionsHandler
	Organizations          OrganizationsHandler
	Cash                   CashHandler
	Payments               PaymentsHandler
	Orders                 OrdersHandler
	Purchasing             PurchasingHandler
	InventoryStock         InventoryStockHandler
	MenuCatalog            MenuCatalogHandler
	Modules                ModulesHandler
	Expenses               ExpensesHandler
	Reporting              ReportingHandler
	Reservations           ReservationsHandler
	Floor                  FloorHandler
	PaymentMethods         PaymentMethodsHandler
	IdentityAdministration IdentityAdministrationHandler
	Identity               IdentityHandler
	Customers              CustomerHandler
	InventoryUnits         UnitHandler
}
type IdentityHandler interface {
	Login(http.ResponseWriter, *http.Request)
	Logout(http.ResponseWriter, *http.Request)
	Auth(http.Handler) http.Handler
	RequirePermission(string, http.Handler) http.Handler
}
type CustomerHandler interface {
	List(http.ResponseWriter, *http.Request)
	Get(http.ResponseWriter, *http.Request)
	Create(http.ResponseWriter, *http.Request)
	Update(http.ResponseWriter, *http.Request)
	SetActive(http.ResponseWriter, *http.Request)
}
type UnitHandler interface {
	List(http.ResponseWriter, *http.Request)
	Create(http.ResponseWriter, *http.Request)
}

func (a *API) login(w http.ResponseWriter, r *http.Request)  { a.handlers.Identity.Login(w, r) }
func (a *API) logout(w http.ResponseWriter, r *http.Request) { a.handlers.Identity.Logout(w, r) }
func (a *API) auth(next http.Handler) http.Handler           { return a.handlers.Identity.Auth(next) }
func (a *API) requirePermission(permission string, next http.Handler) http.Handler {
	return a.handlers.Identity.RequirePermission(permission, next)
}
func (a *API) listCustomers(w http.ResponseWriter, r *http.Request) { a.handlers.Customers.List(w, r) }
func (a *API) getCustomer(w http.ResponseWriter, r *http.Request)   { a.handlers.Customers.Get(w, r) }
func (a *API) createCustomer(w http.ResponseWriter, r *http.Request) {
	a.handlers.Customers.Create(w, r)
}
func (a *API) updateCustomer(w http.ResponseWriter, r *http.Request) {
	a.handlers.Customers.Update(w, r)
}
func (a *API) updateCustomerStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.Customers.SetActive(w, r)
}
func (a *API) listInventoryUnits(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryUnits.List(w, r)
}
func (a *API) createInventoryUnit(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryUnits.Create(w, r)
}

type InventoryStockHandler interface {
	CreateInventoryTransfer(http.ResponseWriter, *http.Request)
	ListInventoryTransfers(http.ResponseWriter, *http.Request)
	CreatePurchaseInventoryItem(http.ResponseWriter, *http.Request)
	ListLinkableProducts(http.ResponseWriter, *http.Request)
	ListInventory(http.ResponseWriter, *http.Request)
	ListInventoryProducts(http.ResponseWriter, *http.Request)
	ListInventoryMovements(http.ResponseWriter, *http.Request)
	CreateInventoryEntry(http.ResponseWriter, *http.Request)
	CreateInventoryAdjustment(http.ResponseWriter, *http.Request)
	CreateInventoryCombination(http.ResponseWriter, *http.Request)
	SavePurchasePresentation(http.ResponseWriter, *http.Request)
	UpdateInventoryLocationSettings(http.ResponseWriter, *http.Request)
	ListInventoryCombinations(http.ResponseWriter, *http.Request)
}
type PurchasingHandler interface {
	ListSuppliers(http.ResponseWriter, *http.Request)
	ListPurchaseOrders(http.ResponseWriter, *http.Request)
	ListPurchaseReceipts(http.ResponseWriter, *http.Request)
	GetPurchaseOrder(http.ResponseWriter, *http.Request)
	GetPurchaseReceipt(http.ResponseWriter, *http.Request)
	CreateSupplier(http.ResponseWriter, *http.Request)
	UpdateSupplier(http.ResponseWriter, *http.Request)
	CreatePurchaseOrder(http.ResponseWriter, *http.Request)
	UpdatePurchaseOrder(http.ResponseWriter, *http.Request)
	ReceivePurchaseOrder(http.ResponseWriter, *http.Request)
	CreatePurchaseReturn(http.ResponseWriter, *http.Request)
	UpdateSupplierStatus(http.ResponseWriter, *http.Request)
	UpdatePurchaseOrderStatus(http.ResponseWriter, *http.Request)
	ApprovePurchaseOrder(http.ResponseWriter, *http.Request)
}
type OrdersHandler interface {
	CompletePaidOrder(http.ResponseWriter, *http.Request)
	ListKitchenTickets(http.ResponseWriter, *http.Request)
	UpdateKitchenTicketStatus(http.ResponseWriter, *http.Request)
	ListOrders(http.ResponseWriter, *http.Request)
	GetOrdersFloor(http.ResponseWriter, *http.Request)
	GetOrder(http.ResponseWriter, *http.Request)
	CloseOrderBill(http.ResponseWriter, *http.Request)
	DeliverOrderServiceItem(http.ResponseWriter, *http.Request)
	CreateOrder(http.ResponseWriter, *http.Request)
	UpdateOrder(http.ResponseWriter, *http.Request)
	UpdateOrderStatus(http.ResponseWriter, *http.Request)
	AppendOrderItems(http.ResponseWriter, *http.Request)
}
type PaymentsHandler interface {
	RequireCollection(http.Handler) http.Handler
	RequireCollectionRead(http.Handler) http.Handler
	CollectionAccess(http.ResponseWriter, *http.Request)
	CreatePayment(http.ResponseWriter, *http.Request)
	RefundPayment(http.ResponseWriter, *http.Request)
	CreatePaymentBatch(http.ResponseWriter, *http.Request)
	ListPOSOrders(http.ResponseWriter, *http.Request)
	GetPOSOrder(http.ResponseWriter, *http.Request)
}
type CashHandler interface {
	CreateCashRegister(http.ResponseWriter, *http.Request)
	UpdateCashRegister(http.ResponseWriter, *http.Request)
	UpdateCashRegisterStatus(http.ResponseWriter, *http.Request)
	OpenCashShift(http.ResponseWriter, *http.Request)
	CreateCashMovement(http.ResponseWriter, *http.Request)
	CloseCashShift(http.ResponseWriter, *http.Request)
	AssignCashShiftUser(http.ResponseWriter, *http.Request)
	CreateCashOperation(http.ResponseWriter, *http.Request)
	ListCashRegisters(http.ResponseWriter, *http.Request)
	GetCurrentCashShift(http.ResponseWriter, *http.Request)
	ListCashShifts(http.ResponseWriter, *http.Request)
	GetCashShift(http.ResponseWriter, *http.Request)
	GetCashShiftReport(http.ResponseWriter, *http.Request)
	ListCashUserOptions(http.ResponseWriter, *http.Request)
	ListCashShiftUsers(http.ResponseWriter, *http.Request)
	UnassignCashShiftUser(http.ResponseWriter, *http.Request)
}
type OrganizationsHandler interface {
	UpdateOrganization(http.ResponseWriter, *http.Request)
	CreateLocation(http.ResponseWriter, *http.Request)
	UpdateLocation(http.ResponseWriter, *http.Request)
	CreateExchangeRate(http.ResponseWriter, *http.Request)
	UpdateOrgSettings(http.ResponseWriter, *http.Request)
	UpdateRestaurantSetup(http.ResponseWriter, *http.Request)
	CreateFiscalProfile(http.ResponseWriter, *http.Request)
	UpdateFiscalProfile(http.ResponseWriter, *http.Request)
	GetOrganization(http.ResponseWriter, *http.Request)
	GetOrgSettings(http.ResponseWriter, *http.Request)
	GetRestaurantSetup(http.ResponseWriter, *http.Request)
	CompleteRestaurantSetup(http.ResponseWriter, *http.Request)
	DeactivateLocation(http.ResponseWriter, *http.Request)
	DeactivateFiscalProfile(http.ResponseWriter, *http.Request)
	DeleteExchangeRate(http.ResponseWriter, *http.Request)
	GetPlatformOnboardingCatalogs(http.ResponseWriter, *http.Request)
	ListLocations(http.ResponseWriter, *http.Request)
	ListFiscalProfiles(http.ResponseWriter, *http.Request)
	ListExchangeRates(http.ResponseWriter, *http.Request)
}
type SubscriptionsHandler interface {
	CreateSubscriptionPlan(http.ResponseWriter, *http.Request)
	UpdateSubscriptionPlan(http.ResponseWriter, *http.Request)
	UpdateOrganizationSubscription(http.ResponseWriter, *http.Request)
	RecordSubscriptionPayment(http.ResponseWriter, *http.Request)
	ListSubscriptionPlans(http.ResponseWriter, *http.Request)
	ListAvailableSubscriptionPlans(http.ResponseWriter, *http.Request)
	GetOrganizationSubscription(http.ResponseWriter, *http.Request)
}
type PlatformHandler interface {
	Authorize(context.Context, string) error
	OnboardTenant(http.ResponseWriter, *http.Request)
	ListPlatformOrganizations(http.ResponseWriter, *http.Request)
	GetPlatformOrganization(http.ResponseWriter, *http.Request)
	ListPlatformWhatsAppChannels(http.ResponseWriter, *http.Request)
	CreatePlatformWhatsAppChannel(http.ResponseWriter, *http.Request)
	UpdatePlatformWhatsAppChannel(http.ResponseWriter, *http.Request)
}
type ConciergeUsageHandler interface {
	OpenConciergeConversation(http.ResponseWriter, *http.Request)
	ReportConciergeUsage(http.ResponseWriter, *http.Request)
	GetConciergeUsageSettings(http.ResponseWriter, *http.Request)
	UpdateConciergeUsageSettings(http.ResponseWriter, *http.Request)
	ListConciergeUsage(http.ResponseWriter, *http.Request)
	GetOrganizationConciergeUsage(http.ResponseWriter, *http.Request)
	UpdateOrganizationConciergeLimit(http.ResponseWriter, *http.Request)
}
type ConciergeHandler interface {
	ServiceAuth(http.Handler) http.Handler
	CreateConciergeOrder(http.ResponseWriter, *http.Request)
	RequestConciergeBill(http.ResponseWriter, *http.Request)
	RequestConciergeHandoff(http.ResponseWriter, *http.Request)
	GetConciergeHandoffStatus(http.ResponseWriter, *http.Request)
	ListOperationalConciergeHandoffs(http.ResponseWriter, *http.Request)
	ResolveOperationalConciergeHandoff(http.ResponseWriter, *http.Request)
	GetConciergeSettings(http.ResponseWriter, *http.Request)
	ListConciergeMenu(http.ResponseWriter, *http.Request)
	GetConciergeCombo(http.ResponseWriter, *http.Request)
	GetConciergeProductModifiers(http.ResponseWriter, *http.Request)
	GetConciergeDeliveryOptions(http.ResponseWriter, *http.Request)
	QuoteConciergeDelivery(http.ResponseWriter, *http.Request)
	CreateConciergeDeliveryOrder(http.ResponseWriter, *http.Request)
	GetConciergeOrderStatus(http.ResponseWriter, *http.Request)
	GetConciergeDeliveryFees(http.ResponseWriter, *http.Request)
	SaveConciergeDeliveryFees(http.ResponseWriter, *http.Request)
	QuoteAdminDelivery(http.ResponseWriter, *http.Request)
}
