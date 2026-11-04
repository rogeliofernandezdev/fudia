package domain

// Requisitos entre Menú y Acciones de un rol. Una pantalla o acción marcada no
// sirve sin las acciones que el API exige para usarla (por ejemplo, Recetas
// necesita ver inventario para listar insumos). Al guardar un rol se agregan
// solas, siempre que el plan las incluya; el editor las muestra con su motivo.

// MenuRequirements: acciones que necesita cada pantalla del Menú para abrirse y
// cargar sus datos.
var MenuRequirements = map[string][]string{
	"dashboard":      {"dashboard.read"},
	"pos":            {"orders.read"},
	"pedidos":        {"orders.read", "menu.read"},
	"cocina":         {"orders.read"},
	"mesas":          {"tables.read"},
	"caja":           {"cash.read"},
	"reservas":       {"reservations.read"},
	"productos":      {"menu.read"},
	"disponibilidad": {"menu.read"},
	"combos":         {"menu.read"},
	"recetas":        {"menu.read", "inventory.read"},
	"inventario":     {"inventory.read"},
	"kardex":         {"inventory.read"},
	"compras":        {"purchases.read"},
	"clientes":       {"customers.read"},
	"locales":        {"organizations.read"},
	"costos":         {"expenses.read"},
	"fiscal":         {"organizations.read"},
	"usuarios":       {"users.read", "organizations.read"},
	"facturacion":    {"organizations.read"},
	"whatsapp_bot":   {"organizations.read"},
}

// PermissionRequirements: una acción que solo funciona junto a otras.
var PermissionRequirements = map[string][]string{
	"users.read":            {"organizations.read"},
	"users.manage":          {"users.read", "organizations.read"},
	"organizations.manage":  {"organizations.read"},
	"menu.manage":           {"menu.read"},
	"recipes.manage":        {"menu.read", "inventory.read"},
	"customers.manage":      {"customers.read"},
	"orders.manage":         {"orders.read", "menu.read"},
	"kitchen.manage":        {"orders.read"},
	"bar.manage":            {"orders.read"},
	"tables.manage":         {"tables.read"},
	"reservations.manage":   {"reservations.read"},
	"cash.manage":           {"cash.read", "orders.read"},
	"cash.registers.manage": {"cash.read"},
	"cash.expected.read":    {"cash.read"},
	"receipts.manage":       {"receipts.read"},
	"inventory.manage":      {"inventory.read"},
	"inventory.transfer":    {"inventory.read"},
	"purchases.manage":      {"purchases.read"},
	"purchases.approve":     {"purchases.read"},
	"purchases.receive":     {"purchases.read"},
	"expenses.manage":       {"expenses.read"},
	"expenses.void":         {"expenses.read"},
	"delivery.manage":       {"delivery.read"},
}

// CompleteRolePermissions agrega a las acciones del rol las que requieren sus
// pantallas y acciones (de forma transitiva), solo si available las permite.
func CompleteRolePermissions(menus, permissions []string, available func(string) bool) []string {
	out := append([]string{}, permissions...)
	seen := map[string]bool{}
	for _, permission := range out {
		seen[permission] = true
	}
	var add func(string)
	add = func(permission string) {
		if seen[permission] || !available(permission) {
			return
		}
		seen[permission] = true
		out = append(out, permission)
		for _, next := range PermissionRequirements[permission] {
			add(next)
		}
	}
	for _, menu := range menus {
		for _, permission := range MenuRequirements[menu] {
			add(permission)
		}
	}
	for _, permission := range permissions {
		for _, next := range PermissionRequirements[permission] {
			add(next)
		}
	}
	return out
}

// RequirementsFor publica solo los requisitos de las pantallas y acciones del
// catálogo efectivo de la empresa: nunca menciona opciones fuera de su plan.
func (catalog EffectiveRoleCatalog) RequirementsFor() (map[string][]string, map[string][]string) {
	menus, permissions := map[string]bool{}, map[string]bool{}
	for _, value := range RoleCatalogValues(catalog.Menus) {
		menus[value] = true
	}
	for _, value := range RoleCatalogValues(catalog.Permissions) {
		permissions[value] = true
	}
	filter := func(source map[string][]string, keys map[string]bool) map[string][]string {
		out := map[string][]string{}
		for key, required := range source {
			if !keys[key] {
				continue
			}
			kept := []string{}
			for _, permission := range required {
				if permissions[permission] {
					kept = append(kept, permission)
				}
			}
			if len(kept) > 0 {
				out[key] = kept
			}
		}
		return out
	}
	return filter(MenuRequirements, menus), filter(PermissionRequirements, permissions)
}
