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

type customerNoticeItem struct {
	ID               string `json:"id"`
	Kind             string `json:"kind"`
	OrderID          string `json:"orderId"`
	OrderCode        string `json:"orderCode"`
	Fulfillment      string `json:"fulfillment"`
	CustomerPhone    string `json:"customerPhone"`
	ConversationID   string `json:"conversationId"`
	OrganizationName string `json:"organizationName"`
	LocationName     string `json:"locationName"`
}

// Fase 2: con todo el pedido listo, el cliente de WhatsApp que pidió en el
// mostrador recibe un aviso. foods-backend lo deja en una bandeja de salida que
// Concierge reclama con su clave, con vencimiento del reclamo y confirmación.
func TestCustomerNoticesOutbox(t *testing.T) {
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
	var product, tradeName string
	if err := pool.QueryRow(ctx, `
		INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control,service_destination)
		VALUES($1,$2,'Ají de gallina',28,true,'prepared','none','kitchen') RETURNING id
	`, s.OrganizationID, fmt.Sprintf("CUS-%d", nonce)).Scan(&product); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT trade_name FROM organizations WHERE id=$1`, s.OrganizationID).Scan(&tradeName); err != nil {
		t.Fatal(err)
	}

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
	integration := func(method, path, key string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		if key != "" {
			req.Header.Set("X-Fudia-Concierge-Key", key)
		}
		rec := httptest.NewRecorder()
		api.Routes().ServeHTTP(rec, req)
		return rec
	}
	// La bandeja es global: otras pruebas pueden dejar avisos; se filtra por pedido.
	claim := func(orderIDs ...string) []customerNoticeItem {
		t.Helper()
		rec := integration("GET", "/v1/integrations/concierge/notices/pending?limit=50", "concierge-test-key")
		var out struct {
			Items []customerNoticeItem `json:"items"`
		}
		if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &out) != nil {
			t.Fatalf("claim: %d %s", rec.Code, rec.Body.String())
		}
		mine := []customerNoticeItem{}
		for _, item := range out.Items {
			for _, id := range orderIDs {
				if item.OrderID == id {
					mine = append(mine, item)
				}
			}
		}
		return mine
	}
	createTable := func(body string) string {
		t.Helper()
		w := call("POST", "/v1/admin/tables", body, api.createTable)
		var out struct {
			QrToken string `json:"qrToken"`
		}
		if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &out) != nil {
			t.Fatalf("create table: %d %s", w.Code, w.Body.String())
		}
		return out.QrToken
	}
	concierge := func(token, conversation, requestID string) string {
		t.Helper()
		body := fmt.Sprintf(`{"customerName":"Ana","customerPhone":"51911111111","conversationId":%q,"requestId":%q,"items":[{"productId":%q,"qty":1}]}`, conversation, requestID, product)
		req := httptest.NewRequest("POST", "/v1/integrations/concierge/"+token+"/orders", strings.NewReader(body))
		req.Header.Set("X-Fudia-Concierge-Key", "concierge-test-key")
		rec := httptest.NewRecorder()
		api.Routes().ServeHTTP(rec, req)
		var out struct {
			ID string `json:"id"`
		}
		if rec.Code != 201 || json.Unmarshal(rec.Body.Bytes(), &out) != nil {
			t.Fatalf("concierge order: %d %s", rec.Code, rec.Body.String())
		}
		return out.ID
	}
	ready := func(orderID string) {
		t.Helper()
		w := call("GET", "/v1/admin/kitchen/tickets?destination=kitchen", "", api.listKitchenTickets)
		var kitchen struct {
			Items []struct {
				ID      string `json:"id"`
				OrderID string `json:"orderId"`
			} `json:"items"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &kitchen) != nil {
			t.Fatalf("kitchen: %d %s", w.Code, w.Body.String())
		}
		found := false
		for _, ticket := range kitchen.Items {
			if ticket.OrderID != orderID {
				continue
			}
			found = true
			for _, status := range []string{"preparando", "listo"} {
				if w = call("PATCH", "/v1/admin/kitchen/tickets/"+ticket.ID+"/status", `{"status":"`+status+`"}`, api.updateKitchenTicketStatus, "id", ticket.ID); w.Code != 204 {
					t.Fatalf("kitchen %s: %d %s", status, w.Code, w.Body.String())
				}
			}
		}
		if !found {
			t.Fatalf("order %s not in kitchen", orderID)
		}
	}
	pending := func(orderID string) int {
		t.Helper()
		var count int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM customer_notices WHERE order_id=$1`, orderID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		return count
	}

	counterToken := createTable(fmt.Sprintf(`{"name":"Mostrador CUS %d","serviceType":"counter"}`, nonce))
	tableToken := createTable(fmt.Sprintf(`{"name":"Mesa CUS %d","seats":2}`, nonce))
	counterOrder := concierge(counterToken, "cus-counter", "cus-c1")
	tableOrder := concierge(tableToken, "cus-table", "cus-t1")

	// Sin preparación terminada no hay aviso.
	if got := claim(counterOrder); len(got) != 0 || pending(counterOrder) != 0 {
		t.Fatalf("notice before ready: %+v", got)
	}
	ready(counterOrder)
	ready(tableOrder)
	// La mesa la atiende el mozo: no se avisa al cliente.
	if pending(tableOrder) != 0 {
		t.Fatal("table orders must not notify the customer")
	}
	// Un pedido de mostrador registrado por el personal no tiene conversación de WhatsApp.
	w := call("POST", "/v1/admin/orders", fmt.Sprintf(`{"channel":"mostrador","customerName":"Rosa","customerPhone":"51999999999","sendToKitchen":true,"items":[{"productId":%q,"qty":1}]}`, product), api.createOrder)
	var manual struct {
		ID string `json:"id"`
	}
	if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &manual) != nil {
		t.Fatalf("manual counter order: %d %s", w.Code, w.Body.String())
	}
	ready(manual.ID)
	if pending(manual.ID) != 0 {
		t.Fatal("staff orders must not notify by WhatsApp")
	}

	// Sin clave de Concierge no se entrega nada.
	if rec := integration("GET", "/v1/integrations/concierge/notices/pending", ""); rec.Code != 401 {
		t.Fatalf("claim without key: %d", rec.Code)
	}
	got := claim(counterOrder)
	if len(got) != 1 {
		t.Fatalf("counter notice: %+v", got)
	}
	notice := got[0]
	var code string
	if err := pool.QueryRow(ctx, `SELECT code FROM orders WHERE id=$1`, counterOrder).Scan(&code); err != nil {
		t.Fatal(err)
	}
	if notice.Kind != "order_ready" || notice.Fulfillment != "counter" || notice.OrderCode != code || notice.CustomerPhone != "51911111111" ||
		notice.ConversationID != "cus-counter" || notice.OrganizationName != tradeName || notice.LocationName != "Principal" {
		t.Fatalf("notice facts: %+v", notice)
	}
	// Reclamado: no se entrega otra vez mientras dura el reclamo.
	if again := claim(counterOrder); len(again) != 0 {
		t.Fatalf("claimed notice delivered twice: %+v", again)
	}
	// Si Concierge no confirma, el reclamo vence y vuelve a entregarse.
	if _, err := pool.Exec(ctx, `UPDATE customer_notices SET claimed_until=now()-interval '1 second' WHERE id=$1`, notice.ID); err != nil {
		t.Fatal(err)
	}
	if again := claim(counterOrder); len(again) != 1 || again[0].ID != notice.ID {
		t.Fatalf("expired claim must be delivered again: %+v", again)
	}
	for range 2 {
		if rec := integration("POST", "/v1/integrations/concierge/notices/"+notice.ID+"/dispatched", "concierge-test-key"); rec.Code != 204 {
			t.Fatalf("dispatched: %d %s", rec.Code, rec.Body.String())
		}
	}
	var status string
	var attempts int
	if err := pool.QueryRow(ctx, `SELECT status,attempts FROM customer_notices WHERE id=$1`, notice.ID).Scan(&status, &attempts); err != nil || status != "dispatched" || attempts != 2 {
		t.Fatalf("dispatched state: %s %d %v", status, attempts, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE customer_notices SET claimed_until=NULL WHERE id=$1`, notice.ID); err != nil {
		t.Fatal(err)
	}
	if again := claim(counterOrder); len(again) != 0 {
		t.Fatalf("dispatched notice delivered again: %+v", again)
	}
	if rec := integration("POST", "/v1/integrations/concierge/notices/00000000-0000-0000-0000-000000000000/dispatched", "concierge-test-key"); rec.Code != 404 {
		t.Fatalf("unknown notice: %d", rec.Code)
	}
}
