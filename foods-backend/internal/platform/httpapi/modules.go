package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
)

// ModuleCatalog define el catálogo de módulos disponibles en la plataforma.
// El orden de las categorías y módulos aquí define el orden de presentación.
type moduleDef struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Category    string `json:"category"`
}

var moduleCatalog = []moduleDef{
	// Control
	{Key: "reportes", Name: "Reportes", Description: "Informes de ventas, financieros y consolidados", Icon: "grid", Category: "Control"},

	// Operación
	{Key: "pos", Name: "Punto de venta", Description: "Cobro rápido, comprobantes y facturación", Icon: "sales", Category: "Operación"},
	{Key: "pedidos", Name: "Pedidos", Description: "Pedidos de salón, para llevar y delivery", Icon: "receipt", Category: "Operación"},
	{Key: "cocina", Name: "Cocina", Description: "KDS con tiempos y estados de preparación", Icon: "kitchen", Category: "Operación"},
	{Key: "mesas", Name: "Mesas y zonas", Description: "Gestión de mesas, zonas y estado de ocupación", Icon: "grid", Category: "Operación"},
	{Key: "caja", Name: "Caja y turnos", Description: "Apertura/cierre de caja, ingresos y egresos", Icon: "sales", Category: "Operación"},
	{Key: "reservas", Name: "Reservas", Description: "Reservas de mesa por fecha, hora y comensales", Icon: "clock", Category: "Operación"},
	{Key: "call_center", Name: "Call center", Description: "Central de pedidos telefónicos", Icon: "receipt", Category: "Operación"},
	{Key: "carta_qr", Name: "Carta digital QR", Description: "Menú escaneable desde la mesa vía QR", Icon: "qr", Category: "Operación"},
	{Key: "kiosco", Name: "Kiosco de autoservicio", Description: "Autopedido en kiosco físico", Icon: "grid", Category: "Operación"},

	// Carta y producción
	{Key: "productos", Name: "Carta y productos", Description: "Productos, categorías, alérgenos e imágenes", Icon: "utensils", Category: "Carta y producción"},
	{Key: "combos", Name: "Menús y combos", Description: "Menús compuestos, grupos de elección y alternativas", Icon: "combo", Category: "Carta y producción"},
	{Key: "recetas", Name: "Recetas y producción", Description: "Recetas base, finales, porcionables y descartables", Icon: "chefHat", Category: "Carta y producción"},

	// Abastecimiento
	{Key: "inventario", Name: "Inventario", Description: "Stock por local, mínimos y alertas", Icon: "stock", Category: "Abastecimiento"},
	{Key: "kardex", Name: "Kardex", Description: "Movimientos detallados de inventario", Icon: "stock", Category: "Abastecimiento"},
	{Key: "compras", Name: "Compras", Description: "Órdenes de compra, proveedores y recepción", Icon: "truck", Category: "Abastecimiento"},
	{Key: "logistica", Name: "Logística", Description: "Distribución entre locales y ventas logísticas", Icon: "truck", Category: "Abastecimiento"},

	// Delivery
	{Key: "delivery", Name: "Delivery propio", Description: "Delivery con flota propia, zonas y tarifas", Icon: "truck", Category: "Delivery"},
	{Key: "delivery_apps", Name: "Apps de delivery", Description: "Integración con Rappi, PedidosYa y UberEats", Icon: "settings", Category: "Delivery"},
	{Key: "repartidores", Name: "App repartidores", Description: "App para repartidores con tracking", Icon: "truck", Category: "Delivery"},

	// Negocio
	{Key: "clientes", Name: "Clientes", Description: "Directorio de clientes e historial", Icon: "users", Category: "Negocio"},
	{Key: "crm", Name: "CRM y fidelización", Description: "Segmentación, campañas y fidelización", Icon: "users", Category: "Negocio"},
	{Key: "puntos", Name: "Plaza puntos", Description: "Sistema de puntos acumulables", Icon: "users", Category: "Negocio"},
	{Key: "ofertas", Name: "Ofertas y descuentos", Description: "Promociones, cupones y descuentos por horario", Icon: "sales", Category: "Negocio"},
	{Key: "personal", Name: "Personal y asistencias", Description: "Reloj checador, horarios y tardanzas", Icon: "users", Category: "Negocio"},
	{Key: "locales", Name: "Locales", Description: "Gestión de sedes y configuración por local", Icon: "store", Category: "Negocio"},

	// Inteligencia
	{Key: "costos", Name: "Costos y gastos", Description: "Costos de producción, gastos operativos y rentabilidad", Icon: "stock", Category: "Inteligencia"},
	{Key: "bi", Name: "Restaurant BI", Description: "Business intelligence y analítica avanzada", Icon: "grid", Category: "Inteligencia"},
	{Key: "app_manager", Name: "App manager", Description: "Supervisión operativa remota", Icon: "settings", Category: "Inteligencia"},

	// Configuración
	{Key: "fiscal", Name: "Fiscal y moneda", Description: "Perfiles fiscales, monedas, impuestos y tasas", Icon: "receipt", Category: "Configuración"},
	{Key: "usuarios", Name: "Usuarios y permisos", Description: "Equipo, roles y accesos", Icon: "users", Category: "Configuración"},
	{Key: "facturacion", Name: "Facturación", Description: "Series y comprobantes electrónicos", Icon: "receipt", Category: "Configuración"},
	{Key: "integraciones", Name: "Integraciones", Description: "WhatsApp, pagos e impresión", Icon: "settings", Category: "Configuración"},
	{Key: "whatsapp_bot", Name: "WhatsApp IA para pedidos", Description: "Bot de pedidos por WhatsApp con IA", Icon: "settings", Category: "Configuración"},
}

func (a *API) listModules(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	rows, err := a.db.Query(r.Context(), `SELECT module_key, active FROM organization_modules WHERE organization_id=$1`, s.OrganizationID)
	if err != nil {
		fail(w, 503, "modules_unavailable", "No pudimos cargar los módulos.")
		return
	}
	defer rows.Close()
	activeMap := map[string]bool{}
	for rows.Next() {
		var key string
		var active bool
		_ = rows.Scan(&key, &active)
		activeMap[key] = active
	}
	// Módulos que no están en la tabla se asumen activos (compatibilidad)
	catalog := make([]map[string]any, 0, len(moduleCatalog))
	for _, m := range moduleCatalog {
		active, exists := activeMap[m.Key]
		if !exists {
			active = true
		}
		catalog = append(catalog, map[string]any{
			"key":         m.Key,
			"name":        m.Name,
			"description": m.Description,
			"icon":        m.Icon,
			"category":    m.Category,
			"active":      active,
		})
	}
	writeJSON(w, 200, map[string]any{"modules": catalog})
}

func (a *API) toggleModule(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		Key    string `json:"key"`
		Active bool   `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.Key == "" {
		fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	// Validar que el módulo existe en el catálogo
	valid := false
	for _, m := range moduleCatalog {
		if m.Key == in.Key {
			valid = true
			break
		}
	}
	if !valid {
		fail(w, 400, "invalid_module", "El módulo no existe.")
		return
	}
	_, err := a.db.Exec(r.Context(), `INSERT INTO organization_modules(organization_id, module_key, active) VALUES($1,$2,$3) ON CONFLICT(organization_id, module_key) DO UPDATE SET active=EXCLUDED.active, updated_at=now()`, s.OrganizationID, in.Key, in.Active)
	if err != nil {
		fail(w, 503, "module_unavailable", "No pudimos actualizar el módulo.")
		return
	}
	a.audit(r, "module.toggled", "module", in.Key)
	w.WriteHeader(204)
}

// activeModules devuelve el set de claves de módulos activos para una organización.
// Se usa para filtrar el sidebar y las rutas disponibles.
func (a *API) activeModules(orgID string) map[string]bool {
	rows, err := a.db.Query(context.Background(), `SELECT module_key, active FROM organization_modules WHERE organization_id=$1`, orgID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := map[string]bool{}
	for rows.Next() {
		var key string
		var active bool
		_ = rows.Scan(&key, &active)
		result[key] = active
	}
	return result
}

// moduleCategories ordena las categorías del catálogo para presentación.
func moduleCategories() []string {
	seen := map[string]bool{}
	cats := []string{}
	for _, m := range moduleCatalog {
		if !seen[m.Category] {
			seen[m.Category] = true
			cats = append(cats, m.Category)
		}
	}
	sort.Strings(cats)
	return cats
}
