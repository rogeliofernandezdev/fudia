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

func TestTableAccountPreparationAppendPaymentAndAutomaticRelease(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	outsider := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	seed := func(query string, args ...any) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, query, args...).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	other := s
	other.UserID = seed(`INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,'other@flow.test','Otro mozo','test') RETURNING id`, s.OrganizationID)
	otherLocal := s
	otherLocal.LocationID = seed(`INSERT INTO locations(organization_id,name,code) VALUES($1,'Segundo','FLOW2') RETURNING id`, s.OrganizationID)
	table := seed(`INSERT INTO tables(organization_id,location_id,name,seats,zone,qr_token) VALUES($1,$2,'Mesa Flujo',4,'Principal',encode(gen_random_bytes(16),'hex')) RETURNING id`, s.OrganizationID, s.LocationID)
	products := map[string]string{}
	for _, dest := range []string{"kitchen", "bar", "direct"} {
		products[dest] = seed(`INSERT INTO products(organization_id,sku,name,price,product_type,quantity_control,service_destination) VALUES($1,$2,$2,10,'prepared','none',$2) RETURNING id`, s.OrganizationID, dest)
	}
	call := func(sc scope, method, id, item, body string, h http.HandlerFunc, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "/", strings.NewReader(body))
		r.SetPathValue("id", id)
		r.SetPathValue("itemId", item)
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, sc))
		w := httptest.NewRecorder()
		h(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: %d want %d: %s", method, id, w.Code, want, w.Body.String())
		}
		return w
	}
	read := func(id string) order {
		t.Helper()
		w := call(s, "GET", id, "", "", api.getOrder, 200)
		var o order
		if err := json.Unmarshal(w.Body.Bytes(), &o); err != nil {
			t.Fatal(err)
		}
		return o
	}
	w := call(s, "POST", "", "", fmt.Sprintf(`{"channel":"salon","tableId":%q,"sendToKitchen":true,"items":[{"productId":%q,"qty":1},{"productId":%q,"qty":1},{"productId":%q,"qty":1}]}`, table, products["kitchen"], products["bar"], products["direct"]), api.createOrder, 201)
	var created order
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	id := created.ID
	o := read(id)
	if o.WaiterID != s.UserID || len(o.ServiceItems) != 3 || o.AccountState != "open" {
		t.Fatalf("assignment/snapshot: %#v", o)
	}
	task := func(dest string) string {
		t.Helper()
		for _, item := range o.ServiceItems {
			if item.Destination == dest {
				return item.ID
			}
		}
		t.Fatal("missing destination " + dest)
		return ""
	}
	direct := task("direct")
	kitchen := task("kitchen")
	bar := task("bar")
	ticketsRequest := httptest.NewRequest("GET", "/", nil).WithContext(context.WithValue(ctx, scopeKey{}, s))
	tickets, err := api.serviceKitchenTickets(ticketsRequest, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(tickets) != 2 {
		t.Fatalf("direct must not enter KDS: %#v", tickets)
	}
	for _, ticket := range tickets {
		for _, item := range ticket.Items {
			if item.ProductID == products["direct"] {
				t.Fatal("direct in KDS")
			}
		}
	}
	register := seed(`INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,'Caja Flujo',$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID)
	call(s, "POST", "", "", fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":0}`, register), api.openCashShift, 201)
	payment := func(amount int, want int) {
		t.Helper()
		call(s, "POST", "", "", fmt.Sprintf(`{"orderId":%q,"method":"card","amount":%d}`, id, amount), api.createPayment, want)
	}
	payment(10, 409) // an open bill is not chargeable
	call(s, "POST", id, "", "", api.closeOrderBill, 409)
	// Never trust a stale aggregate status over the individual product states.
	if _, err := pool.Exec(ctx, `UPDATE orders SET status='entregado',bill_closed_at=now() WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	call(s, "POST", id, "", "", api.closeOrderBill, 409)
	payment(10, 409)
	call(s, "POST", "", "", fmt.Sprintf(`{"orderId":%q,"payments":[{"method":"cash","amount":5},{"method":"card","amount":5}]}`, id), api.createPaymentBatch, 409)
	var prematurePayments int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM payments WHERE order_id=$1`, id).Scan(&prematurePayments); err != nil || prematurePayments != 0 {
		t.Fatalf("undelivered products must block all payment paths: %d %v", prematurePayments, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE orders SET status='confirmado',bill_closed_at=NULL WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	for _, sc := range []scope{other, outsider, otherLocal} {
		want := 404
		if sc.UserID == other.UserID {
			want = 403
		}
		call(sc, "POST", id, "", `{"requestKey":"blocked","items":[]}`, api.closeOrderBill, want)
		call(sc, "PATCH", id, direct, "", api.deliverOrderServiceItem, want)
	}
	call(s, "PATCH", id, direct, "", api.deliverOrderServiceItem, 200)
	call(s, "PATCH", id, kitchen, "", api.deliverOrderServiceItem, 409)
	call(s, "PATCH", id, "", `{"status":"preparando"}`, api.updateKitchenTicketStatus, 204)
	call(s, "POST", id, "", "", api.closeOrderBill, 409)
	o = read(id)
	for _, item := range o.ServiceItems {
		if item.ID == bar && item.Status != "confirmado" {
			t.Fatal("kitchen changed bar")
		}
	}
	call(s, "PATCH", id, "", `{"status":"listo"}`, api.updateKitchenTicketStatus, 204)
	call(s, "PATCH", id, kitchen, "", api.deliverOrderServiceItem, 200)
	call(s, "POST", id, "", "", api.closeOrderBill, 409) // bar still pending
	call(s, "PATCH", id+"~bar", "", `{"status":"preparando"}`, api.updateKitchenTicketStatus, 204)
	call(s, "PATCH", id+"~bar", "", `{"status":"listo"}`, api.updateKitchenTicketStatus, 204)
	call(s, "PATCH", id, bar, "", api.deliverOrderServiceItem, 200)
	o = read(id)
	if o.Status != "entregado" || o.CompletedAt != "" {
		t.Fatal("delivery closed an open account")
	}
	appendBody := fmt.Sprintf(`{"requestKey":"additional-1","items":[{"productId":%q,"qty":2}]}`, products["direct"])
	call(other, "POST", id, "", appendBody, api.appendOrderItems, 403)
	call(s, "POST", id, "", appendBody, api.appendOrderItems, 200)
	call(s, "POST", id, "", appendBody, api.appendOrderItems, 200) // retry never duplicates stock/lines
	o = read(id)
	if len(o.Items) != 4 || len(o.ServiceItems) != 4 || o.Total != "50.00" {
		t.Fatalf("append not idempotent: %#v", o)
	}
	for _, item := range o.ServiceItems {
		if item.ID == direct || item.ID == bar || item.ID == kitchen {
			if item.Status != "entregado" {
				t.Fatal("old consumption resent")
			}
		}
	}
	call(s, "POST", id, "", "", api.closeOrderBill, 409) // new direct consumption must be served
	call(s, "POST", id, "", fmt.Sprintf(`{"requestKey":"additional-2","items":[{"productId":%q,"qty":1}]}`, products["kitchen"]), api.appendOrderItems, 200)
	o = read(id)
	if o.Total != "60.00" || o.RemainingAmount != "60.00" || o.AccountState != "open" {
		t.Fatalf("append to open account: %#v", o)
	}
	call(s, "POST", id, "", "", api.closeOrderBill, 409)
	payment(60, 409)
	call(s, "POST", "", "", fmt.Sprintf(`{"channel":"salon","tableId":%q,"sendToKitchen":true,"items":[{"productId":%q,"qty":1}]}`, table, products["direct"]), api.createOrder, 409)
	tickets, err = api.serviceKitchenTickets(ticketsRequest, "", "kitchen")
	if err != nil || len(tickets) != 1 {
		t.Fatalf("new kitchen round only: %v %#v", err, tickets)
	}
	call(s, "PATCH", tickets[0].ID, "", `{"status":"preparando"}`, api.updateKitchenTicketStatus, 204)
	call(s, "PATCH", tickets[0].ID, "", `{"status":"listo"}`, api.updateKitchenTicketStatus, 204)
	call(s, "POST", id, "", "", api.closeOrderBill, 409) // ready is not delivered
	o = read(id)
	for _, item := range o.ServiceItems {
		if item.Status == "listo" {
			call(s, "PATCH", id, item.ID, "", api.deliverOrderServiceItem, 200)
		}
	}
	o = read(id)
	if o.CompletedAt != "" || o.Status != "entregado" || o.AccountState != "open" {
		t.Fatalf("delivery alone must not close the bill or release the table: %#v", o)
	}
	call(s, "POST", id, "", "", api.closeOrderBill, 200)
	payment(20, 201)
	o = read(id)
	if o.RemainingAmount != "40.00" || o.AccountState != "awaiting_payment" || o.CompletedAt != "" {
		t.Fatalf("partial payment must keep table occupied: %#v", o)
	}
	call(s, "POST", id, "", strings.Replace(appendBody, "additional-1", "after-close", 1), api.appendOrderItems, 409)
	call(s, "POST", id, "", "", api.closeOrderBill, 200) // idempotent close
	payment(40, 201)
	o = read(id)
	if o.CompletedAt == "" || o.Status != "entregado" {
		t.Fatal("closed, paid, delivered must release automatically")
	}
	call(s, "POST", id, "", appendBody, api.appendOrderItems, 409)
	var events int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE organization_id=$1 AND entity_id=$2 AND action='order.completed'`, s.OrganizationID, id).Scan(&events); err != nil || events != 1 {
		t.Fatalf("automatic finalization audit: %d %v", events, err)
	}
	call(other, "POST", "", "", fmt.Sprintf(`{"channel":"salon","tableId":%q,"sendToKitchen":true,"items":[{"productId":%q,"qty":1}]}`, table, products["direct"]), api.createOrder, 201)
}
