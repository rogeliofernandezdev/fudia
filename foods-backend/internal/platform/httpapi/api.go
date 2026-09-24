package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type API struct{ db *pgxpool.Pool }
type scope struct{ UserID, OrganizationID, LocationID, Name string }
type scopeKey struct{}
type apiError struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlationId"`
}

func New(db *pgxpool.Pool) *API { return &API{db: db} }
func (a *API) Routes() *http.ServeMux {
	m := http.NewServeMux()
	m.HandleFunc("POST /v1/auth/login", a.login)
	m.HandleFunc("POST /v1/auth/logout", a.logout)
	m.Handle("POST /v1/platform/organizations", a.auth(a.requirePlatformAdmin(http.HandlerFunc(a.onboardTenant))))
	m.Handle("GET /v1/platform/plans", a.auth(a.requirePlatformAdmin(http.HandlerFunc(a.listSubscriptionPlans))))
	m.Handle("POST /v1/platform/plans", a.auth(a.requirePlatformAdmin(http.HandlerFunc(a.createSubscriptionPlan))))
	m.Handle("PATCH /v1/platform/plans/{id}", a.auth(a.requirePlatformAdmin(http.HandlerFunc(a.updateSubscriptionPlan))))
	m.Handle("PATCH /v1/platform/subscription", a.auth(a.requirePlatformAdmin(http.HandlerFunc(a.updateOrganizationSubscription))))
	m.Handle("POST /v1/platform/subscription/payments", a.auth(a.requirePlatformAdmin(http.HandlerFunc(a.recordSubscriptionPayment))))
	m.Handle("GET /v1/admin/dashboard", a.auth(a.requirePermission("dashboard.read", http.HandlerFunc(a.dashboard))))
	m.Handle("GET /v1/admin/context", a.auth(http.HandlerFunc(a.getContext)))
	m.Handle("GET /v1/admin/me", a.auth(http.HandlerFunc(a.getMyProfile)))
	m.Handle("PATCH /v1/admin/me", a.auth(http.HandlerFunc(a.updateMyProfile)))
	m.Handle("GET /v1/admin/subscription", a.auth(a.requirePermission("subscription.read", http.HandlerFunc(a.getOrganizationSubscription))))
	m.Handle("GET /v1/admin/plans", a.auth(http.HandlerFunc(a.listAvailableSubscriptionPlans)))
	m.Handle("GET /v1/admin/settings", a.auth(a.requirePermission("organizations.read", http.HandlerFunc(a.getOrgSettings))))
	m.Handle("GET /v1/admin/payment-methods", a.auth(a.requirePermission("organizations.read", http.HandlerFunc(a.listPaymentMethodsAdmin))))
	m.Handle("POST /v1/admin/payment-methods", a.auth(a.requirePermission("organizations.manage", http.HandlerFunc(a.createPaymentMethod))))
	m.Handle("PATCH /v1/admin/payment-methods/{code}", a.auth(a.requirePermission("organizations.manage", http.HandlerFunc(a.updatePaymentMethod))))
	m.Handle("PATCH /v1/admin/payment-methods/{code}/status", a.auth(a.requirePermission("organizations.manage", http.HandlerFunc(a.updatePaymentMethodStatus))))
	m.Handle("PATCH /v1/admin/settings", a.auth(a.requirePermission("organizations.manage", http.HandlerFunc(a.updateOrgSettings))))
	m.Handle("GET /v1/admin/organization", a.auth(a.requirePermission("organizations.read", http.HandlerFunc(a.getOrganization))))
	m.Handle("PATCH /v1/admin/organization", a.auth(a.requirePermission("organizations.manage", http.HandlerFunc(a.updateOrganization))))
	m.Handle("GET /v1/admin/fiscal-profiles", a.auth(a.requirePermission("organizations.read", http.HandlerFunc(a.listFiscalProfiles))))
	m.Handle("POST /v1/admin/fiscal-profiles", a.auth(a.requirePermission("organizations.manage", http.HandlerFunc(a.createFiscalProfile))))
	m.Handle("PATCH /v1/admin/fiscal-profiles/{id}", a.auth(a.requirePermission("organizations.manage", http.HandlerFunc(a.updateFiscalProfile))))
	m.Handle("DELETE /v1/admin/fiscal-profiles/{id}", a.auth(a.requirePermission("organizations.manage", http.HandlerFunc(a.deactivateFiscalProfile))))
	m.Handle("GET /v1/admin/locations", a.auth(a.requirePermission("organizations.read", http.HandlerFunc(a.listLocations))))
	m.Handle("POST /v1/admin/locations", a.auth(a.requirePermission("organizations.manage", http.HandlerFunc(a.createLocation))))
	m.Handle("PATCH /v1/admin/locations/{id}", a.auth(a.requirePermission("organizations.manage", http.HandlerFunc(a.updateLocation))))
	m.Handle("DELETE /v1/admin/locations/{id}", a.auth(a.requirePermission("organizations.manage", http.HandlerFunc(a.deactivateLocation))))
	m.Handle("GET /v1/admin/exchange-rates", a.auth(a.requirePermission("organizations.read", http.HandlerFunc(a.listExchangeRates))))
	m.Handle("POST /v1/admin/exchange-rates", a.auth(a.requirePermission("organizations.manage", http.HandlerFunc(a.createExchangeRate))))
	m.Handle("DELETE /v1/admin/exchange-rates/{id}", a.auth(a.requirePermission("organizations.manage", http.HandlerFunc(a.deleteExchangeRate))))
	m.Handle("GET /v1/admin/allergens", a.auth(a.requirePermission("menu.read", http.HandlerFunc(a.listAllergens))))
	m.Handle("GET /v1/admin/categories", a.auth(a.requirePermission("menu.read", http.HandlerFunc(a.listCategories))))
	m.Handle("POST /v1/admin/categories", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.createCategory))))
	m.Handle("PATCH /v1/admin/categories/{id}", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.updateCategory))))
	m.Handle("DELETE /v1/admin/categories/{id}", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.deactivateCategory))))
	m.Handle("GET /v1/admin/products", a.auth(a.requirePermission("menu.read", http.HandlerFunc(a.listProducts))))
	m.Handle("POST /v1/admin/products", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.createProduct))))
	m.Handle("GET /v1/admin/products/{id}", a.auth(a.requirePermission("menu.read", http.HandlerFunc(a.getProduct))))
	m.Handle("PATCH /v1/admin/products/{id}", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.updateProduct))))
	m.Handle("DELETE /v1/admin/products/{id}", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.deactivateProduct))))
	m.Handle("POST /v1/admin/products/{id}/image", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.uploadProductImage))))
	m.Handle("GET /v1/admin/product-availability", a.auth(a.requirePermission("menu.read", http.HandlerFunc(a.listProductAvailability))))
	m.Handle("PATCH /v1/admin/product-availability/{productId}", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.updateProductAvailability))))
	m.Handle("GET /v1/admin/inventory", a.auth(a.requirePermission("inventory.read", http.HandlerFunc(a.listInventory))))
	m.Handle("GET /v1/admin/inventory/products", a.auth(a.requirePermission("inventory.read", http.HandlerFunc(a.listInventoryProducts))))
	m.Handle("POST /v1/admin/inventory/entries", a.auth(a.requirePermission("inventory.manage", http.HandlerFunc(a.createInventoryEntry))))
	m.Handle("POST /v1/admin/inventory/adjustments", a.auth(a.requirePermission("inventory.manage", http.HandlerFunc(a.createInventoryAdjustment))))
	m.Handle("GET /v1/admin/inventory/movements", a.auth(a.requirePermission("inventory.read", http.HandlerFunc(a.listInventoryMovements))))
	m.Handle("PATCH /v1/admin/inventory/items/{id}/settings", a.auth(a.requirePermission("inventory.manage", http.HandlerFunc(a.updateInventoryLocationSettings))))
	m.Handle("GET /v1/admin/inventory/transfers", a.auth(a.requirePermission("inventory.read", http.HandlerFunc(a.listInventoryTransfers))))
	m.Handle("POST /v1/admin/inventory/transfers", a.auth(a.requirePermission("inventory.transfer", http.HandlerFunc(a.createInventoryTransfer))))
	m.Handle("GET /v1/admin/suppliers", a.auth(a.requirePermission("purchases.read", http.HandlerFunc(a.listSuppliers))))
	m.Handle("POST /v1/admin/suppliers", a.auth(a.requirePermission("purchases.manage", http.HandlerFunc(a.createSupplier))))
	m.Handle("PATCH /v1/admin/suppliers/{id}", a.auth(a.requirePermission("purchases.manage", http.HandlerFunc(a.updateSupplier))))
	m.Handle("PATCH /v1/admin/suppliers/{id}/status", a.auth(a.requirePermission("purchases.manage", http.HandlerFunc(a.updateSupplierStatus))))
	m.Handle("GET /v1/admin/purchase-inventory-items", a.auth(a.requirePermission("purchases.read", http.HandlerFunc(a.listInventoryProducts))))
	m.Handle("POST /v1/admin/purchase-inventory-items", a.auth(a.requirePermission("purchases.manage", http.HandlerFunc(a.createPurchaseInventoryItem))))
	m.Handle("GET /v1/admin/purchase-item-categories", a.auth(a.requirePermission("purchases.read", http.HandlerFunc(a.listCategories))))
	m.Handle("GET /v1/admin/purchase-orders", a.auth(a.requirePermission("purchases.read", http.HandlerFunc(a.listPurchaseOrders))))
	m.Handle("POST /v1/admin/purchase-orders", a.auth(a.requirePermission("purchases.manage", http.HandlerFunc(a.createPurchaseOrder))))
	m.Handle("GET /v1/admin/purchase-orders/{id}", a.auth(a.requirePermission("purchases.read", http.HandlerFunc(a.getPurchaseOrder))))
	m.Handle("PATCH /v1/admin/purchase-orders/{id}", a.auth(a.requirePermission("purchases.manage", http.HandlerFunc(a.updatePurchaseOrder))))
	m.Handle("PATCH /v1/admin/purchase-orders/{id}/status", a.auth(a.requirePermission("purchases.manage", http.HandlerFunc(a.updatePurchaseOrderStatus))))
	m.Handle("POST /v1/admin/purchase-orders/{id}/approve", a.auth(a.requirePermission("purchases.approve", http.HandlerFunc(a.approvePurchaseOrder))))
	m.Handle("POST /v1/admin/purchase-orders/{id}/receive", a.auth(a.requirePermission("purchases.receive", http.HandlerFunc(a.receivePurchaseOrder))))
	m.Handle("GET /v1/admin/purchase-receipts", a.auth(a.requirePermission("purchases.read", http.HandlerFunc(a.listPurchaseReceipts))))
	m.Handle("GET /v1/admin/purchase-receipts/{id}", a.auth(a.requirePermission("purchases.read", http.HandlerFunc(a.getPurchaseReceipt))))
	m.Handle("POST /v1/admin/purchase-receipts/{id}/returns", a.auth(a.requirePermission("purchases.manage", http.HandlerFunc(a.createPurchaseReturn))))
	m.Handle("GET /v1/admin/recipes", a.auth(a.requirePermission("menu.read", http.HandlerFunc(a.listRecipes))))
	m.Handle("GET /v1/admin/recipes/{productId}", a.auth(a.requirePermission("menu.read", http.HandlerFunc(a.getRecipe))))
	m.Handle("PUT /v1/admin/recipes/{productId}", a.auth(a.requirePermission("recipes.manage", http.HandlerFunc(a.saveRecipe))))
	m.Handle("GET /v1/admin/expenses", a.auth(a.requirePermission("expenses.read", http.HandlerFunc(a.listExpenses))))
	m.Handle("POST /v1/admin/expenses", a.auth(a.requirePermission("expenses.manage", http.HandlerFunc(a.createExpense))))
	m.Handle("POST /v1/admin/expenses/{id}/void", a.auth(a.requirePermission("expenses.void", http.HandlerFunc(a.voidExpense))))
	m.Handle("GET /v1/admin/expense-categories", a.auth(a.requirePermission("expenses.read", http.HandlerFunc(a.listExpenseCategories))))
	m.Handle("POST /v1/admin/expense-categories", a.auth(a.requirePermission("expenses.manage", http.HandlerFunc(a.createExpenseCategory))))
	m.Handle("PATCH /v1/admin/expense-categories/{id}", a.auth(a.requirePermission("expenses.manage", http.HandlerFunc(a.updateExpenseCategory))))
	m.Handle("PATCH /v1/admin/expense-categories/{id}/status", a.auth(a.requirePermission("expenses.manage", http.HandlerFunc(a.updateExpenseCategoryStatus))))
	m.Handle("GET /v1/admin/cash-registers", a.auth(a.requirePermission("cash.read", http.HandlerFunc(a.listCashRegisters))))
	m.Handle("POST /v1/admin/cash-registers", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.createCashRegister))))
	m.Handle("PATCH /v1/admin/cash-registers/{id}", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.updateCashRegister))))
	m.Handle("PATCH /v1/admin/cash-registers/{id}/status", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.updateCashRegisterStatus))))
	m.Handle("GET /v1/admin/cash-shifts/current", a.auth(a.requirePermission("cash.read", http.HandlerFunc(a.getCurrentCashShift))))
	m.Handle("GET /v1/admin/cash-shifts", a.auth(a.requirePermission("cash.read", http.HandlerFunc(a.listCashShifts))))
	m.Handle("POST /v1/admin/cash-shifts", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.openCashShift))))
	m.Handle("GET /v1/admin/cash-shifts/{id}", a.auth(a.requirePermission("cash.read", http.HandlerFunc(a.getCashShift))))
	m.Handle("POST /v1/admin/cash-shifts/{id}/movements", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.createCashMovement))))
	m.Handle("POST /v1/admin/cash-shifts/{id}/close", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.closeCashShift))))
	m.Handle("GET /v1/admin/cash-users/options", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.listCashUserOptions))))
	m.Handle("GET /v1/admin/cash-shifts/{id}/users", a.auth(a.requirePermission("cash.read", http.HandlerFunc(a.listCashShiftUsers))))
	m.Handle("POST /v1/admin/cash-shifts/{id}/users", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.assignCashShiftUser))))
	m.Handle("DELETE /v1/admin/cash-shifts/{id}/users/{userId}", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.unassignCashShiftUser))))
	m.Handle("POST /v1/admin/cash-shifts/{id}/operations", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.createCashOperation))))
	m.Handle("GET /v1/admin/pos/orders", a.auth(a.requirePermission("orders.read", http.HandlerFunc(a.listPOSOrders))))
	m.Handle("GET /v1/admin/pos/orders/{id}", a.auth(a.requirePermission("orders.read", http.HandlerFunc(a.getPOSOrder))))
	m.Handle("POST /v1/admin/payments", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.createPayment))))
	m.Handle("POST /v1/admin/payments/{id}/refund", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.refundPayment))))
	m.Handle("GET /v1/operations/inventory", a.auth(a.requirePermission("inventory.read", http.HandlerFunc(a.listInventory))))
	m.Handle("GET /v1/operations/products", a.auth(a.requirePermission("menu.read", http.HandlerFunc(a.listProducts))))
	m.Handle("GET /v1/operations/product-availability", a.auth(a.requirePermission("menu.read", http.HandlerFunc(a.listProductAvailability))))
	m.Handle("POST /v1/operations/orders", a.auth(a.requirePermission("orders.manage", http.HandlerFunc(a.createOrder))))
	m.Handle("PATCH /v1/operations/orders/{id}/status", a.auth(a.requirePermission("orders.manage", http.HandlerFunc(a.updateOrderStatus))))
	m.Handle("PATCH /v1/operations/product-availability/{productId}", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.updateProductAvailability))))
	m.Handle("GET /v1/operations/payment-methods", a.auth(http.HandlerFunc(a.listOperationalPaymentMethods)))
	m.Handle("GET /v1/operations/cash-registers", a.auth(a.requirePermission("cash.read", http.HandlerFunc(a.listCashRegisters))))
	m.Handle("GET /v1/operations/cash-shifts/current", a.auth(a.requirePermission("cash.read", http.HandlerFunc(a.getCurrentCashShift))))
	m.Handle("POST /v1/operations/cash-shifts", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.openCashShift))))
	m.Handle("POST /v1/operations/cash-shifts/{id}/movements", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.createCashMovement))))
	m.Handle("POST /v1/operations/cash-shifts/{id}/close", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.closeCashShift))))
	m.Handle("GET /v1/operations/pos/orders", a.auth(a.requirePermission("orders.read", http.HandlerFunc(a.listPOSOrders))))
	m.Handle("GET /v1/operations/pos/orders/{id}", a.auth(a.requirePermission("orders.read", http.HandlerFunc(a.getPOSOrder))))
	m.Handle("POST /v1/operations/payments/batch", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.createPaymentBatch))))
	m.Handle("POST /v1/operations/payments/{id}/refund", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.refundPayment))))
	m.Handle("POST /v1/operations/pos/orders/{id}/complete", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.completePaidOrder))))
	m.Handle("GET /v1/admin/combos", a.auth(a.requirePermission("menu.read", http.HandlerFunc(a.listCombos))))
	m.Handle("GET /v1/admin/combos/{id}", a.auth(a.requirePermission("menu.read", http.HandlerFunc(a.getCombo))))
	m.Handle("GET /v1/admin/order-combos", a.auth(a.requirePermission("orders.manage", http.HandlerFunc(a.listOrderCombos))))
	m.Handle("GET /v1/admin/order-combos/{id}", a.auth(a.requirePermission("orders.manage", http.HandlerFunc(a.getOrderCombo))))
	m.Handle("POST /v1/admin/combos", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.saveCombo))))
	m.Handle("PATCH /v1/admin/combos/{id}", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.saveCombo))))
	m.Handle("PATCH /v1/admin/combos/{id}/status", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.updateComboStatus))))
	m.Handle("GET /v1/admin/customers", a.auth(a.requirePermission("customers.read", http.HandlerFunc(a.listCustomers))))
	m.Handle("POST /v1/admin/customers", a.auth(a.requirePermission("customers.manage", http.HandlerFunc(a.createCustomer))))
	m.Handle("GET /v1/admin/customers/{id}", a.auth(a.requirePermission("customers.read", http.HandlerFunc(a.getCustomer))))
	m.Handle("PATCH /v1/admin/customers/{id}", a.auth(a.requirePermission("customers.manage", http.HandlerFunc(a.updateCustomer))))
	m.Handle("PATCH /v1/admin/customers/{id}/status", a.auth(a.requirePermission("customers.manage", http.HandlerFunc(a.updateCustomerStatus))))
	m.Handle("GET /v1/admin/orders", a.auth(a.requirePermission("orders.read", http.HandlerFunc(a.listOrders))))
	m.Handle("GET /v1/admin/reservations", a.auth(a.requirePermission("reservations.read", http.HandlerFunc(a.listReservations))))
	m.Handle("POST /v1/admin/reservations", a.auth(a.requirePermission("reservations.manage", http.HandlerFunc(a.createReservation))))
	m.Handle("PATCH /v1/admin/reservations/{id}", a.auth(a.requirePermission("reservations.manage", http.HandlerFunc(a.updateReservation))))
	m.Handle("PATCH /v1/admin/reservations/{id}/status", a.auth(a.requirePermission("reservations.manage", http.HandlerFunc(a.updateReservationStatus))))
	m.Handle("GET /v1/admin/orders/floor", a.auth(a.requirePermission("orders.read", http.HandlerFunc(a.getOrdersFloor))))
	m.Handle("POST /v1/admin/orders", a.auth(a.requirePermission("orders.manage", http.HandlerFunc(a.createOrder))))
	m.Handle("GET /v1/admin/orders/{id}", a.auth(a.requirePermission("orders.read", http.HandlerFunc(a.getOrder))))
	m.Handle("GET /v1/admin/kitchen/tickets", a.auth(a.requirePermission("orders.read", http.HandlerFunc(a.listKitchenTickets))))
	m.Handle("PATCH /v1/admin/kitchen/tickets/{id}/status", a.auth(a.requirePermission("kitchen.manage", http.HandlerFunc(a.updateKitchenTicketStatus))))
	m.Handle("PATCH /v1/admin/orders/{id}", a.auth(a.requirePermission("orders.manage", http.HandlerFunc(a.updateOrder))))
	m.Handle("PATCH /v1/admin/orders/{id}/status", a.auth(a.requirePermission("orders.manage", http.HandlerFunc(a.updateOrderStatus))))
	m.Handle("GET /v1/admin/roles", a.auth(a.requirePermission("users.read", http.HandlerFunc(a.listRoles))))
	m.Handle("POST /v1/admin/roles", a.auth(a.requirePermission("users.manage", http.HandlerFunc(a.createRole))))
	m.Handle("PATCH /v1/admin/roles/{id}", a.auth(a.requirePermission("users.manage", http.HandlerFunc(a.updateRole))))
	m.Handle("PATCH /v1/admin/roles/{id}/status", a.auth(a.requirePermission("users.manage", http.HandlerFunc(a.updateRoleStatus))))
	m.Handle("GET /v1/admin/permission-catalog", a.auth(a.requirePermission("users.read", http.HandlerFunc(a.getPermissionCatalog))))
	m.Handle("GET /v1/admin/users", a.auth(a.requirePermission("users.read", http.HandlerFunc(a.listUsers))))
	m.Handle("POST /v1/admin/users", a.auth(a.requirePermission("users.manage", http.HandlerFunc(a.createUser))))
	m.Handle("PATCH /v1/admin/users/{id}", a.auth(a.requirePermission("users.manage", http.HandlerFunc(a.updateUser))))
	m.Handle("PATCH /v1/admin/users/{id}/status", a.auth(a.requirePermission("users.manage", http.HandlerFunc(a.updateUserStatus))))
	m.Handle("GET /v1/admin/tables", a.auth(a.requirePermission("tables.read", http.HandlerFunc(a.listTables))))
	m.Handle("POST /v1/admin/tables", a.auth(a.requirePermission("tables.manage", http.HandlerFunc(a.createTable))))
	m.Handle("POST /v1/admin/tables/batch", a.auth(a.requirePermission("tables.manage", http.HandlerFunc(a.createTablesBatch))))
	m.Handle("PATCH /v1/admin/tables/{id}", a.auth(a.requirePermission("tables.manage", http.HandlerFunc(a.updateTable))))
	m.Handle("DELETE /v1/admin/tables/{id}", a.auth(a.requirePermission("tables.manage", http.HandlerFunc(a.deactivateTable))))
	m.Handle("POST /v1/admin/tables/{id}/qr", a.auth(a.requirePermission("tables.manage", http.HandlerFunc(a.regenerateTableQR))))
	m.Handle("GET /v1/public/tables/{token}", http.HandlerFunc(a.getTableByQR))\n\tm.Handle("GET /v1/public/concierge/{token}/menu", http.HandlerFunc(a.listConciergeMenu))\n\tm.Handle("POST /v1/public/concierge/{token}/orders", http.HandlerFunc(a.createConciergeOrder))
	m.Handle("GET /v1/admin/zones", a.auth(a.requirePermission("menu.read", http.HandlerFunc(a.listZones))))
	m.Handle("POST /v1/admin/zones", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.createZone))))
	m.Handle("PATCH /v1/admin/zones/{id}", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.updateZone))))
	m.Handle("DELETE /v1/admin/zones/{id}", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.deactivateZone))))
	m.Handle("GET /v1/admin/modules", a.auth(a.requirePlatformAdmin(http.HandlerFunc(a.listModules))))
	m.Handle("PATCH /v1/admin/modules", a.auth(a.requirePlatformAdmin(http.HandlerFunc(a.toggleModule))))
	m.Handle("GET /v1/admin/organizations", a.auth(a.requirePlatformAdmin(http.HandlerFunc(a.listOrganizations))))
	m.Handle("GET /v1/admin/organizations/{id}/locations", a.auth(a.requirePlatformAdmin(http.HandlerFunc(a.listOrgLocations))))
	m.Handle("POST /v1/admin/switch-context", a.auth(http.HandlerFunc(a.switchContext)))
	m.Handle("GET /v1/admin/locations/available", a.auth(http.HandlerFunc(a.listAvailableLocations)))
	m.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))
	return m
}
func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("foods_session"); err == nil && cookie.Value != "" {
		sum := sha256.Sum256([]byte(cookie.Value))
		if _, err := a.db.Exec(r.Context(), `UPDATE sessions SET revoked_at=COALESCE(revoked_at,now()) WHERE token_hash=$1`, sum[:]); err != nil {
			fail(w, 503, "session_unavailable", "No pudimos cerrar la sesión. Intenta nuevamente.")
			return
		}
	}
	http.SetCookie(w, &http.Cookie{Name: "foods_session", Value: "", Path: "/", HttpOnly: true, Secure: os.Getenv("FOODS_COOKIE_SECURE") == "true", SameSite: http.SameSiteLaxMode, MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}
func (a *API) requirePermission(permission string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := r.Context().Value(scopeKey{}).(scope)
		var platformAdmin bool
		if err := a.db.QueryRow(r.Context(), `SELECT platform_admin FROM users WHERE id=$1 AND active`, s.UserID).Scan(&platformAdmin); err == nil && platformAdmin {
			next.ServeHTTP(w, r)
			return
		}
		var allowed bool
		err := a.db.QueryRow(r.Context(), `
			SELECT EXISTS(
				SELECT 1
				FROM user_roles ur
				JOIN roles ro ON ro.id=ur.role_id
				WHERE ur.user_id=$1
				  AND ur.location_id=$2
				  AND ro.organization_id=$3
				  AND ro.active
				  AND $4 = ANY(ro.permissions)
			)
		`, s.UserID, s.LocationID, s.OrganizationID, permission).Scan(&allowed)
		if err != nil {
			fail(w, 503, "permissions_unavailable", "No pudimos validar tus permisos.")
			return
		}
		if !allowed {
			fail(w, 403, "forbidden", "No tienes permisos para realizar esta acción.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, code, msg string) {
	correlationID := w.Header().Get("X-Request-ID")
	if correlationID == "" {
		correlationID = "request"
	}
	writeJSON(w, status, apiError{Code: code, Message: msg, CorrelationID: correlationID})
}
func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.Email == "" || in.Password == "" {
		fail(w, 400, "invalid_request", "Correo y contraseña son obligatorios.")
		return
	}

	rows, err := a.db.Query(r.Context(), `
		SELECT id,organization_id,full_name,password_hash,platform_admin
		FROM users
		WHERE lower(email)=lower($1) AND active
		ORDER BY created_at,id
	`, in.Email)
	if err != nil {
		fail(w, 503, "session_unavailable", "No pudimos validar la cuenta.")
		return
	}
	defer rows.Close()

	type loginCandidate struct {
		UserID, OrganizationID, Name, Hash string
		PlatformAdmin bool
	}
	matches := []loginCandidate{}
	for rows.Next() {
		var candidate loginCandidate
		if err = rows.Scan(&candidate.UserID,&candidate.OrganizationID,&candidate.Name,&candidate.Hash,&candidate.PlatformAdmin); err != nil {
			fail(w, 503, "session_unavailable", "No pudimos validar la cuenta.")
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(candidate.Hash), []byte(in.Password)) == nil {
			matches = append(matches, candidate)
		}
	}
	if err = rows.Err(); err != nil {
		fail(w, 503, "session_unavailable", "No pudimos validar la cuenta.")
		return
	}
	if len(matches) == 0 {
		fail(w, 401, "invalid_credentials", "Correo o contraseña incorrectos.")
		return
	}
	if len(matches) > 1 {
		fail(w, 409, "ambiguous_account", "Ese correo pertenece a más de una cuenta con la misma contraseña. Solicita al administrador diferenciar el acceso.")
		return
	}

	candidate := matches[0]
	var locationID string
	if candidate.PlatformAdmin {
		err = a.db.QueryRow(r.Context(), `
			SELECT id
			FROM locations
			WHERE organization_id=$1 AND active
			ORDER BY created_at,id
			LIMIT 1
		`, candidate.OrganizationID).Scan(&locationID)
	} else {
		err = a.db.QueryRow(r.Context(), `
			SELECT ur.location_id
			FROM user_roles ur
			JOIN roles ro ON ro.id=ur.role_id
			JOIN locations l ON l.id=ur.location_id AND l.organization_id=ro.organization_id
			WHERE ur.user_id=$1
			  AND ro.organization_id=$2
			  AND ro.active
			  AND l.active
			ORDER BY l.created_at,l.id,ro.name
			LIMIT 1
		`, candidate.UserID, candidate.OrganizationID).Scan(&locationID)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 403, "account_without_access", "Tu cuenta no tiene un rol activo en ningún local. Contacta a un administrador.")
		return
	}
	if err != nil {
		fail(w, 503, "session_unavailable", "No pudimos determinar tu local de trabajo.")
		return
	}

	tokenBytes := make([]byte, 32)
	if _, err = rand.Read(tokenBytes); err != nil {
		fail(w, 500, "session_error", "No se pudo iniciar la sesión.")
		return
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	sum := sha256.Sum256([]byte(token))
	_, err = a.db.Exec(r.Context(), `
		INSERT INTO sessions(token_hash,user_id,organization_id,location_id,expires_at)
		VALUES($1,$2,$3,$4,$5)
	`, sum[:], candidate.UserID, candidate.OrganizationID, locationID, time.Now().Add(12*time.Hour))
	if err != nil {
		fail(w, 500, "session_error", "No se pudo iniciar la sesión.")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "foods_session", Value: token, Path: "/", HttpOnly: true, Secure: os.Getenv("FOODS_COOKIE_SECURE") == "true", SameSite: http.SameSiteLaxMode, MaxAge: 43200})
	writeJSON(w, 200, map[string]any{"user": map[string]any{"id": candidate.UserID, "name": candidate.Name, "platformAdmin": candidate.PlatformAdmin}, "expiresIn": 43200})
}
func (a *API) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string
		if c, e := r.Cookie("foods_session"); e == nil {
			token = c.Value
		}
		if token == "" {
			token = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		}
		if token == "" {
			fail(w, 401, "unauthenticated", "Inicia sesión para continuar.")
			return
		}
		sum := sha256.Sum256([]byte(token))
		var s scope
		err := a.db.QueryRow(r.Context(), `SELECT s.user_id,s.organization_id,s.location_id,u.full_name FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now() AND u.active`, sum[:]).Scan(&s.UserID, &s.OrganizationID, &s.LocationID, &s.Name)
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, 401, "session_expired", "Tu sesión venció. Ingresa nuevamente.")
			return
		}
		if err != nil {
			fail(w, 503, "session_unavailable", "No pudimos validar la sesión.")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), scopeKey{}, s)))
	})
}
func (a *API) dashboard(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var salesNet, averageTicket string
	var paidOrders, openOrders, critical, purchases, reservationsToday, kitchenPending int

	err := a.db.QueryRow(r.Context(), `
		WITH loc AS (
		  SELECT timezone
		  FROM locations
		  WHERE id=$2 AND organization_id=$1
		),
		today_payments AS (
		  SELECT p.order_id,p.amount
		  FROM payments p
		  CROSS JOIN loc
		  WHERE p.organization_id=$1 AND p.location_id=$2
		    AND (p.created_at AT TIME ZONE loc.timezone)::date=(now() AT TIME ZONE loc.timezone)::date
		),
		today_refunds AS (
		  SELECT pr.amount
		  FROM payment_refunds pr
		  CROSS JOIN loc
		  WHERE pr.organization_id=$1 AND pr.location_id=$2
		    AND (pr.created_at AT TIME ZONE loc.timezone)::date=(now() AT TIME ZONE loc.timezone)::date
		),
		sales AS (
		  SELECT
		    COALESCE((SELECT sum(amount) FROM today_payments),0)
		    - COALESCE((SELECT sum(amount) FROM today_refunds),0) AS net
		),
		paid AS (
		  SELECT count(DISTINCT order_id)::int AS orders
		  FROM today_payments
		)
		SELECT
		  sales.net::text,
		  paid.orders,
		  CASE WHEN paid.orders>0 THEN (sales.net/paid.orders)::text ELSE '0' END,
		  (SELECT count(*) FROM orders
		   WHERE organization_id=$1 AND location_id=$2
		     AND status NOT IN ('entregado','cancelado')),
		  (SELECT count(*)
		   FROM inventory_items i
		   LEFT JOIN stock_balances b
		     ON b.inventory_item_id=i.id
		    AND b.organization_id=i.organization_id
		    AND b.location_id=$2
		   WHERE i.organization_id=$1 AND i.active
		     AND COALESCE(b.quantity,0)<=i.minimum_stock),
		  (SELECT count(*) FROM purchase_orders
		   WHERE organization_id=$1 AND location_id=$2 AND status='pending_approval'),
		  (SELECT count(*)
		   FROM reservations r
		   CROSS JOIN loc
		   WHERE r.organization_id=$1 AND r.location_id=$2
		     AND r.status IN ('pending','confirmed')
		     AND (r.starts_at AT TIME ZONE loc.timezone)::date=(now() AT TIME ZONE loc.timezone)::date),
		  (SELECT count(*) FROM orders
		   WHERE organization_id=$1 AND location_id=$2
		     AND status IN ('confirmado','preparando'))
		FROM sales CROSS JOIN paid
	`, s.OrganizationID, s.LocationID).Scan(
		&salesNet, &paidOrders, &averageTicket, &openOrders, &critical, &purchases, &reservationsToday, &kitchenPending,
	)
	if err != nil {
		fail(w,503,"dashboard_unavailable","No pudimos calcular el resumen operativo.")
		return
	}

	hourly := []map[string]any{}
	rows,err:=a.db.Query(r.Context(), `
		SELECT hour,COALESCE(sum(value),0)::text
		FROM (
		  SELECT extract(hour FROM p.created_at AT TIME ZONE l.timezone)::int AS hour,p.amount AS value
		  FROM payments p JOIN locations l ON l.id=p.location_id AND l.organization_id=p.organization_id
		  WHERE p.organization_id=$1 AND p.location_id=$2
		    AND (p.created_at AT TIME ZONE l.timezone)::date=(now() AT TIME ZONE l.timezone)::date
		  UNION ALL
		  SELECT extract(hour FROM pr.created_at AT TIME ZONE l.timezone)::int AS hour,-pr.amount AS value
		  FROM payment_refunds pr JOIN locations l ON l.id=pr.location_id AND l.organization_id=pr.organization_id
		  WHERE pr.organization_id=$1 AND pr.location_id=$2
		    AND (pr.created_at AT TIME ZONE l.timezone)::date=(now() AT TIME ZONE l.timezone)::date
		) movements
		GROUP BY hour ORDER BY hour
	`,s.OrganizationID,s.LocationID)
	if err==nil {
		defer rows.Close()
		for rows.Next(){var hour int;var total string;if rows.Scan(&hour,&total)==nil{hourly=append(hourly,map[string]any{"hour":hour,"total":total})}}
	}

	topProducts:=[]map[string]any{}
	productRows,err:=a.db.Query(r.Context(), `
		SELECT oi.name,sum(oi.qty)::text,sum(oi.qty*oi.unit_price)::text
		FROM order_items oi
		JOIN orders o ON o.id=oi.order_id AND o.organization_id=oi.organization_id
		WHERE o.organization_id=$1 AND o.location_id=$2 AND o.status<>'cancelado'
		  AND (`+netPaidSQL+`) >= o.total
		  AND EXISTS(
		    SELECT 1 FROM payments today_payment
		    JOIN locations l ON l.id=today_payment.location_id AND l.organization_id=today_payment.organization_id
		    WHERE today_payment.order_id=o.id AND today_payment.organization_id=o.organization_id AND today_payment.location_id=o.location_id
		      AND (today_payment.created_at AT TIME ZONE l.timezone)::date=(now() AT TIME ZONE l.timezone)::date
		  )
		GROUP BY oi.name ORDER BY sum(oi.qty) DESC,oi.name LIMIT 5
	`,s.OrganizationID,s.LocationID)
	if err==nil {
		defer productRows.Close()
		for productRows.Next(){var name,qty,revenue string;if productRows.Scan(&name,&qty,&revenue)==nil{topProducts=append(topProducts,map[string]any{"name":name,"qty":qty,"revenue":revenue})}}
	}
	writeJSON(w,200,map[string]any{
		"salesNet":salesNet,"paidOrders":paidOrders,"averageTicket":averageTicket,"openOrders":openOrders,
		"criticalStock":critical,"purchasesToApprove":purchases,"reservationsToday":reservationsToday,
		"kitchenPending":kitchenPending,"hourlySales":hourly,"topProducts":topProducts,
	})
}

func (a *API) getContext(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var organizationName, locationName, locationCountry, locationTimezone string
	var platformAdmin bool
	var permissions, menuAccess []string
	err := a.db.QueryRow(r.Context(), `
		SELECT o.trade_name,l.name,p.country_code,l.timezone,u.platform_admin,
		       CASE WHEN u.platform_admin THEN ARRAY['*']::text[] ELSE COALESCE((
		         SELECT array_agg(DISTINCT permission ORDER BY permission)
		         FROM user_roles ur
		         JOIN roles ro ON ro.id=ur.role_id AND ro.organization_id=$1 AND ro.active
		         CROSS JOIN LATERAL unnest(ro.permissions) permission
		         WHERE ur.user_id=$3 AND ur.location_id=$2
		       ),ARRAY[]::text[]) END,
		       CASE WHEN u.platform_admin THEN ARRAY['*']::text[] ELSE COALESCE((
		         SELECT array_agg(DISTINCT access_key ORDER BY access_key)
		         FROM user_roles ur
		         JOIN roles ro ON ro.id=ur.role_id AND ro.organization_id=$1 AND ro.active
		         CROSS JOIN LATERAL unnest(ro.menu_access) access_key
		         WHERE ur.user_id=$3 AND ur.location_id=$2
		       ),ARRAY[]::text[]) END
		FROM organizations o
		JOIN locations l ON l.organization_id=o.id
		JOIN organization_fiscal_profiles p ON p.id=l.fiscal_profile_id AND p.organization_id=l.organization_id AND p.active
		JOIN users u ON u.id=$3 AND u.organization_id=o.id AND u.active
		WHERE o.id=$1 AND l.id=$2 AND o.active AND l.active
	`, s.OrganizationID, s.LocationID, s.UserID).Scan(
		&organizationName, &locationName, &locationCountry, &locationTimezone, &platformAdmin,
		&permissions, &menuAccess,
	)
	if err != nil {
		fail(w, 503, "context_unavailable", "No pudimos cargar el contexto de trabajo.")
		return
	}
	modules := a.activeModules(s.OrganizationID)
	if modules == nil {
		fail(w, 503, "modules_unavailable", "No pudimos cargar los módulos de la empresa.")
		return
	}
	writeJSON(w, 200, map[string]any{
		"user": map[string]any{"id": s.UserID, "name": s.Name, "platformAdmin": platformAdmin},
		"organization": map[string]string{"id": s.OrganizationID, "name": organizationName},
		"location": map[string]string{"id": s.LocationID, "name": locationName, "country": locationCountry, "timezone": locationTimezone},
		"modules": modules,
		"menuAccess": menuAccess,
		"permissions": permissions,
	})
}
