package httpapi

import "testing"

func TestNormalizedSurcharge(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: "0"},
		{name: "spaces", in: "   ", want: "0"},
		{name: "amount", in: " 2.50 ", want: "2.50"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizedSurcharge(tt.in); got != tt.want {
				t.Fatalf("normalizedSurcharge(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}


func TestValidateComboRequiresUniqueGroupNames(t *testing.T) {
	in := comboInput{
		Name:  "Menú ejecutivo",
		Price: "25.00",
		Groups: []comboGroupInput{
			{Name: "Entrada", Required: true, MinSelections: 1, MaxSelections: 1, Options: []comboOptionInput{{ProductID: "a"}}},
			{Name: " entrada ", Required: true, MinSelections: 1, MaxSelections: 1, Options: []comboOptionInput{{ProductID: "b"}}},
		},
	}

	if got := validateCombo(in); got != "Cada grupo del menú necesita un nombre único." {
		t.Fatalf("validateCombo duplicate groups = %q", got)
	}
}

func TestValidateComboAcceptsDistinctGroups(t *testing.T) {
	in := comboInput{
		Name:  "Menú ejecutivo",
		Price: "25.00",
		Groups: []comboGroupInput{
			{Name: "Entrada", Required: true, MinSelections: 1, MaxSelections: 1, Options: []comboOptionInput{{ProductID: "a"}}},
			{Name: "Fondo", Required: true, MinSelections: 1, MaxSelections: 1, Options: []comboOptionInput{{ProductID: "b"}}},
		},
	}

	if got := validateCombo(in); got != "" {
		t.Fatalf("validateCombo distinct groups = %q", got)
	}
}
