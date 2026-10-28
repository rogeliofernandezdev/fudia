package domain

import "math"

type TipSettings struct {
	Mode           string   `json:"mode"`
	Percentage     float64  `json:"percentage"`
	ShowSuggestion bool     `json:"showSuggestion"`
	AllowFixed     bool     `json:"allowFixed"`
	Channels       []string `json:"channels"`
}
type TipPolicy struct {
	TipSettings
	ChannelOptions []TipChannelOption `json:"channelOptions"`
	Inherited      bool               `json:"inherited"`
	Country        string             `json:"country"`
	Currency       string             `json:"currency"`
	Decimals       int                `json:"decimals"`
}

type TipChannelOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}
type TipSettingsInput struct {
	TipSettings
	Inherit bool `json:"inherit"`
}
type TipInput struct {
	Kind          string  `json:"kind"`
	Value         float64 `json:"value"`
	DirectToStaff bool    `json:"directToStaff"`
}
type TipView struct {
	ID            string `json:"id"`
	PaymentID     string `json:"paymentId"`
	OrderCode     string `json:"orderCode"`
	Amount        string `json:"amount"`
	MethodName    string `json:"methodName"`
	AffectsCash   bool   `json:"affectsCash"`
	DirectToStaff bool   `json:"directToStaff"`
	CreatedByName string `json:"createdByName"`
	CreatedAt     string `json:"createdAt"`
}
type TipPayoutInput struct {
	RecipientID string  `json:"recipientId"`
	Amount      float64 `json:"amount"`
	Method      string  `json:"method"`
	Note        string  `json:"note"`
	RequestKey  string  `json:"requestKey"`
}
type TipPayoutView struct {
	ShiftID       string `json:"-"`
	CreatedByID   string `json:"-"`
	ID            string `json:"id"`
	RecipientID   string `json:"recipientId"`
	RecipientName string `json:"recipientName"`
	Amount        string `json:"amount"`
	Method        string `json:"method"`
	Note          string `json:"note"`
	CreatedByName string `json:"createdByName"`
	CreatedAt     string `json:"createdAt"`
}
type TipRecipient struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type TipBalance struct {
	ID        string
	Available float64
}
type TipReport struct {
	CashCollected  string          `json:"cashCollected"`
	OtherCollected string          `json:"otherCollected"`
	Direct         string          `json:"direct"`
	CashPaidOut    string          `json:"cashPaidOut"`
	OtherPaidOut   string          `json:"otherPaidOut"`
	Pending        string          `json:"pending"`
	Receipts       []TipView       `json:"receipts"`
	Payouts        []TipPayoutView `json:"payouts"`
}

func ValidMoney(value float64, decimals int) bool {
	scale := math.Pow10(decimals)
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value > 0 && value <= 1e9 && math.Abs(value*scale-math.Round(value*scale)) < 0.00001
}
func TipAmount(in TipInput, base float64, decimals int) (float64, bool) {
	if decimals < 0 || decimals > 2 || !ValidMoney(in.Value, 2) || base <= 0 {
		return 0, false
	}
	amount := in.Value
	if in.Kind == "percentage" {
		if in.Value > 100 {
			return 0, false
		}
		amount = base * in.Value / 100
	} else if in.Kind != "fixed" || !ValidMoney(in.Value, decimals) {
		return 0, false
	}
	scale := math.Pow10(decimals)
	amount = math.Round(amount*scale) / scale
	return amount, ValidMoney(amount, decimals)
}
