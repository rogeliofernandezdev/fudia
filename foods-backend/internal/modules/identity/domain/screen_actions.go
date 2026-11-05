package domain

// ScreenAction es una acción del catálogo tal como se presenta dentro de una
// pantalla del menú: el mismo permiso puede habilitar cosas distintas según la
// pantalla (orders.read lista Comprobantes y también muestra el plano del Salón).
type ScreenAction struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ScreenActions lista, por clave del Menú, solo las acciones que esa pantalla
// valida de verdad (rutas del API o controles de la interfaz), en el orden en que
// se usan: primero ver, luego operar. La etiqueta usa las palabras de la pantalla.
// Cada acción vive en la pantalla de su módulo: un efecto secundario en otra
// pantalla no se repite (el saldo esperado de Caja también se oculta en el
// dashboard, pero solo se muestra en Caja y turnos).
// Al proteger una pantalla con un permiso nuevo, se agrega aquí.
var ScreenActions = map[string][]ScreenAction{
	"dashboard":      {{"dashboard.read", "Ver el dashboard"}},
	"ventas":         {{"orders.read", "Ver ventas y reimprimir comprobantes"}},
	"comprobantes":   {{"orders.read", "Ver y descargar comprobantes"}, {"orders.manage", "Reenviar comprobantes a SUNAT"}},
	"pos":            {{"orders.read", "Ver pedidos por cobrar"}, {"cash.manage", "Cobrar, devolver y cerrar pedidos"}},
	"pedidos":        {{"orders.read", "Ver pedidos y su detalle"}, {"menu.read", "Ver la carta al tomar pedidos"}, {"orders.manage", "Crear, editar, cancelar y avanzar pedidos"}, {"cash.manage", "Cobrar el saldo de un pedido"}},
	"salon":          {{"orders.read", "Ver el plano del salón"}, {"menu.read", "Ver la carta al tomar pedidos"}, {"orders.manage", "Abrir mesas, agregar productos y cerrar cuentas"}},
	"cocina":         {{"orders.read", "Ver las comandas"}, {"kitchen.manage", "Iniciar y marcar listo en Cocina"}, {"bar.manage", "Iniciar y marcar listo en Barra"}},
	"mesas":          {{"tables.read", "Ver mesas"}, {"menu.read", "Ver zonas"}, {"tables.manage", "Crear, editar e imprimir QR de mesas"}, {"menu.manage", "Crear y editar zonas"}},
	"caja":           {{"cash.read", "Ver cajas, turnos y reportes de cierre"}, {"cash.registers.manage", "Registrar y editar cajas"}, {"cash.manage", "Abrir y cerrar turnos, registrar ingresos y egresos"}, {"cash.expected.read", "Ver el saldo esperado en el cierre ciego"}},
	"reservas":       {{"reservations.read", "Ver reservas"}, {"tables.read", "Ver mesas para asignarlas"}, {"reservations.manage", "Crear, confirmar, sentar y cancelar reservas"}},
	"productos":      {{"menu.read", "Ver carta y productos"}, {"menu.manage", "Crear y editar productos y categorías"}},
	"disponibilidad": {{"menu.read", "Ver la disponibilidad"}, {"menu.manage", "Agotar y reponer productos"}},
	"combos":         {{"menu.read", "Ver menús y combos"}, {"menu.manage", "Crear y editar menús y combos"}},
	"recetas":        {{"menu.read", "Ver recetas"}, {"inventory.read", "Ver insumos del inventario"}, {"recipes.manage", "Crear y editar recetas"}},
	"inventario":     {{"inventory.read", "Ver inventario"}, {"inventory.manage", "Registrar ajustes y editar artículos"}, {"inventory.transfer", "Transferir stock entre locales"}},
	"kardex":         {{"inventory.read", "Ver movimientos del kárdex"}},
	"compras":        {{"purchases.read", "Ver compras y proveedores"}, {"purchases.manage", "Crear órdenes y administrar proveedores"}, {"purchases.approve", "Aprobar órdenes de compra"}, {"purchases.receive", "Recibir compras"}},
	"clientes":       {{"customers.read", "Ver clientes"}, {"customers.manage", "Crear y editar clientes"}},
	"locales":        {{"organizations.read", "Ver locales"}, {"organizations.manage", "Crear, editar y desactivar locales"}, {"subscription.read", "Consultar el límite de locales del plan"}},
	"costos":         {{"expenses.read", "Ver gastos"}, {"expenses.manage", "Registrar gastos y categorías"}, {"expenses.void", "Anular gastos"}},
	"mi_restaurante": {{"organizations.read", "Ver los datos del restaurante"}, {"organizations.manage", "Editar los datos del restaurante"}},
	"fiscal":         {{"organizations.read", "Ver perfiles fiscales y tasas"}, {"organizations.manage", "Crear y editar perfiles fiscales y tasas"}},
	"medios_pago":    {{"organizations.read", "Ver medios de pago"}, {"organizations.manage", "Crear y editar medios de pago"}},
	"usuarios":       {{"users.read", "Ver usuarios y roles"}, {"organizations.read", "Ver locales para asignarlos"}, {"users.manage", "Crear y editar usuarios y roles"}, {"subscription.read", "Consultar el límite de usuarios del plan"}},
	"facturacion":    {{"organizations.read", "Ver la configuración de facturación"}, {"organizations.manage", "Configurar series y credenciales SUNAT"}},
	"whatsapp_bot":   {{"organizations.read", "Ver Fudia Concierge"}, {"organizations.manage", "Editar tarifas de delivery"}},
}

// ScreenActionsFor devuelve las acciones por pantalla limitadas a las pantallas y
// acciones del catálogo efectivo (plan de la empresa).
func (catalog EffectiveRoleCatalog) ScreenActionsFor() map[string][]ScreenAction {
	menus, permissions := map[string]bool{}, map[string]bool{}
	for _, value := range RoleCatalogValues(catalog.Menus) {
		menus[value] = true
	}
	for _, value := range RoleCatalogValues(catalog.Permissions) {
		permissions[value] = true
	}
	out := map[string][]ScreenAction{}
	for menu, actions := range ScreenActions {
		if !menus[menu] {
			continue
		}
		kept := []ScreenAction{}
		for _, action := range actions {
			if permissions[action.Value] {
				kept = append(kept, action)
			}
		}
		if len(kept) > 0 {
			out[menu] = kept
		}
	}
	return out
}
