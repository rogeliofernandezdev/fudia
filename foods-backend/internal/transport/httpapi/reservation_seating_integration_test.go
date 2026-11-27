package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// Ubicar al cliente («En mesa») exige mesa: la elegida o la que ya tenía la
// reserva. Terminado el tiempo reservado, el listado la muestra como «Asistió»
// (attended) y el filtro por estado la separa de las que siguen en mesa.
func TestReservationSeatingRequiresTableAndEndsAsAttended(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	var tableID, reservationID string
	if err := pool.QueryRow(ctx, `INSERT INTO tables(organization_id,location_id,name,seats,zone,qr_token) VALUES($1,$2,'Mesa ubicación',4,'Principal',md5(random()::text||clock_timestamp()::text)) RETURNING id`, s.OrganizationID, s.LocationID).Scan(&tableID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO reservations(organization_id,location_id,customer_name,starts_at,guests,status,duration_minutes) VALUES($1,$2,'Rosa Llegada',now()+interval '1 hour',2,'confirmed',90) RETURNING id`, s.OrganizationID, s.LocationID).Scan(&reservationID); err != nil {
		t.Fatal(err)
	}
	patch := func(body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest("PATCH", "/v1/admin/reservations/"+reservationID+"/status", bytes.NewReader([]byte(body)))
		req.SetPathValue("id", reservationID)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		api.updateReservationStatus(rec, req)
		return rec
	}
	list := func(status string) []reservationView {
		t.Helper()
		req := httptest.NewRequest("GET", "/v1/admin/reservations?status="+status+"&page=1&pageSize=20", nil)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		api.listReservations(rec, req)
		if rec.Code != 200 {
			t.Fatalf("list reservations %q: %d %s", status, rec.Code, rec.Body.String())
		}
		var page struct{ Items []reservationView }
		if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		return page.Items
	}

	if rec := patch(`{"status":"seated"}`); rec.Code != 400 || !strings.Contains(rec.Body.String(), "seat_table_required") {
		t.Fatalf("seating without a table must be rejected: %d %s", rec.Code, rec.Body.String())
	}
	if rec := patch(`{"status":"seated","tableId":"` + tableID + `"}`); rec.Code != 204 {
		t.Fatalf("seat at table: %d %s", rec.Code, rec.Body.String())
	}
	seated := list("seated")
	if len(seated) != 1 || seated[0].Status != "seated" || seated[0].TableName != "Mesa ubicación" {
		t.Fatalf("seated reservation must keep its table: %#v", seated)
	}
	if rec := patch(`{"status":"attended"}`); rec.Code != 400 {
		t.Fatalf("attended is read-only: %d %s", rec.Code, rec.Body.String())
	}

	// Terminado el tiempo reservado deja de estar «En mesa» y pasa a «Asistió».
	if _, err := pool.Exec(ctx, `UPDATE reservations SET starts_at=now()-interval '3 hours' WHERE id=$1`, reservationID); err != nil {
		t.Fatal(err)
	}
	if items := list("seated"); len(items) != 0 {
		t.Fatalf("a finished reservation is no longer seated: %#v", items)
	}
	attended := list("attended")
	if len(attended) != 1 || attended[0].Status != "attended" || attended[0].TableName != "Mesa ubicación" {
		t.Fatalf("finished reservation must list as attended with its table: %#v", attended)
	}
}
