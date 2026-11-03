package domain

import "testing"

func TestAudiencesFollowPermissions(t *testing.T) {
	cases := []struct {
		service, cashier bool
		want             []string
	}{
		{false, false, []string{}},
		{true, false, []string{AudienceService, AudienceAll}},
		{false, true, []string{AudienceCashier, AudienceAll}},
		{true, true, []string{AudienceService, AudienceCashier, AudienceAll}},
	}
	for _, c := range cases {
		got := Audiences(c.service, c.cashier)
		if len(got) != len(c.want) {
			t.Fatalf("Audiences(%v,%v)=%v want %v", c.service, c.cashier, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("Audiences(%v,%v)=%v want %v", c.service, c.cashier, got, c.want)
			}
		}
	}
}

func TestNormalizeRejectsUnknownAndClipsText(t *testing.T) {
	if _, ok := Normalize(Input{Kind: "promo", Audience: AudienceAll, Title: "x"}); ok {
		t.Fatal("unknown kind accepted")
	}
	if _, ok := Normalize(Input{Kind: KindBill, Audience: "kitchen", Title: "x"}); ok {
		t.Fatal("unknown audience accepted")
	}
	if _, ok := Normalize(Input{Kind: KindBill, Audience: AudienceAll, Title: "  "}); ok {
		t.Fatal("empty title accepted")
	}
	long := ""
	for range 300 {
		long += "ñ"
	}
	in, ok := Normalize(Input{Kind: KindAssistance, Audience: AudienceService, Title: long, Detail: long})
	if !ok || len([]rune(in.Title)) != 120 || len([]rune(in.Detail)) != 240 {
		t.Fatalf("clip by runes: %d %d", len([]rune(in.Title)), len([]rune(in.Detail)))
	}
}

func TestBuildersWriteStaffText(t *testing.T) {
	table := TableOrder("t1", "Mesa 4", "o1", "Ana", 3, "r1")
	if table.Audience != AudienceService || table.Title != "Mesa 4 · nuevo pedido" || table.Detail != "3 productos · Ana" || table.DedupeKey != "order:o1:r1" {
		t.Fatalf("table order: %+v", table)
	}
	counter := CounterOrder("Mostrador", "o2", "PED-0012", "", 1, Money("S/", "25"))
	if counter.Audience != AudienceCashier || counter.Title != "Nuevo pedido en Mostrador" || counter.Detail != "PED-0012 · 1 producto · S/ 25.00" {
		t.Fatalf("counter order: %+v", counter)
	}
	if remote := RemoteOrder("recojo", "o3", "PED-1", "Ana", 2, ""); remote.Kind != KindPickup || remote.Title != "Nuevo pedido para recoger" {
		t.Fatalf("pickup: %+v", remote)
	}
	if remote := RemoteOrder("delivery", "o3", "PED-1", "Ana", 2, ""); remote.Kind != KindDelivery {
		t.Fatalf("delivery: %+v", remote)
	}
	if bill := BillRequested("t1", "Mesa 4", "o1", "", ""); bill.Detail != "" || bill.DedupeKey != "" || bill.Audience != AudienceAll {
		t.Fatalf("bill without request: %+v", bill)
	}
	salon := OrderReady("round1", "bar", "o1", "t1", "Mesa 4", "PED-1", "Ana")
	if salon.Audience != AudienceService || salon.Title != "Mesa 4 · listo para entregar" || salon.Detail != "Barra · PED-1" || salon.DedupeKey != "ready:round1:bar" {
		t.Fatalf("salon ready: %+v", salon)
	}
	pickup := OrderReady("round2", "kitchen", "o2", "", "", "PED-2", "")
	if pickup.Audience != AudienceAll || pickup.Title != "PED-2 listo para entregar" {
		t.Fatalf("counter ready without name: %+v", pickup)
	}
	if Money("S/", "abc") != "" {
		t.Fatal("invalid amount must be empty")
	}
}
