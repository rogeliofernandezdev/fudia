package domain

import (
	"strings"
	"testing"
	"time"
)

func validEvent() Event {
	return Event{Provider: ProviderYape, EventKey: "yape:op-9f81d2:12:30", SenderName: "Rosa Quispe", Amount: 62.5, OccurredAt: time.Date(2026, 11, 15, 12, 30, 0, 0, time.UTC)}
}

func TestNormalizeEventAceptaYapeYPlinYNormaliza(t *testing.T) {
	cases := map[string]Event{
		"yape": validEvent(),
		"plin": {Provider: " PLIN ", EventKey: "plin-abc12345", SenderName: "  Ana  Villanueva ", Amount: 10, OccurredAt: time.Now()},
	}
	for name, event := range cases {
		clean, err := NormalizeEvent(event)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if clean.Provider != strings.ToLower(strings.TrimSpace(event.Provider)) {
			t.Fatalf("%s: proveedor no normalizado", name)
		}
	}
}

func TestNormalizeEventRechazaLoAmbiguo(t *testing.T) {
	cases := map[string]Event{
		"proveedor desconocido": {Provider: "mercadopago", EventKey: "x-12345678", Amount: 10, OccurredAt: time.Now()},
		"clave corta":           {Provider: ProviderYape, EventKey: "corta", Amount: 10, OccurredAt: time.Now()},
		"clave vacía":           {Provider: ProviderYape, EventKey: "        ", Amount: 10, OccurredAt: time.Now()},
		"monto cero":            {Provider: ProviderYape, EventKey: "yape-12345678", Amount: 0, OccurredAt: time.Now()},
		"monto negativo":        {Provider: ProviderYape, EventKey: "yape-12345678", Amount: -5, OccurredAt: time.Now()},
		"monto irreal":          {Provider: ProviderYape, EventKey: "yape-12345678", Amount: 2e7, OccurredAt: time.Now()},
	}
	for name, event := range cases {
		if _, err := NormalizeEvent(event); err == nil {
			t.Fatalf("%s: se aceptó un evento ambiguo", name)
		}
	}
}

func TestNormalizeEventAcotaElRemitente(t *testing.T) {
	event := validEvent()
	event.SenderName = strings.Repeat("a", SenderNameLimit+30)
	clean, err := NormalizeEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	if runes := len([]rune(clean.SenderName)); runes != SenderNameLimit {
		t.Fatalf("remitente no acotado: %d", runes)
	}
}

func TestNormalizeEventOperationNumber(t *testing.T) {
	for _, value := range []string{"", "0012345678", "IBK-001234", strings.Repeat("1", OperationNumberMax)} {
		event := validEvent()
		event.OperationNumber = " " + value + " "
		clean, err := NormalizeEvent(event)
		if err != nil || clean.OperationNumber != value || clean.EventKey != event.EventKey {
			t.Fatalf("operation %q: %+v, %v", value, clean, err)
		}
	}
	for _, value := range []string{"123 456", "123\n456", strings.Repeat("1", OperationNumberMax+1), "<123>"} {
		event := validEvent()
		event.OperationNumber = value
		if _, err := NormalizeEvent(event); err == nil {
			t.Fatalf("invalid operation accepted: %q", value)
		}
	}
}

func TestPairingCodeSoloVigenteYSinUsar(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	fresh := PairingCode{ExpiresAt: now.Add(PairingCodeMinutes * time.Minute)}
	if !fresh.Redeemable(now) {
		t.Fatal("un código vigente debe canjearse")
	}
	used := fresh
	used.UsedAt = &now
	if used.Redeemable(now) {
		t.Fatal("un código usado no debe canjearse")
	}
	expired := PairingCode{ExpiresAt: now.Add(-time.Minute)}
	if expired.Redeemable(now) {
		t.Fatal("un código vencido no debe canjearse")
	}
}

func TestFormatAmountDosDecimales(t *testing.T) {
	if got := FormatAmount(62.5); got != "62.50" {
		t.Fatalf("FormatAmount = %q", got)
	}
}

func TestPairingStatusOf(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	later := now.Add(5 * time.Minute)
	scanned := now.Add(-time.Minute)
	cases := []struct {
		name  string
		state PairingState
		want  PairingStatus
	}{
		{"pending", PairingState{ExpiresAt: later}, PairingStatus{Status: PairingPending}},
		{"scanned", PairingState{ExpiresAt: later, ScannedAt: &scanned, Model: "Pixel 8"}, PairingStatus{Status: PairingScanned, Model: "Pixel 8"}},
		{"linked wins over expiry", PairingState{ExpiresAt: now.Add(-time.Hour), ScannedAt: &scanned, UsedAt: &scanned, DeviceName: "Caja 1"}, PairingStatus{Status: PairingLinked, DeviceName: "Caja 1"}},
		{"expired after scan", PairingState{ExpiresAt: now.Add(-time.Second), ScannedAt: &scanned}, PairingStatus{Status: PairingExpired}},
	}
	for _, c := range cases {
		got := c.state.StatusOf(now)
		got.ExpiresAt = ""
		if got != c.want {
			t.Fatalf("%s: %#v", c.name, got)
		}
	}
}
