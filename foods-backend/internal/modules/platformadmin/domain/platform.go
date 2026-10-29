package domain

import (
	"net/mail"
	"strings"
)

type TenantOnboardingInput struct {
	TipSettings      *InitialTipSettings `json:"tipSettings,omitempty"`
	LegalName        string              `json:"legalName"`
	TradeName        string              `json:"tradeName"`
	TaxID            string              `json:"taxId"`
	Timezone         string              `json:"timezone"`
	Country          string              `json:"country"`
	Currency         string              `json:"currency"`
	CurrencyPosition string              `json:"currencyPosition"`
	TaxName          string              `json:"taxName"`
	TaxRate          string              `json:"taxRate"`
	TaxIncluded      bool                `json:"taxIncluded"`
	LocationName     string              `json:"locationName"`
	LocationCode     string              `json:"locationCode"`
	Address          string              `json:"address"`
	LocationPhone    string              `json:"locationPhone"`
	LocationHours    string              `json:"locationHours"`
	Latitude         *float64            `json:"latitude,omitempty"`
	Longitude        *float64            `json:"longitude,omitempty"`
	AdminName        string              `json:"adminName"`
	AdminEmail       string              `json:"adminEmail"`
	AdminPassword    string              `json:"adminPassword"`
	PlanID           string              `json:"planId"`
	BillingCycle     string              `json:"billingCycle"`
	TermsAccepted    bool                `json:"termsAccepted"`
}

// Onboarding owns its input. Billing validates and persists it through its port.
type InitialTipSettings struct {
	Mode           string   `json:"mode"`
	Percentage     float64  `json:"percentage"`
	ShowSuggestion bool     `json:"showSuggestion"`
	AllowFixed     bool     `json:"allowFixed"`
	Channels       []string `json:"channels"`
}
type PlatformOrganizationPlan struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}
type PlatformOrganizationSubscription struct {
	Plan         PlatformOrganizationPlan `json:"plan"`
	BillingCycle string                   `json:"billingCycle"`
	PriceAmount  string                   `json:"priceAmount"`
	Currency     string                   `json:"currency"`
	Status       string                   `json:"status"`
	RenewsAt     *string                  `json:"renewsAt"`
	TrialEndsAt  *string                  `json:"trialEndsAt"`
	AutoRenew    bool                     `json:"autoRenew"`
}
type PlatformOrganizationPayment struct {
	Amount   string  `json:"amount"`
	Currency string  `json:"currency"`
	PaidAt   *string `json:"paidAt"`
}
type PlatformOrganizationView struct {
	ID              string                            `json:"id"`
	LegalName       string                            `json:"legalName"`
	TradeName       string                            `json:"tradeName"`
	TaxID           string                            `json:"taxId"`
	Active          bool                              `json:"active"`
	CreatedAt       string                            `json:"createdAt"`
	Subscription    *PlatformOrganizationSubscription `json:"subscription"`
	LastPayment     *PlatformOrganizationPayment      `json:"lastPayment"`
	PaymentStanding string                            `json:"paymentStanding"`
}
type OptionView struct {
	Value string `json:"value"`
	Label string `json:"label"`
}
type PlatformOrganizationUsage struct {
	Locations    int  `json:"locations"`
	Users        int  `json:"users"`
	MaxLocations *int `json:"maxLocations"`
	MaxUsers     *int `json:"maxUsers"`
}
type PlatformOrganizationDetailSubscription struct {
	PlanName        string  `json:"planName"`
	BillingCycle    string  `json:"billingCycle"`
	PriceAmount     string  `json:"priceAmount"`
	Currency        string  `json:"currency"`
	Status          string  `json:"status"`
	StartedAt       string  `json:"startedAt"`
	RenewsAt        *string `json:"renewsAt"`
	TrialEndsAt     *string `json:"trialEndsAt"`
	AutoRenew       bool    `json:"autoRenew"`
	TermsVersion    *string `json:"termsVersion"`
	TermsAcceptedAt *string `json:"termsAcceptedAt"`
	ModuleCount     int     `json:"moduleCount"`
}
type PlatformOrganizationPayments struct {
	PaidCount  int     `json:"paidCount"`
	TotalPaid  string  `json:"totalPaid"`
	LastPaidAt *string `json:"lastPaidAt"`
}
type PlatformOrganizationAdmin struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Active bool   `json:"active"`
}
type PlatformOrganizationLocation struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Code    string `json:"code"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Active  bool   `json:"active"`
}
type PlatformOrganizationDetail struct {
	ID              string                                  `json:"id"`
	LegalName       string                                  `json:"legalName"`
	TradeName       string                                  `json:"tradeName"`
	TaxID           string                                  `json:"taxId"`
	CountryCode     string                                  `json:"countryCode"`
	Currency        string                                  `json:"currency"`
	Timezone        string                                  `json:"timezone"`
	TaxName         string                                  `json:"taxName"`
	TaxRate         string                                  `json:"taxRate"`
	TaxIncluded     bool                                    `json:"taxIncluded"`
	Active          bool                                    `json:"active"`
	CreatedAt       string                                  `json:"createdAt"`
	PaymentStanding string                                  `json:"paymentStanding"`
	Usage           PlatformOrganizationUsage               `json:"usage"`
	Subscription    *PlatformOrganizationDetailSubscription `json:"subscription"`
	Payments        PlatformOrganizationPayments            `json:"payments"`
	Administrators  []PlatformOrganizationAdmin             `json:"administrators"`
	Locations       []PlatformOrganizationLocation          `json:"locations"`
}
type PlatformWhatsAppChannel struct {
	ID          string `json:"id"`
	CountryCode string `json:"countryCode"`
	CountryName string `json:"countryName"`
	PhoneNumber string `json:"phoneNumber"`
	DisplayName string `json:"displayName"`
	Active      bool   `json:"active"`
}
type PlatformWhatsAppChannelInput struct {
	CountryCode string `json:"countryCode"`
	PhoneNumber string `json:"phoneNumber"`
	DisplayName string `json:"displayName"`
	Active      bool   `json:"active"`
}

var PaymentStandingOptions = []OptionView{
	{Value: "up_to_date", Label: "Al día"},
	{Value: "due_soon", Label: "Por vencer"},
	{Value: "overdue", Label: "Vencida"},
	{Value: "trial", Label: "En prueba"},
	{Value: "cancelled", Label: "Cancelada"},
	{Value: "none", Label: "Sin suscripción"},
}

const SubscriptionDueSoonDays = 7

func NormalizeTenantOnboardingInput(in *TenantOnboardingInput) {
	in.LegalName = strings.TrimSpace(in.LegalName)
	in.TradeName = strings.TrimSpace(in.TradeName)
	in.TaxID = strings.TrimSpace(in.TaxID)
	in.Timezone = strings.TrimSpace(in.Timezone)
	in.Country = strings.ToUpper(strings.TrimSpace(in.Country))
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.CurrencyPosition = strings.TrimSpace(in.CurrencyPosition)
	in.TaxName = strings.TrimSpace(in.TaxName)
	in.LocationName = strings.TrimSpace(in.LocationName)
	in.LocationCode = strings.ToUpper(strings.TrimSpace(in.LocationCode))
	in.Address = strings.TrimSpace(in.Address)
	in.LocationPhone = strings.TrimSpace(in.LocationPhone)
	in.LocationHours = strings.TrimSpace(in.LocationHours)
	in.AdminName = strings.TrimSpace(in.AdminName)
	in.AdminEmail = strings.ToLower(strings.TrimSpace(in.AdminEmail))
	in.PlanID = strings.TrimSpace(in.PlanID)
	in.BillingCycle = strings.TrimSpace(in.BillingCycle)
}
func ValidOnboardingEmail(value string) bool {
	if value == "" || len(value) > 180 {
		return false
	}
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}
func ValidOnboardingCoordinates(latitude, longitude *float64) bool {
	if (latitude == nil) != (longitude == nil) {
		return false
	}
	if latitude == nil {
		return true
	}
	return *latitude >= -90 && *latitude <= 90 && *longitude >= -180 && *longitude <= 180
}
func ValidPaymentStanding(value string) bool {
	for _, option := range PaymentStandingOptions {
		if option.Value == value {
			return true
		}
	}
	return false
}
func ValidatePlatformWhatsAppChannel(in *PlatformWhatsAppChannelInput) (string, string) {
	in.CountryCode = strings.ToUpper(strings.TrimSpace(in.CountryCode))
	in.PhoneNumber = strings.TrimSpace(in.PhoneNumber)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if len(in.CountryCode) != 2 {
		return "invalid_country", "Selecciona un país válido."
	}
	if in.PhoneNumber == "" || len(in.PhoneNumber) > 40 {
		return "invalid_phone", "Ingresa un número de WhatsApp válido."
	}
	return "", ""
}
