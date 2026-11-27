package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// La garantía de una reserva se cobra en el turno de caja, se descuenta de la
// cuenta cuando el cliente llega y, si no llega o se cancela reteniéndola, queda
// como venta de mostrador; también puede devolverse desde el turno.
func TestReservationDepositChargeApplyRefundAndRetain(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	grantCashManagementForPaymentTest(t, api, s)
	call := func(method, body, id string, handler http.HandlerFunc) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/", strings.NewReader(body))
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		handler(rec, req)
		return rec
	}
	if _, err := pool.Exec(ctx, `UPDATE locations SET timezone='America/Lima' WHERE id=$1`, s.LocationID); err != nil {
		t.Fatal(err)
	}
	var register cashRegisterView
	_ = json.Unmarshal(call("POST", `{"name":"Caja garantías"}`, "", api.createCashRegister).Body.Bytes(), &register)
	opened := call("POST", fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":100}`, register.ID), "", api.openCashShift)
	if opened.Code != 201 {
		t.Fatalf("open shift: %d %s", opened.Code, opened.Body.String())
	}
	var shift cashShiftView
	_ = json.Unmarshal(opened.Body.Bytes(), &shift)

	reservation := func(name string, deposit int) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO reservations(organization_id,location_id,customer_name,starts_at,guests,status,tolerance_minutes,deposit_amount)
			VALUES($1,$2,$3,now()+interval '10 minutes',2,'confirmed',15,$4) RETURNING id`, s.OrganizationID, s.LocationID, name, deposit).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	charge := func(id, method string) {
		t.Helper()
		if rec := call("POST", fmt.Sprintf(`{"method":%q,"reference":""}`, method), id, api.chargeReservationDeposit); rec.Code != 201 {
			t.Fatalf("charge deposit: %d %s", rec.Code, rec.Body.String())
		}
	}
	var productID string
	if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price) VALUES($1,'DEPOSIT-DISH','Lomo saltado',30) RETURNING id`, s.OrganizationID).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	order := func(total int) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			WITH o AS (INSERT INTO orders(organization_id,location_id,channel,status,customer_name,subtotal,total,created_by)
			  VALUES($1,$2,'mostrador','entregado','Mesa garantía',$3,$3,$4) RETURNING id,organization_id)
			INSERT INTO order_items(organization_id,order_id,product_id,name,qty,unit_price) SELECT organization_id,id,$5,'Lomo saltado',1,$3 FROM o
			RETURNING order_id`, s.OrganizationID, s.LocationID, total, s.UserID, productID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	depositOf := func(id string) (status, method, document string) {
		t.Helper()
		rec := call("GET", "", "", api.listReservations)
		var page struct{ Items []reservationView }
		_ = json.Unmarshal(rec.Body.Bytes(), &page)
		for _, item := range page.Items {
			if item.ID == id {
				return item.DepositStatus, item.DepositMethodName, item.DepositDocument
			}
		}
		t.Fatalf("reservation %s not listed: %s", id, rec.Body.String())
		return
	}

	// Sin garantía no hay nada que cobrar; con garantía se cobra una sola vez.
	free := reservation("Sin garantía", 0)
	if rec := call("POST", `{"method":"cash"}`, free, api.chargeReservationDeposit); rec.Code != 409 || !strings.Contains(rec.Body.String(), "reservation_without_deposit") {
		t.Fatalf("a reservation without deposit cannot be charged: %d %s", rec.Code, rec.Body.String())
	}
	if status, _, _ := depositOf(free); status != "none" {
		t.Fatalf("free reservation deposit status: %s", status)
	}
	ana := reservation("Ana Garantía", 50)
	if status, _, _ := depositOf(ana); status != "pending" {
		t.Fatalf("uncharged deposit must be pending: %s", status)
	}
	charge(ana, "cash")
	if rec := call("POST", `{"method":"cash"}`, ana, api.chargeReservationDeposit); rec.Code != 409 || !strings.Contains(rec.Body.String(), "deposit_already_charged") {
		t.Fatalf("second charge must be rejected: %d %s", rec.Code, rec.Body.String())
	}
	if status, method, _ := depositOf(ana); status != "paid" || method == "" {
		t.Fatalf("charged deposit: %s %q", status, method)
	}
	var cashIn int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM cash_movements WHERE shift_id=$1 AND source_type='reservation_deposit' AND amount=50`, shift.ID).Scan(&cashIn)
	if cashIn != 1 {
		t.Fatalf("cash deposit must enter the shift once: %d", cashIn)
	}
	// La reserva tiene mesa y el pedido no (mostrador o delivery): «suggested» es
	// false y la garantía igual se ofrece. Antes salía NULL y la lista venía vacía.
	var tableID string
	if err := pool.QueryRow(ctx, `INSERT INTO tables(organization_id,location_id,name,seats,zone,qr_token) VALUES($1,$2,'Mesa garantía',4,'Principal',md5(random()::text||clock_timestamp()::text)) RETURNING id`, s.OrganizationID, s.LocationID).Scan(&tableID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE reservations SET table_id=$2 WHERE id=$1`, ana, tableID); err != nil {
		t.Fatal(err)
	}

	// Cobros la ofrece y la descuenta: S/ 180 − S/ 50 = S/ 130 con tarjeta.
	dinner := order(180)
	var detail posOrderDetail
	_ = json.Unmarshal(call("GET", "", dinner, api.getPOSOrder).Body.Bytes(), &detail)
	var anaPayment string
	for _, option := range detail.Deposits {
		if option.CustomerName == "Ana Garantía" && option.Amount == "50.00" && !option.Suggested && option.TableName == "Mesa garantía" {
			anaPayment = option.PaymentID
		}
	}
	if anaPayment == "" {
		t.Fatalf("checkout must offer the held deposit: %+v", detail.Deposits)
	}
	if rec := call("POST", fmt.Sprintf(`{"orderId":%q,"method":"card","amount":140,"depositPaymentId":%q}`, dinner, anaPayment), "", api.createPayment); rec.Code != 409 || !strings.Contains(rec.Body.String(), "payment_exceeds_remaining") {
		t.Fatalf("payment above the discounted balance must be rejected: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call("POST", fmt.Sprintf(`{"orderId":%q,"method":"card","amount":130,"depositPaymentId":%q}`, dinner, anaPayment), "", api.createPayment); rec.Code != 201 {
		t.Fatalf("payment with deposit: %d %s", rec.Code, rec.Body.String())
	}
	var paid string
	var reservationStatus string
	_ = pool.QueryRow(ctx, `SELECT paid_amount::numeric(12,2)::text FROM orders WHERE id=$1`, dinner).Scan(&paid)
	_ = pool.QueryRow(ctx, `SELECT status FROM reservations WHERE id=$1`, ana).Scan(&reservationStatus)
	if paid != "180.00" || reservationStatus != "seated" {
		t.Fatalf("deposit must settle the bill and seat the reservation: paid %s status %s", paid, reservationStatus)
	}
	if status, _, _ := depositOf(ana); status != "applied" {
		t.Fatalf("applied deposit status: %s", status)
	}

	// Una garantía que cubre todo el saldo admite un cobro de cero.
	exact := reservation("Cubre todo", 50)
	charge(exact, "card")
	covered := order(50)
	var exactPayment string
	_ = pool.QueryRow(ctx, `SELECT id FROM payments WHERE reservation_id=$1`, exact).Scan(&exactPayment)
	if rec := call("POST", fmt.Sprintf(`{"orderId":%q,"method":"cash","amount":0,"depositPaymentId":%q}`, covered, exactPayment), "", api.createPayment); rec.Code != 201 {
		t.Fatalf("zero payment with covering deposit: %d %s", rec.Code, rec.Body.String())
	}

	// Mayor que el consumo no se descuenta: se devuelve desde el turno.
	luis := reservation("Luis Devolución", 50)
	charge(luis, "cash")
	small := order(30)
	var luisPayment string
	_ = pool.QueryRow(ctx, `SELECT id FROM payments WHERE reservation_id=$1`, luis).Scan(&luisPayment)
	if rec := call("POST", fmt.Sprintf(`{"orderId":%q,"method":"cash","amount":0,"depositPaymentId":%q}`, small, luisPayment), "", api.createPayment); rec.Code != 409 || !strings.Contains(rec.Body.String(), "deposit_exceeds_remaining") {
		t.Fatalf("deposit above the bill must be rejected: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call("POST", "", luis, api.refundReservationDeposit); rec.Code != 204 {
		t.Fatalf("refund deposit: %d %s", rec.Code, rec.Body.String())
	}
	if status, _, _ := depositOf(luis); status != "refunded" {
		t.Fatalf("refunded deposit status: %s", status)
	}
	var cashOut int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM cash_movements WHERE shift_id=$1 AND source_type='reservation_deposit_refund' AND movement_type='expense' AND amount=50`, shift.ID).Scan(&cashOut)
	if cashOut != 1 {
		t.Fatalf("cash refund must leave the shift once: %d", cashOut)
	}

	// Cancelar con la garantía cobrada exige decidir; retenerla la vuelve venta.
	carla := reservation("Carla Cancela", 40)
	charge(carla, "card")
	if rec := call("PATCH", `{"status":"cancelled"}`, carla, api.updateReservationStatus); rec.Code != 409 || !strings.Contains(rec.Body.String(), "deposit_decision_required") {
		t.Fatalf("cancel must ask what to do with the deposit: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call("PATCH", `{"status":"cancelled","deposit":"retain"}`, carla, api.updateReservationStatus); rec.Code != 204 {
		t.Fatalf("cancel retaining: %d %s", rec.Code, rec.Body.String())
	}
	var item, retainedPaid string
	if err := pool.QueryRow(ctx, `
		SELECT i.name,o.paid_amount::numeric(12,2)::text FROM payments p
		JOIN orders o ON o.id=p.order_id JOIN order_items i ON i.order_id=o.id
		WHERE p.reservation_id=$1 AND p.deposit_outcome='retained' AND o.channel='mostrador' AND o.completed_at IS NOT NULL`, carla).Scan(&item, &retainedPaid); err != nil {
		t.Fatal(err)
	}
	if item != "Garantía de reserva no utilizada" || retainedPaid != "40.00" {
		t.Fatalf("retained deposit order: %q %s", item, retainedPaid)
	}
	if status, _, _ := depositOf(carla); status != "retained" {
		t.Fatalf("retained deposit status: %s", status)
	}

	// No llegó: el barrido la vence, avisa con la garantía y la retiene.
	diego := reservation("Diego No Llegó", 50)
	charge(diego, "cash")
	if _, err := pool.Exec(ctx, `UPDATE reservations SET starts_at=now()-interval '1 hour',tolerance_minutes=0 WHERE id=$1`, diego); err != nil {
		t.Fatal(err)
	}
	if _, err := api.reservations.ExpireOverdue(ctx); err != nil {
		t.Fatal(err)
	}
	var detailText string
	if err := pool.QueryRow(ctx, `SELECT detail FROM staff_notices WHERE organization_id=$1 AND location_id=$2 AND kind='reservation_no_show' AND title LIKE '%Diego%'`, s.OrganizationID, s.LocationID).Scan(&detailText); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(detailText, "Garantía retenida S/ 50.00") {
		t.Fatalf("no-show notice must mention the retained deposit: %q", detailText)
	}
	if _, err := api.payments.RetainForfeitedDeposits(ctx); err != nil {
		t.Fatal(err)
	}
	if status, _, _ := depositOf(diego); status != "retained" {
		t.Fatalf("no-show deposit must be retained: %s", status)
	}
	if again, err := api.payments.RetainForfeitedDeposits(ctx); err != nil || again != 0 {
		t.Fatalf("a retained deposit is retained once: %d %v", again, err)
	}

	// El reporte del turno lista la garantía devuelta (sin pedido) por su nombre.
	var report cashShiftReport
	_ = json.Unmarshal(call("GET", "", shift.ID, api.getCashShiftReport).Body.Bytes(), &report)
	found := false
	for _, payment := range report.Payments {
		found = found || payment.OrderCode == "Garantía de reserva"
	}
	if !found || report.CollectedAmount == "" {
		t.Fatalf("shift report must list order-less deposits: %+v", report.Payments)
	}
}

// Con facturación electrónica activa, la garantía retenida sale con su boleta
// y Reservas muestra su número.
func TestRetainedReservationDepositGetsReceipt(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	grantCashManagementForPaymentTest(t, api, s)
	e := api.handlers.EInvoicing
	call := func(handler http.HandlerFunc, method, body string, path map[string]string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "/", strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
		for key, value := range path {
			r.SetPathValue(key, value)
		}
		w := httptest.NewRecorder()
		handler(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: want %d, got %d %s", method, body, want, w.Code, w.Body.String())
		}
		return w
	}
	var registerID, profileID, reservationID string
	if err := pool.QueryRow(ctx, `INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,'Caja boleta garantía',$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&registerID); err != nil {
		t.Fatal(err)
	}
	call(api.openCashShift, "POST", fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":0}`, registerID), nil, 201)
	if _, err := pool.Exec(ctx, `INSERT INTO organization_modules(organization_id,module_key,active) VALUES($1,'facturacion',true)
		ON CONFLICT (organization_id,module_key) DO UPDATE SET active=true`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE organizations SET tax_id=$2 WHERE id=$1`, s.OrganizationID, testRUC()); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT fiscal_profile_id FROM locations WHERE id=$1`, s.LocationID).Scan(&profileID); err != nil {
		t.Fatal(err)
	}
	call(e.SaveProfile, "PUT", `{"provider":"fake","environment":"test"}`, map[string]string{"id": profileID}, 200)
	call(e.SaveLocation, "PUT", `{"defaultDocument":"receipt","series":{"receipt":"B001","invoice":"F001"}}`, map[string]string{"id": s.LocationID}, 200)

	if err := pool.QueryRow(ctx, `
		INSERT INTO reservations(organization_id,location_id,customer_name,starts_at,guests,status,tolerance_minutes,deposit_amount)
		VALUES($1,$2,'Elena Boleta',now()+interval '1 hour',2,'pending',15,60) RETURNING id`, s.OrganizationID, s.LocationID).Scan(&reservationID); err != nil {
		t.Fatal(err)
	}
	call(api.chargeReservationDeposit, "POST", `{"method":"cash","reference":""}`, map[string]string{"id": reservationID}, 201)
	call(api.updateReservationStatus, "PATCH", `{"status":"cancelled","deposit":"retain"}`, map[string]string{"id": reservationID}, 204)

	var number, status, total string
	if err := pool.QueryRow(ctx, `
		SELECT d.series||'-'||d.number,d.status,d.total::text FROM payments p
		JOIN fiscal_documents d ON d.order_id=p.order_id AND d.organization_id=p.organization_id
		WHERE p.reservation_id=$1 AND d.document_type='receipt'`, reservationID).Scan(&number, &status, &total); err != nil {
		t.Fatal(err)
	}
	if number != "B001-1" || status != "ACCEPTED" || total != "60.00" {
		t.Fatalf("retained deposit receipt: %s %s %s", number, status, total)
	}
	var page struct{ Items []reservationView }
	_ = json.Unmarshal(call(api.listReservations, "GET", "", nil, 200).Body.Bytes(), &page)
	for _, item := range page.Items {
		if item.ID == reservationID && (item.DepositStatus != "retained" || item.DepositDocument != "B001-1") {
			t.Fatalf("reservation must show the retained deposit receipt: %+v", item)
		}
	}
}
