package domain

import "testing"

func TestPaymentReadyKeepsSalonAndPickupRulesAndFreesCounterSales(t *testing.T) {
	cases := []struct {
		channel, status      string
		completed, billClose bool
		want                 bool
	}{
		{"salon", "entregado", false, false, false},
		{"salon", "entregado", false, true, true},
		{"whatsapp", "confirmado", false, true, false},
		{"recojo", "confirmado", false, false, false},
		{"recojo", "listo", false, false, true},
		{"delivery", "en_camino", false, false, true},
		{"mostrador", "confirmado", false, false, true},
		{"mostrador", "preparando", false, false, true},
		{"mostrador", "cancelado", false, false, false},
		{"mostrador", "entregado", true, false, false},
	}
	for _, tc := range cases {
		if got := PaymentReady(tc.channel, tc.status, tc.completed, tc.billClose); got != tc.want {
			t.Fatalf("%+v: got %v", tc, got)
		}
	}
}
