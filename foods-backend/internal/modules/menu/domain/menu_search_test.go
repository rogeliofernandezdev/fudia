package domain

import (
	"reflect"
	"testing"
)

func TestSearchTerms(t *testing.T) {
	cases := []struct {
		name, q string
		want    []string
	}{
		{name: "pregunta general pide la carta completa", q: "¿Qué tienes pra hoy?", want: []string{}},
		{name: "quiero comer también es la carta", q: "Pero quiero comer algo", want: []string{}},
		{name: "sin tildes y en minúsculas", q: "AJÍ de Gallina", want: []string{"aji", "gallina"}},
		{name: "plurales simples", q: "ceviches, limones y papas", want: []string{"cevich", "limon", "papa"}},
		{name: "palabras cortas no se recortan", q: "res tres mas", want: []string{"res", "tre", "mas"}},
		{name: "signos y comodines se descartan", q: "lomo% _saltado!!", want: []string{"lomo", "saltado"}},
		{name: "sin repetidos", q: "lomo lomos LOMO", want: []string{"lomo"}},
		{name: "vacío", q: "   ", want: []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SearchTerms(tc.q); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("SearchTerms(%q) = %#v, want %#v", tc.q, got, tc.want)
			}
		})
	}
}

func TestSearchTermsAreBounded(t *testing.T) {
	got := SearchTerms("uno dos tres cuatro cinco seis siete ocho nueve diez")
	if len(got) != maxSearchTerms {
		t.Fatalf("expected %d terms, got %#v", maxSearchTerms, got)
	}
}

func TestNormalizeSearchTextMatchesSQLTranslate(t *testing.T) {
	if got := NormalizeSearchText("Ají Ñandú PÚRÉ"); got != "aji nandu pure" {
		t.Fatalf("unexpected normalization: %q", got)
	}
}
