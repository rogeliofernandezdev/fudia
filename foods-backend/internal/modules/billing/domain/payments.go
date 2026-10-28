package domain

type PaymentView struct {
	ID               string `json:"id"`
	OrderID          string `json:"orderId"`
	OrderCode        string `json:"orderCode"`
	ShiftID          string `json:"shiftId"`
	CashRegisterName string `json:"cashRegisterName"`
	Method           string `json:"method"`
	MethodName       string `json:"methodName"`
	AffectsCash      bool   `json:"affectsCash"`
	Amount           string `json:"amount"`
	RefundedAmount   string `json:"refundedAmount"`
	NetAmount        string `json:"netAmount"`
	Reference        string `json:"reference"`
	CreatedByName    string `json:"createdByName"`
	CreatedAt        string `json:"createdAt"`
}
type PosOrderSummary struct {
	WaiterID        string   `json:"waiterId"`
	BillClosed      bool     `json:"billClosed"`
	ID              string   `json:"id"`
	Code            string   `json:"code"`
	Channel         string   `json:"channel"`
	Status          string   `json:"status"`
	CustomerName    string   `json:"customerName"`
	TableName       string   `json:"tableName"`
	Total           string   `json:"total"`
	PaidAmount      string   `json:"paidAmount"`
	RemainingAmount string   `json:"remainingAmount"`
	PaymentStatus   string   `json:"paymentStatus"`
	PaymentMethods  []string `json:"paymentMethods"`
	CreatedAt       string   `json:"createdAt"`
}
type PaymentReceiptContext struct {
	OrganizationName string `json:"organizationName"`
	LegalName        string `json:"legalName"`
	TaxID            string `json:"taxId"`
	LocationName     string `json:"locationName"`
	Address          string `json:"address"`
	Phone            string `json:"phone"`
	Country          string `json:"country"`
	Timezone         string `json:"timezone"`
	Currency         string `json:"currency"`
	CurrencySymbol   string `json:"currencySymbol"`
	CurrencyPosition string `json:"currencyPosition"`
	CurrencyDecimals int    `json:"currencyDecimals"`
}

func PaymentStatus(total, paid float64) string {
	if paid <= 0.00001 {
		return "pending"
	}
	if paid+0.00001 >= total {
		return "paid"
	}
	return "partial"
}
