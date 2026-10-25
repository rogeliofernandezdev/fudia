// Package contracts exposes stable integration DTOs for the Orders bounded context.
// Consumers do not depend on Orders use-case implementations.
package contracts

import "github.com/foods-platform/foods-backend/internal/modules/orders/domain"

type QROrderInput struct {
	CustomerName   string                  `json:"customerName"`
	CustomerPhone  string                  `json:"customerPhone"`
	ConversationID string                  `json:"conversationId"`
	RequestID      string                  `json:"requestId"`
	Notes          string                  `json:"notes"`
	Items          []domain.OrderItemInput `json:"items"`
}

type TableSubmission struct {
	Order   domain.Order
	Created bool
}

// Read-side projections published by Orders. Aliases keep existing JSON and
// transaction contracts stable while consumers stop importing domain directly.
type Order = domain.Order
type OrderItem = domain.OrderItem
type OrderItemInput = domain.OrderItemInput

var Channels = domain.Channels

// NormalizeChannel publica la compatibilidad de lectura de tipos históricos.
func NormalizeChannel(channel string) string { return domain.NormalizeChannel(channel) }

// ConciergeDeliveryInput crea un pedido de delivery o recojo sin mesa.
// Concierge calcula DeliveryFee y DistanceKm con las tarifas del local antes
// de enviarlo; Orders no recibe precios de productos desde el cliente.
type ConciergeDeliveryInput struct {
	Channel        string                  `json:"channel"`
	CustomerName   string                  `json:"customerName"`
	CustomerPhone  string                  `json:"customerPhone"`
	ConversationID string                  `json:"conversationId"`
	RequestID      string                  `json:"requestId"`
	Address        string                  `json:"address"`
	Reference      string                  `json:"reference"`
	Notes          string                  `json:"notes"`
	Latitude       *float64                `json:"latitude"`
	Longitude      *float64                `json:"longitude"`
	DistanceKm     *float64                `json:"distanceKm"`
	DeliveryFee    float64                 `json:"deliveryFee"`
	Items          []domain.OrderItemInput `json:"items"`
}

// ConciergeOrderState indica si el pedido de una conversación sigue en
// atención (mesa abierta o delivery/recojo no entregado ni cancelado).
type ConciergeOrderState struct {
	Order  domain.Order
	Active bool
}
