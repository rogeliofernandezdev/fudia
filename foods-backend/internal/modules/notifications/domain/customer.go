package domain

import "strings"

// Avisos al cliente por WhatsApp. Concierge redacta el mensaje; aquí solo
// viajan los hechos del pedido.
const CustomerOrderReady = "order_ready"

// Formas de entrega que avisan «pedido listo»: el cliente recoge en el local.
const (
	FulfillmentCounter = "counter"
	FulfillmentPickup  = "pickup"
)

// CustomerInput registra un aviso pendiente para el cliente de una conversación.
type CustomerInput struct {
	Kind, OrderID, CustomerPhone, ConversationID string
}

// CustomerNotice es un aviso reclamado por Concierge para enviarlo.
type CustomerNotice struct {
	ID               string `json:"id"`
	Kind             string `json:"kind"`
	OrderID          string `json:"orderId"`
	OrderCode        string `json:"orderCode"`
	Fulfillment      string `json:"fulfillment"`
	CustomerPhone    string `json:"customerPhone"`
	ConversationID   string `json:"conversationId"`
	OrganizationName string `json:"organizationName"`
	LocationName     string `json:"locationName"`
	CreatedAt        string `json:"createdAt"`
}

// FulfillmentFor traduce el canal del pedido; vacío si el canal no avisa al cliente.
func FulfillmentFor(channel string) string {
	switch channel {
	case "mostrador":
		return FulfillmentCounter
	case "recojo":
		return FulfillmentPickup
	}
	return ""
}

// OrderReadyForCustomer arma el aviso «pedido listo» solo para mostrador o
// recojo con un cliente de WhatsApp identificado.
func OrderReadyForCustomer(orderID, channel, phone, conversationID string) (CustomerInput, bool) {
	in := CustomerInput{
		Kind: CustomerOrderReady, OrderID: strings.TrimSpace(orderID),
		CustomerPhone: strings.TrimSpace(phone), ConversationID: strings.TrimSpace(conversationID),
	}
	ok := FulfillmentFor(channel) != "" && in.OrderID != "" && in.CustomerPhone != "" &&
		len(in.CustomerPhone) <= 32 && in.ConversationID != "" && len(in.ConversationID) <= 160
	return in, ok
}
