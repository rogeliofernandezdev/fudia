package domain

import "sort"

type ModuleDef struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Category    string `json:"category"`
}

const (
	ModuleAvailabilityReady       = "ready"
	ModuleAvailabilityDevelopment = "development"
	ModuleAvailabilityPlanned     = "planned"
)

var MvpModuleKeys = map[string]bool{
	"reportes": true, "pos": true, "pedidos": true, "cocina": true, "mesas": true, "caja": true, "reservas": true,
	"productos": true, "combos": true, "recetas": true, "inventario": true, "kardex": true, "compras": true,
	"clientes": true, "locales": true, "fiscal": true, "usuarios": true, "costos": true, "whatsapp_bot": true,
}
var DevelopmentModuleKeys = map[string]bool{
	"carta_qr":      true,
	"facturacion":   true,
	"integraciones": true,
}
var ModuleCatalog = []ModuleDef{
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
	{Key: "productos", Name: "Carta y productos", Description: "Productos, categorías, disponibilidad, alérgenos e imágenes", Icon: "utensils", Category: "Carta y producción"},
	{Key: "combos", Name: "Menús y combos", Description: "Menús compuestos, grupos de elección y alternativas", Icon: "combo", Category: "Carta y producción"},
	{Key: "recetas", Name: "Recetas", Description: "Recetas base, finales, porcionables y descartables", Icon: "cookingPot", Category: "Carta y producción"},

	// Abastecimiento
	{Key: "inventario", Name: "Inventario", Description: "Stock por local, mínimos y alertas", Icon: "stock", Category: "Abastecimiento"},
	{Key: "kardex", Name: "Kardex", Description: "Movimientos detallados de inventario", Icon: "ledger", Category: "Abastecimiento"},
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
	{Key: "usuarios", Name: "Usuarios y roles", Description: "Equipo, roles y accesos", Icon: "users", Category: "Configuración"},
	{Key: "facturacion", Name: "Facturación", Description: "Series y comprobantes electrónicos", Icon: "receipt", Category: "Configuración"},
	{Key: "integraciones", Name: "Integraciones", Description: "WhatsApp, pagos e impresión", Icon: "settings", Category: "Configuración"},
	{Key: "whatsapp_bot", Name: "Fudia Concierge", Description: "Pedidos conversacionales iniciados desde el QR por WhatsApp", Icon: "settings", Category: "Configuración"},
}

func ModuleAvailability(key string) string {
	if MvpModuleKeys[key] {
		return ModuleAvailabilityReady
	}
	if DevelopmentModuleKeys[key] {
		return ModuleAvailabilityDevelopment
	}
	return ModuleAvailabilityPlanned
}
func ModuleCanActivate(key string) bool {
	return ModuleAvailability(key) == ModuleAvailabilityReady
}
func ModuleCategories() []string {
	seen := map[string]bool{}
	cats := []string{}
	for _, m := range ModuleCatalog {
		if !seen[m.Category] {
			seen[m.Category] = true
			cats = append(cats, m.Category)
		}
	}
	sort.Strings(cats)
	return cats
}

type ModuleView struct {
	ModuleDef
	Availability string `json:"availability"`
	Active       bool   `json:"active"`
}
