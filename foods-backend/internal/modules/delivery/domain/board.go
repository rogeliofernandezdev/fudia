package domain

// Opción «Delivery» del restaurante: tablero de sus deliveries, centrales que
// cubren su local y entregas fallidas. Son lecturas con el alcance de la sesión.

// BoardHistoryHours es cuánto conserva el tablero los deliveries ya cerrados.
const BoardHistoryHours = 24

// FailedHistoryDays es cuánto lista el local sus entregas fallidas.
const FailedHistoryDays = 90

// BoardOrder es un delivery del local con su despacho a las centrales.
type BoardOrder struct {
	ID             string `json:"id"`
	Code           string `json:"code"`
	Status         string `json:"status"`
	CustomerName   string `json:"customerName"`
	CustomerPhone  string `json:"customerPhone"`
	Address        string `json:"address"`
	Total          string `json:"total"`
	CurrencySymbol string `json:"currencySymbol"`
	CreatedAt      string `json:"createdAt"`
	// DispatchStatus vacío: el pedido no usa centrales (delivery propio).
	DispatchStatus          string `json:"dispatchStatus"`
	PartnerName             string `json:"partnerName"`
	PartnerPhone            string `json:"partnerPhone"`
	CourierName             string `json:"courierName"`
	PickupWaitSeconds       int    `json:"pickupWaitSeconds"`
	PickupAlert             string `json:"pickupAlert"`
	PickupAlertAcknowledged bool   `json:"pickupAlertAcknowledged"`
	IssueReportedAt         string `json:"issueReportedAt,omitempty"`
	// FailedDelivery: la central registró la entrega fallida.
	FailedDelivery bool   `json:"failedDelivery"`
	AcceptedAt     string `json:"acceptedAt,omitempty"`
	PickedUpAt     string `json:"pickedUpAt,omitempty"`
	DeliveredAt    string `json:"deliveredAt,omitempty"`
}

// LocationPartner es una central que cubre el distrito del local.
type LocationPartner struct {
	Name          string `json:"name"`
	WhatsAppPhone string `json:"whatsappPhone"`
}

// LocationPartners responde qué centrales atienden el local.
type LocationPartners struct {
	// Enabled: Delivery Fudia está activo para la empresa.
	Enabled      bool              `json:"enabled"`
	DistrictName string            `json:"districtName"`
	Items        []LocationPartner `json:"items"`
}

// LocationFailedDelivery es una entrega fallida vista por el local.
type LocationFailedDelivery struct {
	FailedDelivery
	PartnerName string `json:"partnerName"`
}
