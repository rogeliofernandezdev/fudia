package httpapi

import "testing"

func TestValidatePlatformWhatsAppChannelUsesEnvironmentManagedMetaConfiguration(t *testing.T) {
	in := platformWhatsAppChannelInput{
		CountryCode: "pe",
		PhoneNumber: " +51914832364 ",
		DisplayName: " FudIA Perú ",
		Active:      true,
	}

	if code, message := validatePlatformWhatsAppChannel(&in); code != "" {
		t.Fatalf("active channel must be valid without database-managed Meta configuration: %s %s", code, message)
	}
	if in.CountryCode != "PE" || in.PhoneNumber != "+51914832364" || in.DisplayName != "FudIA Perú" {
		t.Fatalf("channel input was not normalized: %#v", in)
	}
}
