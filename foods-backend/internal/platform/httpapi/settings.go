package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

type orgSettings struct {
	Country          string           `json:"country"`
	Currency         string           `json:"currency"`
	CurrencySymbol   string           `json:"currencySymbol"`
	CurrencyPosition string           `json:"currencyPosition"`
	CurrencyDecimals int              `json:"currencyDecimals"`
	TaxName          string           `json:"taxName"`
	TaxRate          float64          `json:"taxRate"`
	TaxIncluded      bool             `json:"taxIncluded"`
	CurrencyOptions  []currencyOption `json:"currencyOptions"`
	CountryOptions   []countryOption  `json:"countryOptions"`
}
type orgSettingsInput struct {
	Country          string  `json:"country"`
	Currency         string  `json:"currency"`
	CurrencyPosition string  `json:"currencyPosition"`
	TaxName          string  `json:"taxName"`
	TaxRate          float64 `json:"taxRate"`
	TaxIncluded      bool    `json:"taxIncluded"`
}

func countryByCode(code string) (countryOption, bool) {
	for _, item := range supportedCountries {
		if item.Code == code {
			return item, true
		}
	}
	return countryOption{}, false
}

func currencyByCode(code string) (currencyOption, bool) {
	for _, item := range supportedCurrencies {
		if item.Code == code {
			return item, true
		}
	}
	return currencyOption{}, false
}
func (a *API) getOrgSettings(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var out orgSettings
	err := a.db.QueryRow(r.Context(), `SELECT p.country_code,p.currency,p.currency_symbol,p.currency_position,p.currency_decimals,p.tax_name,p.tax_rate::float8,p.tax_included FROM organization_fiscal_profiles p JOIN organizations o ON o.id=p.organization_id WHERE p.organization_id=$1 AND p.is_default AND p.active AND o.active`, s.OrganizationID).Scan(&out.Country, &out.Currency, &out.CurrencySymbol, &out.CurrencyPosition, &out.CurrencyDecimals, &out.TaxName, &out.TaxRate, &out.TaxIncluded)
	if err != nil {
		fail(w, 503, "settings_unavailable", "No pudimos cargar la configuración financiera.")
		return
	}
	out.CurrencyOptions = supportedCurrencies
	out.CountryOptions = supportedCountries
	writeJSON(w, 200, out)
}
func (a *API) updateOrgSettings(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in orgSettingsInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.Country = strings.ToUpper(strings.TrimSpace(in.Country))
	in.CurrencyPosition = strings.TrimSpace(in.CurrencyPosition)
	in.TaxName = strings.TrimSpace(in.TaxName)
	currency, validCurrency := currencyByCode(in.Currency)
	_, validCountry := countryByCode(in.Country)
	if !validCountry || !validCurrency || (in.CurrencyPosition != "before" && in.CurrencyPosition != "after") || in.TaxName == "" || len(in.TaxName) > 30 || in.TaxRate < 0 || in.TaxRate > 1 {
		fail(w, 400, "invalid_settings", "Revisa la moneda y el porcentaje de impuesto.")
		return
	}
	tx, err := a.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err == nil {
		var previousID string
		err = tx.QueryRow(r.Context(), `SELECT id FROM organization_fiscal_profiles WHERE organization_id=$1 AND is_default FOR UPDATE`, s.OrganizationID).Scan(&previousID)
		if err == nil {
			_, err = tx.Exec(r.Context(), `UPDATE organization_fiscal_profiles SET is_default=false,updated_at=now() WHERE organization_id=$1 AND is_default`, s.OrganizationID)
		}
		var profileID string
		if err == nil {
			err = tx.QueryRow(r.Context(), `INSERT INTO organization_fiscal_profiles(organization_id,country_code,currency,currency_symbol,currency_position,currency_decimals,tax_name,tax_rate,tax_included,is_default,active) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,true,true) ON CONFLICT(organization_id,country_code) DO UPDATE SET currency=EXCLUDED.currency,currency_symbol=EXCLUDED.currency_symbol,currency_position=EXCLUDED.currency_position,currency_decimals=EXCLUDED.currency_decimals,tax_name=EXCLUDED.tax_name,tax_rate=EXCLUDED.tax_rate,tax_included=EXCLUDED.tax_included,is_default=true,active=true,updated_at=now() RETURNING id`, s.OrganizationID, in.Country, currency.Code, currency.Symbol, in.CurrencyPosition, currency.Decimals, in.TaxName, in.TaxRate, in.TaxIncluded).Scan(&profileID)
		}
		if err == nil {
			_, err = tx.Exec(r.Context(), `UPDATE locations SET fiscal_profile_id=$1 WHERE organization_id=$2 AND fiscal_profile_id=$3`, profileID, s.OrganizationID, previousID)
		}
		if err == nil {
			err = tx.Commit(r.Context())
		} else {
			_ = tx.Rollback(r.Context())
		}
	}
	if err != nil {
		fail(w, 503, "settings_unavailable", "No pudimos guardar la configuración financiera.")
		return
	}
	a.audit(r, "organization.settings.updated", "organization", s.OrganizationID)
	a.getOrgSettings(w, r)
}
