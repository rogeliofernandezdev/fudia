package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Enlace/QR de delivery del local: se resuelve como un QR con
// serviceType=delivery, solo toma pedidos de delivery o recojo (sin mesa,
// cuenta ni llamado al mozo) y regenerarlo invalida el token anterior.
func TestLocationDeliveryLink(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	nonce := time.Now().UnixNano()
	t.Setenv("FUDIA_CONCIERGE_API_KEY", "concierge-test-key")
	var productID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control)
		VALUES($1,$2,'Lomo para llevar',30,true,'prepared','none') RETURNING id`,
		s.OrganizationID, fmt.Sprintf("LINK-%d", nonce)).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO organization_modules(organization_id,module_key,active) VALUES($1,'whatsapp_bot',true)
		ON CONFLICT(organization_id,module_key) DO UPDATE SET active=true,updated_at=now()`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM concierge_delivery_requests WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(bg, `DELETE FROM location_delivery_links WHERE organization_id=$1`, s.OrganizationID)
	})
	d := api.handlers.Delivery
	admin := func(handler func(http.ResponseWriter, *http.Request), method, id string, subject scope) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/", nil)
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, subject))
		rec := httptest.NewRecorder()
		handler(rec, req)
		return rec
	}
	link := func(rec *httptest.ResponseRecorder) string {
		t.Helper()
		var out struct {
			Token  string
			Active bool
		}
		if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &out) != nil {
			t.Fatalf("link: %d %s", rec.Code, rec.Body.String())
		}
		return out.Token
	}
	if token := link(admin(d.DeliveryLink, "GET", s.LocationID, s)); token != "" {
		t.Fatalf("no link yet, got %q", token)
	}
	token := link(admin(d.RotateDeliveryLink, "POST", s.LocationID, s))
	if len(token) != 32 || link(admin(d.DeliveryLink, "GET", s.LocationID, s)) != token {
		t.Fatalf("created link: %q", token)
	}
	call := func(method, path, body, qr string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if strings.HasPrefix(path, "/v1/integrations/") {
			req.Header.Set("X-Fudia-Concierge-Key", "concierge-test-key")
		}
		rec := httptest.NewRecorder()
		api.Routes().ServeHTTP(rec, req)
		return rec
	}
	public := call("GET", "/v1/public/tables/"+token, "", token)
	if public.Code != 200 || !strings.Contains(public.Body.String(), `"serviceType":"delivery"`) || !strings.Contains(public.Body.String(), `"conciergeEnabled":true`) {
		t.Fatalf("public resolve: %d %s", public.Code, public.Body.String())
	}
	base := "/v1/integrations/concierge/" + token
	if rec := call("GET", base+"/menu", "", token); rec.Code != 200 || !strings.Contains(rec.Body.String(), "Lomo para llevar") {
		t.Fatalf("menu by link: %d %s", rec.Code, rec.Body.String())
	}
	conversation := fmt.Sprintf("conv-link-%d", nonce)
	items := fmt.Sprintf(`[{"productId":%q,"qty":1,"unitPrice":0.01,"note":"","selections":[]}]`, productID)
	tableOrder := fmt.Sprintf(`{"customerName":"Ana","customerPhone":"51999999999","conversationId":%q,"requestId":"t-%d","items":%s}`, conversation, nonce, items)
	if rec := call("POST", base+"/orders", tableOrder, token); rec.Code != 409 || !strings.Contains(rec.Body.String(), "delivery_link_only") {
		t.Fatalf("table order by link: %d %s", rec.Code, rec.Body.String())
	}
	bill := fmt.Sprintf(`{"conversationId":%q,"customerPhone":"51999999999","requestId":"b-%d"}`, conversation, nonce)
	if rec := call("POST", base+"/bill", bill, token); rec.Code != 409 || !strings.Contains(rec.Body.String(), "delivery_link_only") {
		t.Fatalf("bill by link: %d %s", rec.Code, rec.Body.String())
	}
	pickup := fmt.Sprintf(`{"fulfillment":"pickup","customerName":"Ana","customerPhone":"51999999999","conversationId":%q,"requestId":"p-%d","items":%s}`, conversation, nonce, items)
	rec := call("POST", base+"/delivery-orders", pickup, token)
	if rec.Code != 201 || !strings.Contains(rec.Body.String(), `"channel":"recojo"`) {
		t.Fatalf("pickup by link: %d %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID, TableID string
	}
	if json.Unmarshal(rec.Body.Bytes(), &created) != nil || created.TableID != "" {
		t.Fatalf("pickup must not use a table: %s", rec.Body.String())
	}
	if rec := call("POST", base+"/conversations", fmt.Sprintf(`{"conversationId":"%s-2","kind":"delivery"}`, conversation), token); rec.Code >= 400 {
		t.Fatalf("delivery conversation by link: %d %s", rec.Code, rec.Body.String())
	}

	// Regenerar invalida el QR impreso anterior.
	rotated := link(admin(d.RotateDeliveryLink, "POST", s.LocationID, s))
	if rotated == token {
		t.Fatal("rotation must change the token")
	}
	if rec := call("GET", "/v1/public/tables/"+token, "", token); rec.Code != 404 {
		t.Fatalf("old token still open: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call("GET", base+"/menu", "", token); rec.Code != 404 {
		t.Fatalf("old token still serves menu: %d %s", rec.Code, rec.Body.String())
	}
	foreign := seedInventoryScope(t, pool)
	if rec := admin(d.DeliveryLink, "GET", s.LocationID, foreign); rec.Code != 404 {
		t.Fatalf("foreign company sees link: %d %s", rec.Code, rec.Body.String())
	}
	if rec := admin(d.RotateDeliveryLink, "POST", s.LocationID, foreign); rec.Code != 404 {
		t.Fatalf("foreign company rotates link: %d %s", rec.Code, rec.Body.String())
	}
}
