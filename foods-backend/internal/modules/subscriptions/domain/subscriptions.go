package domain

import "time"

type SubscriptionPlanView struct {
	ID           string   `json:"id"`
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Currency     string   `json:"currency"`
	MonthlyPrice string   `json:"monthlyPrice"`
	AnnualPrice  string   `json:"annualPrice"`
	TrialDays    int      `json:"trialDays"`
	MaxLocations *int     `json:"maxLocations"`
	MaxUsers     *int     `json:"maxUsers"`
	ModuleKeys   []string `json:"moduleKeys"`
	TermsVersion string   `json:"termsVersion"`
	Active       bool     `json:"active"`
}
type SubscriptionPlanInput struct {
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Currency     string   `json:"currency"`
	MonthlyPrice string   `json:"monthlyPrice"`
	AnnualPrice  string   `json:"annualPrice"`
	TrialDays    int      `json:"trialDays"`
	MaxLocations *int     `json:"maxLocations"`
	MaxUsers     *int     `json:"maxUsers"`
	ModuleKeys   []string `json:"moduleKeys"`
	TermsVersion string   `json:"termsVersion"`
	Active       bool     `json:"active"`
}
type SubscriptionPaymentView struct {
	ID                string  `json:"id"`
	Amount            string  `json:"amount"`
	Currency          string  `json:"currency"`
	Status            string  `json:"status"`
	Provider          string  `json:"provider"`
	ExternalReference *string `json:"externalReference"`
	PaidAt            *string `json:"paidAt"`
	CreatedAt         string  `json:"createdAt"`
}
type SubscriptionOrganizationView struct {
	ID        string `json:"id"`
	TradeName string `json:"tradeName"`
	LegalName string `json:"legalName"`
	TaxID     string `json:"taxId"`
	Active    bool   `json:"active"`
}
type OrganizationSubscriptionView struct {
	ID                    string                       `json:"id"`
	Organization          SubscriptionOrganizationView `json:"organization"`
	Plan                  SubscriptionPlanView         `json:"plan"`
	BillingCycle          string                       `json:"billingCycle"`
	PriceAmount           string                       `json:"priceAmount"`
	Currency              string                       `json:"currency"`
	Status                string                       `json:"status"`
	TrialStartsAt         *string                      `json:"trialStartsAt"`
	TrialEndsAt           *string                      `json:"trialEndsAt"`
	CurrentPeriodStartsAt *string                      `json:"currentPeriodStartsAt"`
	CurrentPeriodEndsAt   *string                      `json:"currentPeriodEndsAt"`
	RenewsAt              *string                      `json:"renewsAt"`
	AutoRenew             bool                         `json:"autoRenew"`
	TermsVersion          *string                      `json:"termsVersion"`
	TermsAcceptedAt       *string                      `json:"termsAcceptedAt"`
	Usage                 SubscriptionUsage            `json:"usage"`
	Payments              []SubscriptionPaymentView    `json:"payments"`
}
type SubscriptionUsage struct {
	Locations int `json:"locations"`
	Users     int `json:"users"`
}
type SubscriptionUpdateInput struct {
	PlanID        string `json:"planId"`
	BillingCycle  string `json:"billingCycle"`
	Status        string `json:"status"`
	AutoRenew     bool   `json:"autoRenew"`
	TermsAccepted bool   `json:"termsAccepted"`
}
type SubscriptionPaymentInput struct {
	Amount            string `json:"amount"`
	Currency          string `json:"currency"`
	Status            string `json:"status"`
	Provider          string `json:"provider"`
	ExternalReference string `json:"externalReference"`
	PaidAt            string `json:"paidAt"`
}

func SubscriptionPeriodEnd(start time.Time, cycle string) time.Time {
	if cycle == "annual" {
		return start.AddDate(1, 0, 0)
	}
	return start.AddDate(0, 1, 0)
}
