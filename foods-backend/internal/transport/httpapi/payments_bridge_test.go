package httpapi

import (
	billingapp "github.com/foods-platform/foods-backend/internal/modules/billing/application"
	billingdomain "github.com/foods-platform/foods-backend/internal/modules/billing/domain"
)

type paymentView = billingdomain.PaymentView
type posOrderSummary = billingdomain.PosOrderSummary
type posOrderDetail = billingapp.POSOrderDetail
