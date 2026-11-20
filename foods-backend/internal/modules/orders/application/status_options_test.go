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
