package httpapi

import "testing"

func TestValidKitchenTransition(t *testing.T) {
	tests := []struct {
		current string
		next    string
		want    bool
	}{
		{"confirmado", "preparando", true},
		{"preparando", "listo", true},
		{"confirmado", "listo", false},
		{"listo", "entregado", false},
		{"nuevo", "preparando", false},
		{"preparando", "cancelado", false},
	}
	for _, tt := range tests {
		if got := validKitchenTransition(tt.current, tt.next); got != tt.want {
			t.Fatalf("validKitchenTransition(%q,%q)=%v, want %v", tt.current, tt.next, got, tt.want)
		}
	}
}
