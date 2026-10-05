package httpapi

import "testing"

func TestValidatePlatformWhatsAppChannelUsesEnvironmentManagedCredentials(t *testing.T) {
	phoneNumberID := "123456789"
	in := platformWhatsAppChannelInput{
		CountryCode:   "pe",
		PhoneNumber:   " +51914832364 ",
		PhoneNumberID: &phoneNumberID,
		DisplayName:   " FudIA Perú ",
		Active:        true,
	}

	if code, message := validatePlatformWhatsAppChannel(&in); code != "" {
		t.Fatalf("active channel with phone number ID must be valid without a database secret reference: %s %s", code, message)
	}
	if in.CountryCode != "PE" || in.PhoneNumber != "+51914832364" || in.DisplayName != "FudIA Perú" {
		t.Fatalf("channel input was not normalized: %#v", in)
	}
}

func TestValidatePlatformWhatsAppChannelRequiresPhoneNumberIDWhenActive(t *testing.T) {
	in := platformWhatsAppChannelInput{
		CountryCode: "PE",
		PhoneNumber: "+51914832364",
		DisplayName: "FudIA Perú",
		Active:      true,
	}

	if code, _ := validatePlatformWhatsAppChannel(&in); code != "channel_incomplete" {
		t.Fatalf("expected channel_incomplete, got %q", code)
	}
}
