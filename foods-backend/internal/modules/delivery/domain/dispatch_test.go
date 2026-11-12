package domain

import "testing"

func TestNormalizePartnerRequiresCountryCode(t *testing.T) {
	in, problem := NormalizePartner(PartnerInput{Name: "  Rápido   Express ", WhatsAppPhone: "+51 987 654 321", CoverageCodes: []string{"150101"}})
	if problem != "" || in.Name != "Rápido Express" || in.WhatsAppPhone != "51987654321" {
		t.Fatalf("got %#v %q", in, problem)
	}
	for _, phone := range []string{"0987654321", "1234567", "1234567890123456", ""} {
		if _, problem := NormalizePartner(PartnerInput{Name: "Central", WhatsAppPhone: phone}); problem == "" {
			t.Fatalf("phone %q accepted", phone)
		}
	}
	if _, problem := NormalizePartner(PartnerInput{Name: "C", WhatsAppPhone: "51987654321"}); problem == "" {
		t.Fatal("one-letter name accepted")
	}
}

func TestNormalizeOrder(t *testing.T) {
	ids, problem := NormalizeOrder(PartnerOrderInput{PartnerIDs: []string{" a ", "", "b"}})
	if problem != "" || len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
		t.Fatalf("got %#v %q", ids, problem)
	}
	if _, problem := NormalizeOrder(PartnerOrderInput{PartnerIDs: []string{"a", "a"}}); problem == "" {
		t.Fatal("duplicates accepted")
	}
	if _, problem := NormalizeOrder(PartnerOrderInput{}); problem == "" {
		t.Fatal("empty order accepted")
	}
}

func TestPartnerCoverageMustBeExplicit(t *testing.T) {
	for _, codes := range [][]string{nil, {}, {"Lima"}, {"15"}, {"1501"}, {"150"}, {"1501011"}} {
		if _, problem := NormalizePartner(PartnerInput{Name: "Central", WhatsAppPhone: "51987654321", CoverageCodes: codes}); problem == "" {
			t.Fatalf("invalid coverage accepted: %#v", codes)
		}
	}
	in, problem := NormalizePartner(PartnerInput{Name: "Central", WhatsAppPhone: "51987654321", CoverageCodes: []string{"150101", "150102", "150122", "150101"}})
	if problem != "" || len(in.CoverageCodes) != 3 {
		t.Fatalf("valid coverage: %#v %s", in, problem)
	}
}

func TestOrderDispatchable(t *testing.T) {
	for _, status := range []string{"confirmado", "preparando", "listo"} {
		if !OrderDispatchable("delivery", status, false) {
			t.Fatalf("%s should be dispatchable", status)
		}
	}
	if OrderDispatchable("delivery", "nuevo", false) || OrderDispatchable("delivery", "listo", true) ||
		OrderDispatchable("recojo", "confirmado", false) || OrderDispatchable("delivery", "en_camino", false) {
		t.Fatal("unexpected dispatchable order")
	}
}

func TestRetryableAndActive(t *testing.T) {
	if !Retryable(DispatchUnassigned) || !Retryable(DispatchOwn) || Retryable(DispatchSearching) || Retryable(DispatchAccepted) {
		t.Fatal("retryable states")
	}
	if !ActiveDispatch(DispatchSearching) || !ActiveDispatch(DispatchAccepted) || ActiveDispatch(DispatchPickedUp) || ActiveDispatch(DispatchOwn) {
		t.Fatal("active states")
	}
}
