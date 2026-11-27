package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

// Registrar una reserva con teléfono deja prerregistrado al cliente en
// Clientes; el mismo número, escrito con o sin código de país, reutiliza ese
// cliente. Sin teléfono la reserva queda sin cliente.
func TestReservationPreRegistersCustomerByPhone(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	starts := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Minute).Format(time.RFC3339)
	create := func(name, phone string) reservationView {
		t.Helper()
		body := fmt.Sprintf(`{"customerName":%q,"customerPhone":%q,"startsAt":%q,"guests":2}`, name, phone, starts)
		req := httptest.NewRequest("POST", "/v1/admin/reservations", bytes.NewReader([]byte(body)))
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		api.createReservation(rec, req)
		if rec.Code != 201 {
			t.Fatalf("create reservation: %d %s", rec.Code, rec.Body.String())
		}
		var item reservationView
		if err := json.Unmarshal(rec.Body.Bytes(), &item); err != nil {
			t.Fatal(err)
		}
		return item
	}

	first := create("Ana Reserva", "987 654 321")
	if first.CustomerID == nil || *first.CustomerID == "" {
		t.Fatalf("a reservation with phone must pre-register its customer: %#v", first)
	}
	var name, phone string
	if err := pool.QueryRow(ctx, `SELECT display_name,phone FROM customers WHERE id=$1 AND organization_id=$2`, *first.CustomerID, s.OrganizationID).Scan(&name, &phone); err != nil {
		t.Fatal(err)
	}
	if name != "Ana Reserva" || phone != "987 654 321" {
		t.Fatalf("pre-registered customer keeps name and phone as typed: %q %q", name, phone)
	}
	again := create("Ana R.", "+51 987654321")
	if again.CustomerID == nil || *again.CustomerID != *first.CustomerID {
		t.Fatalf("same phone with country code must reuse the customer: %#v", again)
	}
	if noPhone := create("Sin Teléfono", ""); noPhone.CustomerID != nil && *noPhone.CustomerID != "" {
		t.Fatalf("a reservation without phone must not create a customer: %#v", noPhone)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM customers WHERE organization_id=$1`, s.OrganizationID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("only one customer must exist for the organization, got %d", count)
	}
}
