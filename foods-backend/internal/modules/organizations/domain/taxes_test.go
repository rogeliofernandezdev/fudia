package domain

import "testing"

func peruMype() *SpecialTaxRegime {
	to2026, to2027, to2028 := "2026-01-01", "2027-01-01", "2028-01-01"
	return &SpecialTaxRegime{Code: "pe_mype_restaurants", Country: "PE", Name: "Tasa especial MYPE de restaurantes y hoteles", Rates: []SpecialTaxRate{
		{ValidFrom: "2025-01-01", ValidTo: &to2026, Rate: "0.100000"},
		{ValidFrom: "2026-01-01", ValidTo: &to2027, Rate: "0.105000"},
		{ValidFrom: "2027-01-01", ValidTo: &to2028, Rate: "0.150000"},
	}}
}

func TestResolveEffectiveTaxUsesTheSpecialRateOnlyWhileItIsValid(t *testing.T) {
	general := TaxComponent{Code: GeneralTaxCode, Name: "IGV", Rate: "0.18"}
	cases := []struct{ date, source, rate string }{
		{"2024-12-31", TaxSourceGeneral, "0.180000"},
		{"2025-06-30", TaxSourceSpecial, "0.100000"},
		{"2026-01-01", TaxSourceSpecial, "0.105000"},
		{"2026-12-31", TaxSourceSpecial, "0.105000"},
		{"2027-01-01", TaxSourceSpecial, "0.150000"},
		{"2028-01-01", TaxSourceGeneral, "0.180000"},
	}
	for _, c := range cases {
		got := ResolveEffectiveTax(c.date, general, true, peruMype())
		if got.Source != c.source || got.TotalRate != c.rate || len(got.Components) != 1 || got.Components[0].Rate != c.rate || got.Components[0].Name != "IGV" || !got.TaxIncluded {
			t.Fatalf("%s: %+v", c.date, got)
		}
		if (c.source == TaxSourceSpecial) != (got.SpecialRegime != nil) {
			t.Fatalf("%s: special regime name %v", c.date, got.SpecialRegime)
		}
	}
}

func TestResolveEffectiveTaxWithoutEnrollmentUsesTheGeneralRate(t *testing.T) {
	got := ResolveEffectiveTax("2026-05-01", TaxComponent{Code: GeneralTaxCode, Name: "IGV", Rate: "0.180000"}, false, nil)
	if got.Source != TaxSourceGeneral || got.TotalRate != "0.180000" || got.SpecialRegime != nil {
		t.Fatalf("general: %+v", got)
	}
}

func TestSpecialTaxStatusWarnsBeforeTheRegisteredRatesEnd(t *testing.T) {
	cases := []struct{ today, state, rate, coverageEnd string }{
		{"2026-11-01", SpecialTaxCurrent, "0.105000", "2028-01-01"},
		{"2027-12-01", SpecialTaxCurrent, "0.150000", "2028-01-01"},
		{"2027-12-02", SpecialTaxExpiring, "0.150000", "2028-01-01"},
		{"2028-01-01", SpecialTaxExpired, "", ""},
	}
	for _, c := range cases {
		got := SpecialTaxStatusOn(*peruMype(), c.today)
		rate, end := "", ""
		if got.Rate != nil {
			rate = *got.Rate
		}
		if got.CoverageEnd != nil {
			end = *got.CoverageEnd
		}
		if got.State != c.state || rate != c.rate || end != c.coverageEnd {
			t.Fatalf("%s: %+v", c.today, got)
		}
	}
	// A gap in the calendar ends the coverage at the gap.
	gap := peruMype()
	gap.Rates = gap.Rates[:2]
	if got := SpecialTaxStatusOn(*gap, "2026-12-15"); got.State != SpecialTaxExpiring || *got.CoverageEnd != "2027-01-01" {
		t.Fatalf("gap: %+v", got)
	}
}

func TestNormalizeSpecialTaxRateInput(t *testing.T) {
	to := " 2029-01-01 "
	in := SpecialTaxRateInput{ValidFrom: "2028-01-01", ValidTo: &to, Rate: "0.125", LegalBasis: " Ley 99999 "}
	if !NormalizeSpecialTaxRateInput(&in) || in.Rate != "0.125000" || *in.ValidTo != "2029-01-01" || in.LegalBasis != "Ley 99999" {
		t.Fatalf("valid rate: %+v", in)
	}
	bad := []SpecialTaxRateInput{
		{ValidFrom: "2028-01-01", Rate: "0.1", LegalBasis: ""},
		{ValidFrom: "2028-13-01", Rate: "0.1", LegalBasis: "Ley"},
		{ValidFrom: "2028-01-01", ValidTo: &[]string{"2027-12-31"}[0], Rate: "0.1", LegalBasis: "Ley"},
		{ValidFrom: "2028-01-01", Rate: "1.5", LegalBasis: "Ley"},
		{ValidFrom: "2028-01-01", Rate: "0.1234567", LegalBasis: "Ley"},
	}
	for _, in := range bad {
		if NormalizeSpecialTaxRateInput(&in) {
			t.Fatalf("accepted %+v", in)
		}
	}
}
