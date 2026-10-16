package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeliveryContactValidation(t *testing.T) {
	for _, tc := range []struct {
		name, phone, address, reference string
		valid                           bool
	}{
		{"Ana", "+51 (999) 123-456", "Av. Perú 100", "Puerta azul", true},
		{"José", "999123456", "Calle 1", "", true},
		{"", "999123456", "Calle 1", "", false},
		{"Ana", "", "Calle 1", "", false},
		{"Ana", "123", "Calle 1", "", false},
		{"Ana", "1234567890123456", "Calle 1", "", false},
		{"Ana", "abc999123456", "Calle 1", "", false},
		{"Ana", "999+123456", "Calle 1", "", false},
		{"Ana", "999123456", "   ", "", false},
		{strings.Repeat("a", 161), "999123456", "Calle 1", "", false},
		{"Ana", "999123456", strings.Repeat("a", 241), "", false},
		{"Ana", "999123456", "Calle 1", strings.Repeat("a", 241), false},
	} {
		message := validateDeliveryContact("delivery", tc.name, tc.phone, tc.address, tc.reference)
		if (message == "") != tc.valid {
			t.Fatalf("validation %+v: %q", tc, message)
		}
	}
	if message := validateDeliveryContact("salon", "", "", "", ""); message != "" {
		t.Fatal("salon must retain optional contact", message)
	}
}

func TestCreateDeliveryRejectsInvalidContactBeforeDatabase(t *testing.T) {
	api := New(nil)
	for _, fields := range []map[string]any{
		{"customerName": "", "customerPhone": "999123456", "address": "Calle 1"},
		{"customerName": "Ana", "customerPhone": "abc", "address": "Calle 1"},
		{"customerName": "Ana", "customerPhone": "999123456", "address": ""},
	} {
		fields["channel"] = "delivery"
		fields["items"] = []map[string]any{{"productId": "unused", "qty": 1}}
		body, _ := json.Marshal(fields)
		request := httptest.NewRequest("POST", "/v1/admin/orders", bytes.NewReader(body))
		request = request.WithContext(context.WithValue(request.Context(), scopeKey{}, scope{}))
		recorder := httptest.NewRecorder()
		api.createOrder(recorder, request)
		if recorder.Code != 400 || !strings.Contains(recorder.Body.String(), "invalid_order") {
			t.Fatalf("invalid delivery contact: %d %s", recorder.Code, recorder.Body.String())
		}
	}
}
