package domain

import (
	"math/big"
	"strings"
	"time"
)

// TaxComponent is one tax line of a document. Rates are exact fractions
// serialized with six decimals ("0.105000" = 10.5%).
type TaxComponent struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Rate string `json:"rate"`
}

// A special tax regime is set by law for a country (e.g. Peru's MYPE
// restaurants). Platform administrators register its dated rates; a fiscal
// profile only says whether the company is enrolled.
type SpecialTaxRate struct {
	ID         string  `json:"id"`
	ValidFrom  string  `json:"validFrom"`
	ValidTo    *string `json:"validTo"`
	Rate       string  `json:"rate"`
	LegalBasis string  `json:"legalBasis"`
}
type SpecialTaxRegime struct {
	Code    string           `json:"code"`
	Country string           `json:"country"`
	Name    string           `json:"name"`
	Rates   []SpecialTaxRate `json:"rates"`
}
type SpecialTaxRateInput struct {
	ValidFrom  string  `json:"validFrom"`
	ValidTo    *string `json:"validTo"`
	Rate       string  `json:"rate"`
	LegalBasis string  `json:"legalBasis"`
}

// SpecialTaxStatus tells whether an enrolled company still has a special rate:
// "current" while the registered rates cover more than SpecialTaxWarnDays,
// "expiring" when they end sooner and "expired" when none covers today (the
// general rate applies). Dates are exclusive ends, like ValidTo.
type SpecialTaxStatus struct {
	State       string  `json:"state"`
	Rate        *string `json:"rate"`
	RateValidTo *string `json:"rateValidTo"`
	CoverageEnd *string `json:"coverageEnd"`
}

// EffectiveTax is the tax that applies to a fiscal profile on a business date.
// Source is "special" when the enrolled regime has a rate for the date and
// "general" otherwise. There is always a single component: the receipt shows
// one tax with its total rate.
type EffectiveTax struct {
	Date          string         `json:"date"`
	Source        string         `json:"source"`
	SpecialRegime *string        `json:"specialRegime"`
	Components    []TaxComponent `json:"components"`
	TotalRate     string         `json:"totalRate"`
	TaxIncluded   bool           `json:"taxIncluded"`
}

const (
	TaxSourceGeneral       = "general"
	TaxSourceSpecial       = "special"
	GeneralTaxCode         = "general"
	SpecialTaxCode         = "special"
	SpecialTaxCurrent      = "current"
	SpecialTaxExpiring     = "expiring"
	SpecialTaxExpired      = "expired"
	SpecialTaxWarnDays     = 30
	taxRateDecimals        = 6
	businessDateLayout     = "2006-01-02"
	maxLegalBasisRunes     = 120
	specialRateMaxDecimals = 1_000_000
)

func ValidBusinessDate(value string) bool {
	_, err := time.Parse(businessDateLayout, value)
	return err == nil
}

// NormalizeSpecialTaxRateInput trims the input and reports whether it is a
// valid dated rate: end after start, rate between 0 and 1 with at most six
// decimals and a legal basis.
func NormalizeSpecialTaxRateInput(in *SpecialTaxRateInput) bool {
	in.ValidFrom, in.Rate, in.LegalBasis = strings.TrimSpace(in.ValidFrom), strings.TrimSpace(in.Rate), strings.TrimSpace(in.LegalBasis)
	from, err := time.Parse(businessDateLayout, in.ValidFrom)
	if err != nil || in.LegalBasis == "" || len([]rune(in.LegalBasis)) > maxLegalBasisRunes {
		return false
	}
	if in.ValidTo != nil {
		value := strings.TrimSpace(*in.ValidTo)
		if value == "" {
			in.ValidTo = nil
		} else if to, err := time.Parse(businessDateLayout, value); err != nil || !to.After(from) {
			return false
		} else {
			in.ValidTo = &value
		}
	}
	rate, ok := new(big.Rat).SetString(in.Rate)
	if !ok || rate.Sign() < 0 || rate.Cmp(big.NewRat(1, 1)) > 0 || !new(big.Rat).Mul(rate, big.NewRat(specialRateMaxDecimals, 1)).IsInt() {
		return false
	}
	in.Rate = rate.FloatString(taxRateDecimals)
	return true
}

func rateOn(regime SpecialTaxRegime, date string) (SpecialTaxRate, bool) {
	for _, rate := range regime.Rates {
		if rate.ValidFrom <= date && (rate.ValidTo == nil || date < *rate.ValidTo) {
			return rate, true
		}
	}
	return SpecialTaxRate{}, false
}

// SpecialRateOn returns the regime rate that covers date (validFrom inclusive,
// validTo exclusive).
func SpecialRateOn(regime SpecialTaxRegime, date string) (string, bool) {
	rate, ok := rateOn(regime, date)
	return rate.Rate, ok
}

// SpecialTaxStatusOn follows the rates chained from today (each starting where
// the previous one ends) to find when the special rate stops applying.
func SpecialTaxStatusOn(regime SpecialTaxRegime, today string) SpecialTaxStatus {
	current, ok := rateOn(regime, today)
	if !ok {
		return SpecialTaxStatus{State: SpecialTaxExpired}
	}
	rate, rateValidTo := current.Rate, current.ValidTo
	out := SpecialTaxStatus{State: SpecialTaxCurrent, Rate: &rate, RateValidTo: rateValidTo}
	end := current.ValidTo
	for end != nil {
		next, ok := rateOn(regime, *end)
		if !ok {
			break
		}
		end = next.ValidTo
	}
	if end == nil {
		return out
	}
	coverageEnd := *end
	out.CoverageEnd = &coverageEnd
	day, _ := time.Parse(businessDateLayout, today)
	if coverageEnd <= day.AddDate(0, 0, SpecialTaxWarnDays).Format(businessDateLayout) {
		out.State = SpecialTaxExpiring
	}
	return out
}

// ResolveEffectiveTax keeps the general tax name and swaps its rate for the
// special one while the enrolled regime covers date.
func ResolveEffectiveTax(date string, general TaxComponent, taxIncluded bool, regime *SpecialTaxRegime) EffectiveTax {
	out := EffectiveTax{Date: date, Source: TaxSourceGeneral, Components: []TaxComponent{general}, TaxIncluded: taxIncluded}
	if regime != nil {
		if rate, ok := SpecialRateOn(*regime, date); ok {
			name := regime.Name
			out.Source, out.SpecialRegime = TaxSourceSpecial, &name
			out.Components = []TaxComponent{{Code: SpecialTaxCode, Name: general.Name, Rate: rate}}
		}
	}
	out.TotalRate = canonicalRate(out.Components[0].Rate)
	out.Components[0].Rate = out.TotalRate
	return out
}

func canonicalRate(value string) string {
	rate, ok := new(big.Rat).SetString(value)
	if !ok {
		return value
	}
	return rate.FloatString(taxRateDecimals)
}
