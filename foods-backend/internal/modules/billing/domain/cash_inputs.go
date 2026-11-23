package domain

type RegisterInput struct {
	Name       string `json:"name"`
	BlindClose bool   `json:"blindClose"`
}

type RegisterStatusInput struct {
	Active bool `json:"active"`
}

type OpenShiftInput struct {
	CashRegisterID string  `json:"cashRegisterId"`
	OpeningAmount  float64 `json:"openingAmount"`
	Note           string  `json:"note"`
}

type CashMovementInput struct {
	MovementType string  `json:"movementType"`
	Amount       float64 `json:"amount"`
	Reason       string  `json:"reason"`
	Note         string  `json:"note"`
	// CategoryID convierte el egreso en un gasto cuando la empresa tiene el
	// módulo `costos`: el gasto y el movimiento de caja son un solo registro.
	CategoryID string `json:"categoryId,omitempty"`
}

type CloseShiftInput struct {
	TipPayout     *TipPayoutInput  `json:"tipPayout,omitempty"`
	CountedAmount *float64         `json:"countedAmount"`
	Note          string           `json:"note"`
	Counts        []CountLineInput `json:"counts"`
}

type ShiftUserInput struct {
	UserID string `json:"userId"`
}

type CashOperationInput struct {
	OperationType string  `json:"operationType"`
	TargetShiftID string  `json:"targetShiftId"`
	Amount        float64 `json:"amount"`
	Reason        string  `json:"reason"`
	Note          string  `json:"note"`
}

type CountLineInput struct {
	Denomination float64 `json:"denomination"`
	Quantity     int     `json:"quantity"`
}
