package httpapi

import "net/http"

// Handlers injects the extracted verticals. Only cmd/api composes production adapters.
type Handlers struct {
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
