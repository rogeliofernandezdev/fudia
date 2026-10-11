package httpapi

import (
	orderdomain "github.com/foods-platform/foods-backend/internal/modules/orders/domain"
)

type orderItem = orderdomain.OrderItem
type order = orderdomain.Order
type preparedOrderSelection = orderdomain.PreparedOrderSelection
type preparedOrderItem = orderdomain.PreparedOrderItem
type orderPreparationError struct {
	Status  int
	Code    string
	Message string
}

var canManageOrderService = orderdomain.CanManageOrderService
var editableOrderStatus = orderdomain.EditableOrderStatus
