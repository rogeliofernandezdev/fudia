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
	m.Handle("GET /v1/admin/dashboard", a.auth(http.HandlerFunc(a.dashboard)))
	m.Handle("GET /v1/admin/context", a.auth(http.HandlerFunc(a.getContext)))
	m.Handle("GET /v1/admin/settings", a.auth(a.requirePermission("organizations.read", http.HandlerFunc(a.getOrgSettings))))
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
	m.Handle("POST /v1/admin/purchase-orders/{id}/receive", a.auth(a.requirePermission("purchases.receive", http.HandlerFunc(a.receivePurchaseOrder))))
	m.Handle("GET /v1/admin/cash-registers", a.auth(a.requirePermission("cash.read", http.HandlerFunc(a.listCashRegisters))))
	m.Handle("POST /v1/admin/cash-registers", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.createCashRegister))))
	m.Handle("GET /v1/admin/cash-shifts/current", a.auth(a.requirePermission("cash.read", http.HandlerFunc(a.getCurrentCashShift))))
	m.Handle("GET /v1/admin/cash-shifts", a.auth(a.requirePermission("cash.read", http.HandlerFunc(a.listCashShifts))))
	m.Handle("POST /v1/admin/cash-shifts", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.openCashShift))))
	m.Handle("GET /v1/admin/cash-shifts/{id}", a.auth(a.requirePermission("cash.read", http.HandlerFunc(a.getCashShift))))
	m.Handle("POST /v1/admin/cash-shifts/{id}/movements", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.createCashMovement))))
	m.Handle("POST /v1/admin/cash-shifts/{id}/close", a.auth(a.requirePermission("cash.manage", http.HandlerFunc(a.closeCashShift))))
	m.Handle("GET /v1/operations/product-availability", a.auth(a.requirePermission("menu.read", http.HandlerFunc(a.listProductAvailability))))
	m.Handle("PATCH /v1/operations/product-availability/{productId}", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.updateProductAvailability))))
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
	m.Handle("GET /v1/public/tables/{token}", http.HandlerFunc(a.getTableByQR))
	m.Handle("GET /v1/admin/zones", a.auth(a.requirePermission("menu.read", http.HandlerFunc(a.listZones))))
	m.Handle("POST /v1/admin/zones", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.createZone))))
	m.Handle("PATCH /v1/admin/zones/{id}", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.updateZone))))
	m.Handle("DELETE /v1/admin/zones/{id}", a.auth(a.requirePermission("menu.manage", http.HandlerFunc(a.deactivateZone))))
	m.Handle("GET /v1/admin/modules", a.auth(a.requirePermission("organizations.read", http.HandlerFunc(a.listModules))))
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
		err := a.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id=ur.role_id WHERE ur.user_id=$1 AND (ur.location_id IS NULL OR ur.location_id=$2) AND ro.active AND (ro.permissions @> ARRAY['*']::text[] OR ro.permissions @> ARRAY[$3]::text[]))`, s.UserID, s.LocationID, permission).Scan(&allowed)
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
	writeJSON(w, status, apiError{Code: code, Message: msg, CorrelationID: "request"})
}
func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	var s scope
	var hash string
	var platformAdmin bool
	err := a.db.QueryRow(r.Context(), `SELECT u.id,u.organization_id,COALESCE(ur.location_id,l.id),u.full_name,u.password_hash,u.platform_admin FROM users u JOIN locations l ON l.organization_id=u.organization_id AND l.active LEFT JOIN user_roles ur ON ur.user_id=u.id AND ur.location_id=l.id WHERE lower(u.email)=lower($1) AND u.active ORDER BY (ur.location_id IS NULL), l.created_at LIMIT 1`, in.Email).Scan(&s.UserID, &s.OrganizationID, &s.LocationID, &s.Name, &hash, &platformAdmin)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		fail(w, 401, "invalid_credentials", "Correo o contraseña incorrectos.")
		return
	}
	tokenBytes := make([]byte, 32)
	if _, err = rand.Read(tokenBytes); err != nil {
		fail(w, 500, "session_error", "No se pudo iniciar la sesión.")
		return
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	sum := sha256.Sum256([]byte(token))
	_, err = a.db.Exec(r.Context(), `INSERT INTO sessions(token_hash,user_id,organization_id,location_id,expires_at) VALUES($1,$2,$3,$4,$5)`, sum[:], s.UserID, s.OrganizationID, s.LocationID, time.Now().Add(12*time.Hour))
	if err != nil {
		fail(w, 500, "session_error", "No se pudo iniciar la sesión.")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "foods_session", Value: token, Path: "/", HttpOnly: true, Secure: os.Getenv("FOODS_COOKIE_SECURE") == "true", SameSite: http.SameSiteLaxMode, MaxAge: 43200})
	writeJSON(w, 200, map[string]any{"user": map[string]any{"id": s.UserID, "name": s.Name, "platformAdmin": platformAdmin}, "expiresIn": 43200})
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
	var products, critical, purchases int
	_ = a.db.QueryRow(r.Context(), `SELECT count(*) FROM products WHERE organization_id=$1 AND active`, s.OrganizationID).Scan(&products)
	_ = a.db.QueryRow(r.Context(), `SELECT count(*) FROM inventory_items i LEFT JOIN stock_balances b ON b.inventory_item_id=i.id AND b.location_id=$2 WHERE i.organization_id=$1 AND COALESCE(b.quantity,0)<=i.minimum_stock`, s.OrganizationID, s.LocationID).Scan(&critical)
	_ = a.db.QueryRow(r.Context(), `SELECT count(*) FROM purchase_orders WHERE organization_id=$1 AND location_id=$2 AND status='pending_approval'`, s.OrganizationID, s.LocationID).Scan(&purchases)
	var currency string
	_ = a.db.QueryRow(r.Context(), `SELECT p.currency FROM locations l JOIN organization_fiscal_profiles p ON p.id=l.fiscal_profile_id AND p.organization_id=l.organization_id WHERE l.id=$1 AND l.organization_id=$2 AND l.active AND p.active`, s.LocationID, s.OrganizationID).Scan(&currency)
	writeJSON(w, 200, map[string]any{"products": products, "criticalStock": critical, "purchasesToApprove": purchases, "currency": currency})
}

func (a *API) getContext(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var organizationName, locationName, locationCountry, locationTimezone string
	var platformAdmin bool
	err := a.db.QueryRow(r.Context(), `SELECT o.trade_name,l.name,p.country_code,l.timezone,u.platform_admin FROM organizations o JOIN locations l ON l.organization_id=o.id JOIN organization_fiscal_profiles p ON p.id=l.fiscal_profile_id AND p.organization_id=l.organization_id AND p.active JOIN users u ON u.id=$3 WHERE o.id=$1 AND l.id=$2 AND o.active AND l.active`, s.OrganizationID, s.LocationID, s.UserID).Scan(&organizationName, &locationName, &locationCountry, &locationTimezone, &platformAdmin)
	if err != nil {
		fail(w, 503, "context_unavailable", "No pudimos cargar el contexto de trabajo.")
		return
	}
	modules := a.activeModules(s.OrganizationID)
	permissions := []string{"*"}
	menuAccess := []string{"*"}
	if !platformAdmin {
		permissions = []string{}
		if err = a.db.QueryRow(r.Context(), `SELECT COALESCE(array_agg(DISTINCT permission), ARRAY[]::text[]) FROM user_roles ur JOIN roles ro ON ro.id=ur.role_id CROSS JOIN LATERAL unnest(ro.permissions) permission WHERE ur.user_id=$1 AND (ur.location_id IS NULL OR ur.location_id=$2) AND ro.active`, s.UserID, s.LocationID).Scan(&permissions); err != nil {
			fail(w, 503, "permissions_unavailable", "No pudimos cargar tus permisos.")
			return
		}
		menuAccess = []string{}
		if err = a.db.QueryRow(r.Context(), `SELECT COALESCE(array_agg(DISTINCT access_key), ARRAY[]::text[]) FROM user_roles ur JOIN roles ro ON ro.id=ur.role_id CROSS JOIN LATERAL unnest(ro.menu_access) access_key WHERE ur.user_id=$1 AND (ur.location_id IS NULL OR ur.location_id=$2) AND ro.active`, s.UserID, s.LocationID).Scan(&menuAccess); err != nil {
			fail(w, 503, "access_unavailable", "No pudimos cargar tus accesos al sistema.")
			return
		}
	}
	writeJSON(w, 200, map[string]any{"user": map[string]any{"id": s.UserID, "name": s.Name, "platformAdmin": platformAdmin}, "organization": map[string]string{"id": s.OrganizationID, "name": organizationName}, "location": map[string]string{"id": s.LocationID, "name": locationName, "country": locationCountry, "timezone": locationTimezone}, "modules": modules, "menuAccess": menuAccess, "permissions": permissions})
}
