package domain

import "strings"

const MetaCloudAPI = "META_CLOUD_API"
const Whatsmeow = "WHATSMEOW"
const FudiaWeb = "FUDIA_WEB"

var WhatsAppProviderOptions = []OptionView{
	{Value: MetaCloudAPI, Label: "Meta Cloud API"},
	{Value: Whatsmeow, Label: "whatsmeow"},
	{Value: FudiaWeb, Label: "Chat web de Fudia"},
}

// WhatsAppRoute is private service data. Provider credentials are never part of it.
type WhatsAppRoute struct {
	RoutingKey  string `json:"routingKey"`
	CountryCode string `json:"countryCode"`
	Provider    string `json:"provider"`
	ProviderKey string `json:"providerKey"`
	PhoneNumber string `json:"phoneNumber"`
	Active      bool   `json:"active"`
	Legacy      bool   `json:"legacy"`
}

type WhatsAppConnection struct {
	Provider    string `json:"provider"`
	CountryCode string `json:"countryCode"`
	SessionRef  string `json:"sessionRef,omitempty"`
	State       string `json:"state"`
	PhoneNumber string `json:"phoneNumber"`
	ProviderKey string `json:"providerKey,omitempty"`
	QRCode      string `json:"qrCode,omitempty"`
	ExpiresAt   string `json:"expiresAt,omitempty"`
}

type WhatsAppLinkInput struct {
	CountryCode string `json:"countryCode"`
	SessionRef  string `json:"sessionRef,omitempty"`
}

func WhatsAppDigits(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, value)
}
