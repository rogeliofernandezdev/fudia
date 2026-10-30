package domain

import (
	"math/big"
	"strings"
	"time"
)

type OrganizationView struct {
	ID        string `json:"id"`
	LegalName string `json:"legalName"`
	TradeName string `json:"tradeName"`
	TaxID     string `json:"taxId"`
	Timezone  string `json:"timezone"`
	Active    bool   `json:"active"`
}
type OrganizationInput struct {
	LegalName string `json:"legalName"`
	TradeName string `json:"tradeName"`
	TaxID     string `json:"taxId"`
	Timezone  string `json:"timezone"`
}
type FiscalProfile struct {
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
type FiscalProfileInput struct {
	Country          string `json:"country"`
	Currency         string `json:"currency"`
	CurrencyPosition string `json:"currencyPosition"`
	TaxName          string `json:"taxName"`
	TaxRate          string `json:"taxRate"`
	TaxIncluded      bool   `json:"taxIncluded"`
	Default          bool   `json:"default"`
}
type LocationView struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Code            string   `json:"code"`
	Address         string   `json:"address"`
	Phone           string   `json:"phone"`
	OpeningHours    string   `json:"openingHours"`
	OpenedOn        *string  `json:"openedOn,omitempty"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	Timezone        string   `json:"timezone"`
	FiscalProfileID string   `json:"fiscalProfileId"`
	Country         string   `json:"country"`
	Currency        string   `json:"currency"`
	Active          bool     `json:"active"`
}
type LocationInput struct {
	Name            string   `json:"name"`
	Code            string   `json:"code"`
	Address         string   `json:"address"`
	Phone           string   `json:"phone"`
	OpeningHours    string   `json:"openingHours"`
	OpenedOn        *string  `json:"openedOn,omitempty"`
	Latitude        *float64 `json:"latitude,omitempty"`
	Longitude       *float64 `json:"longitude,omitempty"`
	Timezone        string   `json:"timezone"`
	FiscalProfileID string   `json:"fiscalProfileId"`
	Active          *bool    `json:"active,omitempty"`
}
type ExchangeRate struct {
	ID                string  `json:"id"`
	BaseCurrency      string  `json:"baseCurrency"`
	QuoteCurrency     string  `json:"quoteCurrency"`
	Rate              string  `json:"rate"`
	EffectiveAt       string  `json:"effectiveAt"`
	Source            string  `json:"source"`
	ProviderReference *string `json:"providerReference"`
	CreatedByName     *string `json:"createdByName"`
}
type ExchangeRateInput struct {
	BaseCurrency      string  `json:"baseCurrency"`
	QuoteCurrency     string  `json:"quoteCurrency"`
	Rate              string  `json:"rate"`
	EffectiveAt       string  `json:"effectiveAt"`
	Source            string  `json:"source"`
	ProviderReference *string `json:"providerReference"`
}
type OrgSettings struct {
	Country          string           `json:"country"`
	Currency         string           `json:"currency"`
	CurrencySymbol   string           `json:"currencySymbol"`
	CurrencyPosition string           `json:"currencyPosition"`
	CurrencyDecimals int              `json:"currencyDecimals"`
	TaxName          string           `json:"taxName"`
	TaxRate          float64          `json:"taxRate"`
	TaxIncluded      bool             `json:"taxIncluded"`
	CurrencyOptions  []CurrencyOption `json:"currencyOptions"`
	CountryOptions   []CountryOption  `json:"countryOptions"`
}
type OrgSettingsInput struct {
	Country          string  `json:"country"`
	Currency         string  `json:"currency"`
	CurrencyPosition string  `json:"currencyPosition"`
	TaxName          string  `json:"taxName"`
	TaxRate          float64 `json:"taxRate"`
	TaxIncluded      bool    `json:"taxIncluded"`
}
type RestaurantSetupCounts struct {
	Categories     int `json:"categories"`
	Products       int `json:"products"`
	Tables         int `json:"tables"`
	CashRegisters  int `json:"cashRegisters"`
	Users          int `json:"users"`
	InventoryItems int `json:"inventoryItems"`
	Recipes        int `json:"recipes"`
	Suppliers      int `json:"suppliers"`
}
type RestaurantSetupView struct {
	ServiceMode string                `json:"serviceMode"`
	CompletedAt *time.Time            `json:"completedAt"`
	CoreReady   bool                  `json:"coreReady"`
	Counts      RestaurantSetupCounts `json:"counts"`
	Modules     map[string]bool       `json:"modules"`
}
type PlatformCountryOption struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	DefaultCurrency string `json:"defaultCurrency"`
	CallingCode     string `json:"callingCode"`
	DefaultTimezone string `json:"defaultTimezone"`
}

func ValidateLocationInput(in *LocationInput) bool {
	if in.OpenedOn != nil && *in.OpenedOn != "" {
		if _, err := time.Parse("2006-01-02", *in.OpenedOn); err != nil {
			return false
		}
	}
	in.Name, in.Code, in.Address, in.Phone, in.OpeningHours, in.Timezone, in.FiscalProfileID = strings.TrimSpace(in.Name), strings.ToUpper(strings.TrimSpace(in.Code)), strings.TrimSpace(in.Address), strings.TrimSpace(in.Phone), strings.TrimSpace(in.OpeningHours), strings.TrimSpace(in.Timezone), strings.TrimSpace(in.FiscalProfileID)
	return in.Name != "" && in.Code != "" && len(in.Code) <= 30 && in.Timezone != "" && in.FiscalProfileID != ""
}
func NormalizePlatformFiscalInput(in *FiscalProfileInput) bool {
	in.Country = strings.ToUpper(strings.TrimSpace(in.Country))
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.CurrencyPosition = strings.TrimSpace(in.CurrencyPosition)
	in.TaxName = strings.TrimSpace(in.TaxName)
	rate, validRate := new(big.Rat).SetString(in.TaxRate)
	return validRate && rate.Sign() >= 0 && rate.Cmp(big.NewRat(1, 1)) <= 0 && (in.CurrencyPosition == "before" || in.CurrencyPosition == "after") && in.TaxName != "" && len(in.TaxName) <= 30
}
func ValidServiceMode(value string) bool {
	return value == "counter" || value == "dine_in" || value == "mixed"
}
func RestaurantSetupCoreReady(serviceMode string, counts RestaurantSetupCounts) bool {
	requiresTables := serviceMode == "dine_in" || serviceMode == "mixed"
	return ValidServiceMode(serviceMode) &&
		counts.Categories > 0 && counts.Products > 0 && counts.CashRegisters > 0 &&
		(!requiresTables || counts.Tables > 0)
}
func CountryByCode(code string) (CountryOption, bool) {
	for _, item := range SupportedCountries {
		if item.Code == code {
			return item, true
		}
	}
	return CountryOption{}, false
}
func CurrencyByCode(code string) (CurrencyOption, bool) {
	for _, item := range SupportedCurrencies {
		if item.Code == code {
			return item, true
		}
	}
	return CurrencyOption{}, false
}
func ValidateFiscalInput(in *FiscalProfileInput) (CurrencyOption, bool) {
	in.Country = strings.ToUpper(strings.TrimSpace(in.Country))
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.CurrencyPosition = strings.TrimSpace(in.CurrencyPosition)
	in.TaxName = strings.TrimSpace(in.TaxName)
	currency, validCurrency := CurrencyByCode(in.Currency)
	_, validCountry := CountryByCode(in.Country)
	rate, validRate := new(big.Rat).SetString(in.TaxRate)
	return currency, validCountry && validCurrency && validRate && rate.Sign() >= 0 && rate.Cmp(big.NewRat(1, 1)) <= 0 && (in.CurrencyPosition == "before" || in.CurrencyPosition == "after") && in.TaxName != "" && len(in.TaxName) <= 30
}
