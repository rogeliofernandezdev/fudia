package domain

import "testing"

func TestNotificationQuery(t *testing.T) {
	if got := NotificationQuery("  Rosa   Quispe "); got != "Rosa Quispe" {
		t.Fatalf("espacios: %q", got)
	}
	long := ""
	for i := 0; i < 100; i++ {
		long += "á"
	}
	if got := []rune(NotificationQuery(long)); len(got) != 80 {
		t.Fatalf("límite de 80 caracteres sin cortar letras: %d", len(got))
	}
}

func TestNotificationAmountPrefix(t *testing.T) {
	cases := map[string]string{
		"25":      "25",
		"25.5":    "25.5",
		"25,50":   "25.50",
		"S/ 1.00": "1.00",
		"s/1":     "1",
		"25.":     "25.",
		"Rosa":    "",
		"25.555":  "",
		"1 2":     "",
		"":        "",
	}
	for in, want := range cases {
		if got := NotificationAmountPrefix(in); got != want {
			t.Fatalf("%q: quería %q, obtuve %q", in, want, got)
		}
	}
}
