package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Los pedidos de Fudia Concierge quedan guardados (como «Guardar comanda»): Cocina
// no los ve hasta que el mozo pulsa «Enviar comanda». Quien la envía queda como
// responsable de la mesa, cada ronda nueva del bot vuelve a esperar su envío y cada
// WhatsApp recibe «en preparación» por lo que pidió.
func TestBotOrdersWaitForStaffSend(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
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
	staff := func(name string) scope {
		t.Helper()
		var userID string
		if err := pool.QueryRow(ctx, `
			WITH u AS (
			  INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,$2,$3,'test') RETURNING id
			), ro AS (
			  INSERT INTO roles(organization_id,name,permissions,menu_access) VALUES($1,$3,ARRAY['orders.read','orders.manage','tables.read'],ARRAY['pedidos']) RETURNING id
			), ur AS (
			  INSERT INTO user_roles(user_id,role_id,location_id) SELECT u.id,ro.id,$4 FROM u,ro
			) SELECT id FROM u`, s.OrganizationID, fmt.Sprintf("%s-%d@example.test", strings.ToLower(name), nonce), name, s.LocationID).Scan(&userID); err != nil {
			t.Fatal(err)
		}
		return scope{OrganizationID: s.OrganizationID, LocationID: s.LocationID, UserID: userID, Name: name}
	}
	ana, beto := staff("Ana"), staff("Beto")
	var product string
	if err := pool.QueryRow(ctx, `
		INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control,service_destination)
		VALUES($1,$2,'Lomo saltado',30,true,'prepared','none','kitchen') RETURNING id
	`, s.OrganizationID, fmt.Sprintf("WAIT-%d", nonce)).Scan(&product); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/v1/admin/tables", strings.NewReader(fmt.Sprintf(`{"name":"Mesa W %d","seats":4}`, nonce)))
	api.createTable(w, r.WithContext(context.WithValue(r.Context(), scopeKey{}, s)))
	var table struct {
		QrToken string `json:"qrToken"`
	}
	if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &table) != nil {
		t.Fatalf("create table: %d %s", w.Code, w.Body.String())
	}
	type order struct {
		ID       string `json:"id"`
		Status   string `json:"status"`
		WaiterID string `json:"waiterId"`
	}
	botOrder := func(conversation, phone, request string) order {
		t.Helper()
		body := fmt.Sprintf(`{"customerName":"Lucía","customerPhone":%q,"conversationId":%q,"requestId":%q,"items":[{"productId":%q,"qty":1}]}`, phone, conversation, request, product)
		req := httptest.NewRequest("POST", "/v1/integrations/concierge/"+table.QrToken+"/orders", strings.NewReader(body))
		req.Header.Set("X-Fudia-Concierge-Key", "concierge-test-key")
		rec := httptest.NewRecorder()
		api.Routes().ServeHTTP(rec, req)
		var out order
		if (rec.Code != 201 && rec.Code != 200) || json.Unmarshal(rec.Body.Bytes(), &out) != nil {
			t.Fatalf("bot order: %d %s", rec.Code, rec.Body.String())
		}
		return out
	}
	send := func(owner scope, orderID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/v1/admin/orders/"+orderID+"/send", nil)
		req.SetPathValue("id", orderID)
		rec := httptest.NewRecorder()
		api.handlers.Orders.SendOrder(rec, req.WithContext(context.WithValue(req.Context(), scopeKey{}, owner)))
		return rec
	}
	inKitchen := func(orderID string) int {
		t.Helper()
		req := httptest.NewRequest("GET", "/v1/admin/kitchen/tickets", nil)
		rec := httptest.NewRecorder()
		api.listKitchenTickets(rec, req.WithContext(context.WithValue(req.Context(), scopeKey{}, s)))
		var kitchen struct {
			Items []struct {
				OrderID string `json:"orderId"`
				Items   []struct {
					ID string `json:"id"`
				} `json:"items"`
			} `json:"items"`
		}
		if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &kitchen) != nil {
			t.Fatalf("kitchen: %d %s", rec.Code, rec.Body.String())
		}
		count := 0
		for _, ticket := range kitchen.Items {
			if ticket.OrderID == orderID {
				count += len(ticket.Items)
			}
		}
		return count
	}
	sentNotices := func(orderID string) []string {
		t.Helper()
		rows, err := pool.Query(ctx, `SELECT conversation_id FROM customer_notices WHERE order_id=$1 AND kind='order_sent' ORDER BY created_at`, orderID)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := []string{}
		for rows.Next() {
			var conversation string
			if err := rows.Scan(&conversation); err != nil {
				t.Fatal(err)
			}
			out = append(out, conversation)
		}
		return out
	}

	// 1. El cliente confirma en el bot: la comanda queda guardada, sin mozo y fuera de Cocina.
	first := botOrder("w-lucia", "51911111111", "w-r1")
	if first.Status != "nuevo" || first.WaiterID != "" || inKitchen(first.ID) != 0 {
		t.Fatalf("bot order must wait for the waiter: %+v kitchen=%d", first, inKitchen(first.ID))
	}
	// 2. Ana pulsa «Enviar comanda»: pasa a Cocina, Ana queda a cargo y Lucía recibe «en preparación».
	rec := send(ana, first.ID)
	var sent order
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &sent) != nil || sent.Status != "confirmado" || sent.WaiterID != ana.UserID {
		t.Fatalf("send: %d %s", rec.Code, rec.Body.String())
	}
	if inKitchen(first.ID) != 1 {
		t.Fatalf("sent order must reach the kitchen: %d", inKitchen(first.ID))
	}
	if got := sentNotices(first.ID); len(got) != 1 || got[0] != "w-lucia" {
		t.Fatalf("sent notice: %v", got)
	}
	if rec = send(ana, first.ID); rec.Code != 409 || !strings.Contains(rec.Body.String(), "nothing_to_send") {
		t.Fatalf("nothing left to send: %d %s", rec.Code, rec.Body.String())
	}

	// 3. Un amigo pide otra ronda por el bot: se suma a la cuenta pero espera el envío.
	round := botOrder("w-amigo", "51922222222", "w-r2")
	if round.ID != first.ID || round.Status != "confirmado" || inKitchen(first.ID) != 1 {
		t.Fatalf("bot round must wait for the waiter: %+v kitchen=%d", round, inKitchen(first.ID))
	}
	// Solo la mesera a cargo puede enviarla.
	if rec = send(beto, first.ID); rec.Code != 403 {
		t.Fatalf("another waiter must not send: %d %s", rec.Code, rec.Body.String())
	}
	if rec = send(ana, first.ID); rec.Code != 200 || inKitchen(first.ID) != 2 {
		t.Fatalf("send round: %d %s kitchen=%d", rec.Code, rec.Body.String(), inKitchen(first.ID))
	}
	if got := sentNotices(first.ID); len(got) != 2 || got[1] != "w-amigo" {
		t.Fatalf("each round notifies who ordered it: %v", got)
	}
}
