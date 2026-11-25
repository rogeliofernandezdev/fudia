package domain

import "testing"

func TestOperationalStatusKeepsShippingPendingUntilCustomerReceipt(t *testing.T) {
	for _, option := range Statuses {
		status := option["value"]
		want := status
		if status == "en_camino" {
			want = "listo"
		}
		if got := OperationalOrderStatus(status); got != want {
			t.Errorf("operational status for %s = %s; want %s", status, got, want)
		}
	}
	if !contains(Transitions["listo"], "en_camino") || !contains(Transitions["en_camino"], "entregado") {
		t.Fatal("operational projection must preserve the delivery lifecycle")
	}
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
