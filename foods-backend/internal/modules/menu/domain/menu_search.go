package domain

import (
	"strings"
	"unicode"
)

// maxSearchTerms acota cuántas palabras de una búsqueda llegan a SQL.
const maxSearchTerms = 8

var searchAccents = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
	"à", "a", "è", "e", "ì", "i", "ò", "o", "ù", "u",
)

// searchStopWords son palabras de una pregunta general («¿qué tienes para
// hoy?», «quiero comer algo») que no nombran un producto. Una búsqueda hecha
// solo de ellas equivale a pedir la carta completa.
var searchStopWords = map[string]bool{
	"a": true, "al": true, "algo": true, "ahora": true, "buenas": true, "bueno": true,
	"buenos": true, "carta": true, "comer": true, "con": true, "dame": true, "de": true,
	"deme": true, "del": true, "deseo": true, "dia": true, "disponible": true,
	"disponibles": true, "el": true, "en": true, "es": true, "favor": true, "hay": true,
	"hola": true, "hoy": true, "la": true, "las": true, "lo": true, "los": true, "me": true,
	"mi": true, "noches": true, "o": true, "pa": true, "para": true, "pero": true, "plato": true,
	"platos": true, "por": true, "porfa": true, "porfavor": true, "pra": true,
	"producto": true, "productos": true, "que": true, "queria": true, "quiero": true,
	"quisiera": true, "recomienda": true, "recomiendas": true, "se": true, "sus": true,
	"tardes": true, "te": true, "tiene": true, "tienen": true, "tienes": true,
	"tomar": true, "traeme": true, "tu": true, "un": true, "una": true, "unas": true,
	"unos": true, "ver": true, "y": true,
}

// NormalizeSearchText lleva un texto a minúsculas y sin tildes, igual que la
// comparación SQL de la carta pública.
func NormalizeSearchText(value string) string {
	return searchAccents.Replace(strings.ToLower(value))
}

// SearchTerms separa una búsqueda de la carta en palabras comparables: sin
// tildes ni signos, sin palabras de relleno y con plurales simples reducidos
// («ceviches» → «cevich», «limones» → «limon», «papas» → «papa»). Sin
// términos, la búsqueda devuelve la carta completa.
func SearchTerms(q string) []string {
	words := strings.FieldsFunc(NormalizeSearchText(q), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	terms := []string{}
	seen := map[string]bool{}
	for _, word := range words {
		if searchStopWords[word] {
			continue
		}
		term := singularSearchTerm(word)
		if seen[term] {
			continue
		}
		seen[term] = true
		terms = append(terms, term)
		if len(terms) == maxSearchTerms {
			break
		}
	}
	return terms
}

func singularSearchTerm(word string) string {
	switch {
	case len(word) > 4 && strings.HasSuffix(word, "es"):
		return strings.TrimSuffix(word, "es")
	case len(word) > 3 && strings.HasSuffix(word, "s"):
		return strings.TrimSuffix(word, "s")
	}
	return word
}
