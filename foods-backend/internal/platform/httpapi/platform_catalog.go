package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (a *API) listPlatformCurrencies(ctx context.Context) ([]currencyOption, error) {
	rows, err := a.db.Query(ctx, `SELECT code,name,symbol,decimals FROM platform_currencies WHERE active ORDER BY name,code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []currencyOption{}
	for rows.Next() {
		var item currencyOption
		if err := rows.Scan(&item.Code, &item.Name, &item.Symbol, &item.Decimals); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (a *API) listPlatformCountries(ctx context.Context) ([]countryOption, error) {
	rows, err := a.db.Query(ctx, `
		SELECT c.code,c.name,c.default_currency
		FROM platform_countries c
		WHERE c.active
		ORDER BY c.name,c.code
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []countryOption{}
	for rows.Next() {
		var item countryOption
		if err := rows.Scan(&item.Code, &item.Name, &item.DefaultCurrency); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (a *API) listOnboardingCountries(ctx context.Context) ([]countryOption, error) {
	rows, err := a.db.Query(ctx, `
		SELECT c.code,c.name,c.default_currency
		FROM platform_countries c
		WHERE c.active
		  AND EXISTS (
		    SELECT 1 FROM platform_whatsapp_channels w
		    WHERE w.country_code=c.code
		      AND w.active
		  )
		ORDER BY c.name,c.code
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []countryOption{}
	for rows.Next() {
		var item countryOption
		if err := rows.Scan(&item.Code, &item.Name, &item.DefaultCurrency); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (a *API) platformCurrencyByCode(ctx context.Context, code string) (currencyOption, bool, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	var item currencyOption
	err := a.db.QueryRow(ctx, `SELECT code,name,symbol,decimals FROM platform_currencies WHERE code=$1 AND active`, code).Scan(&item.Code, &item.Name, &item.Symbol, &item.Decimals)
	if err == nil {
		return item, true, nil
	}
	if err == pgx.ErrNoRows {
		return currencyOption{}, false, nil
	}
	return currencyOption{}, false, err
}

func (a *API) onboardingCountryByCode(ctx context.Context, code string) (countryOption, bool, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	var item countryOption
	err := a.db.QueryRow(ctx, `
		SELECT c.code,c.name,c.default_currency
		FROM platform_countries c
		WHERE c.code=$1 AND c.active
		  AND EXISTS (
		    SELECT 1 FROM platform_whatsapp_channels w
		    WHERE w.country_code=c.code
		      AND w.active
		  )
	`, code).Scan(&item.Code, &item.Name, &item.DefaultCurrency)
	if err == nil {
		return item, true, nil
	}
	if err == pgx.ErrNoRows {
		return countryOption{}, false, nil
	}
	return countryOption{}, false, err
}

func (a *API) getPlatformOnboardingCatalogs(w http.ResponseWriter, r *http.Request) {
	var countries []countryOption
	var err error
	if r.URL.Query().Get("scope") == "all" {
		countries, err = a.listPlatformCountries(r.Context())
	} else {
		countries, err = a.listOnboardingCountries(r.Context())
	}
	if err != nil {
		fail(w, 503, "catalogs_unavailable", "No pudimos cargar los países disponibles.")
		return
	}
	currencies, err := a.listPlatformCurrencies(r.Context())
	if err != nil {
		fail(w, 503, "catalogs_unavailable", "No pudimos cargar las monedas disponibles.")
		return
	}
	writeJSON(w, 200, map[string]any{"countryOptions": countries, "currencyOptions": currencies})
}
