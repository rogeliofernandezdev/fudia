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

// Cocina marca cada plato listo uno por uno: la comanda sigue en preparación
// hasta el último plato, y ese último plato tiene el mismo efecto que «Marcar
// listo» sobre toda la comanda (ronda, pedido y aviso al personal).
func TestKitchenItemsReadyOneByOne(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	nonce := time.Now().UnixNano()

	product := func(name, destination string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control,service_destination)
			VALUES($1,$2,$3,20,true,'prepared','none',$4) RETURNING id
		`, s.OrganizationID, fmt.Sprintf("KIT-%s-%d", destination, nonce)+name[:3], name, destination).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	lomo, ceviche, chicha := product("Lomo saltado", "kitchen"), product("Ceviche", "kitchen"), product("Chicha morada", "bar")

	call := func(method, target, body string, handler http.HandlerFunc, pathValues ...string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, target, bytes.NewReader([]byte(body)))
		for i := 0; i+1 < len(pathValues); i += 2 {
			r.SetPathValue(pathValues[i], pathValues[i+1])
		}
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
		w := httptest.NewRecorder()
		handler(w, r)
		return w
	}
	w := call("POST", "/v1/admin/orders", fmt.Sprintf(`{"channel":"mostrador","customerName":"Rosa","sendToKitchen":true,"items":[{"productId":%q,"qty":2},{"productId":%q,"qty":1},{"productId":%q,"qty":1}]}`, lomo, ceviche, chicha), api.createOrder)
	var order struct {
		ID   string `json:"id"`
		Code string `json:"code"`
	}
	if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &order) != nil {
		t.Fatalf("create order: %d %s", w.Code, w.Body.String())
	}

	type plate struct {
		ID, Name, Qty, Status string
	}
	type ticket struct {
		ID, OrderID, Status string
		Items               []plate
	}
	board := func(destination string) ticket {
		t.Helper()
		w := call("GET", "/v1/admin/kitchen/tickets?destination="+destination, "", api.listKitchenTickets)
		var page struct{ Items []ticket }
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil {
			t.Fatalf("kitchen: %d %s", w.Code, w.Body.String())
		}
		for _, item := range page.Items {
			if item.OrderID == order.ID {
				return item
			}
		}
		t.Fatalf("ticket of %s missing in %s: %s", order.ID, destination, w.Body.String())
		return ticket{}
	}
	byName := func(tk ticket, name string) plate {
		t.Helper()
		for _, p := range tk.Items {
			if p.Name == name {
				return p
			}
		}
		t.Fatalf("plate %s missing: %+v", name, tk)
		return plate{}
	}
	mark := func(ticketID, itemID, status string) *httptest.ResponseRecorder {
		t.Helper()
		return call("PATCH", "/v1/admin/kitchen/tickets/"+ticketID+"/items/"+itemID+"/status", `{"status":"`+status+`"}`, api.updateKitchenItemStatus, "id", ticketID, "itemId", itemID)
	}
	readyNotices := func() int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM staff_notices WHERE order_id=$1 AND kind='order_ready'`, order.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	orderStatus := func() string {
		t.Helper()
		var status string
		if err := pool.QueryRow(ctx, `SELECT status FROM orders WHERE id=$1`, order.ID).Scan(&status); err != nil {
			t.Fatal(err)
		}
		return status
	}

	kitchen := board("kitchen")
	if kitchen.Status != "confirmado" || len(kitchen.Items) != 2 {
		t.Fatalf("kitchen ticket: %+v", kitchen)
	}
	for _, p := range kitchen.Items {
		if p.ID == "" || p.Status != "confirmado" {
			t.Fatalf("each plate exposes its id and status: %+v", kitchen.Items)
		}
	}
	bar := board("bar")
	if !strings.HasSuffix(bar.ID, "~bar") || len(bar.Items) != 1 {
		t.Fatalf("bar ticket: %+v", bar)
	}
	if w = call("PATCH", "/v1/admin/kitchen/tickets/"+kitchen.ID+"/status", `{"status":"preparando"}`, api.updateKitchenTicketStatus, "id", kitchen.ID); w.Code != 204 {
		t.Fatalf("start ticket: %d %s", w.Code, w.Body.String())
	}
	lomoPlate, cevichePlate := byName(kitchen, "Lomo saltado"), byName(kitchen, "Ceviche")

	// Un plato listo: la comanda sigue en preparación y nadie recibe aviso.
	if w = mark(kitchen.ID, lomoPlate.ID, "listo"); w.Code != 204 {
		t.Fatalf("lomo ready: %d %s", w.Code, w.Body.String())
	}
	kitchen = board("kitchen")
	if kitchen.Status != "preparando" || byName(kitchen, "Lomo saltado").Status != "listo" || byName(kitchen, "Lomo saltado").Qty != "2.00" || byName(kitchen, "Ceviche").Status != "preparando" {
		t.Fatalf("partial progress: %+v", kitchen)
	}
	if readyNotices() != 0 || orderStatus() != "preparando" {
		t.Fatalf("one plate must not close the ticket: notices=%d status=%s", readyNotices(), orderStatus())
	}

	// Rechazos: repetir, plato de otra estación, comanda inexistente, estado inválido.
	for _, c := range []struct {
		ticket, item, status string
		code                 int
		errorCode            string
	}{
		{kitchen.ID, lomoPlate.ID, "listo", 409, "kitchen_item_unavailable"},
		{kitchen.ID, bar.Items[0].ID, "listo", 409, "kitchen_item_unavailable"},
		{"00000000-0000-0000-0000-000000000001", lomoPlate.ID, "listo", 404, "kitchen_ticket_not_found"},
		{kitchen.ID, lomoPlate.ID, "entregado", 400, "invalid_kitchen_status"},
	} {
		if w = mark(c.ticket, c.item, c.status); w.Code != c.code || !strings.Contains(w.Body.String(), c.errorCode) {
			t.Fatalf("%s/%s %s: %d %s", c.ticket, c.item, c.status, w.Code, w.Body.String())
		}
	}

	// Deshacer mientras la comanda sigue abierta y volver a marcar.
	if w = mark(kitchen.ID, lomoPlate.ID, "preparando"); w.Code != 204 || byName(board("kitchen"), "Lomo saltado").Status != "preparando" {
		t.Fatalf("undo lomo: %d %s", w.Code, w.Body.String())
	}
	if w = mark(kitchen.ID, lomoPlate.ID, "listo"); w.Code != 204 {
		t.Fatalf("lomo ready again: %d %s", w.Code, w.Body.String())
	}

	// Último plato: igual que «Marcar listo» sobre toda la comanda.
	if w = mark(kitchen.ID, cevichePlate.ID, "listo"); w.Code != 204 {
		t.Fatalf("ceviche ready: %d %s", w.Code, w.Body.String())
	}
	if kitchen = board("kitchen"); kitchen.Status != "listo" {
		t.Fatalf("last plate closes the ticket: %+v", kitchen)
	}
	var detail string
	if err := pool.QueryRow(ctx, `SELECT detail FROM staff_notices WHERE order_id=$1 AND kind='order_ready'`, order.ID).Scan(&detail); err != nil || detail != "Cocina · "+order.Code {
		t.Fatalf("ready notice like the whole ticket: %q %v (count %d)", detail, err, readyNotices())
	}
	// La barra sigue pendiente: el pedido aún no está listo.
	if status := orderStatus(); status != "confirmado" {
		t.Fatalf("order waits for the bar: %s", status)
	}
	if w = mark(kitchen.ID, cevichePlate.ID, "preparando"); w.Code != 409 || !strings.Contains(w.Body.String(), "kitchen_ticket_ready") {
		t.Fatalf("a ready ticket cannot be reopened by plate: %d %s", w.Code, w.Body.String())
	}

	// Barra: un plato confirmado puede marcarse listo directamente.
	if w = mark(bar.ID, bar.Items[0].ID, "listo"); w.Code != 204 {
		t.Fatalf("bar ready: %d %s", w.Code, w.Body.String())
	}
	if board("bar").Status != "listo" || orderStatus() != "listo" || readyNotices() != 2 {
		t.Fatalf("bar closes the order: status=%s notices=%d", orderStatus(), readyNotices())
	}
}
