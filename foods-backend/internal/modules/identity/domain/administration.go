package domain

import "strings"

type RoleView struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	SystemKey   *string  `json:"systemKey"`
	Description string   `json:"description"`
	MenuAccess  []string `json:"menuAccess"`
	Permissions []string `json:"permissions"`
	Active      bool     `json:"active"`
	UserCount   int      `json:"userCount"`
}
type RoleInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	MenuAccess  []string `json:"menuAccess"`
	Permissions []string `json:"permissions"`
}
type UserAssignment struct {
	RoleID       string `json:"roleId"`
	RoleName     string `json:"roleName,omitempty"`
	LocationID   string `json:"locationId"`
	LocationName string `json:"locationName,omitempty"`
}
type UserView struct {
	ID            string           `json:"id"`
	FullName      string           `json:"fullName"`
	Email         string           `json:"email"`
	Active        bool             `json:"active"`
	PlatformAdmin bool             `json:"platformAdmin"`
	Assignments   []UserAssignment `json:"assignments"`
}
type UserInput struct {
	FullName    string           `json:"fullName"`
	Email       string           `json:"email"`
	Password    string           `json:"password"`
	Assignments []UserAssignment `json:"assignments"`
}

var PermissionCatalog = []map[string]any{
	{"group": "Administración", "items": []map[string]string{{"value": "dashboard.read", "label": "Ver resumen operativo"}, {"value": "users.read", "label": "Ver usuarios y roles"}, {"value": "users.manage", "label": "Administrar usuarios y roles"}, {"value": "organizations.read", "label": "Ver empresa y locales"}, {"value": "organizations.manage", "label": "Administrar empresa y locales"}, {"value": "fiscal.read", "label": "Ver información fiscal"}, {"value": "subscription.read", "label": "Ver plan y suscripción de la empresa"}}},
	{"group": "Carta y clientes", "items": []map[string]string{{"value": "menu.read", "label": "Ver carta y productos"}, {"value": "menu.manage", "label": "Administrar carta y productos"}, {"value": "recipes.manage", "label": "Administrar recetas"}, {"value": "customers.read", "label": "Ver clientes"}, {"value": "customers.manage", "label": "Administrar clientes"}}},
	{"group": "Operación", "items": []map[string]string{{"value": "orders.read", "label": "Ver pedidos y comandas"}, {"value": "orders.manage", "label": "Gestionar pedidos"}, {"value": "kitchen.manage", "label": "Preparar en Cocina"}, {"value": "bar.manage", "label": "Preparar en Barra"}, {"value": "tables.read", "label": "Ver mesas"}, {"value": "tables.manage", "label": "Administrar mesas"}, {"value": "reservations.read", "label": "Ver reservas"}, {"value": "reservations.manage", "label": "Gestionar reservas"}, {"value": "cash.read", "label": "Ver caja"}, {"value": "cash.registers.manage", "label": "Administrar cajas"}, {"value": "cash.manage", "label": "Operar turnos y cobros"}, {"value": "cash.expected.read", "label": "Ver saldo esperado en cierre ciego"}, {"value": "receipts.read", "label": "Ver comprobantes"}, {"value": "receipts.manage", "label": "Emitir comprobantes"}}},
	{"group": "Abastecimiento y análisis", "items": []map[string]string{{"value": "inventory.read", "label": "Ver inventario"}, {"value": "inventory.manage", "label": "Administrar inventario"}, {"value": "inventory.transfer", "label": "Transferir stock entre locales"}, {"value": "purchases.read", "label": "Ver compras"}, {"value": "purchases.manage", "label": "Crear y administrar compras"}, {"value": "purchases.approve", "label": "Aprobar órdenes de compra"}, {"value": "purchases.receive", "label": "Recibir compras"}, {"value": "expenses.read", "label": "Ver gastos"}, {"value": "expenses.manage", "label": "Registrar y administrar gastos"}, {"value": "expenses.void", "label": "Anular gastos"}, {"value": "reports.read", "label": "Ver reportes"}, {"value": "audit.read", "label": "Ver auditoría"}}},
	{"group": "Delivery", "items": []map[string]string{{"value": "delivery.read", "label": "Ver entregas asignadas"}, {"value": "delivery.manage", "label": "Actualizar entregas"}}},
}
var MenuAccessCatalog = []map[string]any{
	{"group": "Control", "items": []map[string]string{{"value": "dashboard", "label": "Reportes"}}},
	{"group": "Operación", "items": []map[string]string{{"value": "pos", "label": "Punto de venta"}, {"value": "pedidos", "label": "Pedidos"}, {"value": "cocina", "label": "Cocina"}, {"value": "mesas", "label": "Mesas y zonas"}, {"value": "caja", "label": "Caja y turnos"}, {"value": "reservas", "label": "Reservas"}, {"value": "call_center", "label": "Call center"}, {"value": "carta_qr", "label": "Carta digital QR"}, {"value": "kiosco", "label": "Kiosco de autoservicio"}}},
	{"group": "Carta y producción", "items": []map[string]string{{"value": "productos", "label": "Carta y productos"}, {"value": "disponibilidad", "label": "Disponibilidad de la carta"}, {"value": "combos", "label": "Menús y combos"}, {"value": "recetas", "label": "Recetas y producción"}}},
	{"group": "Abastecimiento", "items": []map[string]string{{"value": "inventario", "label": "Inventario"}, {"value": "kardex", "label": "Kardex"}, {"value": "compras", "label": "Compras"}, {"value": "logistica", "label": "Logística"}}},
	{"group": "Delivery", "items": []map[string]string{{"value": "delivery", "label": "Delivery propio"}, {"value": "delivery_apps", "label": "Apps de delivery"}, {"value": "repartidores", "label": "App repartidores"}}},
	{"group": "Negocio", "items": []map[string]string{{"value": "clientes", "label": "Clientes"}, {"value": "crm", "label": "CRM y fidelización"}, {"value": "puntos", "label": "Plaza puntos"}, {"value": "ofertas", "label": "Ofertas y descuentos"}, {"value": "personal", "label": "Personal y asistencias"}, {"value": "locales", "label": "Locales"}}},
	{"group": "Inteligencia", "items": []map[string]string{{"value": "costos", "label": "Costos y gastos"}, {"value": "bi", "label": "Restaurant BI"}, {"value": "app_manager", "label": "App manager"}}},
	{"group": "Configuración", "items": []map[string]string{{"value": "fiscal", "label": "Fiscal y moneda"}, {"value": "usuarios", "label": "Usuarios y permisos"}, {"value": "facturacion", "label": "Facturación"}, {"value": "integraciones", "label": "Integraciones"}, {"value": "whatsapp_bot", "label": "WhatsApp IA para pedidos"}}},
}

type MyProfilePlanView struct {
	ID     string `json:"id"`
	Code   string `json:"code"`
	Name   string `json:"name"`
	Status string `json:"status"`
}
type MyProfileView struct {
	FullName      string             `json:"fullName"`
	Email         string             `json:"email"`
	PlatformAdmin bool               `json:"platformAdmin"`
	RoleNames     []string           `json:"roleNames"`
	Permissions   []string           `json:"permissions"`
	CurrentPlan   *MyProfilePlanView `json:"currentPlan"`
}
type MyProfileInput struct {
	FullName        string `json:"fullName"`
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}
type OrgSummary struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Active        bool   `json:"active"`
	LocationCount int    `json:"locationCount"`
}
type LocationSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Code   string `json:"code"`
	Active bool   `json:"active"`
}
type SwitchContextInput struct {
	OrganizationID string `json:"organizationId"`
	LocationID     string `json:"locationId"`
}
type EffectiveRoleCatalog struct {
	Menus, Permissions []map[string]any
	Unrestricted       bool
}

var RolePermissionModules = map[string][]string{
	"dashboard": {"reportes"}, "users": {"usuarios"},
	"organizations": {"locales", "fiscal", "whatsapp_bot"}, "fiscal": {"fiscal"},
	"menu": {"productos", "combos", "recetas"}, "recipes": {"recetas"}, "customers": {"clientes"},
	"orders": {"pedidos", "pos", "cocina", "reportes"}, "kitchen": {"cocina"}, "bar": {"cocina"},
	"tables": {"mesas"}, "reservations": {"reservas"}, "cash": {"caja", "pos"},
	"receipts": {"facturacion"}, "inventory": {"inventario"}, "purchases": {"compras"},
	"expenses": {"costos"}, "reports": {"reportes"}, "audit": {"reportes"},
	"delivery": {"delivery", "repartidores"},
}

type OrganizationRoleSeed struct {
	Name        string
	SystemKey   string
	Description string
	MenuAccess  []string
	Permissions []string
}

var CompanyAdministratorMenuAccess = []string{
	"dashboard", "pos", "pedidos", "cocina", "mesas", "caja", "reservas", "productos", "disponibilidad", "combos", "recetas",
	"inventario", "kardex", "compras", "logistica", "clientes", "locales", "fiscal", "usuarios", "call_center", "carta_qr", "kiosco",
	"delivery", "delivery_apps", "repartidores", "crm", "puntos", "ofertas", "personal", "costos", "bi", "app_manager", "facturacion",
	"integraciones", "whatsapp_bot",
}
var CompanyAdministratorPermissions = []string{
	"dashboard.read", "users.read", "users.manage", "organizations.read", "organizations.manage", "fiscal.read", "subscription.read",
	"menu.read", "menu.manage", "recipes.manage", "customers.read", "customers.manage",
	"orders.read", "orders.manage", "kitchen.manage", "bar.manage", "tables.read", "tables.manage", "reservations.read", "reservations.manage",
	"cash.read", "cash.registers.manage", "cash.manage", "cash.expected.read", "receipts.read", "receipts.manage",
	"inventory.read", "inventory.manage", "inventory.transfer",
	"purchases.read", "purchases.manage", "purchases.approve", "purchases.receive", "expenses.read", "expenses.manage", "expenses.void",
	"reports.read", "audit.read", "delivery.read", "delivery.manage",
}
var DefaultOrganizationRoles = []OrganizationRoleSeed{
	{Name: "Administrador de empresa", SystemKey: "administrator", Description: "Administra la empresa, sus locales, equipo y operación sin privilegios de plataforma", MenuAccess: CompanyAdministratorMenuAccess, Permissions: CompanyAdministratorPermissions},
	{Name: "Gerente de local", SystemKey: "location_manager", Description: "Gestiona la operación completa de los locales asignados", MenuAccess: []string{"dashboard", "pos", "pedidos", "cocina", "mesas", "caja", "reservas", "productos", "disponibilidad", "combos", "recetas", "inventario", "kardex", "compras", "clientes", "locales", "fiscal", "usuarios", "costos"}, Permissions: []string{"dashboard.read", "organizations.read", "organizations.manage", "users.read", "users.manage", "menu.read", "menu.manage", "recipes.manage", "customers.read", "customers.manage", "orders.read", "orders.manage", "kitchen.manage", "tables.read", "tables.manage", "cash.read", "cash.manage", "cash.expected.read", "reservations.read", "reservations.manage", "inventory.read", "inventory.manage", "inventory.transfer", "purchases.read", "purchases.manage", "purchases.approve", "purchases.receive", "expenses.read", "expenses.manage", "expenses.void", "reports.read"}},
	{Name: "Supervisor de turno", SystemKey: "shift_supervisor", Description: "Supervisa atención, comandas, mesas, caja y reservas", MenuAccess: []string{"dashboard", "pos", "pedidos", "cocina", "mesas", "caja", "reservas", "productos", "disponibilidad", "combos", "clientes"}, Permissions: []string{"dashboard.read", "menu.read", "customers.read", "customers.manage", "orders.read", "orders.manage", "kitchen.manage", "tables.read", "tables.manage", "cash.read", "cash.manage", "cash.expected.read", "reservations.read", "reservations.manage"}},
	{Name: "Cajero", SystemKey: "cashier", Description: "Opera caja, cobros y arqueos", MenuAccess: []string{"pos", "pedidos", "mesas", "caja", "clientes"}, Permissions: []string{"menu.read", "customers.read", "customers.manage", "orders.read", "orders.manage", "tables.read", "cash.read", "cash.manage"}},
	{Name: "Mesero", SystemKey: "waiter", Description: "Gestiona mesas, clientes, pedidos y reservas", MenuAccess: []string{"pedidos", "mesas", "reservas"}, Permissions: []string{"menu.read", "customers.read", "customers.manage", "orders.read", "orders.manage", "tables.read", "reservations.read", "reservations.manage"}},
	{Name: "Encargado de barra", SystemKey: "bartender", Description: "Prepara las bebidas de Barra", MenuAccess: []string{"cocina"}, Permissions: []string{"menu.read", "orders.read", "bar.manage"}},
	{Name: "Cocinero", SystemKey: "cook", Description: "Consulta y actualiza comandas de cocina", MenuAccess: []string{"cocina", "disponibilidad"}, Permissions: []string{"menu.read", "orders.read", "kitchen.manage"}},
	{Name: "Jefe de cocina", SystemKey: "kitchen_manager", Description: "Gestiona comandas, disponibilidad, recetas y producción", MenuAccess: []string{"cocina", "productos", "disponibilidad", "recetas", "inventario"}, Permissions: []string{"menu.read", "menu.manage", "orders.read", "kitchen.manage", "recipes.manage", "inventory.read"}},
	{Name: "Almacenero", SystemKey: "warehouse", Description: "Gestiona inventario, kardex y recepción", MenuAccess: []string{"inventario", "kardex", "compras"}, Permissions: []string{"inventory.read", "inventory.manage", "inventory.transfer", "purchases.read", "purchases.receive"}},
	{Name: "Compras", SystemKey: "buyer", Description: "Gestiona proveedores y órdenes de compra", MenuAccess: []string{"inventario", "kardex", "compras"}, Permissions: []string{"inventory.read", "purchases.read", "purchases.manage", "purchases.approve", "purchases.receive"}},
	{Name: "Contabilidad", SystemKey: "accounting", Description: "Consulta cierres, ventas, gastos e información fiscal", MenuAccess: []string{"dashboard", "caja", "fiscal", "costos"}, Permissions: []string{"dashboard.read", "cash.read", "cash.expected.read", "expenses.read", "expenses.manage", "expenses.void", "reports.read", "fiscal.read", "organizations.read"}},
	{Name: "Auditor", SystemKey: "auditor", Description: "Acceso de solo lectura a operación e historial", MenuAccess: []string{"dashboard", "productos", "clientes", "inventario", "kardex", "compras", "costos"}, Permissions: []string{"dashboard.read", "menu.read", "customers.read", "orders.read", "cash.read", "inventory.read", "purchases.read", "expenses.read", "reports.read", "audit.read"}},
}

func NormalizeRoleCatalogValues(values []string, catalog []map[string]any) ([]string, bool) {
	allowed := map[string]bool{}
	for _, group := range catalog {
		items, ok := group["items"].([]map[string]string)
		if !ok {
			continue
		}
		for _, item := range items {
			allowed[item["value"]] = true
		}
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" || value == "*" || !allowed[value] || seen[value] {
			if value == "" || value == "*" || !allowed[value] {
				return nil, false
			}
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out, len(out) > 0
}
func PermissionAvailableForModules(permission string, modules map[string]bool) bool {
	// Subscription is a company-level account capability, not an optional module.
	if permission == "subscription.read" {
		return true
	}
	if permission == "inventory.read" && modules["kardex"] {
		return true
	}
	prefix, _, _ := strings.Cut(permission, ".")
	for _, module := range RolePermissionModules[prefix] {
		if modules[module] && (module != "recetas" || modules["inventario"]) {
			return true
		}
	}
	return false
}
func FilterRoleCatalog(groups []map[string]any, allowed func(string) bool) []map[string]any {
	result := []map[string]any{}
	for _, group := range groups {
		items := []map[string]string{}
		for _, item := range group["items"].([]map[string]string) {
			if allowed(item["value"]) {
				items = append(items, item)
			}
		}
		if len(items) > 0 {
			result = append(result, map[string]any{"group": group["group"], "items": items})
		}
	}
	return result
}
func RoleCatalogForModules(modules map[string]bool) EffectiveRoleCatalog {
	menus := FilterRoleCatalog(MenuAccessCatalog, func(key string) bool {
		switch key {
		case "dashboard":
			key = "reportes"
		case "disponibilidad":
			key = "productos"
		}
		return modules[key] && (key != "recetas" || modules["inventario"])
	})
	permissions := FilterRoleCatalog(PermissionCatalog, func(key string) bool {
		return PermissionAvailableForModules(key, modules)
	})
	return EffectiveRoleCatalog{Menus: menus, Permissions: permissions}
}
func RoleCatalogValues(groups []map[string]any) []string {
	values := []string{}
	for _, group := range groups {
		for _, item := range group["items"].([]map[string]string) {
			values = append(values, item["value"])
		}
	}
	return values
}
func EffectiveRoleValues(values []string, groups []map[string]any) []string {
	allowed := RoleCatalogValues(groups)
	selected := map[string]bool{}
	for _, value := range values {
		selected[value] = true
	}
	result := []string{}
	for _, value := range allowed {
		if selected["*"] || selected[value] {
			result = append(result, value)
		}
	}
	return result
}
func (catalog EffectiveRoleCatalog) Project(role RoleView) RoleView {
	if !catalog.Unrestricted {
		role.MenuAccess = EffectiveRoleValues(role.MenuAccess, catalog.Menus)
		role.Permissions = EffectiveRoleValues(role.Permissions, catalog.Permissions)
	}
	return role
}

type ContextRecord struct {
	OrganizationName, LocationName, Country, Timezone string
	PlatformAdmin                                     bool
	Permissions, MenuAccess                           []string
}
type WorkContext struct {
	User           map[string]any    `json:"user"`
	Organization   map[string]string `json:"organization"`
	Location       map[string]string `json:"location"`
	Modules        map[string]bool   `json:"modules"`
	MenuAccess     []string          `json:"menuAccess"`
	Permissions    []string          `json:"permissions"`
	SetupRequired  bool              `json:"setupRequired"`
	CashCollection bool              `json:"cashCollection"`
}
