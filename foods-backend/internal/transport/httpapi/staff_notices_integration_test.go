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

type noticeFeed struct {
	Items []struct {
		ID      string `json:"id"`
		Kind    string `json:"kind"`
		Title   string `json:"title"`
		Detail  string `json:"detail"`
		OrderID string `json:"orderId"`
		TableID string `json:"tableId"`
		Read    bool   `json:"read"`
	} `json:"items"`
	Unread int `json:"unread"`
}

// Campana del personal: los pedidos por QR, la cuenta, el llamado al mozo y
// la comanda lista avisan a quien debe actuar. Cada usuario ve solo las
// audiencias de sus permisos en su local y lleva su propia lectura.
func TestStaffNoticesEndToEnd(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	foreign := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	nonce := time.Now().UnixNano()
	t.Setenv("FUDIA_CONCIERGE_API_KEY", "concierge-test-key")

	if _, err := pool.Exec(ctx, `
		INSERT INTO organization_modules(organization_id,module_key,active) VALUES($1,'whatsapp_bot',true)
		ON CONFLICT(organization_id,module_key) DO UPDATE SET active=true,updated_at=now()
	`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	staff := func(name string, permissions ...string) scope {
		t.Helper()
		var userID string
		if err := pool.QueryRow(ctx, `
			WITH u AS (
			  INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,$2,$3,'test') RETURNING id
			), ro AS (
			  INSERT INTO roles(organization_id,name,permissions,menu_access) VALUES($1,$3,$4,ARRAY['pedidos']) RETURNING id
			), ur AS (
			  INSERT INTO user_roles(user_id,role_id,location_id) SELECT u.id,ro.id,$5 FROM u,ro
			) SELECT id FROM u`, s.OrganizationID, fmt.Sprintf("%s-%d@example.test", strings.ToLower(name), nonce), name+" prueba", permissions, s.LocationID).Scan(&userID); err != nil {
			t.Fatal(err)
		}
		return scope{OrganizationID: s.OrganizationID, LocationID: s.LocationID, UserID: userID, Name: name}
	}
	waiter := staff("Mozo", "orders.read", "orders.manage", "tables.read")
	cashier := staff("Cajero", "orders.read", "cash.read", "cash.manage")
	cook := staff("Cocinero", "orders.read", "kitchen.manage")

	var product string
	if err := pool.QueryRow(ctx, `
		INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control,service_destination)
		VALUES($1,$2,'Lomo saltado',30,true,'prepared','none','kitchen') RETURNING id
	`, s.OrganizationID, fmt.Sprintf("NTC-%d", nonce)).Scan(&product); err != nil {
		t.Fatal(err)
	}

	call := func(owner scope, method, target, body string, handler http.HandlerFunc, pathValues ...string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, target, bytes.NewReader([]byte(body)))
		for i := 0; i+1 < len(pathValues); i += 2 {
			r.SetPathValue(pathValues[i], pathValues[i+1])
		}
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, owner))
		w := httptest.NewRecorder()
		handler(w, r)
		return w
	}
	feed := func(owner scope) noticeFeed {
		t.Helper()
		w := call(owner, "GET", "/v1/admin/notices", "", api.handlers.Notices.Feed)
		var out noticeFeed
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil {
			t.Fatalf("feed: %d %s", w.Code, w.Body.String())
		}
		return out
	}
	kinds := func(f noticeFeed) map[string]int {
		out := map[string]int{}
		for _, item := range f.Items {
			out[item.Kind]++
		}
		return out
	}
	createTable := func(body string) (string, string) {
		t.Helper()
		w := call(s, "POST", "/v1/admin/tables", body, api.createTable)
		var out struct {
			ID      string `json:"id"`
			QrToken string `json:"qrToken"`
		}
		if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &out) != nil {
			t.Fatalf("create table: %d %s", w.Code, w.Body.String())
		}
		return out.ID, out.QrToken
	}
	concierge := func(token, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/v1/integrations/concierge/"+token+path, strings.NewReader(body))
		req.Header.Set("X-Fudia-Concierge-Key", "concierge-test-key")
		rec := httptest.NewRecorder()
		api.Routes().ServeHTTP(rec, req)
		return rec
	}
	orderBody := func(name, conversation, requestID string, qty int) string {
		return fmt.Sprintf(`{"customerName":%q,"customerPhone":"51911111111","conversationId":%q,"requestId":%q,"items":[{"productId":%q,"qty":%d}]}`, name, conversation, requestID, product, qty)
	}

	tableName := fmt.Sprintf("Mesa NTC %d", nonce)
	tableID, tableToken := createTable(fmt.Sprintf(`{"name":%q,"seats":4}`, tableName))
	_, counterToken := createTable(fmt.Sprintf(`{"name":"Mostrador NTC %d","serviceType":"counter"}`, nonce))

	// Pedido por QR de mesa y su reintento: un solo aviso a los mozos.
	for range 2 {
		if rec := concierge(tableToken, "/orders", orderBody("Lucía", "ntc-table", "ntc-r1", 2)); rec.Code != 201 && rec.Code != 200 {
			t.Fatalf("table order: %d %s", rec.Code, rec.Body.String())
		}
	}
	got := feed(waiter)
	if len(got.Items) != 1 || got.Unread != 1 || got.Items[0].Kind != "table_order" || got.Items[0].TableID != tableID ||
		got.Items[0].Title != tableName+" · nuevo pedido" || got.Items[0].Detail != "2 productos · Lucía" {
		t.Fatalf("table order notice: %+v", got)
	}
	// Una ronda adicional es otro aviso.
	if rec := concierge(tableToken, "/orders", orderBody("Lucía", "ntc-table", "ntc-r2", 1)); rec.Code != 200 {
		t.Fatalf("second round: %d %s", rec.Code, rec.Body.String())
	}
	if got = feed(waiter); kinds(got)["table_order"] != 2 {
		t.Fatalf("each round notifies once: %+v", got)
	}
	if got = feed(cashier); kinds(got)["table_order"] != 0 {
		t.Fatalf("cashier must not see table orders: %+v", got)
	}

	// Venta por mostrador: avisa a caja, no a los mozos.
	rec := concierge(counterToken, "/orders", orderBody("Ana", "ntc-counter", "ntc-c1", 1))
	var counterOrder struct {
		ID   string `json:"id"`
		Code string `json:"code"`
	}
	if rec.Code != 201 || json.Unmarshal(rec.Body.Bytes(), &counterOrder) != nil {
		t.Fatalf("counter order: %d %s", rec.Code, rec.Body.String())
	}
	got = feed(cashier)
	if len(got.Items) != 1 || got.Items[0].Kind != "counter_order" || got.Items[0].OrderID != counterOrder.ID ||
		!strings.HasPrefix(got.Items[0].Title, "Nuevo pedido en Mostrador NTC") ||
		got.Items[0].Detail != counterOrder.Code+" · Ana · 1 producto · S/ 30.00" {
		t.Fatalf("counter order notice: %+v", got)
	}
	if kinds(feed(waiter))["counter_order"] != 0 {
		t.Fatal("waiters must not see counter orders")
	}

	// La cuenta avisa a mozos y caja; el reintento no duplica. El llamado al mozo solo a mozos.
	for range 2 {
		if rec = concierge(tableToken, "/bill", `{"conversationId":"ntc-table","requestId":"ntc-bill-1"}`); rec.Code != 200 {
			t.Fatalf("bill: %d %s", rec.Code, rec.Body.String())
		}
	}
	if rec = concierge(tableToken, "/handoffs", `{"conversationId":"ntc-table","reason":"Necesito cubiertos"}`); rec.Code != 200 {
		t.Fatalf("handoff: %d %s", rec.Code, rec.Body.String())
	}
	waiterKinds, cashierKinds := kinds(feed(waiter)), kinds(feed(cashier))
	if waiterKinds["bill_requested"] != 1 || cashierKinds["bill_requested"] != 1 || waiterKinds["assistance_requested"] != 1 || cashierKinds["assistance_requested"] != 0 {
		t.Fatalf("bill/handoff audiences: waiter=%v cashier=%v", waiterKinds, cashierKinds)
	}

	// Cocina marca lista la comanda de mostrador: avisa a mozos y caja.
	w := call(cook, "GET", "/v1/admin/kitchen/tickets?destination=kitchen", "", api.listKitchenTickets)
	var kitchen struct {
		Items []struct {
			ID      string `json:"id"`
			OrderID string `json:"orderId"`
		} `json:"items"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &kitchen) != nil {
		t.Fatalf("kitchen: %d %s", w.Code, w.Body.String())
	}
	ready := 0
	for _, ticket := range kitchen.Items {
		if ticket.OrderID != counterOrder.ID {
			continue
		}
		for _, status := range []string{"preparando", "listo"} {
			if w = call(cook, "PATCH", "/v1/admin/kitchen/tickets/"+ticket.ID+"/status", `{"status":"`+status+`"}`, api.updateKitchenTicketStatus, "id", ticket.ID); w.Code != 204 {
				t.Fatalf("kitchen %s: %d %s", status, w.Code, w.Body.String())
			}
		}
		ready++
	}
	if ready != 1 {
		t.Fatalf("counter ticket not found in kitchen: %+v", kitchen)
	}
	for _, owner := range []scope{waiter, cashier} {
		f := feed(owner)
		if kinds(f)["order_ready"] != 1 || f.Items[0].Title != "Pedido de Ana listo para entregar" || f.Items[0].Detail != "Cocina · "+counterOrder.Code {
			t.Fatalf("ready notice for %s: %+v", owner.Name, f)
		}
	}

	// Sin audiencias (cocina) o de otra empresa: campana vacía.
	for _, owner := range []scope{cook, foreign} {
		if f := feed(owner); len(f.Items) != 0 || f.Unread != 0 {
			t.Fatalf("%s must not see notices: %+v", owner.Name, f)
		}
	}

	// Lectura por usuario: leer un aviso compartido no lo marca para el otro.
	got = feed(waiter)
	before := got.Unread
	shared := got.Items[0].ID
	if w = call(waiter, "POST", "/v1/admin/notices/"+shared+"/read", "", api.handlers.Notices.MarkRead, "id", shared); w.Code != 204 {
		t.Fatalf("mark read: %d %s", w.Code, w.Body.String())
	}
	if got = feed(waiter); got.Unread != before-1 || !got.Items[0].Read {
		t.Fatalf("waiter read state: %+v", got)
	}
	if f := feed(cashier); f.Items[0].ID != shared || f.Items[0].Read {
		t.Fatalf("cashier keeps the shared notice unread: %+v", f)
	}
	// Un aviso que el usuario no ve (o de otra empresa) no existe para él.
	counterNotice := feed(cashier).Items[len(feed(cashier).Items)-1].ID
	for _, owner := range []scope{waiter, foreign} {
		if w = call(owner, "POST", "/v1/admin/notices/"+counterNotice+"/read", "", api.handlers.Notices.MarkRead, "id", counterNotice); w.Code != 404 {
			t.Fatalf("%s must not read a hidden notice: %d %s", owner.Name, w.Code, w.Body.String())
		}
	}
	if w = call(waiter, "POST", "/v1/admin/notices/read", "", api.handlers.Notices.MarkAllRead); w.Code != 204 {
		t.Fatalf("mark all read: %d %s", w.Code, w.Body.String())
	}
	if got = feed(waiter); got.Unread != 0 || feed(cashier).Unread == 0 {
		t.Fatalf("mark all read is per user: waiter=%+v", got)
	}
}
