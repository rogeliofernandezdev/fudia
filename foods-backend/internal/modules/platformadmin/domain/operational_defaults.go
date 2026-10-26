package domain

import (
	"errors"
	"strings"
)

// TenantRecords son los registros base que crea el onboarding en un solo envío.
type TenantRecords struct {
	OrganizationID  string
	FiscalProfileID string
	LocationID      string
	AdministratorID string
}

// DefaultTable es una mesa precargada en el primer local de una empresa nueva.
// Como al crearla desde Mesas, nace activa y con su QR habilitado; queda libre
// porque la ocupación se deriva de los pedidos abiertos.
type DefaultTable struct {
	Name      string
	Seats     int
	QREnabled bool
}

// TenantOperationalDefaults describe lo que el paso seed_operational_defaults
// agrega a una empresa nueva, además de las plantillas de gastos, zonas y cajas.
// Solo se aplica dentro del onboarding: no se usa para empresas existentes.
type TenantOperationalDefaults struct {
	ProductCategory      string
	ProductCategoryScope string
	TableZone            string
	Tables               []DefaultTable
	ServiceMode          string
}

// NewTenantOperationalDefaults devuelve la precarga vigente: categoría «General»
// para cualquier tipo de producto, «Mesa 1» y «Mesa 2» para 2 personas en la zona
// «Principal» y atención en Salón. No incluye productos, pedidos ni turnos de caja.
func NewTenantOperationalDefaults() TenantOperationalDefaults {
	return TenantOperationalDefaults{
		ProductCategory:      "General",
		ProductCategoryScope: "both",
		TableZone:            "Principal",
		Tables:               []DefaultTable{{Name: "Mesa 1", Seats: 2, QREnabled: true}, {Name: "Mesa 2", Seats: 2, QREnabled: true}},
		ServiceMode:          "dine_in",
	}
}

// Validate protege los invariantes que el esquema exige antes de abrir la escritura.
func (d TenantOperationalDefaults) Validate() error {
	if strings.TrimSpace(d.ProductCategory) == "" {
		return errors.New("default product category is required")
	}
	switch d.ProductCategoryScope {
	case "prepared", "retail", "both":
	default:
		return errors.New("default product category scope is invalid")
	}
	switch d.ServiceMode {
	case "counter", "dine_in", "mixed":
	default:
		return errors.New("default service mode is invalid")
	}
	if len(d.Tables) > 0 && strings.TrimSpace(d.TableZone) == "" {
		return errors.New("default tables require a zone")
	}
	seen := make(map[string]bool, len(d.Tables))
	for _, table := range d.Tables {
		name := strings.TrimSpace(table.Name)
		if name == "" || table.Seats < 1 {
			return errors.New("default table requires a name and at least one seat")
		}
		if seen[name] {
			return errors.New("default table names must be unique")
		}
		seen[name] = true
	}
	return nil
}

// TableNames, TableSeats y TableQREnabled preparan las mesas como arreglos
// paralelos para SQL.
func (d TenantOperationalDefaults) TableNames() []string {
	names := make([]string, len(d.Tables))
	for i, table := range d.Tables {
		names[i] = table.Name
	}
	return names
}
func (d TenantOperationalDefaults) TableSeats() []int32 {
	seats := make([]int32, len(d.Tables))
	for i, table := range d.Tables {
		seats[i] = int32(table.Seats)
	}
	return seats
}
func (d TenantOperationalDefaults) TableQREnabled() []bool {
	enabled := make([]bool, len(d.Tables))
	for i, table := range d.Tables {
		enabled[i] = table.QREnabled
	}
	return enabled
}
