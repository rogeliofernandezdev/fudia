package domain

import "testing"

func TestDistanceKmIsStraightLineRoundedTo10Meters(t *testing.T) {
	// Plaza de Armas de Lima -> Parque Kennedy (Miraflores): ~8.6 km en línea recta.
	got := DistanceKm(-12.045974, -77.030601, -12.121948, -77.029680)
	if got < 8.3 || got > 8.7 {
		t.Fatalf("unexpected distance: %v", got)
	}
	if !roundedTo(got) {
		t.Fatalf("distance must be rounded to 0.01 km: %v", got)
	}
	if DistanceKm(-12.1, -77.0, -12.1, -77.0) != 0 {
		t.Fatal("same point must be 0 km")
	}
}

func TestQuoteFeeUsesFirstCoveringTier(t *testing.T) {
	tiers := []FeeTier{{MaxDistanceKm: 6, Fee: 12}, {MaxDistanceKm: 2, Fee: 5}, {MaxDistanceKm: 4, Fee: 8}}
	cases := []struct {
		distance float64
		fee      float64
		covered  bool
	}{
		{0, 5, true}, {2, 5, true}, {2.01, 8, true}, {4, 8, true}, {5.5, 12, true}, {6, 12, true}, {6.01, 0, false},
	}
	for _, c := range cases {
		tier, ok := QuoteFee(tiers, c.distance)
		if ok != c.covered || tier.Fee != c.fee {
			t.Fatalf("distance %v: got %v %v", c.distance, tier, ok)
		}
	}
	if _, ok := QuoteFee(nil, 1); ok {
		t.Fatal("no tiers means no delivery coverage")
	}
	if CoverageKm(tiers) != 6 || CoverageKm(nil) != 0 {
		t.Fatal("coverage is the largest tier limit")
	}
}

func TestValidateFeeTiers(t *testing.T) {
	valid := []FeeTier{{MaxDistanceKm: 2, Fee: 5}, {MaxDistanceKm: 4.5, Fee: 0}}
	if message := ValidateFeeTiers(valid); message != "" {
		t.Fatalf("valid tiers rejected: %s", message)
	}
	invalid := [][]FeeTier{
		{{MaxDistanceKm: 0, Fee: 5}},
		{{MaxDistanceKm: 51, Fee: 5}},
		{{MaxDistanceKm: 2, Fee: -1}},
		{{MaxDistanceKm: 2, Fee: 1000}},
		{{MaxDistanceKm: 2.001, Fee: 5}},
		{{MaxDistanceKm: 2, Fee: 5}, {MaxDistanceKm: 2, Fee: 6}},
		make([]FeeTier, MaxFeeTiers+1),
	}
	for _, tiers := range invalid {
		if ValidateFeeTiers(tiers) == "" {
			t.Fatalf("invalid tiers accepted: %#v", tiers)
		}
	}
	if !ValidCoordinates(-12.1, -77) || ValidCoordinates(91, 0) || ValidCoordinates(0, -181) {
		t.Fatal("coordinate validation")
	}
}
