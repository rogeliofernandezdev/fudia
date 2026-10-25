package domain

import "testing"

func TestServiceChannelGroupsOriginsByService(t *testing.T) {
	for stored, service := range map[string]string{"whatsapp": "salon", "mostrador": "recojo", "salon": "salon", "recojo": "recojo", "delivery": "delivery"} {
		if got := ServiceChannel(stored); got != service {
			t.Fatalf("ServiceChannel(%q)=%q, want %q", stored, got, service)
		}
	}
	if NormalizeChannel("whatsapp") != "whatsapp" || NormalizeChannel("mostrador") != "recojo" {
		t.Fatal("writes keep the WhatsApp origin and map legacy Mostrador to Recojo")
	}
	for _, option := range Channels {
		if option["value"] == "mostrador" {
			t.Fatalf("Mostrador is no longer an order channel: %v", option)
		}
	}
	for _, option := range ServiceChannels {
		if option["value"] == "whatsapp" || option["value"] == "mostrador" {
			t.Fatalf("service tabs must not list origins: %v", option)
		}
	}
}
