package httpapi

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

type organizationRoleSeed struct {
	Name        string
	SystemKey   string
	Description string
	MenuAccess  []string
	Permissions []string
}

var companyAdministratorMenuAccess = []string{
	"dashboard","pos","pedidos","cocina","mesas","caja","reservas","productos","disponibilidad","combos","recetas",
	"inventario","kardex","compras","logistica","clientes","locales","fiscal","usuarios","call_center","carta_qr","kiosco",
	"delivery","delivery_apps","repartidores","crm","puntos","ofertas","personal","costos","bi","app_manager","facturacion",
	"integraciones","whatsapp_bot",
}

var companyAdministratorPermissions = []string{
	"dashboard.read","users.read","users.manage","organizations.read","organizations.manage","fiscal.read",
	"menu.read","menu.manage","recipes.manage","customers.read","customers.manage",
	"orders.read","orders.manage","kitchen.manage","tables.read","tables.manage","reservations.read","reservations.manage",
	"cash.read","cash.manage","cash.expected.read","receipts.read","receipts.manage",
	"inventory.read","inventory.manage","inventory.transfer",
	"purchases.read","purchases.manage","purchases.approve","purchases.receive",
	"reports.read","audit.read","delivery.read","delivery.manage",
}

var defaultOrganizationRoles = []organizationRoleSeed{
	{Name:"Administrador de empresa",SystemKey:"administrator",Description:"Administra la empresa, sus locales, equipo y operación sin privilegios de plataforma",MenuAccess:companyAdministratorMenuAccess,Permissions:companyAdministratorPermissions},
	{Name:"Gerente de local",SystemKey:"location_manager",Description:"Gestiona la operación completa de los locales asignados",MenuAccess:[]string{"dashboard","pos","pedidos","cocina","mesas","caja","reservas","productos","disponibilidad","combos","recetas","inventario","kardex","compras","clientes","locales","fiscal","usuarios"},Permissions:[]string{"dashboard.read","organizations.read","organizations.manage","users.read","users.manage","menu.read","menu.manage","recipes.manage","customers.read","customers.manage","orders.read","orders.manage","kitchen.manage","tables.read","tables.manage","cash.read","cash.manage","cash.expected.read","reservations.read","reservations.manage","inventory.read","inventory.manage","inventory.transfer","purchases.read","purchases.manage","purchases.approve","purchases.receive","reports.read"}},
	{Name:"Supervisor de turno",SystemKey:"shift_supervisor",Description:"Supervisa atención, comandas, mesas, caja y reservas",MenuAccess:[]string{"dashboard","pos","pedidos","cocina","mesas","caja","reservas","productos","disponibilidad","combos","clientes"},Permissions:[]string{"dashboard.read","menu.read","customers.read","customers.manage","orders.read","orders.manage","kitchen.manage","tables.read","tables.manage","cash.read","cash.manage","cash.expected.read","reservations.read","reservations.manage"}},
	{Name:"Cajero",SystemKey:"cashier",Description:"Opera caja, cobros y arqueos",MenuAccess:[]string{"pos","pedidos","mesas","caja","clientes"},Permissions:[]string{"menu.read","customers.read","customers.manage","orders.read","orders.manage","tables.read","cash.read","cash.manage"}},
	{Name:"Mesero",SystemKey:"waiter",Description:"Gestiona mesas, clientes, pedidos y reservas",MenuAccess:[]string{"pedidos","mesas","reservas"},Permissions:[]string{"menu.read","customers.read","customers.manage","orders.read","orders.manage","tables.read","reservations.read","reservations.manage"}},
	{Name:"Cocinero",SystemKey:"cook",Description:"Consulta y actualiza comandas de cocina",MenuAccess:[]string{"cocina","disponibilidad"},Permissions:[]string{"menu.read","orders.read","kitchen.manage"}},
	{Name:"Jefe de cocina",SystemKey:"kitchen_manager",Description:"Gestiona comandas, disponibilidad, recetas y producción",MenuAccess:[]string{"cocina","productos","disponibilidad","recetas","inventario"},Permissions:[]string{"menu.read","menu.manage","orders.read","kitchen.manage","recipes.manage","inventory.read"}},
	{Name:"Almacenero",SystemKey:"warehouse",Description:"Gestiona inventario, kardex y recepción",MenuAccess:[]string{"inventario","kardex","compras"},Permissions:[]string{"inventory.read","inventory.manage","inventory.transfer","purchases.read","purchases.receive"}},
	{Name:"Compras",SystemKey:"buyer",Description:"Gestiona proveedores y órdenes de compra",MenuAccess:[]string{"inventario","kardex","compras"},Permissions:[]string{"inventory.read","purchases.read","purchases.manage","purchases.approve","purchases.receive"}},
	{Name:"Contabilidad",SystemKey:"accounting",Description:"Consulta cierres, ventas e información fiscal",MenuAccess:[]string{"dashboard","caja","fiscal"},Permissions:[]string{"dashboard.read","cash.read","cash.expected.read","reports.read","fiscal.read","organizations.read"}},
	{Name:"Auditor",SystemKey:"auditor",Description:"Acceso de solo lectura a operación e historial",MenuAccess:[]string{"dashboard","productos","clientes","inventario","kardex","compras"},Permissions:[]string{"dashboard.read","menu.read","customers.read","orders.read","cash.read","inventory.read","purchases.read","reports.read","audit.read"}},
}

func seedOrganizationRoles(ctx context.Context, tx pgx.Tx, organizationID string) (string, error) {
	var administratorID string
	for _, seed := range defaultOrganizationRoles {
		var id string
		err := tx.QueryRow(ctx, `
			INSERT INTO roles(organization_id,name,system_key,description,menu_access,permissions,active)
			VALUES($1,$2,$3,$4,$5,$6,true)
			ON CONFLICT(organization_id,name) DO NOTHING
			RETURNING id
		`, organizationID, seed.Name, seed.SystemKey, seed.Description, seed.MenuAccess, seed.Permissions).Scan(&id)
		if err != nil && err != pgx.ErrNoRows {
			return "", err
		}
		if err == pgx.ErrNoRows {
			if err = tx.QueryRow(ctx, `SELECT id FROM roles WHERE organization_id=$1 AND name=$2`, organizationID, seed.Name).Scan(&id); err != nil {
				return "", err
			}
		}
		if seed.SystemKey == "administrator" {
			administratorID = id
		}
	}
	return administratorID, nil
}

func normalizeRoleCatalogValues(values []string, catalog []map[string]any) ([]string, bool) {
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

type identityQueryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func tenantHasAdministrator(ctx context.Context, q identityQueryRower, organizationID string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM users u
			JOIN user_roles ur ON ur.user_id=u.id
			JOIN roles r ON r.id=ur.role_id AND r.organization_id=u.organization_id
			JOIN locations l ON l.id=ur.location_id AND l.organization_id=u.organization_id
			WHERE u.organization_id=$1
			  AND u.active
			  AND NOT u.platform_admin
			  AND r.active
			  AND l.active
			  AND r.permissions @> ARRAY['users.manage']::text[]
		)
	`, organizationID).Scan(&ok)
	return ok, err
}

func actorIsPlatformAdmin(ctx context.Context, q identityQueryRower, userID string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT platform_admin FROM users WHERE id=$1 AND active`, userID).Scan(&ok)
	return ok, err
}
