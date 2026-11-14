package domain

import (
	"testing"
	"time"
)

func TestPickupAlertLevel(t *testing.T) {
	for _, tc := range []struct {
		wait  time.Duration
		level int
		name  string
	}{
		{0, 0, ""},
		{179 * time.Second, 0, ""},
		{3 * time.Minute, 1, PickupAlertWarning},
		{5 * time.Minute, 2, PickupAlertUrgent},
		{8 * time.Minute, 3, PickupAlertCritical},
		{40 * time.Minute, 3, PickupAlertCritical},
	} {
		if level, name := PickupAlertLevel(tc.wait); level != tc.level || name != tc.name {
			t.Fatalf("%s: got %d %q", tc.wait, level, name)
		}
	}
}

func TestNormalizeCourierName(t *testing.T) {
	if name, ok := NormalizeCourierName("  Luis   Pérez "); !ok || name != "Luis Pérez" {
		t.Fatalf("got %q %v", name, ok)
	}
	if _, ok := NormalizeCourierName("L"); ok {
		t.Fatal("short name accepted")
	}
	if !ValidCourierToken("0123456789abcdef0123456789abcdef") || ValidCourierToken("../x") {
		t.Fatal("courier token pattern")
	}
	if !ValidCourierAction(ActionNotDelivered) || ValidCourierAction(ActionAccept) || ValidCourierAction(ActionRelease) {
		t.Fatal("courier actions")
	}
	if !ValidPanelAction(ActionRelease) || !ValidPanelAction(ActionKeep) || !ValidPanelAction(ActionFail) || ValidPanelAction(ActionNotDelivered) {
		t.Fatal("panel actions")
	}
}

func TestSettleFailedDelivery(t *testing.T) {
	near := 120
	far := 900
	valid := FailedDeliveryFacts{FoodAmount: 60, DistanceMeters: &near, DestinationKnown: true, CustomerNotified: true}
	got := SettleFailedDelivery(valid)
	if !got.FudiaShareEligible || got.CourierReimbursement != 60 || got.FudiaShare != 30 || got.PartnerShare != 30 || len(got.ValidationNotes) != 0 {
		t.Fatalf("validated incident: %#v", got)
	}
	capped := valid
	capped.FoodAmount = 250
	if got = SettleFailedDelivery(capped); got.FudiaShare != FailedDeliveryFudiaCap || got.PartnerShare != 150 || got.CourierReimbursement != 250 || len(got.ValidationNotes) != 1 {
		t.Fatalf("capped incident: %#v", got)
	}
	odd := valid
	odd.FoodAmount = 45.55
	if got = SettleFailedDelivery(odd); got.FudiaShare+got.PartnerShare != 45.55 {
		t.Fatalf("shares must add up: %#v", got)
	}
	for name, facts := range map[string]FailedDeliveryFacts{
		"no location":    {FoodAmount: 60, DestinationKnown: true, CustomerNotified: true},
		"far":            {FoodAmount: 60, DistanceMeters: &far, DestinationKnown: true, CustomerNotified: true},
		"no destination": {FoodAmount: 60, DistanceMeters: &near, CustomerNotified: true},
		"not notified":   {FoodAmount: 60, DistanceMeters: &near, DestinationKnown: true},
		"over limit":     {FoodAmount: 60, DistanceMeters: &near, DestinationKnown: true, CustomerNotified: true, CoveredRecently: FailedDeliveryMonthlyLimit},
	} {
		got := SettleFailedDelivery(facts)
		if got.FudiaShareEligible || got.FudiaShare != 0 || got.PartnerShare != 60 || got.CourierReimbursement != 60 || len(got.ValidationNotes) == 0 {
			t.Fatalf("%s: %#v", name, got)
		}
	}
}
