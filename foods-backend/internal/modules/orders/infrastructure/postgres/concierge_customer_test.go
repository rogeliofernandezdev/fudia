package postgres

import "testing"

func TestPhoneDigitsKeepsOnlyNumbers(t *testing.T) {
	cases := map[string]string{
		"51999999999":     "51999999999",
		"+51 999 999 999": "51999999999",
		"(01) 234-5678":   "012345678",
		"":                "",
		"sin número":      "",
	}
	for in, want := range cases {
		if got := phoneDigits(in); got != want {
			t.Fatalf("phoneDigits(%q)=%q, want %q", in, got, want)
		}
	}
}
