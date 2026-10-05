package httpapi

import (
	"context"
	"math/big"
	"strings"

	"github.com/jackc/pgx/v5"
)

func normalizePlatformFiscalInput(in *fiscalProfileInput) bool {
	in.Country = strings.ToUpper(strings.TrimSpace(in.Country))
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.CurrencyPosition = strings.TrimSpace(in.CurrencyPosition)
	in.TaxName = strings.TrimSpace(in.TaxName)
	rate, validRate := new(big.Rat).SetString(in.TaxRate)
	return validRate && rate.Sign() >= 0 && rate.Cmp(big.NewRat(1, 1)) <= 0 && (in.CurrencyPosition == "before" || in.CurrencyPosition == "after") && in.TaxName != "" && len(in.TaxName) <= 30
}

func (a *API) platformCountryByCode(ctx context.Context, code string) (countryOption, bool, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	var item countryOption
	err := a.db.QueryRow(ctx, `SELECT code,name,default_currency FROM platform_countries WHERE code=$1 AND active`, code).Scan(&item.Code, &item.Name, &item.DefaultCurrency)
	if err == nil {
		return item, true, nil
	}
	if err == pgx.ErrNoRows {
		return countryOption{}, false, nil
	}
	return countryOption{}, false, err
}

func (a *API) validatePlatformFiscalInput(ctx context.Context, in *fiscalProfileInput, requireOperationalWhatsApp bool) (currencyOption, bool, error) {
	if !normalizePlatformFiscalInput(in) {
		return currencyOption{}, false, nil
	}

	var countryOK bool
	var err error
	if requireOperationalWhatsApp {
		_, countryOK, err = a.onboardingCountryByCode(ctx, in.Country)
	} else {
		_, countryOK, err = a.platformCountryByCode(ctx, in.Country)
	}
	if err != nil || !countryOK {
		return currencyOption{}, false, err
	}

	currency, currencyOK, err := a.platformCurrencyByCode(ctx, in.Currency)
	if err != nil {
		return currencyOption{}, false, err
	}
	return currency, currencyOK, nil
}
