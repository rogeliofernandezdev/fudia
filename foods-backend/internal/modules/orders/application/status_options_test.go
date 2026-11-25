package application

import "testing"

func TestPresentStatusOptionsKeepsCatalogOrderAndSelection(t *testing.T) {
	values := func(options []map[string]string) []string {
		out := []string{}
		for _, option := range options {
			out = append(out, option["value"])
		}
		return out
	}
	got := values(PresentStatusOptions(map[string]bool{"listo": true, "nuevo": true}, ""))
	if len(got) != 2 || got[0] != "nuevo" || got[1] != "listo" {
		t.Fatalf("only present statuses in catalog order: %v", got)
	}
	got = values(PresentStatusOptions(map[string]bool{"nuevo": true}, "cancelado"))
	if len(got) != 2 || got[1] != "cancelado" {
		t.Fatalf("the selected status stays even without orders: %v", got)
	}
	if got = values(PresentStatusOptions(nil, "abiertos")); len(got) != 0 {
		t.Fatalf("abiertos is not a catalog status: %v", got)
	}
}

func TestPresentStatusOptionsGroupsShippingUnderReady(t *testing.T) {
	for _, selected := range []string{"", "listo", "en_camino"} {
		options := PresentStatusOptions(map[string]bool{"en_camino": true, "nuevo": false}, selected)
		if len(options) != 1 || options[0]["value"] != "listo" || options[0]["label"] != "Por entregar" {
			t.Fatalf("shipping stays pending delivery without a duplicate option: %+v", options)
		}
	}
	options := PresentStatusOptions(map[string]bool{"listo": true, "en_camino": true, "entregado": true}, "")
	if len(options) != 2 || options[0]["value"] != "listo" || options[1]["value"] != "entregado" {
		t.Fatalf("ready and shipping must share one operational option: %+v", options)
	}
}
