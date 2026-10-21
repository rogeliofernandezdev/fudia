package httpapi

import "testing"

func TestOrderServiceOwnershipDoesNotTransferOnRead(t *testing.T) {
	for _, tc := range []struct {
		user, channel, waiter string
		allowed               bool
	}{
		{"ana", "salon", "ana", true}, {"luis", "salon", "ana", false},
		{"", "salon", "ana", false}, {"luis", "salon", "", true},
		{"luis", "delivery", "", true}, {"luis", "whatsapp", "", true},
	} {
		if got := canManageOrderService(tc.user, tc.channel, tc.waiter); got != tc.allowed {
			t.Errorf("ownership(%q,%q,%q)=%v, want %v", tc.user, tc.channel, tc.waiter, got, tc.allowed)
		}
	}
}
