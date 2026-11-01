package domain

// FiscalRequestInput asks for an electronic document on the payment that
// completes the account; documentType "none" (or omitted) issues nothing.
type FiscalRequestInput struct {
	DocumentType string `json:"documentType"`
	Customer     struct {
		DocumentType   string `json:"documentType"`
		DocumentNumber string `json:"documentNumber"`
		Name           string `json:"name"`
		Address        string `json:"address"`
	} `json:"customer"`
}

type PaymentInput struct {
	Fiscal    *FiscalRequestInput `json:"fiscal,omitempty"`
	ShiftID   string              `json:"shiftId,omitempty"`
	Tip       *TipInput           `json:"tip,omitempty"`
	OrderID   string              `json:"orderId"`
	Method    string              `json:"method"`
	Amount    float64             `json:"amount"`
	Reference string              `json:"reference"`
}

type RefundInput struct {
	Amount float64 `json:"amount"`
	Reason string  `json:"reason"`
	Note   string  `json:"note"`
}

type PaymentBatchInput struct {
	Fiscal   *FiscalRequestInput `json:"fiscal,omitempty"`
	ShiftID  string              `json:"shiftId,omitempty"`
	OrderID  string              `json:"orderId"`
	Payments []PaymentLine       `json:"payments"`
}

type PaymentLine struct {
	Tip       *TipInput `json:"tip,omitempty"`
	Method    string    `json:"method"`
	Amount    float64   `json:"amount"`
	Reference string    `json:"reference"`
}
