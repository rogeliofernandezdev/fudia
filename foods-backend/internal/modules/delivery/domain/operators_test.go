package domain

import (
	"strings"
	"testing"
)

func TestNormalizeOperator(t *testing.T) {
	in, problem := NormalizeOperator(OperatorInput{FullName: "  Lucía   Ramos ", Email: " Lucia@Central.PE "})
	if problem != "" || in.FullName != "Lucía Ramos" || in.Email != "lucia@central.pe" {
		t.Fatalf("normalized operator: %#v %q", in, problem)
	}
	for _, bad := range []OperatorInput{
		{FullName: "L", Email: "l@central.pe"},
		{FullName: strings.Repeat("x", 81), Email: "l@central.pe"},
		{FullName: "Lucía", Email: "sin-arroba"},
		{FullName: "Lucía", Email: "a@b"},
		{FullName: "Lucía", Email: strings.Repeat("a", 176) + "@x.pe"},
	} {
		if _, problem := NormalizeOperator(bad); problem == "" {
			t.Fatalf("invalid operator accepted: %#v", bad)
		}
	}
}

func TestOperatorPasswordProblem(t *testing.T) {
	if OperatorPasswordProblem("1234567") == "" || OperatorPasswordProblem(strings.Repeat("a", 73)) == "" {
		t.Fatal("password limits not enforced")
	}
	if OperatorPasswordProblem("ocho-car") != "" || OperatorPasswordProblem("ñandú-12") != "" {
		t.Fatal("valid password rejected")
	}
}

func TestTemporaryPasswordUsesUnambiguousCharacters(t *testing.T) {
	random := make([]byte, 64)
	for i := range random {
		random[i] = byte(i * 7)
	}
	password, ok := TemporaryPassword(random)
	if !ok || len(password) != TemporaryPasswordLength {
		t.Fatalf("temporary password: %q %v", password, ok)
	}
	for _, r := range password {
		if !strings.ContainsRune(TemporaryPasswordAlphabet, r) || strings.ContainsRune("0O1lI", r) {
			t.Fatalf("unexpected character %q in %q", r, password)
		}
	}
	// Los bytes que sesgarían el reparto se descartan.
	if _, ok := TemporaryPassword([]byte{255, 254, 253, 252}); ok {
		t.Fatal("biased bytes must be discarded")
	}
}
