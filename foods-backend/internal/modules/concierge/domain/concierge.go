package domain

type BillRequestInput struct {
	ConversationID string `json:"conversationId"`
	CustomerPhone  string `json:"customerPhone"`
	// RequestID identifica idempotentemente una solicitud de cuenta: repetirlo
	// devuelve la cuenta vigente sin registrar otra solicitud.
	RequestID string `json:"requestId"`
}
type SettingsView struct {
	Active            bool   `json:"active"`
	OrganizationName  string `json:"organizationName"`
	LocationName      string `json:"locationName"`
	ManagedByPlatform bool   `json:"managedByPlatform"`
}
type HandoffView struct {
	ID             string  `json:"id"`
	TableID        string  `json:"tableId"`
	TableName      string  `json:"tableName"`
	ConversationID string  `json:"conversationId"`
	CustomerPhone  string  `json:"customerPhone"`
	Reason         string  `json:"reason"`
	Status         string  `json:"status"`
	RequestedAt    string  `json:"requestedAt"`
	ResolvedAt     *string `json:"resolvedAt"`
	ResolvedByName *string `json:"resolvedByName"`
}
type HandoffInput struct {
	ConversationID string `json:"conversationId"`
	CustomerPhone  string `json:"customerPhone"`
	Reason         string `json:"reason"`
}

// FeeTierView es un rango de tarifa: hasta MaxDistanceKm cuesta Fee.
type FeeTierView struct {
	MaxDistanceKm string `json:"maxDistanceKm"`
	Fee           string `json:"fee"`
}

// DeliveryOptions describe cómo puede recibir su pedido un cliente cuya
// atención en mesa terminó. Recojo siempre está disponible; delivery requiere
// ubicación del local y al menos un rango de tarifa.
type DeliveryOptions struct {
	LocationName      string        `json:"locationName"`
	LocationAddress   string        `json:"locationAddress"`
	LocationPhone     string        `json:"locationPhone"`
	CurrencySymbol    string        `json:"currencySymbol"`
	PickupAvailable   bool          `json:"pickupAvailable"`
	DeliveryAvailable bool          `json:"deliveryAvailable"`
	MaxDistanceKm     string        `json:"maxDistanceKm"`
	Tiers             []FeeTierView `json:"tiers"`
}
type DeliveryQuoteInput struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}
type DeliveryQuote struct {
	Covered        bool   `json:"covered"`
	DistanceKm     string `json:"distanceKm"`
	Fee            string `json:"fee"`
	MaxDistanceKm  string `json:"maxDistanceKm"`
	CurrencySymbol string `json:"currencySymbol"`
}

// OrderStatusView indica a Concierge si la atención de su pedido sigue activa.
type OrderStatusView struct {
	OrderID       string `json:"orderId"`
	Code          string `json:"code"`
	Channel       string `json:"channel"`
	Status        string `json:"status"`
	Subtotal      string `json:"subtotal"`
	DeliveryFee   string `json:"deliveryFee"`
	Total         string `json:"total"`
	PaymentStatus string `json:"paymentStatus"`
	Active        bool   `json:"active"`
}

// DeliveryFeesView es la configuración administrativa del local actual.
type DeliveryFeesView struct {
	LocationName   string        `json:"locationName"`
	HasCoordinates bool          `json:"hasCoordinates"`
	CurrencySymbol string        `json:"currencySymbol"`
	Tiers          []FeeTierView `json:"tiers"`
}
type DeliveryFeeTierInput struct {
	MaxDistanceKm float64 `json:"maxDistanceKm"`
	Fee           float64 `json:"fee"`
}
type DeliveryFeesInput struct {
	Tiers []DeliveryFeeTierInput `json:"tiers"`
}
