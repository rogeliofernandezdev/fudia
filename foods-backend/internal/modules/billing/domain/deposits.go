package domain

// Garantía de reserva (2026-11-26). Es un cobro normal del turno que la recibe,
// todavía sin pedido; al llegar el cliente se aplica a su cuenta y, si no llega,
// se retiene en un pedido de mostrador con su boleta.

// DepositOutcome values stored in payments.deposit_outcome.
const (
	DepositApplied  = "applied"
	DepositRetained = "retained"
)

// RetainedDepositItem is the line of the counter order that keeps a guarantee
// the customer did not use; the receipt prints it as is.
const RetainedDepositItem = "Garantía de reserva no utilizada"

// DepositInput charges the guarantee of a reservation in the cashier's shift.
type DepositInput struct {
	ShiftID   string `json:"shiftId,omitempty"`
	Method    string `json:"method"`
	Reference string `json:"reference"`
}

// DepositReceipt answers a charged guarantee.
type DepositReceipt struct {
	PaymentID  string `json:"paymentId"`
	Amount     string `json:"amount"`
	MethodName string `json:"methodName"`
}

// DepositReservation is the reservation locked to charge its guarantee.
type DepositReservation struct {
	Status       string
	Amount       float64
	CustomerName string
	Charged      bool
}

// HeldDeposit is a charged guarantee still waiting for its customer.
type HeldDeposit struct {
	PaymentID         string
	ReservationID     string
	ReservationStatus string
	Amount            float64
	AffectsCash       bool
	CustomerName      string
	CustomerPhone     string
}

// DepositOption is a held guarantee the checkout can discount from an order.
// Suggested marks the reservation seated at the order's table.
type DepositOption struct {
	PaymentID    string `json:"paymentId"`
	CustomerName string `json:"customerName"`
	Amount       string `json:"amount"`
	MethodName   string `json:"methodName"`
	StartsAt     string `json:"startsAt"`
	TableName    string `json:"tableName"`
	Suggested    bool   `json:"suggested"`
}

// ForfeitedDeposit is a held guarantee whose reservation ended without the
// customer (No asistió or cancelled): the sweep retains it.
type ForfeitedDeposit struct {
	OrganizationID string
	LocationID     string
	ReservationID  string
}
