package domain

type PaymentInput struct {
	OrderID   string  `json:"orderId"`
	Method    string  `json:"method"`
	Amount    float64 `json:"amount"`
	Reference string  `json:"reference"`
}

type RefundInput struct {
	Amount float64 `json:"amount"`
	Reason string  `json:"reason"`
	Note   string  `json:"note"`
}

type PaymentBatchInput struct {
	OrderID  string        `json:"orderId"`
	Payments []PaymentLine `json:"payments"`
}

type PaymentLine struct {
	Method    string  `json:"method"`
	Amount    float64 `json:"amount"`
	Reference string  `json:"reference"`
}
