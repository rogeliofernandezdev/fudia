package domain

// CollectionShift exposes only what a POS operator needs, never cash balances.
type CollectionShift struct {
	ID               string `json:"id"`
	CashRegisterName string `json:"cashRegisterName"`
	OpenedByName     string `json:"openedByName"`
}

type CashMovementView struct {
	ID            string  `json:"id"`
	MovementType  string  `json:"movementType"`
	SourceType    string  `json:"sourceType"`
	SourceID      *string `json:"sourceId"`
	Amount        string  `json:"amount"`
	Reason        string  `json:"reason"`
	Note          string  `json:"note"`
	CreatedByName string  `json:"createdByName"`
	CreatedAt     string  `json:"createdAt"`
}
type CashShiftView struct {
	ID                    string             `json:"id"`
	Code                  string             `json:"code"`
	CashRegisterID        string             `json:"cashRegisterId"`
	CashRegisterName      string             `json:"cashRegisterName"`
	BlindClose            bool               `json:"blindClose"`
	Status                string             `json:"status"`
	OpeningAmount         string             `json:"openingAmount"`
	IncomeAmount          string             `json:"incomeAmount"`
	ExpenseAmount         string             `json:"expenseAmount"`
	ExpectedAmount        string             `json:"expectedAmount"`
	ExpectedVisible       bool               `json:"expectedVisible"`
	DetailsVisible        bool               `json:"detailsVisible"`
	ClosingExpectedAmount *string            `json:"closingExpectedAmount"`
	ClosingCountedAmount  *string            `json:"closingCountedAmount"`
	VarianceAmount        *string            `json:"varianceAmount"`
	OpeningNote           string             `json:"openingNote"`
	ClosingNote           string             `json:"closingNote"`
	OpenedByID            string             `json:"openedById"`
	OpenedByName          string             `json:"openedByName"`
	ClosedByName          string             `json:"closedByName"`
	ActiveUserNames       []string           `json:"activeUserNames"`
	BusinessDate          string             `json:"businessDate"`
	OpenedAt              string             `json:"openedAt"`
	ClosedAt              *string            `json:"closedAt"`
	MovementCount         int                `json:"movementCount"`
	Movements             []CashMovementView `json:"movements,omitempty"`
}
type CashRegisterView struct {
	ID         string         `json:"id"`
	Code       string         `json:"code"`
	Name       string         `json:"name"`
	Active     bool           `json:"active"`
	BlindClose bool           `json:"blindClose"`
	OpenShift  *CashShiftView `json:"openShift"`
}
type CashShiftUserView struct {
	UserID     string `json:"userId"`
	Name       string `json:"name"`
	AssignedAt string `json:"assignedAt"`
}
type CashUserOption struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	AssignedShiftID *string `json:"assignedShiftId"`
}
type CashOperationView struct {
	ID            string `json:"id"`
	OperationType string `json:"operationType"`
	Amount        string `json:"amount"`
	Reason        string `json:"reason"`
	Note          string `json:"note"`
	CreatedAt     string `json:"createdAt"`
}
type CashReportSale struct {
	ID         string `json:"id"`
	OrderCode  string `json:"orderCode"`
	Customer   string `json:"customer"`
	Name       string `json:"name"`
	Details    string `json:"details"`
	Quantity   string `json:"quantity"`
	UnitPrice  string `json:"unitPrice"`
	Total      string `json:"total"`
	CreatedAt  string `json:"createdAt"`
	WaiterName string `json:"waiterName"`
}
type CashReportPayment struct {
	ID            string `json:"id"`
	OrderCode     string `json:"orderCode"`
	MethodName    string `json:"methodName"`
	Amount        string `json:"amount"`
	Reference     string `json:"reference"`
	CreatedByName string `json:"createdByName"`
	CreatedAt     string `json:"createdAt"`
}
type CashReportCount struct {
	Denomination string `json:"denomination"`
	Quantity     int    `json:"quantity"`
	Total        string `json:"total"`
}
type CashReportMethod struct {
	Name   string `json:"name"`
	Amount string `json:"amount"`
}
type CashShiftReport struct {
	Tips               *TipReport          `json:"tips,omitempty"`
	Version            int                 `json:"version"`
	Persisted          bool                `json:"persisted"`
	GeneratedAt        string              `json:"generatedAt"`
	OrganizationName   string              `json:"organizationName"`
	LocationName       string              `json:"locationName"`
	Timezone           string              `json:"timezone"`
	Country            string              `json:"country"`
	Currency           string              `json:"currency"`
	CurrencySymbol     string              `json:"currencySymbol"`
	CurrencyPosition   string              `json:"currencyPosition"`
	CurrencyDecimals   int                 `json:"currencyDecimals"`
	Shift              CashShiftView       `json:"shift"`
	Sales              []CashReportSale    `json:"sales"`
	Payments           []CashReportPayment `json:"payments"`
	Refunds            []CashReportPayment `json:"refunds"`
	Methods            []CashReportMethod  `json:"methods"`
	Counts             []CashReportCount   `json:"counts"`
	CollectedAmount    string              `json:"collectedAmount"`
	RefundedAmount     string              `json:"refundedAmount"`
	NetCollectedAmount string              `json:"netCollectedAmount"`
}
