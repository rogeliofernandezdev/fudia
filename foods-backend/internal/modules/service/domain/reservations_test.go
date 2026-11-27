package domain

import "testing"

func TestReservationDepositIsOptionalWithTwoDecimals(t *testing.T) {
	for _, value := range []string{"0", "0.00", "50", "50.5", "1234567890.99"} {
		if !ValidDepositAmount(value) {
			t.Fatalf("%q must be a valid deposit", value)
		}
	}
	for _, value := range []string{"-1", "50.555", "abc", "", "12345678901"} {
		if ValidDepositAmount(value) {
			t.Fatalf("%q must be rejected", value)
		}
	}
	blank := "  "
	in := ReservationInput{CustomerName: "Ana", StartsAt: "2999-01-01T20:00:00Z", Guests: 2, DepositAmount: &blank}
	if _, ok := ParseReservationInput(&in); !ok || *in.DepositAmount != "0" {
		t.Fatalf("blank deposit means no cost: %v", *in.DepositAmount)
	}
	tolerance := 241
	in = ReservationInput{CustomerName: "Ana", StartsAt: "2999-01-01T20:00:00Z", Guests: 2, ToleranceMinutes: &tolerance}
	if _, ok := ParseReservationInput(&in); ok {
		t.Fatal("tolerance above 240 must be rejected")
	}
}
