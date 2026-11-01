package domain

import "testing"

func peruMypeInput() TaxRegimeInput {
	to := "2027-01-01"
	return TaxRegimeInput{
		Name: " Régimen MYPE restaurantes ", ValidFrom: "2026-01-01", ValidTo: &to,
		Components: []TaxComponent{{Code: " IGV ", Name: "IGV", Rate: "0.08"}, {Code: "ipm", Name: "IPM", Rate: "0.025"}},
	}
}

func TestNormalizeTaxRegimeInputCanonicalizes(t *testing.T) {
	in := peruMypeInput()
	if !NormalizeTaxRegimeInput(&in) {
		t.Fatal("valid regime rejected")
	}
	if in.Name != "Régimen MYPE restaurantes" || in.Components[0].Code != "igv" || in.Components[0].Rate != "0.080000" || in.Components[1].Rate != "0.025000" {
		t.Fatalf("not canonical: %+v", in)
	}
	if total := SumTaxRates(in.Components); total != "0.105000" {
		t.Fatalf("total must be exact, got %s", total)
	}
}

func TestNormalizeTaxRegimeInputRejects(t *testing.T) {
	cases := map[string]func(*TaxRegimeInput){
		"empty name":        func(in *TaxRegimeInput) { in.Name = " " },
		"bad date":          func(in *TaxRegimeInput) { in.ValidFrom = "2026-13-01" },
		"end before start":  func(in *TaxRegimeInput) { to := "2025-12-31"; in.ValidTo = &to },
		"end equals start":  func(in *TaxRegimeInput) { to := "2026-01-01"; in.ValidTo = &to },
		"no components":     func(in *TaxRegimeInput) { in.Components = nil },
		"duplicate code":    func(in *TaxRegimeInput) { in.Components[1].Code = "IGV" },
		"bad code":          func(in *TaxRegimeInput) { in.Components[0].Code = "1gv" },
		"negative rate":     func(in *TaxRegimeInput) { in.Components[0].Rate = "-0.01" },
		"too many decimals": func(in *TaxRegimeInput) { in.Components[0].Rate = "0.0800001" },
		"sum above 100%":    func(in *TaxRegimeInput) { in.Components[0].Rate = "0.99" },
		"too many components": func(in *TaxRegimeInput) {
			for _, code := range []string{"a", "b", "c", "d"} {
				in.Components = append(in.Components, TaxComponent{Code: code, Name: code, Rate: "0"})
			}
		},
	}
	for name, mutate := range cases {
		in := peruMypeInput()
		mutate(&in)
		if NormalizeTaxRegimeInput(&in) {
			t.Errorf("%s must be rejected", name)
		}
	}
}

func TestNormalizeTaxRegimeInputOpenEnded(t *testing.T) {
	in := peruMypeInput()
	empty := " "
	in.ValidTo = &empty
	if !NormalizeTaxRegimeInput(&in) || in.ValidTo != nil {
		t.Fatalf("blank end date means no end: %+v", in.ValidTo)
	}
}

func TestResolveEffectiveTax(t *testing.T) {
	general := TaxComponent{Code: GeneralTaxCode, Name: "IGV", Rate: "0.180000"}
	end2026, end2027 := "2027-01-01", "2028-01-01"
	regimes := []TaxRegime{
		{ID: "r2027", Name: "MYPE 2027", ValidFrom: "2027-01-01", ValidTo: &end2027, Components: []TaxComponent{{Code: "igv", Name: "IGV", Rate: "0.120000"}, {Code: "ipm", Name: "IPM", Rate: "0.030000"}}},
		{ID: "r2026", Name: "MYPE 2026", ValidFrom: "2026-01-01", ValidTo: &end2026, Components: []TaxComponent{{Code: "igv", Name: "IGV", Rate: "0.080000"}, {Code: "ipm", Name: "IPM", Rate: "0.025000"}}},
	}
	cases := []struct{ date, source, total string }{
		{"2025-12-31", TaxSourceGeneral, "0.180000"},
		{"2026-01-01", TaxSourceRegime, "0.105000"},
		{"2026-12-31", TaxSourceRegime, "0.105000"},
		{"2027-01-01", TaxSourceRegime, "0.150000"},
		{"2028-01-01", TaxSourceGeneral, "0.180000"},
	}
	for _, c := range cases {
		got := ResolveEffectiveTax(c.date, general, true, regimes)
		if got.Source != c.source || got.TotalRate != c.total || !got.TaxIncluded || got.Date != c.date {
			t.Errorf("%s: got %+v", c.date, got)
		}
	}
	if got := ResolveEffectiveTax("2026-06-01", general, false, regimes); got.RegimeName == nil || *got.RegimeName != "MYPE 2026" {
		t.Fatalf("regime name missing: %+v", got)
	}
}
