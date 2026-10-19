package domain

import (
	"strings"
	"testing"
)

func TestResolveAvailabilityReason(t *testing.T) {
	cases := []struct {
		name, code, detail   string
		wantCode, wantReason string
		wantInvalid          bool
	}{
		{name: "motivo de un toque guarda su etiqueta", code: "no_supplies", wantCode: "no_supplies", wantReason: "Sin insumos"},
		{name: "el detalle se ignora fuera de Otro", code: "sold_out", detail: "texto", wantCode: "sold_out", wantReason: "Se acabó"},
		{name: "Otro guarda el detalle", code: "other", detail: "El proveedor no llegó", wantCode: "other", wantReason: "El proveedor no llegó"},
		{name: "Otro exige detalle", code: "other", wantInvalid: true},
		{name: "detalle demasiado largo", code: "other", detail: strings.Repeat("á", 241), wantInvalid: true},
		{name: "cliente anterior con texto libre se trata como Otro", detail: "Corrección", wantCode: "other", wantReason: "Corrección"},
		{name: "sin motivo", wantInvalid: true},
		{name: "código de otro catálogo", code: "extra_production", wantInvalid: true},
	}
	for _, c := range cases {
		code, reason, invalid := ResolveAvailabilityReason(AvailabilitySoldOutReasons, c.code, c.detail)
		if (invalid != "") != c.wantInvalid || code != c.wantCode || reason != c.wantReason {
			t.Fatalf("%s: code=%q reason=%q invalid=%q", c.name, code, reason, invalid)
		}
	}
	if code, reason, invalid := ResolveAvailabilityReason(AvailabilityQuotaReasons, "count_fix", ""); invalid != "" || code != "count_fix" || reason != "Corrección de conteo" {
		t.Fatalf("quota reason: %q %q %q", code, reason, invalid)
	}
	if AvailabilitySoldOutReasons[0].Value != "sold_out" {
		t.Fatal("«Se acabó» encabeza el catálogo: es el motivo preseleccionado en hora punta")
	}
}
