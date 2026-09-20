package httpapi

import (
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type organizationView struct {
	ID        string `json:"id"`
	LegalName string `json:"legalName"`
	TradeName string `json:"tradeName"`
	TaxID     string `json:"taxId"`
	Timezone  string `json:"timezone"`
	Active    bool   `json:"active"`
}

type organizationInput struct {
	LegalName string `json:"legalName"`
	TradeName string `json:"tradeName"`
	TaxID     string `json:"taxId"`
	Timezone  string `json:"timezone"`
}

type fiscalProfile struct {
	ID                string `json:"id"`
	Country           string `json:"country"`
	CountryName       string `json:"countryName"`
	Currency          string `json:"currency"`
	CurrencySymbol    string `json:"currencySymbol"`
	CurrencyPosition  string `json:"currencyPosition"`
	CurrencyDecimals  int    `json:"currencyDecimals"`
	TaxName           string `json:"taxName"`
	TaxRate           string `json:"taxRate"`
	TaxIncluded       bool   `json:"taxIncluded"`
	Default           bool   `json:"default"`
	Active            bool   `json:"active"`
	AssignedLocations int    `json:"assignedLocations"`
}

type fiscalProfileInput struct {
	Country          string `json:"country"`
	Currency         string `json:"currency"`
	CurrencyPosition string `json:"currencyPosition"`
	TaxName          string `json:"taxName"`
	TaxRate          string `json:"taxRate"`
	TaxIncluded      bool   `json:"taxIncluded"`
	Default          bool   `json:"default"`
}

type locationView struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Code            string   `json:"code"`
	Address         string   `json:"address"`
	Phone           string   `json:"phone"`
	OpeningHours    string   `json:"openingHours"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	Timezone        string   `json:"timezone"`
	FiscalProfileID string   `json:"fiscalProfileId"`
	Country         string   `json:"country"`
	Currency        string   `json:"currency"`
	Active          bool     `json:"active"`
}

type locationInput struct {
	Name            string   `json:"name"`
	Code            string   `json:"code"`
	Address         string   `json:"address"`
	Phone           string   `json:"phone"`
	OpeningHours    string   `json:"openingHours"`
	Latitude        *float64 `json:"latitude,omitempty"`
	Longitude       *float64 `json:"longitude,omitempty"`
	Timezone        string   `json:"timezone"`
	FiscalProfileID string   `json:"fiscalProfileId"`
	Active          *bool    `json:"active,omitempty"`
}

type exchangeRate struct {
	ID                string  `json:"id"`
	BaseCurrency      string  `json:"baseCurrency"`
	QuoteCurrency     string  `json:"quoteCurrency"`
	Rate              string  `json:"rate"`
	EffectiveAt       string  `json:"effectiveAt"`
	Source            string  `json:"source"`
	ProviderReference *string `json:"providerReference"`
	CreatedByName     *string `json:"createdByName"`
}

type exchangeRateInput struct {
	BaseCurrency      string  `json:"baseCurrency"`
	QuoteCurrency     string  `json:"quoteCurrency"`
	Rate              string  `json:"rate"`
	EffectiveAt       string  `json:"effectiveAt"`
	Source            string  `json:"source"`
	ProviderReference *string `json:"providerReference"`
}

func (a *API) getOrganization(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var out organizationView
	err := a.db.QueryRow(r.Context(), `SELECT id,legal_name,trade_name,tax_id,timezone,active FROM organizations WHERE id=$1`, s.OrganizationID).Scan(&out.ID, &out.LegalName, &out.TradeName, &out.TaxID, &out.Timezone, &out.Active)
	if err != nil {
		fail(w, 503, "organization_unavailable", "No pudimos cargar los datos de la empresa.")
		return
	}
	writeJSON(w, 200, out)
}

func (a *API) updateOrganization(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in organizationInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	in.LegalName, in.TradeName, in.TaxID, in.Timezone = strings.TrimSpace(in.LegalName), strings.TrimSpace(in.TradeName), strings.TrimSpace(in.TaxID), strings.TrimSpace(in.Timezone)
	if in.LegalName == "" || in.TradeName == "" || len(in.TaxID) < 6 || len(in.TaxID) > 32 || in.Timezone == "" {
		fail(w, 400, "invalid_organization", "Completa los datos obligatorios de la empresa.")
		return
	}
	var out organizationView
	err := a.db.QueryRow(r.Context(), `UPDATE organizations SET legal_name=$1,trade_name=$2,tax_id=$3,timezone=$4,updated_at=now() WHERE id=$5 AND active RETURNING id,legal_name,trade_name,tax_id,timezone,active`, in.LegalName, in.TradeName, in.TaxID, in.Timezone, s.OrganizationID).Scan(&out.ID, &out.LegalName, &out.TradeName, &out.TaxID, &out.Timezone, &out.Active)
	if err != nil {
		fail(w, 409, "organization_conflict", "No pudimos guardar la empresa. Verifica su identificación fiscal.")
		return
	}
	a.audit(r, "organization.updated", "organization", out.ID)
	writeJSON(w, 200, out)
}

func scanFiscalProfile(rows pgx.Rows, item *fiscalProfile) error {
	return rows.Scan(&item.ID, &item.Country, &item.Currency, &item.CurrencySymbol, &item.CurrencyPosition, &item.CurrencyDecimals, &item.TaxName, &item.TaxRate, &item.TaxIncluded, &item.Default, &item.Active, &item.AssignedLocations)
}

func (a *API) listFiscalProfiles(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	rows, err := a.db.Query(r.Context(), `SELECT p.id,p.country_code,p.currency,p.currency_symbol,p.currency_position,p.currency_decimals,p.tax_name,p.tax_rate::text,p.tax_included,p.is_default,p.active,count(l.id) FROM organization_fiscal_profiles p LEFT JOIN locations l ON l.fiscal_profile_id=p.id AND l.organization_id=p.organization_id WHERE p.organization_id=$1 GROUP BY p.id ORDER BY p.is_default DESC,p.country_code LIMIT $2 OFFSET $3`, s.OrganizationID, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "profiles_unavailable", "No pudimos cargar los perfiles fiscales.")
		return
	}
	defer rows.Close()
	items := []fiscalProfile{}
	for rows.Next() {
		var item fiscalProfile
		if err = scanFiscalProfile(rows, &item); err != nil {
			fail(w, 503, "profiles_unavailable", "No pudimos cargar los perfiles fiscales.")
			return
		}
		if country, ok := countryByCode(item.Country); ok {
			item.CountryName = country.Name
		} else {
			item.CountryName = item.Country
		}
		items = append(items, item)
	}
	var total int
	_ = a.db.QueryRow(r.Context(), `SELECT count(*) FROM organization_fiscal_profiles WHERE organization_id=$1`, s.OrganizationID).Scan(&total)
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size, "countryOptions": supportedCountries, "currencyOptions": supportedCurrencies})
}

func validateFiscalInput(in *fiscalProfileInput) (currencyOption, bool) {
	in.Country = strings.ToUpper(strings.TrimSpace(in.Country))
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.CurrencyPosition = strings.TrimSpace(in.CurrencyPosition)
	in.TaxName = strings.TrimSpace(in.TaxName)
	currency, validCurrency := currencyByCode(in.Currency)
	_, validCountry := countryByCode(in.Country)
	rate, validRate := new(big.Rat).SetString(in.TaxRate)
	return currency, validCountry && validCurrency && validRate && rate.Sign() >= 0 && rate.Cmp(big.NewRat(1, 1)) <= 0 && (in.CurrencyPosition == "before" || in.CurrencyPosition == "after") && in.TaxName != "" && len(in.TaxName) <= 30
}

func (a *API) saveFiscalProfile(w http.ResponseWriter, r *http.Request, create bool) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in fiscalProfileInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	currency, valid := validateFiscalInput(&in)
	if !valid {
		fail(w, 400, "invalid_profile", "Revisa país, moneda e impuesto.")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "profile_unavailable", "No pudimos guardar el perfil fiscal.")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	if in.Default {
		_, err = tx.Exec(r.Context(), `UPDATE organization_fiscal_profiles SET is_default=false,updated_at=now() WHERE organization_id=$1`, s.OrganizationID)
	}
	var id string
	if create && err == nil {
		err = tx.QueryRow(r.Context(), `INSERT INTO organization_fiscal_profiles(organization_id,country_code,currency,currency_symbol,currency_position,currency_decimals,tax_name,tax_rate,tax_included,is_default) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`, s.OrganizationID, in.Country, currency.Code, currency.Symbol, in.CurrencyPosition, currency.Decimals, in.TaxName, in.TaxRate, in.TaxIncluded, in.Default).Scan(&id)
	} else if err == nil {
		id = r.PathValue("id")
		command, updateErr := tx.Exec(r.Context(), `UPDATE organization_fiscal_profiles SET country_code=$1,currency=$2,currency_symbol=$3,currency_position=$4,currency_decimals=$5,tax_name=$6,tax_rate=$7,tax_included=$8,is_default=$9,active=true,updated_at=now() WHERE id=$10 AND organization_id=$11`, in.Country, currency.Code, currency.Symbol, in.CurrencyPosition, currency.Decimals, in.TaxName, in.TaxRate, in.TaxIncluded, in.Default, id, s.OrganizationID)
		err = updateErr
		if err == nil && command.RowsAffected() == 0 {
			err = pgx.ErrNoRows
		}
	}
	if err == nil && !in.Default {
		var defaults int
		err = tx.QueryRow(r.Context(), `SELECT count(*) FROM organization_fiscal_profiles WHERE organization_id=$1 AND is_default`, s.OrganizationID).Scan(&defaults)
		if err == nil && defaults == 0 {
			err = errors.New("default fiscal profile required")
		}
	}
	if err != nil || tx.Commit(r.Context()) != nil {
		fail(w, 409, "profile_conflict", "El país ya existe o la empresa debe conservar un perfil predeterminado.")
		return
	}
	a.audit(r, map[bool]string{true: "fiscal_profile.created", false: "fiscal_profile.updated"}[create], "fiscal_profile", id)
	if create {
		writeJSON(w, 201, map[string]string{"id": id})
	} else {
		w.WriteHeader(204)
	}
}

func (a *API) createFiscalProfile(w http.ResponseWriter, r *http.Request) {
	a.saveFiscalProfile(w, r, true)
}
func (a *API) updateFiscalProfile(w http.ResponseWriter, r *http.Request) {
	a.saveFiscalProfile(w, r, false)
}

func (a *API) deactivateFiscalProfile(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	command, err := a.db.Exec(r.Context(), `UPDATE organization_fiscal_profiles p SET active=false,updated_at=now() WHERE p.id=$1 AND p.organization_id=$2 AND NOT p.is_default AND NOT EXISTS(SELECT 1 FROM locations l WHERE l.fiscal_profile_id=p.id AND l.active)`, r.PathValue("id"), s.OrganizationID)
	if err != nil || command.RowsAffected() == 0 {
		fail(w, 409, "profile_in_use", "No puedes desactivar el perfil predeterminado ni uno asignado a locales activos.")
		return
	}
	a.audit(r, "fiscal_profile.deactivated", "fiscal_profile", r.PathValue("id"))
	w.WriteHeader(204)
}

func (a *API) listLocations(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	rows, err := a.db.Query(r.Context(), `SELECT l.id,l.name,l.code,l.address,l.phone,l.opening_hours,l.latitude,l.longitude,l.timezone,l.fiscal_profile_id,p.country_code,p.currency,l.active FROM locations l JOIN organization_fiscal_profiles p ON p.id=l.fiscal_profile_id AND p.organization_id=l.organization_id WHERE l.organization_id=$1 ORDER BY l.active DESC,l.name LIMIT $2 OFFSET $3`, s.OrganizationID, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "locations_unavailable", "No pudimos cargar los locales.")
		return
	}
	defer rows.Close()
	items := []locationView{}
	for rows.Next() {
		var item locationView
		if err = rows.Scan(&item.ID, &item.Name, &item.Code, &item.Address, &item.Phone, &item.OpeningHours, &item.Latitude, &item.Longitude, &item.Timezone, &item.FiscalProfileID, &item.Country, &item.Currency, &item.Active); err != nil {
			fail(w, 503, "locations_unavailable", "No pudimos cargar los locales.")
			return
		}
		items = append(items, item)
	}
	var total int
	_ = a.db.QueryRow(r.Context(), `SELECT count(*) FROM locations WHERE organization_id=$1`, s.OrganizationID).Scan(&total)
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size})
}

func validateLocationInput(in *locationInput) bool {
	in.Name, in.Code, in.Address, in.Phone, in.OpeningHours, in.Timezone, in.FiscalProfileID = strings.TrimSpace(in.Name), strings.ToUpper(strings.TrimSpace(in.Code)), strings.TrimSpace(in.Address), strings.TrimSpace(in.Phone), strings.TrimSpace(in.OpeningHours), strings.TrimSpace(in.Timezone), strings.TrimSpace(in.FiscalProfileID)
	return in.Name != "" && in.Code != "" && len(in.Code) <= 30 && in.Timezone != "" && in.FiscalProfileID != ""
}

func (a *API) createLocation(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in locationInput
	if json.NewDecoder(r.Body).Decode(&in) != nil || !validateLocationInput(&in) {
		fail(w, 400, "invalid_location", "Completa los datos obligatorios del local.")
		return
	}
	var out locationView
	err := a.db.QueryRow(r.Context(), `INSERT INTO locations(organization_id,name,code,address,phone,opening_hours,latitude,longitude,timezone,fiscal_profile_id) SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,p.id FROM organization_fiscal_profiles p WHERE p.id=$10 AND p.organization_id=$1 AND p.active RETURNING id,name,code,address,phone,opening_hours,latitude,longitude,timezone,fiscal_profile_id,(SELECT country_code FROM organization_fiscal_profiles WHERE id=fiscal_profile_id),(SELECT currency FROM organization_fiscal_profiles WHERE id=fiscal_profile_id),active`, s.OrganizationID, in.Name, in.Code, in.Address, in.Phone, in.OpeningHours, in.Latitude, in.Longitude, in.Timezone, in.FiscalProfileID).Scan(&out.ID, &out.Name, &out.Code, &out.Address, &out.Phone, &out.OpeningHours, &out.Latitude, &out.Longitude, &out.Timezone, &out.FiscalProfileID, &out.Country, &out.Currency, &out.Active)
	if err != nil {
		fail(w, 409, "location_conflict", "El código ya existe o el perfil fiscal no pertenece a la empresa.")
		return
	}
	a.audit(r, "location.created", "location", out.ID)
	writeJSON(w, 201, out)
}

func (a *API) updateLocation(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in locationInput
	if json.NewDecoder(r.Body).Decode(&in) != nil || !validateLocationInput(&in) {
		fail(w, 400, "invalid_location", "Completa los datos obligatorios del local.")
		return
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	command, err := a.db.Exec(r.Context(), `UPDATE locations l SET name=$1,code=$2,address=$3,phone=$4,opening_hours=$5,latitude=$6,longitude=$7,timezone=$8,fiscal_profile_id=p.id,active=$9,updated_at=now() FROM organization_fiscal_profiles p WHERE l.id=$10 AND l.organization_id=$11 AND p.id=$12 AND p.organization_id=l.organization_id AND p.active`, in.Name, in.Code, in.Address, in.Phone, in.OpeningHours, in.Latitude, in.Longitude, in.Timezone, active, r.PathValue("id"), s.OrganizationID, in.FiscalProfileID)
	if err != nil || command.RowsAffected() == 0 {
		fail(w, 409, "location_conflict", "No pudimos actualizar el local.")
		return
	}
	a.audit(r, "location.updated", "location", r.PathValue("id"))
	w.WriteHeader(204)
}

func (a *API) deactivateLocation(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	if r.PathValue("id") == s.LocationID {
		fail(w, 409, "active_location", "No puedes desactivar el local de tu sesión actual.")
		return
	}
	command, err := a.db.Exec(r.Context(), `UPDATE locations SET active=false,updated_at=now() WHERE id=$1 AND organization_id=$2 AND active`, r.PathValue("id"), s.OrganizationID)
	if err != nil || command.RowsAffected() == 0 {
		fail(w, 404, "location_not_found", "No encontramos el local activo.")
		return
	}
	a.audit(r, "location.deactivated", "location", r.PathValue("id"))
	w.WriteHeader(204)
}

func (a *API) listExchangeRates(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	rows, err := a.db.Query(r.Context(), `SELECT e.id,e.base_currency,e.quote_currency,e.rate::text,to_char(e.effective_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),e.source,e.provider_reference,u.full_name FROM organization_exchange_rates e LEFT JOIN users u ON u.id=e.created_by WHERE e.organization_id=$1 ORDER BY e.effective_at DESC LIMIT $2 OFFSET $3`, s.OrganizationID, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "rates_unavailable", "No pudimos cargar los tipos de cambio.")
		return
	}
	defer rows.Close()
	items := []exchangeRate{}
	for rows.Next() {
		var item exchangeRate
		if err = rows.Scan(&item.ID, &item.BaseCurrency, &item.QuoteCurrency, &item.Rate, &item.EffectiveAt, &item.Source, &item.ProviderReference, &item.CreatedByName); err != nil {
			fail(w, 503, "rates_unavailable", "No pudimos cargar los tipos de cambio.")
			return
		}
		items = append(items, item)
	}
	var total int
	_ = a.db.QueryRow(r.Context(), `SELECT count(*) FROM organization_exchange_rates WHERE organization_id=$1`, s.OrganizationID).Scan(&total)
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size, "currencyOptions": supportedCurrencies})
}

func (a *API) createExchangeRate(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in exchangeRateInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	in.BaseCurrency, in.QuoteCurrency, in.Rate, in.Source = strings.ToUpper(strings.TrimSpace(in.BaseCurrency)), strings.ToUpper(strings.TrimSpace(in.QuoteCurrency)), strings.TrimSpace(in.Rate), strings.TrimSpace(in.Source)
	rate, rateOK := new(big.Rat).SetString(in.Rate)
	effective, timeErr := time.Parse(time.RFC3339, in.EffectiveAt)
	_, baseOK := currencyByCode(in.BaseCurrency)
	_, quoteOK := currencyByCode(in.QuoteCurrency)
	if !baseOK || !quoteOK || in.BaseCurrency == in.QuoteCurrency || !rateOK || rate.Sign() <= 0 || timeErr != nil || (in.Source != "manual" && in.Source != "provider") {
		fail(w, 400, "invalid_rate", "Revisa las monedas, la tasa, la fecha y la fuente.")
		return
	}
	var out exchangeRate
	err := a.db.QueryRow(r.Context(), `INSERT INTO organization_exchange_rates(organization_id,base_currency,quote_currency,rate,effective_at,source,provider_reference,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id,base_currency,quote_currency,rate::text,to_char(effective_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),source,provider_reference`, s.OrganizationID, in.BaseCurrency, in.QuoteCurrency, in.Rate, effective, in.Source, in.ProviderReference, s.UserID).Scan(&out.ID, &out.BaseCurrency, &out.QuoteCurrency, &out.Rate, &out.EffectiveAt, &out.Source, &out.ProviderReference)
	if err != nil {
		fail(w, 409, "rate_conflict", "Ya existe un tipo de cambio para ese par y fecha.")
		return
	}
	out.CreatedByName = &s.Name
	a.audit(r, "exchange_rate.created", "exchange_rate", out.ID)
	writeJSON(w, 201, out)
}

func (a *API) deleteExchangeRate(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	command, err := a.db.Exec(r.Context(), `DELETE FROM organization_exchange_rates WHERE id=$1 AND organization_id=$2`, r.PathValue("id"), s.OrganizationID)
	if err != nil || command.RowsAffected() == 0 {
		fail(w, 404, "rate_not_found", "No encontramos el tipo de cambio.")
		return
	}
	a.audit(r, "exchange_rate.deleted", "exchange_rate", r.PathValue("id"))
	w.WriteHeader(204)
}
