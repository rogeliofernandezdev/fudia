package httpapi

import "testing"

func TestEditableOrderStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{status: "nuevo", want: true},
		{status: "confirmado", want: true},
		{status: "preparando", want: false},
		{status: "listo", want: false},
		{status: "en_camino", want: false},
		{status: "entregado", want: false},
		{status: "cancelado", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := editableOrderStatus(tt.status); got != tt.want {
				t.Fatalf("editableOrderStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
