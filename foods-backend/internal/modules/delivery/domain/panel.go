package domain

import "regexp"

// Panel web del proveedor de Delivery Fudia. La central abre su panel con un
// enlace privado: ve los pedidos que se le ofrecen, confirma la atención, la
// recogida y la entrega, y comparte la ubicación del repartidor.

// DispatchDelivered: la central confirmó la entrega al cliente.
const DispatchDelivered = "delivered"

// ActionDeliver confirma la entrega; solo existe en el panel web.
const ActionDeliver = "deliver"

// Resultados propios de la entrega.
const (
	OutcomeDelivered   = "delivered"
	OutcomeNotPickedUp = "not_picked_up"
)

// Etapas de un pedido en el panel de la central.
const (
	StageOffer     = "offer"
	StageAccepted  = "accepted"
	StagePickedUp  = "picked_up"
	StageDelivered = "delivered"
)

// PanelHistoryHours es cuánto conserva el panel los pedidos ya entregados.
const PanelHistoryHours = 12

// CourierLocationMaxAgeMinutes: una ubicación más antigua ya no se muestra al cliente.
const CourierLocationMaxAgeMinutes = 15

var panelTokenPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

// ValidPanelToken valida la forma del enlace privado antes de consultar la base.
func ValidPanelToken(token string) bool { return panelTokenPattern.MatchString(token) }

// ValidPanelAction acepta las acciones del panel: las de la oferta y la entrega.
func ValidPanelAction(action string) bool {
	return ValidAction(action) || action == ActionDeliver
}

// WithPartner indica si el reparto está en manos de una central.
func WithPartner(status string) bool {
	return status == DispatchAccepted || status == DispatchPickedUp || status == DispatchDelivered
}

// PanelLink es el enlace privado del panel de una central.
type PanelLink struct {
	PartnerID  string `json:"partnerId"`
	PanelToken string `json:"panelToken"`
}

// PanelOrder es un pedido ofrecido o en reparto, con lo que la central
// necesita para recogerlo y entregarlo.
type PanelOrder struct {
	// Token identifica la oferta en las acciones del panel.
	Token             string        `json:"token"`
	Stage             string        `json:"stage"`
	OrderCode         string        `json:"orderCode"`
	OrderStatus       string        `json:"orderStatus"`
	OrganizationName  string        `json:"organizationName"`
	LocationName      string        `json:"locationName"`
	LocationAddress   string        `json:"locationAddress"`
	LocationLatitude  *float64      `json:"locationLatitude"`
	LocationLongitude *float64      `json:"locationLongitude"`
	CustomerName      string        `json:"customerName"`
	CustomerPhone     string        `json:"customerPhone"`
	Address           string        `json:"address"`
	Reference         string        `json:"reference"`
	Latitude          *float64      `json:"latitude"`
	Longitude         *float64      `json:"longitude"`
	DistanceKm        *float64      `json:"distanceKm"`
	Notes             string        `json:"notes"`
	Items             []MessageItem `json:"items"`
	CurrencySymbol    string        `json:"currencySymbol"`
	// Subtotal es lo que la central paga al local; Total, lo que cobra al cliente.
	Subtotal    string `json:"subtotal"`
	DeliveryFee string `json:"deliveryFee"`
	Total       string `json:"total"`
	// ExpiresInSeconds: tiempo restante para responder la oferta (0 fuera de la oferta).
	ExpiresInSeconds int    `json:"expiresInSeconds"`
	CreatedAt        string `json:"createdAt"`
	AcceptedAt       string `json:"acceptedAt,omitempty"`
	PickedUpAt       string `json:"pickedUpAt,omitempty"`
	DeliveredAt      string `json:"deliveredAt,omitempty"`
}

// PartnerPanel es lo que ve la central al abrir su panel.
type PartnerPanel struct {
	PartnerName string `json:"partnerName"`
	// Active false: la central está desactivada y ya no recibe ofertas.
	Active bool         `json:"active"`
	Orders []PanelOrder `json:"orders"`
}

// PanelActionInput es una acción del panel sobre una oferta.
type PanelActionInput struct {
	Action string `json:"action"`
	Token  string `json:"token"`
}

// CourierLocation es la ubicación que comparte el dispositivo del repartidor.
type CourierLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Point es una coordenada del seguimiento.
type Point struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Tracking es el seguimiento de un pedido de delivery para el cliente: de
// dónde sale, a dónde va y, si una central comparte su ubicación, por dónde va.
type Tracking struct {
	// DispatchStatus vacío: el pedido no usa centrales (delivery propio).
	DispatchStatus     string `json:"dispatchStatus"`
	PartnerName        string `json:"partnerName"`
	OriginName         string `json:"originName"`
	OriginAddress      string `json:"originAddress"`
	Origin             *Point `json:"origin"`
	DestinationAddress string `json:"destinationAddress"`
	Destination        *Point `json:"destination"`
	Courier            *Point `json:"courier"`
	CourierLocatedAt   string `json:"courierLocatedAt,omitempty"`
	AcceptedAt         string `json:"acceptedAt,omitempty"`
	PickedUpAt         string `json:"pickedUpAt,omitempty"`
	DeliveredAt        string `json:"deliveredAt,omitempty"`
}
