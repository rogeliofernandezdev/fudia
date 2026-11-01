package domain

import (
	"math/big"
	"regexp"
	"strings"
	"time"
)

// A tax regime replaces the fiscal profile's general rate during its validity.
// Rates are exact fractions serialized with six decimals ("0.080000" = 8%).
type TaxComponent struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Rate string `json:"rate"`
}
type TaxRegime struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	ValidFrom  string         `json:"validFrom"`
	ValidTo    *string        `json:"validTo"`
	Components []TaxComponent `json:"components"`
	TotalRate  string         `json:"totalRate"`
}
type TaxRegimeInput struct {
	Name       string         `json:"name"`
	ValidFrom  string         `json:"validFrom"`
	ValidTo    *string        `json:"validTo"`
	Components []TaxComponent `json:"components"`
}

// EffectiveTax is the tax that applies to a fiscal profile on a business date.
// Source is "regime" when a regime covers the date and "general" otherwise.
type EffectiveTax struct {
	Date        string         `json:"date"`
	Source      string         `json:"source"`
	RegimeID    *string        `json:"regimeId"`
	RegimeName  *string        `json:"regimeName"`
	Components  []TaxComponent `json:"components"`
	TotalRate   string         `json:"totalRate"`
	TaxIncluded bool           `json:"taxIncluded"`
}

const (
	MaxTaxComponents   = 5
	TaxSourceGeneral   = "general"
	TaxSourceRegime    = "regime"
	GeneralTaxCode     = "general"
	taxRateDecimals    = 6
	businessDateLayout = "2006-01-02"
)

var TaxComponentCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,19}$`)

// NormalizeTaxRegimeInput trims and canonicalizes the input, and reports
// whether it is a valid regime: dated range, 1-5 components with unique codes
// and exact rates whose sum does not exceed 100%.
func NormalizeTaxRegimeInput(in *TaxRegimeInput) bool {
	in.Name = strings.TrimSpace(in.Name)
	in.ValidFrom = strings.TrimSpace(in.ValidFrom)
	if in.Name == "" || len([]rune(in.Name)) > 60 {
		return false
	}
	from, err := time.Parse(businessDateLayout, in.ValidFrom)
	if err != nil {
		return false
	}
	if in.ValidTo != nil {
		value := strings.TrimSpace(*in.ValidTo)
		if value == "" {
			in.ValidTo = nil
		} else {
			to, err := time.Parse(businessDateLayout, value)
			if err != nil || !to.After(from) {
				return false
			}
			in.ValidTo = &value
		}
	}
	if len(in.Components) == 0 || len(in.Components) > MaxTaxComponents {
		return false
	}
	seen := map[string]bool{}
	total := new(big.Rat)
	for i := range in.Components {
		c := &in.Components[i]
		c.Code = strings.ToLower(strings.TrimSpace(c.Code))
		c.Name = strings.TrimSpace(c.Name)
		rate, ok := ParseTaxRate(c.Rate)
		if !ok || !TaxComponentCodePattern.MatchString(c.Code) || seen[c.Code] || c.Name == "" || len([]rune(c.Name)) > 30 {
			return false
		}
		seen[c.Code] = true
		c.Rate = rate.FloatString(taxRateDecimals)
		total.Add(total, rate)
	}
	return total.Cmp(big.NewRat(1, 1)) <= 0
}

// ParseTaxRate accepts a fraction between 0 and 1 with at most six decimals,
// the precision stored by numeric(7,6); anything finer would be rounded silently.
func ParseTaxRate(value string) (*big.Rat, bool) {
	rate, ok := new(big.Rat).SetString(strings.TrimSpace(value))
	if !ok || rate.Sign() < 0 || rate.Cmp(big.NewRat(1, 1)) > 0 {
		return nil, false
	}
	scaled := new(big.Rat).Mul(rate, big.NewRat(1_000_000, 1))
	return rate, scaled.IsInt()
}

// SumTaxRates adds component rates exactly. Rates come from storage or from
// NormalizeTaxRegimeInput, so they are always valid decimals.
func SumTaxRates(components []TaxComponent) string {
	total := new(big.Rat)
	for _, c := range components {
		if rate, ok := new(big.Rat).SetString(c.Rate); ok {
			total.Add(total, rate)
		}
	}
	return total.FloatString(taxRateDecimals)
}

func ValidBusinessDate(value string) bool {
	_, err := time.Parse(businessDateLayout, value)
	return err == nil
}

// ResolveEffectiveTax picks the regime that covers date (validFrom inclusive,
// validTo exclusive) or falls back to the profile's general rate.
func ResolveEffectiveTax(date string, general TaxComponent, taxIncluded bool, regimes []TaxRegime) EffectiveTax {
	for _, regime := range regimes {
		if regime.ValidFrom <= date && (regime.ValidTo == nil || date < *regime.ValidTo) {
			id, name := regime.ID, regime.Name
			return EffectiveTax{Date: date, Source: TaxSourceRegime, RegimeID: &id, RegimeName: &name, Components: regime.Components, TotalRate: SumTaxRates(regime.Components), TaxIncluded: taxIncluded}
		}
	}
	components := []TaxComponent{general}
	return EffectiveTax{Date: date, Source: TaxSourceGeneral, Components: components, TotalRate: SumTaxRates(components), TaxIncluded: taxIncluded}
}
