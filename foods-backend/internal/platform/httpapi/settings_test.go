package httpapi

import "testing"

func TestSupportedCurrenciesUsesCompleteActiveCatalog(t *testing.T) {
	if len(supportedCurrencies) < 150 {
		t.Fatalf("expected complete ISO catalog, got %d currencies", len(supportedCurrencies))
	}
	seen := map[string]bool{}
	for _, item := range supportedCurrencies {
		if seen[item.Code] {
			t.Fatalf("duplicated currency %s", item.Code)
		}
		seen[item.Code] = true
		if len(item.Code) != 3 || item.Name == "" || item.Symbol == "" || item.Decimals < 0 || item.Decimals > 4 {
			t.Fatalf("invalid currency entry: %+v", item)
		}
	}
	for _, code := range []string{"PEN", "USD", "EUR", "JPY", "XAF", "XOF", "XPF", "XCG", "ZWG"} {
		if !seen[code] {
			t.Fatalf("missing active ISO currency %s", code)
		}
	}
	for _, code := range []string{"XAU", "XTS", "XXX"} {
		if seen[code] {
			t.Fatalf("non-operational code %s must not be selectable", code)
		}
	}
}

func TestSupportedCountriesDefaultAndCoverage(t *testing.T) {
	if len(supportedCountries) < 240 {
		t.Fatalf("expected complete country catalog, got %d", len(supportedCountries))
	}
	peru, found := countryByCode("PE")
	if !found || peru.DefaultCurrency != "PEN" {
		t.Fatalf("expected Peru with PEN, got %+v", peru)
	}
}
