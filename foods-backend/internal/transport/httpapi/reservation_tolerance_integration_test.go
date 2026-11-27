package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	auditpg "github.com/foods-platform/foods-backend/internal/modules/audit/infrastructure/postgres"
	noticeapp "github.com/foods-platform/foods-backend/internal/modules/notifications/application"
	noticepg "github.com/foods-platform/foods-backend/internal/modules/notifications/infrastructure/postgres"
	serviceapp "github.com/foods-platform/foods-backend/internal/modules/service/application"
	servicepg "github.com/foods-platform/foods-backend/internal/modules/service/infrastructure/postgres"
	"github.com/foods-platform/foods-backend/internal/platform/database"
)

// Una reserva toma la tolerancia del local, puede ajustarla y, vencida sin que
// el cliente llegue, pasa sola a «No asistió» con un aviso único para caja.
func TestReservationToleranceExpiresAsNoShowAndNotifiesCashier(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE locations SET reservation_tolerance_minutes=20,reservation_deposit_amount=0,timezone='America/Lima' WHERE id=$1`, s.LocationID); err != nil {
		t.Fatal(err)
	}
	create := func(body string) reservationView {
		t.Helper()
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
	starts := time.Now().UTC().Add(time.Hour).Truncate(time.Minute).Format(time.RFC3339)
	inherited := create(fmt.Sprintf(`{"customerName":"Ana Tolerancia","startsAt":%q,"guests":4}`, starts))
	if inherited.ToleranceMinutes != 20 || inherited.Expired || inherited.DepositAmount != "0.00" {
		t.Fatalf("new reservation must take the location tolerance and no deposit by default: %#v", inherited)
	}
	custom := create(fmt.Sprintf(`{"customerName":"Luis Puntual","startsAt":%q,"guests":2,"toleranceMinutes":0,"depositAmount":"50.5"}`, starts))
	if custom.ToleranceMinutes != 0 || custom.DepositAmount != "50.50" {
		t.Fatalf("explicit zero tolerance and deposit must be kept: %#v", custom)
	}
	if _, err := pool.Exec(ctx, `UPDATE locations SET reservation_deposit_amount=30 WHERE id=$1`, s.LocationID); err != nil {
		t.Fatal(err)
	}
	if withDeposit := create(fmt.Sprintf(`{"customerName":"Con garantía","startsAt":%q,"guests":2}`, starts)); withDeposit.DepositAmount != "30.00" {
		t.Fatalf("new reservation must take the location deposit: %#v", withDeposit)
	}
	if _, err := pool.Exec(ctx, `UPDATE locations SET reservation_deposit_amount=0 WHERE id=$1`, s.LocationID); err != nil {
		t.Fatal(err)
	}
	invalid := httptest.NewRequest("POST", "/v1/admin/reservations", bytes.NewReader([]byte(fmt.Sprintf(`{"customerName":"X","startsAt":%q,"guests":2,"toleranceMinutes":241}`, starts))))
	invalid = invalid.WithContext(context.WithValue(invalid.Context(), scopeKey{}, s))
	invalidRec := httptest.NewRecorder()
	api.createReservation(invalidRec, invalid)
	if invalidRec.Code != 400 || !strings.Contains(invalidRec.Body.String(), "invalid_reservation") {
		t.Fatalf("tolerance above 240 must be rejected: %d %s", invalidRec.Code, invalidRec.Body.String())
	}

	// Editar sin toleranceMinutes conserva la de la reserva.
	edit := httptest.NewRequest("PATCH", "/v1/admin/reservations/"+inherited.ID, bytes.NewReader([]byte(fmt.Sprintf(`{"customerName":"Ana Tolerancia","startsAt":%q,"guests":3}`, starts))))
	edit.SetPathValue("id", inherited.ID)
	edit = edit.WithContext(context.WithValue(edit.Context(), scopeKey{}, s))
	editRec := httptest.NewRecorder()
	api.updateReservation(editRec, edit)
	if editRec.Code != 200 || !strings.Contains(editRec.Body.String(), `"toleranceMinutes":20`) {
		t.Fatalf("edit must keep the tolerance: %d %s", editRec.Code, editRec.Body.String())
	}

	// La reserva heredada empezó hace 25 min (vence a los 20); la puntual empieza
	// dentro de 5 min y no vence aún; una sentada nunca vence.
	var seatedID string
	if _, err := pool.Exec(ctx, `UPDATE reservations SET starts_at=now()-interval '25 minutes' WHERE id=$1`, inherited.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE reservations SET starts_at=now()+interval '5 minutes' WHERE id=$1`, custom.ID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO reservations(organization_id,location_id,customer_name,starts_at,guests,status,tolerance_minutes)
		VALUES($1,$2,'Mesa sentada',now()-interval '2 hours',2,'seated',0) RETURNING id
	`, s.OrganizationID, s.LocationID).Scan(&seatedID); err != nil {
		t.Fatal(err)
	}

	reservations := serviceapp.NewReservations(servicepg.NewReservations(pool), database.NewUnitOfWork(pool), auditpg.New(pool)).
		WithNotices(noticeapp.New(noticepg.New(pool), nil))
	if _, err := reservations.ExpireOverdue(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := reservations.ExpireOverdue(ctx); err != nil {
		t.Fatal(err)
	}
	status := func(id string) (string, bool) {
		var value string
		var expired bool
		if err := pool.QueryRow(ctx, `SELECT status,expired_at IS NOT NULL FROM reservations WHERE id=$1`, id).Scan(&value, &expired); err != nil {
			t.Fatal(err)
		}
		return value, expired
	}
	if value, expired := status(inherited.ID); value != "no_show" || !expired {
		t.Fatalf("overdue reservation must expire as no_show: %s %v", value, expired)
	}
	if value, expired := status(custom.ID); value != "pending" || expired {
		t.Fatalf("future reservation must stay open: %s %v", value, expired)
	}
	if value, expired := status(seatedID); value != "seated" || expired {
		t.Fatalf("seated reservation must not expire: %s %v", value, expired)
	}
	var notices int
	var title, detail, audience string
	if err := pool.QueryRow(ctx, `
		SELECT count(*) OVER (),title,detail,audience FROM staff_notices
		WHERE organization_id=$1 AND location_id=$2 AND kind='reservation_no_show'
	`, s.OrganizationID, s.LocationID).Scan(&notices, &title, &detail, &audience); err != nil {
		t.Fatal(err)
	}
	if notices != 1 || audience != "cashier" || title != "Reserva cancelada: Ana Tolerancia no llegó" || !strings.Contains(detail, "3 personas") {
		t.Fatalf("one cashier notice expected: %d %q %q %q", notices, title, detail, audience)
	}

	// Una reserva vencida ya no se edita.
	again := httptest.NewRequest("PATCH", "/v1/admin/reservations/"+inherited.ID, bytes.NewReader([]byte(fmt.Sprintf(`{"customerName":"Ana","startsAt":%q,"guests":3}`, starts))))
	again.SetPathValue("id", inherited.ID)
	again = again.WithContext(context.WithValue(again.Context(), scopeKey{}, s))
	againRec := httptest.NewRecorder()
	api.updateReservation(againRec, again)
	if againRec.Code != 409 {
		t.Fatalf("expired reservation must not be editable: %d %s", againRec.Code, againRec.Body.String())
	}
	list := httptest.NewRequest("GET", "/v1/admin/reservations?status=no_show&page=1&pageSize=20", nil)
	list = list.WithContext(context.WithValue(list.Context(), scopeKey{}, s))
	listRec := httptest.NewRecorder()
	api.listReservations(listRec, list)
	if listRec.Code != 200 || !strings.Contains(listRec.Body.String(), `"expired":true`) || !strings.Contains(listRec.Body.String(), `"defaultToleranceMinutes":20`) || !strings.Contains(listRec.Body.String(), `"defaultDepositAmount":"0.00"`) {
		t.Fatalf("list must expose expiry and default tolerance: %d %s", listRec.Code, listRec.Body.String())
	}
}
