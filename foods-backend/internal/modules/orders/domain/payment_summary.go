package domain

import (
	"math"
	"strconv"
)

func ApplyPaymentSummary(o *Order, paid float64) error {
	if o.BillClosedAt == "" {
		o.AccountState = "open"
	} else {
		o.AccountState = "awaiting_payment"
	}
	total, err := strconv.ParseFloat(o.Total, 64)
	if err != nil {
		return err
	}
	remaining := math.Max(0, total-paid)
	o.PaidAmount = strconv.FormatFloat(paid, 'f', 2, 64)
	o.RemainingAmount = strconv.FormatFloat(remaining, 'f', 2, 64)
	o.PaymentStatus = PaymentStatus(total, paid)
	if o.BillClosedAt != "" && paid >= total {
		o.AccountState = "paid"
	}
	return nil
}
