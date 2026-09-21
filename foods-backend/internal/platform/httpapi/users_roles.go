package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type roleView struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	SystemKey   *string  `json:"systemKey"`
	Description string   `json:"description"`
	MenuAccess  []string `json:"menuAccess"`
	Permissions []string `json:"permissions"`
	Active      bool     `json:"active"`
	UserCount   int      `json:"userCount"`
}
type roleInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	MenuAccess  []string `json:"menuAccess"`
	Permissions []string `json:"permissions"`
}
type userAssignment struct {
	RoleID       string `json:"roleId"`
	RoleName     string `json:"roleName,omitempty"`
	LocationID   string `json:"locationId"`
	LocationName string `json:"locationName,omitempty"`
}
type userView struct {
	ID            string           `json:"id"`
	FullName      string           `json:"fullName"`
	Email         string           `json:"email"`
	Active        bool             `json:"active"`
	PlatformAdmin bool             `json:"platformAdmin"`
	Assignments   []userAssignment `json:"assignments"`
}
type userInput struct {
	FullName    string           `json:"fullName"`
	Email       string           `json:"email"`
	Password    string           `json:"password"`
	Assignments []userAssignment `json:"assignments"`
}

var permissionCatalog = []map[string]any{
	{"group": "Administración", "items": []map[string]string{{"value": "dashboard.read", "label": "Ver resumen operativo"}, {"value": "users.read", "label": "Ver usuarios y roles"}, {"value": "users.manage", "label": "Administrar usuarios y roles"}, {"value": "organizations.read", "label": "Ver empresa y locales"}, {"value": "organizations.manage", "label": "Administrar empresa y locales"}, {"value": "fiscal.read", "label": "Ver información fiscal"}}},
	{"group": "Carta y clientes", "items": []map[string]string{{"value": "menu.read", "label": "Ver carta"}, {"value": "menu.manage", "label": "Administrar carta"}, {"value": "recipes.manage", "label": "Administrar recetas"}, {"value": "customers.read", "label": "Ver clientes"}, {"value": "customers.manage", "label": "Administrar clientes"}}},
	{"group": "Operación", "items": []map[string]string{{"value": "orders.read", "label": "Ver pedidos y comandas"}, {"value": "orders.manage", "label": "Gestionar pedidos"}, {"value": "kitchen.manage", "label": "Actualizar comandas"}, {"value": "tables.read", "label": "Ver mesas"}, {"value": "tables.manage", "label": "Administrar mesas"}, {"value": "cash.read", "label": "Ver caja"}, {"value": "cash.manage", "label": "Operar caja"}, {"value": "cash.expected.read", "label": "Ver saldo esperado en cierre ciego"}, {"value": "receipts.read", "label": "Ver comprobantes"}, {"value": "receipts.manage", "label": "Emitir comprobantes"}}},
	{"group": "Abastecimiento y análisis", "items": []map[string]string{{"value": "inventory.read", "label": "Ver inventario"}, {"value": "inventory.manage", "label": "Administrar inventario"}, {"value": "purchases.read", "label": "Ver compras"}, {"value": "purchases.manage", "label": "Administrar compras"}, {"value": "purchases.receive", "label": "Recibir compras"}, {"value": "reports.read", "label": "Ver reportes"}, {"value": "audit.read", "label": "Ver auditoría"}}},
	{"group": "Delivery", "items": []map[string]string{{"value": "delivery.read", "label": "Ver entregas asignadas"}, {"value": "delivery.manage", "label": "Actualizar entregas"}}},
}

var menuAccessCatalog = []map[string]any{
	{"group": "Control", "items": []map[string]string{{"value": "dashboard", "label": "Reportes"}}},
	{"group": "Operación", "items": []map[string]string{{"value": "pos", "label": "Punto de venta"}, {"value": "pedidos", "label": "Pedidos"}, {"value": "cocina", "label": "Cocina"}, {"value": "mesas", "label": "Mesas y zonas"}, {"value": "caja", "label": "Caja y turnos"}, {"value": "reservas", "label": "Reservas"}, {"value": "call_center", "label": "Call center"}, {"value": "carta_qr", "label": "Carta digital QR"}, {"value": "kiosco", "label": "Kiosco de autoservicio"}}},
	{"group": "Carta y producción", "items": []map[string]string{{"value": "productos", "label": "Carta y productos"}, {"value": "disponibilidad", "label": "Disponibilidad de la carta"}, {"value": "combos", "label": "Menús y combos"}, {"value": "recetas", "label": "Recetas y producción"}}},
	{"group": "Abastecimiento", "items": []map[string]string{{"value": "inventario", "label": "Inventario"}, {"value": "kardex", "label": "Kardex"}, {"value": "compras", "label": "Compras"}, {"value": "logistica", "label": "Logística"}}},
	{"group": "Delivery", "items": []map[string]string{{"value": "delivery", "label": "Delivery propio"}, {"value": "delivery_apps", "label": "Apps de delivery"}, {"value": "repartidores", "label": "App repartidores"}}},
	{"group": "Negocio", "items": []map[string]string{{"value": "clientes", "label": "Clientes"}, {"value": "crm", "label": "CRM y fidelización"}, {"value": "puntos", "label": "Plaza puntos"}, {"value": "ofertas", "label": "Ofertas y descuentos"}, {"value": "personal", "label": "Personal y asistencias"}, {"value": "locales", "label": "Locales"}}},
	{"group": "Inteligencia", "items": []map[string]string{{"value": "costos", "label": "Costos y gastos"}, {"value": "bi", "label": "Restaurant BI"}, {"value": "app_manager", "label": "App manager"}}},
	{"group": "Configuración", "items": []map[string]string{{"value": "fiscal", "label": "Fiscal y moneda"}, {"value": "usuarios", "label": "Usuarios y permisos"}, {"value": "facturacion", "label": "Facturación"}, {"value": "integraciones", "label": "Integraciones"}, {"value": "whatsapp_bot", "label": "WhatsApp IA para pedidos"}}},
}

func (a *API) listRoles(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	rows, err := a.db.Query(r.Context(), `SELECT r.id,r.name,r.system_key,r.description,r.menu_access,r.permissions,r.active,count(DISTINCT ur.user_id) FROM roles r LEFT JOIN user_roles ur ON ur.role_id=r.id WHERE r.organization_id=$1 GROUP BY r.id ORDER BY (r.system_key='administrator') DESC,r.name`, s.OrganizationID)
	if err != nil {
		fail(w, 503, "roles_unavailable", "No pudimos cargar los roles.")
		return
	}
	defer rows.Close()
	items := []roleView{}
	for rows.Next() {
		var v roleView
		if rows.Scan(&v.ID, &v.Name, &v.SystemKey, &v.Description, &v.MenuAccess, &v.Permissions, &v.Active, &v.UserCount) != nil {
			fail(w, 503, "roles_unavailable", "No pudimos cargar los roles.")
			return
		}
		items = append(items, v)
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": len(items)})
}
func (a *API) getPermissionCatalog(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"groups": permissionCatalog, "menuGroups": menuAccessCatalog})
}
func (a *API) createRole(w http.ResponseWriter, r *http.Request) { a.saveRole(w, r, false) }
func (a *API) updateRole(w http.ResponseWriter, r *http.Request) { a.saveRole(w, r, true) }
func (a *API) saveRole(w http.ResponseWriter, r *http.Request, update bool) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in roleInput
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Name) == "" || len(in.MenuAccess) == 0 || len(in.Permissions) == 0 {
		fail(w, 400, "invalid_role", "Ingresa un nombre, un acceso al sistema y un permiso.")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	var v roleView
	var err error
	if update {
		err = a.db.QueryRow(r.Context(), `UPDATE roles SET name=CASE WHEN system_key IS NULL THEN $3 ELSE name END,description=CASE WHEN system_key IS NULL THEN $4 ELSE description END,menu_access=$5,permissions=$6,updated_at=now() WHERE id=$1 AND organization_id=$2 AND COALESCE(system_key,'')<>'administrator' RETURNING id,name,system_key,description,menu_access,permissions,active,0`, r.PathValue("id"), s.OrganizationID, in.Name, strings.TrimSpace(in.Description), in.MenuAccess, in.Permissions).Scan(&v.ID, &v.Name, &v.SystemKey, &v.Description, &v.MenuAccess, &v.Permissions, &v.Active, &v.UserCount)
	} else {
		err = a.db.QueryRow(r.Context(), `INSERT INTO roles(organization_id,name,description,menu_access,permissions) VALUES($1,$2,$3,$4,$5) RETURNING id,name,system_key,description,menu_access,permissions,active,0`, s.OrganizationID, in.Name, strings.TrimSpace(in.Description), in.MenuAccess, in.Permissions).Scan(&v.ID, &v.Name, &v.SystemKey, &v.Description, &v.MenuAccess, &v.Permissions, &v.Active, &v.UserCount)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 409, "protected_role", "El rol Administrador con acceso total no puede modificarse.")
		return
	}
	if err != nil {
		fail(w, 409, "role_conflict", "Ya existe un rol con ese nombre.")
		return
	}
	a.audit(r, map[bool]string{true: "role.updated", false: "role.created"}[update], "role", v.ID)
	writeJSON(w, map[bool]int{true: 200, false: 201}[update], v)
}
func (a *API) updateRoleStatus(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		Active bool `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_status", "Estado inválido.")
		return
	}
	tag, err := a.db.Exec(r.Context(), `UPDATE roles SET active=$3,updated_at=now() WHERE id=$1 AND organization_id=$2 AND COALESCE(system_key,'')<>'administrator'`, r.PathValue("id"), s.OrganizationID, in.Active)
	if err != nil || tag.RowsAffected() == 0 {
		fail(w, 409, "protected_role", "El rol Administrador no puede desactivarse.")
		return
	}
	a.audit(r, "role.status_updated", "role", r.PathValue("id"))
	w.WriteHeader(204)
}

func (a *API) listUsers(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := r.URL.Query().Get("status")
	var total int
	_ = a.db.QueryRow(r.Context(), `SELECT count(*) FROM users WHERE organization_id=$1 AND ($2='' OR full_name ILIKE '%'||$2||'%' OR email ILIKE '%'||$2||'%') AND ($3='' OR ($3='active' AND active) OR ($3='inactive' AND NOT active))`, s.OrganizationID, q, status).Scan(&total)
	rows, err := a.db.Query(r.Context(), `SELECT id,full_name,email,active,platform_admin FROM users WHERE organization_id=$1 AND ($2='' OR full_name ILIKE '%'||$2||'%' OR email ILIKE '%'||$2||'%') AND ($3='' OR ($3='active' AND active) OR ($3='inactive' AND NOT active)) ORDER BY full_name LIMIT $4 OFFSET $5`, s.OrganizationID, q, status, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "users_unavailable", "No pudimos cargar los usuarios.")
		return
	}
	defer rows.Close()
	items := []userView{}
	for rows.Next() {
		var v userView
		if rows.Scan(&v.ID, &v.FullName, &v.Email, &v.Active, &v.PlatformAdmin) != nil {
			continue
		}
		v.Assignments = []userAssignment{}
		ar, _ := a.db.Query(r.Context(), `SELECT ur.role_id,r.name,ur.location_id,l.name FROM user_roles ur JOIN roles r ON r.id=ur.role_id JOIN locations l ON l.id=ur.location_id WHERE ur.user_id=$1 AND r.organization_id=$2 ORDER BY l.name,r.name`, v.ID, s.OrganizationID)
		if ar != nil {
			for ar.Next() {
				var x userAssignment
				if ar.Scan(&x.RoleID, &x.RoleName, &x.LocationID, &x.LocationName) == nil {
					v.Assignments = append(v.Assignments, x)
				}
			}
			ar.Close()
		}
		items = append(items, v)
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size})
}
func (a *API) createUser(w http.ResponseWriter, r *http.Request) { a.saveUser(w, r, false) }
func (a *API) updateUser(w http.ResponseWriter, r *http.Request) { a.saveUser(w, r, true) }
func (a *API) saveUser(w http.ResponseWriter, r *http.Request, update bool) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in userInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_user", "Revisa los datos enviados.")
		return
	}
	in.FullName = strings.TrimSpace(in.FullName)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.FullName == "" || !strings.Contains(in.Email, "@") || (!update && len(in.Password) < 8) || len(in.Assignments) == 0 {
		fail(w, 400, "invalid_user", "Completa nombre, correo, rol, local y una contraseña de al menos 8 caracteres.")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "user_unavailable", "No pudimos guardar el usuario.")
		return
	}
	defer tx.Rollback(r.Context())
	var id string
	if update {
		if in.Password != "" {
			if len(in.Password) < 8 {
				fail(w, 400, "invalid_password", "La contraseña debe tener al menos 8 caracteres.")
				return
			}
			hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
			err = tx.QueryRow(r.Context(), `UPDATE users SET full_name=$3,email=$4,password_hash=$5,updated_at=now() WHERE id=$1 AND organization_id=$2 RETURNING id`, r.PathValue("id"), s.OrganizationID, in.FullName, in.Email, string(hash)).Scan(&id)
		} else {
			err = tx.QueryRow(r.Context(), `UPDATE users SET full_name=$3,email=$4,updated_at=now() WHERE id=$1 AND organization_id=$2 RETURNING id`, r.PathValue("id"), s.OrganizationID, in.FullName, in.Email).Scan(&id)
		}
	} else {
		hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		err = tx.QueryRow(r.Context(), `INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,$2,$3,$4) RETURNING id`, s.OrganizationID, in.Email, in.FullName, string(hash)).Scan(&id)
	}
	if err != nil {
		fail(w, 409, "user_conflict", "El correo ya está registrado.")
		return
	}
	if update {
		_, err = tx.Exec(r.Context(), `DELETE FROM user_roles WHERE user_id=$1`, id)
	}
	for _, x := range in.Assignments {
		if err != nil {
			break
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO user_roles(user_id,role_id,location_id) SELECT $1,r.id,l.id FROM roles r JOIN locations l ON l.organization_id=r.organization_id WHERE r.id=$2 AND l.id=$3 AND r.organization_id=$4 AND r.active AND l.active`, id, x.RoleID, x.LocationID, s.OrganizationID)
	}
	if err != nil || tx.Commit(r.Context()) != nil {
		fail(w, 400, "invalid_assignment", "El rol o local seleccionado no es válido.")
		return
	}
	a.audit(r, map[bool]string{true: "user.updated", false: "user.created"}[update], "user", id)
	writeJSON(w, map[bool]int{true: 200, false: 201}[update], map[string]string{"id": id})
}
func (a *API) updateUserStatus(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	if r.PathValue("id") == s.UserID {
		fail(w, 409, "self_deactivation", "No puedes desactivar tu propia cuenta.")
		return
	}
	var in struct {
		Active bool `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_status", "Estado inválido.")
		return
	}
	tag, err := a.db.Exec(r.Context(), `UPDATE users SET active=$3,updated_at=now() WHERE id=$1 AND organization_id=$2`, r.PathValue("id"), s.OrganizationID, in.Active)
	if err != nil || tag.RowsAffected() == 0 {
		fail(w, 404, "user_not_found", "El usuario no existe.")
		return
	}
	w.WriteHeader(204)
}
