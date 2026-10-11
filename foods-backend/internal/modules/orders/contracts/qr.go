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
var Channels = domain.Channels
