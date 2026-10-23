package postgres

import "testing"

func TestContainsPatternMatchesWildcardsLiterally(t *testing.T) {
	cases := map[string]string{
		" arroz ":   "%arroz%",
		"50%":       `%50\%%`,
		"agua_mesa": `%agua\_mesa%`,
		`a\b`:       `%a\\b%`,
		"":          "%%",
	}
	for query, want := range cases {
		if got := containsPattern(query); got != want {
			t.Fatalf("containsPattern(%q)=%q, want %q", query, got, want)
		}
	}
}
