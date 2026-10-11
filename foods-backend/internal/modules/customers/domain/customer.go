package domain

import "strings"

type Address struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Address     string `json:"address"`
	Reference   string `json:"reference"`
	District    string `json:"district"`
	City        string `json:"city"`
	CountryCode string `json:"countryCode"`
	Default     bool   `json:"default"`
}
type Customer struct {
	ID               string    `json:"id"`
	Code             string    `json:"code"`
	CustomerType     string    `json:"customerType"`
	DisplayName      string    `json:"displayName"`
	DocumentType     string    `json:"documentType"`
	DocumentNumber   string    `json:"documentNumber"`
	Phone            string    `json:"phone"`
	Email            string    `json:"email"`
	PreferredChannel string    `json:"preferredChannel"`
	Preferences      string    `json:"preferences"`
	Notes            string    `json:"notes"`
	MarketingConsent bool      `json:"marketingConsent"`
	VIPOverride      bool      `json:"vipOverride"`
	VisitCount       int       `json:"visitCount"`
	TotalSpent       string    `json:"totalSpent"`
	LastPurchaseAt   *string   `json:"lastPurchaseAt"`
	Active           bool      `json:"active"`
	Segment          string    `json:"segment"`
	Addresses        []Address `json:"addresses,omitempty"`
}
type Input struct {
	CustomerType     string    `json:"customerType"`
	DisplayName      string    `json:"displayName"`
	DocumentType     string    `json:"documentType"`
	DocumentNumber   string    `json:"documentNumber"`
	Phone            string    `json:"phone"`
	Email            string    `json:"email"`
	PreferredChannel string    `json:"preferredChannel"`
	Preferences      string    `json:"preferences"`
	Notes            string    `json:"notes"`
	MarketingConsent bool      `json:"marketingConsent"`
	VIPOverride      bool      `json:"vipOverride"`
	Addresses        []Address `json:"addresses"`
}

func Normalize(in *Input) bool {
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.CustomerType = strings.TrimSpace(in.CustomerType)
	if in.CustomerType == "" {
		in.CustomerType = "person"
	}
	if in.PreferredChannel == "" {
		in.PreferredChannel = "none"
	}
	in.DocumentType = strings.TrimSpace(in.DocumentType)
	in.DocumentNumber = strings.TrimSpace(in.DocumentNumber)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Email = strings.TrimSpace(in.Email)
	in.Preferences = strings.TrimSpace(in.Preferences)
	in.Notes = strings.TrimSpace(in.Notes)
	addresses := make([]Address, 0, len(in.Addresses))
	for _, address := range in.Addresses {
		if strings.TrimSpace(address.Address) == "" {
			continue
		}
		if address.Label == "" {
			address.Label = "Principal"
		}
		if address.CountryCode == "" {
			address.CountryCode = "PE"
		}
		address.CountryCode = strings.ToUpper(address.CountryCode)
		addresses = append(addresses, address)
	}
	in.Addresses = addresses
	return in.DisplayName != "" && (in.CustomerType == "person" || in.CustomerType == "company") && (in.PreferredChannel == "none" || in.PreferredChannel == "whatsapp" || in.PreferredChannel == "phone" || in.PreferredChannel == "email")
}
