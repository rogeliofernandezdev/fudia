package domain

import "testing"

func TestPresentationContentErrorRejectsFractionsOfCountableUnits(t *testing.T) {
	bottle := PresentationRule{Allowed: true, Countable: true, UnitName: "Botella"}
	kilo := PresentationRule{Allowed: true, Countable: false, UnitName: "Kilogramo"}
	cases := []struct {
		name    string
		rule    PresentationRule
		kind    string
		factor  float64
		invalid bool
	}{
		{"package of 0.001 bottles", bottle, "package", 0.001, true},
		{"package of half a bottle", bottle, "package", 0.5, true},
		{"package of one bottle", bottle, "package", 1, false},
		{"package of six bottles", bottle, "package", 6, false},
		{"base unit", bottle, "unit", 1, false},
		{"bag of half a kilo", kilo, "package", 0.5, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := PresentationContentError(tc.rule, tc.kind, tc.factor)
			if (got != "") != tc.invalid {
				t.Fatalf("invalid=%v, got %q", tc.invalid, got)
			}
			if tc.invalid && got != "La presentación debe contener al menos 1 botella. Las fracciones solo se usan con unidades de medida como kilos o litros." {
				t.Fatalf("unexpected message %q", got)
			}
		})
	}
}
