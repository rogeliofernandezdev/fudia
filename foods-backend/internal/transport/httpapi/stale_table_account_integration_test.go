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
)

// Una cuenta de mesa sin rondas nuevas por horas se trata como una cuenta anterior
// sin cerrar: un QR nuevo no le suma pedidos, el bot recibe table_account_stale y el
// personal ve «cuenta anterior sin cerrar» hasta cobrarla, cancelarla o agregar
// productos. Dentro de ese tiempo, el grupo sigue pidiendo a la misma cuenta.
func TestStaleTableAccountBlocksNewQRDiner(t *testing.T) {
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
	var waiterID, product string
	if err := pool.QueryRow(ctx, `
		WITH u AS (
		  INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,$2,'Mozo prueba','test') RETURNING id
		), ro AS (
		  INSERT INTO roles(organization_id,name,permissions,menu_access) VALUES($1,'Mozo',ARRAY['orders.read','orders.manage','tables.read'],ARRAY['pedidos']) RETURNING id
		), ur AS (
		  INSERT INTO user_roles(user_id,role_id,location_id) SELECT u.id,ro.id,$3 FROM u,ro
		) SELECT id FROM u`, s.OrganizationID, fmt.Sprintf("stale-%d@example.test", nonce), s.LocationID).Scan(&waiterID); err != nil {
		t.Fatal(err)
	}
	waiter := scope{OrganizationID: s.OrganizationID, LocationID: s.LocationID, UserID: waiterID, Name: "Mozo"}
	if err := pool.QueryRow(ctx, `
		INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control,service_destination)
		VALUES($1,$2,'Ceviche',25,true,'prepared','none','kitchen') RETURNING id
	`, s.OrganizationID, fmt.Sprintf("STALE-%d", nonce)).Scan(&product); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/v1/admin/tables", strings.NewReader(fmt.Sprintf(`{"name":"Mesa ST %d","seats":4}`, nonce)))
	r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
	api.createTable(w, r)
	var table struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		QrToken string `json:"qrToken"`
	}
	if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &table) != nil {
		t.Fatalf("create table: %d %s", w.Code, w.Body.String())
	}
	concierge := func(path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/v1/integrations/concierge/"+table.QrToken+path, strings.NewReader(body))
		req.Header.Set("X-Fudia-Concierge-Key", "concierge-test-key")
		rec := httptest.NewRecorder()
		api.Routes().ServeHTTP(rec, req)
		return rec
	}
	order := func(conversation, request string) *httptest.ResponseRecorder {
		return concierge("/orders", fmt.Sprintf(`{"customerName":"Lucía","customerPhone":"51911111111","conversationId":%q,"requestId":%q,"items":[{"productId":%q,"qty":1}]}`, conversation, request, product))
	}
	open := func(conversation string) *httptest.ResponseRecorder {
		return concierge("/conversations", fmt.Sprintf(`{"conversationId":%q,"kind":"table"}`, conversation))
	}
	staleNotices := func() int {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/v1/admin/notices", bytes.NewReader(nil))
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, waiter))
		api.handlers.Notices.Feed(rec, req)
		var feed noticeFeed
		if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &feed) != nil {
			t.Fatalf("feed: %d %s", rec.Code, rec.Body.String())
		}
		count := 0
		for _, item := range feed.Items {
			if item.Kind == "stale_account" {
				count++
				if item.Title != table.Name+" · cuenta anterior sin cerrar" {
					t.Fatalf("stale title: %q", item.Title)
				}
			}
		}
		return count
	}

	// Cliente anterior: abre la mesa y pide.
	if rec := open("st-first"); rec.Code != 201 && rec.Code != 200 {
		t.Fatalf("first conversation: %d %s", rec.Code, rec.Body.String())
	}
	rec := order("st-first", "st-r1")
	var placed struct {
		ID string `json:"id"`
	}
	if rec.Code != 201 || json.Unmarshal(rec.Body.Bytes(), &placed) != nil {
		t.Fatalf("first order: %d %s", rec.Code, rec.Body.String())
	}
	// Un amigo en la misma mesa, dentro del tiempo: se suma a la misma cuenta.
	if rec = open("st-friend"); rec.Code != 201 && rec.Code != 200 {
		t.Fatalf("friend conversation: %d %s", rec.Code, rec.Body.String())
	}
	if rec = order("st-friend", "st-r2"); rec.Code != 200 {
		t.Fatalf("friend round: %d %s", rec.Code, rec.Body.String())
	}

	// Pasan 4 horas sin rondas nuevas: el cliente se fue dejando la cuenta abierta.
	if _, err := pool.Exec(ctx, `UPDATE orders SET created_at=created_at-interval '4 hours' WHERE id=$1`, placed.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE order_kitchen_rounds SET created_at=created_at-interval '4 hours' WHERE order_id=$1`, placed.ID); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if rec = open(fmt.Sprintf("st-next-%d", time.Now().UnixNano())); rec.Code != 409 || !strings.Contains(rec.Body.String(), "table_account_stale") {
			t.Fatalf("new diner must not join a stale account: %d %s", rec.Code, rec.Body.String())
		}
	}
	if got := staleNotices(); got != 1 {
		t.Fatalf("one stale notice per episode, got %d", got)
	}
	// Una atención abierta antes tampoco suma pedidos a la cuenta inactiva.
	if rec = order("st-friend", "st-r3"); rec.Code != 409 || !strings.Contains(rec.Body.String(), "table_account_stale") {
		t.Fatalf("stale account must reject rounds: %d %s", rec.Code, rec.Body.String())
	}

	// El personal cobra o cancela la cuenta: el aviso se resuelve y la mesa recibe al nuevo cliente.
	if _, err := pool.Exec(ctx, `UPDATE orders SET status='cancelado' WHERE id=$1`, placed.ID); err != nil {
		t.Fatal(err)
	}
	if got := staleNotices(); got != 0 {
		t.Fatalf("attended stale account must leave the feed, got %d", got)
	}
	if rec = open("st-new-diner"); rec.Code != 201 && rec.Code != 200 {
		t.Fatalf("table must open for the next diner: %d %s", rec.Code, rec.Body.String())
	}
}
