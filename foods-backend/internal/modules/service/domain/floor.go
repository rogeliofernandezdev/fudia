package domain

// ServiceType es el tipo de atención de una mesa: una mesa tradicional con
// cuenta abierta u un punto de venta por mostrador con pedidos independientes.
type ServiceType string

const (
	ServiceTypeTable   ServiceType = "table"
	ServiceTypeCounter ServiceType = "counter"
	// ServiceTypeDelivery identifica el enlace/QR de delivery de un local
	// (location_delivery_links). No es una mesa: nunca se guarda en tables.
	ServiceTypeDelivery ServiceType = "delivery"
)

// Valid acepta solo los tipos persistidos en tables.service_type.
func (t ServiceType) Valid() bool {
	return t == ServiceTypeTable || t == ServiceTypeCounter
}

// ServiceTypeOrDefault conserva la mesa tradicional cuando el cliente no envía el tipo.
func ServiceTypeOrDefault(value *ServiceType) ServiceType {
	if value == nil || *value == "" {
		return ServiceTypeTable
	}
	return *value
}

type Zone struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder"`
	Active    bool   `json:"active"`
}
type ZoneInput struct {
	Name      string `json:"name"`
	SortOrder *int   `json:"sortOrder"`
	Active    *bool  `json:"active"`
}
type Table struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Seats       int         `json:"seats"`
	Zone        string      `json:"zone"`
	Active      bool        `json:"active"`
	QrToken     string      `json:"qrToken"`
	QrEnabled   bool        `json:"qrEnabled"`
	ServiceType ServiceType `json:"serviceType"`
}
type TableInput struct {
	Name        string       `json:"name"`
	Seats       *int         `json:"seats"`
	Zone        string       `json:"zone"`
	Active      *bool        `json:"active"`
	QrEnabled   *bool        `json:"qrEnabled"`
	ServiceType *ServiceType `json:"serviceType"`
}
type TableBatchInput struct {
	Items []TableInput `json:"items"`
}
type TableBatchItem struct {
	Name        string      `json:"name"`
	Seats       int         `json:"seats"`
	Zone        string      `json:"zone"`
	QrEnabled   bool        `json:"qrEnabled"`
	ServiceType ServiceType `json:"serviceType"`
}
type TableQR struct {
	CountryCode      string      `json:"countryCode"`
	OrganizationID   string      `json:"organizationId"`
	LocationID       string      `json:"locationId"`
	Name             string      `json:"name"`
	Seats            int         `json:"seats"`
	Zone             string      `json:"zone"`
	OrgName          string      `json:"organizationName"`
	LocName          string      `json:"locationName"`
	ConciergeEnabled bool        `json:"conciergeEnabled"`
	ServiceType      ServiceType `json:"serviceType"`
}
type TableBatchResult struct {
	Items []Table `json:"items"`
}
