package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Delivery Fudia: con la empresa activada, un pedido confirmado se ofrece a las
// centrales en el orden global; un rechazo o 3 minutos sin respuesta pasan a la siguiente; sin
// centrales queda sin asignar con aviso a caja; «Reintentar» abre otra vuelta;
// la central que acepta recibe «listo para recoger» y «Recoger» lo pone en camino.
func TestDeliveryFudiaPartnerCascade(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE locations SET district_code='150101' WHERE id=$1`, s.LocationID); err != nil {
		t.Fatal(err)
	}
	// Teléfonos únicos por ejecución: 51 + 9 + 8 dígitos del nonce + índice.
	prefix := fmt.Sprintf("519%08d", time.Now().UnixNano()%100000000)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM organization_delivery_settings WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM delivery_dispatches WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM delivery_partners WHERE whatsapp_phone LIKE $1`, prefix+"%")
	})
	d := api.handlers.Delivery
	call := func(handler func(http.ResponseWriter, *http.Request), method, body, id string, subject scope) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/", bytes.NewReader([]byte(body)))
		if id != "" {
			req.SetPathValue("id", id)
		}
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, subject))
		rec := httptest.NewRecorder()
		handler(rec, req)
		return rec
	}
	decode := func(rec *httptest.ResponseRecorder, out any) {
		t.Helper()
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("decode %s: %v", rec.Body.String(), err)
		}
	}
	phone := func(i int) string { return fmt.Sprintf("%s%d", prefix, i) }
	partner := func(name, number string) string {
		rec := call(d.CreatePartner, "POST", fmt.Sprintf(`{"name":%q,"whatsappPhone":"+%s","coverageCodes":["150101"]}`, name, number), "", s)
		if rec.Code != 201 {
			t.Fatalf("create partner: %d %s", rec.Code, rec.Body.String())
		}
		var out struct{ ID, WhatsappPhone string }
		decode(rec, &out)
		if out.WhatsappPhone != number {
			t.Fatalf("phone not normalized: %s", out.WhatsappPhone)
		}
		return out.ID
	}
	first, second := partner("Central Norte", phone(1)), partner("Central Sur", phone(2))
	for _, code := range []string{"15", "1501", "999999"} {
		rec := call(d.CreatePartner, "POST", fmt.Sprintf(`{"name":"Cobertura inválida","whatsappPhone":%q,"coverageCodes":[%q]}`, phone(3), code), "", s)
		if rec.Code != 400 {
			t.Fatalf("only catalog districts can be checked: %s: %d %s", code, rec.Code, rec.Body.String())
		}
	}
	if rec := call(d.CreatePartner, "POST", fmt.Sprintf(`{"name":"Copia","whatsappPhone":%q,"coverageCodes":["150101"]}`, phone(1)), "", s); rec.Code != 409 {
		t.Fatalf("duplicate phone: %d %s", rec.Code, rec.Body.String())
	}
	// El orden es global: la base de prueba solo conserva activas las centrales de esta ejecución.
	if _, err := pool.Exec(ctx, `UPDATE delivery_partners SET active=false WHERE whatsapp_phone NOT LIKE $1`, prefix+"%"); err != nil {
		t.Fatal(err)
	}
	all := call(d.ListPartners, "GET", "", "", s)
	var listed struct{ Items []struct{ ID string } }
	decode(all, &listed)
	order := []string{first, second}
	for _, item := range listed.Items {
		if item.ID != first && item.ID != second {
			order = append(order, item.ID)
		}
	}
	orderBody, _ := json.Marshal(map[string][]string{"partnerIds": order})
	if rec := call(d.ReorderPartners, "PUT", `{"partnerIds":[]}`, "", s); rec.Code != 400 {
		t.Fatalf("empty order: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(d.ReorderPartners, "PUT", fmt.Sprintf(`{"partnerIds":[%q]}`, first), "", s); rec.Code != 409 {
		t.Fatalf("partial order: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(d.ReorderPartners, "PUT", string(orderBody), "", s); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"priority":2`) {
		t.Fatalf("reorder: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(d.OrganizationDelivery, "GET", "", s.OrganizationID, s); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"enabled":false`) {
		t.Fatalf("delivery off by default: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(d.SetOrganizationDelivery, "PUT", `{"enabled":true}`, s.OrganizationID, s); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"enabled":true`) {
		t.Fatalf("enable delivery: %d %s", rec.Code, rec.Body.String())
	}

	var productID string
	if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control) VALUES($1,'FUDIA-DELIVERY','Lomo saltado',30,true,'prepared','none') RETURNING id`, s.OrganizationID).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	createOrder := func() string {
		body := fmt.Sprintf(`{"channel":"delivery","customerName":"Ana","customerPhone":"999123456","address":"Av. Perú 100","reference":"Puerta azul","deliveryFee":5,"sendToKitchen":true,"items":[{"productId":%q,"qty":2,"note":"Sin ají"}]}`, productID)
		req := httptest.NewRequest("POST", "/v1/admin/orders", bytes.NewReader([]byte(body)))
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		api.createOrder(rec, req)
		if rec.Code != 201 {
			t.Fatalf("create delivery: %d %s", rec.Code, rec.Body.String())
		}
		var out struct{ ID string }
		decode(rec, &out)
		return out.ID
	}
	type message struct {
		ID, Kind, PartnerName, PartnerPhone, ActionToken, OrderCode, Total string
		ExpiresInSeconds                                                   int
		Items                                                              []struct {
			Quantity float64
			Name     string
			Note     string
		}
	}
	claim := func() []message {
		rec := call(d.ClaimMessages, "GET", "", "", scope{})
		if rec.Code != 200 {
			t.Fatalf("claim: %d %s", rec.Code, rec.Body.String())
		}
		var out struct{ Items []message }
		decode(rec, &out)
		mine := []message{}
		for _, m := range out.Items {
			if strings.HasPrefix(m.PartnerPhone, prefix) {
				mine = append(mine, m)
				if done := call(d.MarkMessageDispatched, "POST", "", m.ID, scope{}); done.Code != 204 {
					t.Fatalf("dispatched: %d %s", done.Code, done.Body.String())
				}
			}
		}
		return mine
	}
	act := func(action, token, from string) string {
		rec := call(d.PartnerAction, "POST", fmt.Sprintf(`{"action":%q,"token":%q,"phone":%q}`, action, token, from), "", scope{})
		if rec.Code != 200 {
			t.Fatalf("%s: %d %s", action, rec.Code, rec.Body.String())
		}
		var out struct{ Outcome string }
		decode(rec, &out)
		return out.Outcome
	}
	dispatchStatus := func(orderID string) string {
		var status string
		if err := pool.QueryRow(ctx, `SELECT status FROM delivery_dispatches WHERE order_id=$1`, orderID).Scan(&status); err != nil {
			t.Fatalf("dispatch of %s: %v", orderID, err)
		}
		return status
	}
	orderStatus := func(orderID string) string {
		var status string
		if err := pool.QueryRow(ctx, `SELECT status FROM orders WHERE id=$1`, orderID).Scan(&status); err != nil {
			t.Fatal(err)
		}
		return status
	}

	orderID := createOrder()
	offers := claim()
	if len(offers) != 1 || offers[0].Kind != "offer" || offers[0].PartnerName != "Central Norte" || offers[0].ExpiresInSeconds != 180 ||
		offers[0].Total != "65.00" || len(offers[0].Items) != 1 || offers[0].Items[0].Note != "Sin ají" {
		t.Fatalf("first offer: %#v", offers)
	}
	firstToken := offers[0].ActionToken
	if got := act("reject", firstToken, phone(2)); got != "not_found" {
		t.Fatalf("another phone answered: %s", got)
	}
	if got := act("reject", firstToken, "+"+phone(1)); got != "rejected" {
		t.Fatalf("reject: %s", got)
	}
	offers = claim()
	if len(offers) != 1 || offers[0].PartnerName != "Central Sur" {
		t.Fatalf("cascade to second: %#v", offers)
	}
	secondToken := offers[0].ActionToken
	if got := act("accept", firstToken, phone(1)); got != "taken" {
		t.Fatalf("late accept after reject: %s", got)
	}
	// La 2.ª central no responde en 3 minutos: sin más centrales, queda sin asignar.
	if _, err := pool.Exec(ctx, `UPDATE delivery_dispatch_offers SET expires_at=now()-interval '1 second' WHERE action_token=$1`, secondToken); err != nil {
		t.Fatal(err)
	}
	if err := api.dispatcher.Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	if got := dispatchStatus(orderID); got != "unassigned" {
		t.Fatalf("after expiry: %s", got)
	}
	var notices int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM staff_notices WHERE order_id=$1 AND kind='delivery_unassigned'`, orderID).Scan(&notices); err != nil || notices != 1 {
		t.Fatalf("unassigned notice: %d %v", notices, err)
	}
	if got := act("accept", secondToken, phone(2)); got != "expired" {
		t.Fatalf("accept after expiry: %s", got)
	}

	retry := call(d.RetryDispatch, "POST", "", orderID, s)
	if retry.Code != 200 || !strings.Contains(retry.Body.String(), `"round":2`) {
		t.Fatalf("retry: %d %s", retry.Code, retry.Body.String())
	}
	offers = claim()
	if len(offers) != 1 || offers[0].PartnerName != "Central Norte" {
		t.Fatalf("retry offer: %#v", offers)
	}
	acceptedToken := offers[0].ActionToken
	if got := act("accept", acceptedToken, phone(1)); got != "accepted" {
		t.Fatalf("accept: %s", got)
	}
	if got := act("accept", acceptedToken, phone(1)); got != "already_accepted" {
		t.Fatalf("double accept: %s", got)
	}
	money := func(orderID string) (total, fee string) {
		if err := pool.QueryRow(ctx, `SELECT total::text,delivery_fee::text FROM orders WHERE id=$1`, orderID).Scan(&total, &fee); err != nil {
			t.Fatal(err)
		}
		return total, fee
	}
	// La central paga al local solo el pedido: el envío sale del total del restaurante.
	if total, fee := money(orderID); total != "60.00" || fee != "0.00" {
		t.Fatalf("fee must move to the central: total %s fee %s", total, fee)
	}
	if got := act("pickup", acceptedToken, phone(1)); got != "not_ready" {
		t.Fatalf("pickup before kitchen: %s", got)
	}
	var detail struct{ DeliveryDispatchStatus, DeliveryPartnerName, DeliveryPartnerFee, Total, RemainingAmount string }
	getReq := httptest.NewRequest("GET", "/v1/admin/orders/"+orderID, nil)
	getReq.SetPathValue("id", orderID)
	getReq = getReq.WithContext(context.WithValue(getReq.Context(), scopeKey{}, s))
	getRec := httptest.NewRecorder()
	api.getOrder(getRec, getReq)
	decode(getRec, &detail)
	if detail.DeliveryDispatchStatus != "accepted" || detail.DeliveryPartnerName != "Central Norte" ||
		detail.DeliveryPartnerFee != "5.00" || detail.Total != "60.00" {
		t.Fatalf("order read model: %#v", detail)
	}

	for _, status := range []string{"preparando", "listo"} {
		req := httptest.NewRequest("PATCH", "/", bytes.NewReader([]byte(fmt.Sprintf(`{"status":%q}`, status))))
		req.SetPathValue("id", orderID)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		api.updateKitchenTicketStatus(rec, req)
		if rec.Code != 204 {
			t.Fatalf("kitchen %s: %d %s", status, rec.Code, rec.Body.String())
		}
	}
	ready := claim()
	if len(ready) != 1 || ready[0].Kind != "ready" || ready[0].ActionToken != acceptedToken || ready[0].ExpiresInSeconds != 0 || ready[0].Total != "65.00" {
		t.Fatalf("ready message: %#v", ready)
	}
	if got := act("pickup", acceptedToken, phone(1)); got != "picked_up" {
		t.Fatalf("pickup: %s", got)
	}
	if got := act("pickup", acceptedToken, phone(1)); got != "picked_up" {
		t.Fatalf("double pickup: %s", got)
	}
	if dispatchStatus(orderID) != "picked_up" || orderStatus(orderID) != "en_camino" {
		t.Fatalf("after pickup: %s %s", dispatchStatus(orderID), orderStatus(orderID))
	}
	history := call(d.OrderDispatch, "GET", "", orderID, s)
	if history.Code != 200 || strings.Count(history.Body.String(), `"partnerName"`) != 4 {
		t.Fatalf("history: %d %s", history.Code, history.Body.String())
	}

	// Delivery propio: el local retira la oferta y la central ya no la recibe.
	ownID := createOrder()
	if own := call(d.UseOwnDelivery, "POST", "", ownID, s); own.Code != 200 || !strings.Contains(own.Body.String(), `"status":"own"`) {
		t.Fatalf("use own: %d %s", own.Code, own.Body.String())
	}
	if late := claim(); len(late) != 0 {
		t.Fatalf("withdrawn offer still sent: %#v", late)
	}

	// Aceptado y luego delivery propio: el envío vuelve al pedido del local.
	backID := createOrder()
	offers = claim()
	if len(offers) != 1 || act("accept", offers[0].ActionToken, phone(1)) != "accepted" {
		t.Fatalf("accept before own: %#v", offers)
	}
	if total, fee := money(backID); total != "60.00" || fee != "0.00" {
		t.Fatalf("released fee: %s %s", total, fee)
	}
	if own := call(d.UseOwnDelivery, "POST", "", backID, s); own.Code != 200 {
		t.Fatalf("own after accept: %d %s", own.Code, own.Body.String())
	}
	if total, fee := money(backID); total != "65.00" || fee != "5.00" {
		t.Fatalf("restored fee: %s %s", total, fee)
	}
	if notice := claim(); len(notice) != 1 || notice[0].Kind != "cancelled" {
		t.Fatalf("central must be told: %#v", notice)
	}

	t.Run("stale_buttons_cannot_take_reassigned_order", func(t *testing.T) {
		oldToken := offers[0].ActionToken
		if got := act("accept", oldToken, phone(1)); got != "taken" {
			t.Fatalf("old acceptance after own: %s", got)
		}
		if rec := call(d.RetryDispatch, "POST", "", backID, s); rec.Code != 200 {
			t.Fatal(rec.Body.String())
		}
		next := claim()
		if len(next) != 1 {
			t.Fatalf("retry: %#v", next)
		}
		act("reject", next[0].ActionToken, phone(1))
		next = claim()
		if len(next) != 1 {
			t.Fatalf("second: %#v", next)
		}
		if got := act("accept", next[0].ActionToken, phone(2)); got != "accepted" {
			t.Fatal(got)
		}
		for _, status := range []string{"preparando", "listo"} {
			rec := call(api.updateKitchenTicketStatus, "PATCH", fmt.Sprintf(`{"status":%q}`, status), backID, s)
			if rec.Code != 204 {
				t.Fatal(rec.Body.String())
			}
		}
		claim()
		if got := act("pickup", oldToken, phone(1)); got != "taken" {
			t.Fatalf("old central picked up: %s", got)
		}
		if orderStatus(backID) != "listo" || dispatchStatus(backID) != "accepted" {
			t.Fatal("stale button changed order")
		}
		if got := act("pickup", next[0].ActionToken, phone(2)); got != "picked_up" {
			t.Fatalf("current central: %s", got)
		}
	})

	t.Run("failed_fee_transfer_rolls_back_acceptance", func(t *testing.T) {
		id := createOrder()
		messages := claim()
		if len(messages) != 1 {
			t.Fatalf("offer: %#v", messages)
		}
		_, err := pool.Exec(ctx, fmt.Sprintf(`CREATE FUNCTION delivery_fee_review_failure() RETURNS trigger LANGUAGE plpgsql AS $test$ BEGIN IF NEW.id='%s'::uuid AND OLD.delivery_fee>0 AND NEW.delivery_fee=0 THEN RAISE EXCEPTION 'simulated fee failure'; END IF; RETURN NEW; END $test$;
   CREATE TRIGGER delivery_fee_review_failure BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION delivery_fee_review_failure();`, id))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DROP TRIGGER IF EXISTS delivery_fee_review_failure ON orders; DROP FUNCTION IF EXISTS delivery_fee_review_failure();`)
		})
		rec := call(d.PartnerAction, "POST", fmt.Sprintf(`{"action":"accept","token":%q,"phone":%q}`, messages[0].ActionToken, phone(1)), "", scope{})
		if rec.Code != 503 {
			t.Fatalf("failed money transfer accepted: %d %s", rec.Code, rec.Body.String())
		}
		var feeReleased bool
		var offerStatus string
		if err := pool.QueryRow(ctx, `SELECT d.fee_released,f.status FROM delivery_dispatches d JOIN delivery_dispatch_offers f ON f.dispatch_id=d.id WHERE d.order_id=$1`, id).Scan(&feeReleased, &offerStatus); err != nil {
			t.Fatal(err)
		}
		total, fee := money(id)
		if dispatchStatus(id) != "searching" || offerStatus != "pending" || feeReleased || total != "65.00" || fee != "5.00" {
			t.Fatalf("non-atomic acceptance: %s %s %v %s %s", dispatchStatus(id), offerStatus, feeReleased, total, fee)
		}
		call(d.UseOwnDelivery, "POST", "", id, s)
	})

	t.Run("paid_orders_are_not_assigned_or_discounted", func(t *testing.T) {
		grantCashManagementForPaymentTest(t, api, s)
		var registerID string
		if err := pool.QueryRow(ctx, `INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,'Caja reparto',$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&registerID); err != nil {
			t.Fatal(err)
		}
		if rec := call(api.openCashShift, "POST", fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":0}`, registerID), "", s); rec.Code != 201 {
			t.Fatal(rec.Body.String())
		}
		for _, amount := range []int{10, 65} {
			id := createOrder()
			messages := claim()
			if len(messages) != 1 {
				t.Fatalf("offer: %#v", messages)
			}
			for _, status := range []string{"preparando", "listo"} {
				rec := call(api.updateKitchenTicketStatus, "PATCH", fmt.Sprintf(`{"status":%q}`, status), id, s)
				if rec.Code != 204 {
					t.Fatal(rec.Body.String())
				}
			}
			if rec := call(api.createPayment, "POST", fmt.Sprintf(`{"orderId":%q,"method":"card","amount":%d}`, id, amount), "", s); rec.Code != 201 {
				t.Fatal(rec.Body.String())
			}
			if got := act("accept", messages[0].ActionToken, phone(1)); got != "payment_conflict" {
				t.Fatalf("paid acceptance: %s", got)
			}
			total, fee := money(id)
			var released bool
			if err := pool.QueryRow(ctx, `SELECT fee_released FROM delivery_dispatches WHERE order_id=$1`, id).Scan(&released); err != nil {
				t.Fatal(err)
			}
			if total != "65.00" || fee != "5.00" || released || dispatchStatus(id) != "own" {
				t.Fatalf("money diverged: total=%s fee=%s released=%v dispatch=%s", total, fee, released, dispatchStatus(id))
			}
			if messages := claim(); len(messages) != 0 {
				t.Fatalf("paid order still offered: %#v", messages)
			}
			if rec := call(d.RetryDispatch, "POST", "", id, s); rec.Code != 409 {
				t.Fatalf("retry paid order: %d", rec.Code)
			}
		}
	})
	updatePartner := func(id string, active bool, codes string) {
		number := phone(1)
		name := "Central Norte"
		if id == second {
			number = phone(2)
			name = "Central Sur"
		}
		rec := call(d.UpdatePartner, "PATCH", fmt.Sprintf(`{"name":%q,"whatsappPhone":%q,"active":%t,"coverageCodes":%s}`, name, number, active, codes), id, s)
		if rec.Code != 200 {
			t.Fatalf("update: %d %s", rec.Code, rec.Body.String())
		}
	}
	t.Run("deactivation_withdraws_pending_offer_and_cascades", func(t *testing.T) {
		id := createOrder()
		var old string
		if err := pool.QueryRow(ctx, `SELECT f.action_token::text FROM delivery_dispatch_offers f JOIN delivery_dispatches d ON d.id=f.dispatch_id WHERE d.order_id=$1 AND f.status='pending'`, id).Scan(&old); err != nil {
			t.Fatal(err)
		}
		updatePartner(first, false, `["150101"]`)
		messages := claim()
		if len(messages) != 1 || messages[0].PartnerPhone != phone(2) {
			t.Fatalf("inactive central still receives order: %#v", messages)
		}
		if got := act("accept", old, phone(1)); got != "taken" {
			t.Fatalf("inactive button: %s", got)
		}
		call(d.UseOwnDelivery, "POST", "", id, s)
		claim()
		updatePartner(first, true, `["150101"]`)
	})
	t.Run("coverage_matches_only_checked_districts", func(t *testing.T) {
		updatePartner(first, true, `["160101"]`)
		updatePartner(second, true, `["150101"]`)
		id := createOrder()
		messages := claim()
		if len(messages) != 1 || messages[0].PartnerPhone != phone(2) {
			t.Fatalf("checked district match: %#v", messages)
		}
		call(d.UseOwnDelivery, "POST", "", id, s)
		updatePartner(second, true, `["150102"]`)
		unmatched := createOrder()
		if dispatchStatus(unmatched) != "unassigned" || len(claim()) != 0 {
			t.Fatal("wrong district received order")
		}
		updatePartner(first, true, `["150201","150101"]`)
		matched := createOrder()
		messages = claim()
		if len(messages) != 1 || messages[0].PartnerPhone != phone(1) {
			t.Fatalf("district match: %#v", messages)
		}
		call(d.UseOwnDelivery, "POST", "", matched, s)
		updatePartner(first, true, `["150101"]`)
		updatePartner(second, true, `["150101"]`)
		if _, err := pool.Exec(ctx, `UPDATE locations SET district_code=NULL WHERE id=$1`, s.LocationID); err != nil {
			t.Fatal(err)
		}
		missing := createOrder()
		if dispatchStatus(missing) != "unassigned" || len(claim()) != 0 {
			t.Fatal("missing district treated as national coverage")
		}
		if _, err := pool.Exec(ctx, `UPDATE locations SET district_code='150101' WHERE id=$1`, s.LocationID); err != nil {
			t.Fatal(err)
		}
	})

	// Cancelado después de aceptar: la central recibe el aviso de cancelación.
	cancelID := createOrder()
	offers = claim()
	if len(offers) != 1 || act("accept", offers[0].ActionToken, phone(1)) != "accepted" {
		t.Fatalf("accept before cancel: %#v", offers)
	}
	req := httptest.NewRequest("PATCH", "/", bytes.NewReader([]byte(`{"status":"cancelado"}`)))
	req.SetPathValue("id", cancelID)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.updateOrderStatus(rec, req)
	if rec.Code != 200 {
		t.Fatalf("cancel: %d %s", rec.Code, rec.Body.String())
	}
	cancelled := claim()
	if dispatchStatus(cancelID) != "cancelled" || len(cancelled) != 1 || cancelled[0].Kind != "cancelled" {
		t.Fatalf("cancel notice: %s %#v", dispatchStatus(cancelID), cancelled)
	}

	// Otra empresa no ve ni reintenta el despacho.
	foreign := seedInventoryScope(t, pool)
	if hidden := call(d.OrderDispatch, "GET", "", orderID, foreign); hidden.Code != 404 {
		t.Fatalf("foreign dispatch visible: %d %s", hidden.Code, hidden.Body.String())
	}
	if missing := call(d.OrganizationDelivery, "GET", "", "00000000-0000-0000-0000-000000000000", s); missing.Code != 404 {
		t.Fatalf("unknown company: %d %s", missing.Code, missing.Body.String())
	}

	// Sin Delivery Fudia activo en la empresa, un pedido nuevo no se ofrece a centrales.
	if rec := call(d.SetOrganizationDelivery, "PUT", `{"enabled":false}`, s.OrganizationID, s); rec.Code != 200 {
		t.Fatalf("disable delivery: %d %s", rec.Code, rec.Body.String())
	}
	offID := createOrder()
	var dispatches int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM delivery_dispatches WHERE order_id=$1`, offID).Scan(&dispatches); err != nil || dispatches != 0 {
		t.Fatalf("disabled company dispatched: %d %v", dispatches, err)
	}
}
