package domain

import "strings"

// Avisos al cliente por WhatsApp. Concierge redacta el mensaje; aquí solo
// viajan los hechos del pedido.
const CustomerOrderReady = "order_ready"

// CustomerOrderSent avisa que el personal envió a Cocina lo que el cliente pidió
// por WhatsApp (los pedidos del bot esperan «Enviar comanda»).
const CustomerOrderSent = "order_sent"

// Formas de entrega que avisan «pedido listo»: el cliente recoge en el local.
// «En preparación» también avisa en mesa y delivery.
const (
	FulfillmentCounter  = "counter"
	FulfillmentPickup   = "pickup"
	FulfillmentTable    = "table"
	FulfillmentDelivery = "delivery"
)

// CustomerInput registra un aviso pendiente para el cliente de una conversación.
type CustomerInput struct {
	Kind, OrderID, CustomerPhone, ConversationID string
	// DedupeKey distingue los avisos de un mismo pedido y tipo (una ronda enviada).
	DedupeKey string
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

// ChannelFulfillment traduce cualquier canal del pedido a su forma de entrega.
func ChannelFulfillment(channel string) string {
	switch channel {
	case "mostrador":
		return FulfillmentCounter
	case "recojo":
		return FulfillmentPickup
	case "delivery":
		return FulfillmentDelivery
	}
	return FulfillmentTable
}

// OrderSentForCustomer arma el aviso «en preparación» para el WhatsApp que pidió
// una ronda (key) cuando el personal la envía a Cocina.
func OrderSentForCustomer(orderID, phone, conversationID, key string) (CustomerInput, bool) {
	in := CustomerInput{
		Kind: CustomerOrderSent, OrderID: strings.TrimSpace(orderID),
		CustomerPhone: strings.TrimSpace(phone), ConversationID: strings.TrimSpace(conversationID),
		DedupeKey: strings.TrimSpace(key),
	}
	ok := in.OrderID != "" && in.CustomerPhone != "" && len(in.CustomerPhone) <= 32 &&
		in.ConversationID != "" && len(in.ConversationID) <= 160 && len(in.DedupeKey) <= 200
	return in, ok
}
