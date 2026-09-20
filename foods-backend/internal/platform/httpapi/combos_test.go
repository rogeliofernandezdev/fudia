package httpapi

import "testing"

func TestNormalizedSurcharge(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: "0"},
		{name: "spaces", in: "   ", want: "0"},
		{name: "amount", in: " 2.50 ", want: "2.50"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizedSurcharge(tt.in); got != tt.want {
				t.Fatalf("normalizedSurcharge(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
