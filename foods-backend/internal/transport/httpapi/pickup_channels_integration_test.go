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

func TestPickupUnifiesLegacyOrdersWithoutRewritingHistory(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	foreign := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	seed := func(owner scope, channel string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO orders(organization_id,location_id,channel,status,total,created_by) VALUES($1,$2,$3,'confirmado',10,$4) RETURNING id`, owner.OrganizationID, owner.LocationID, channel, owner.UserID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	legacyID := seed(s, "mostrador")
	seed(s, "recojo")
	seed(s, "delivery")
	seed(foreign, "mostrador")
	otherLocal := s
	if err := pool.QueryRow(ctx, `INSERT INTO locations(organization_id,name,code) VALUES($1,'Otro','PICKUP-OTHER') RETURNING id`, s.OrganizationID).Scan(&otherLocal.LocationID); err != nil {
		t.Fatal(err)
	}
	seed(otherLocal, "mostrador")
	call := func(target, body string, handler http.HandlerFunc) *httptest.ResponseRecorder {
		t.Helper()
		method := "GET"
		if body != "" {
			method = "POST"
		}
		r := httptest.NewRequest(method, target, strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
		w := httptest.NewRecorder()
		handler(w, r)
		if w.Code >= 400 {
			t.Fatalf("%s: %d %s", target, w.Code, w.Body.String())
		}
		return w
	}
	for _, alias := range []string{"recojo", "mostrador"} {
		w := call("/v1/admin/orders?channel="+alias+"&status=abiertos&page=1&pageSize=20", "", api.listOrders)
		var out struct {
			Items   []order             `json:"items"`
			Total   int                 `json:"total"`
			Counts  map[string]int      `json:"channelCounts"`
			Options []map[string]string `json:"channelOptions"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if out.Total != 2 || len(out.Items) != 2 || out.Counts["recojo"] != 2 || out.Counts["mostrador"] != 0 || out.Counts["delivery"] != 1 {
			t.Fatalf("unified scoped counts: %+v", out)
		}
		for _, item := range out.Items {
			if item.Channel != "recojo" {
				t.Fatalf("legacy channel leaked: %+v", item)
			}
		}
		for _, option := range out.Options {
			if option["value"] == "mostrador" {
				t.Fatal("duplicate pickup option")
			}
		}
		w = call("/v1/admin/kitchen/tickets?channel="+alias, "", api.listKitchenTickets)
		var kitchen struct {
			Items []struct {
				Channel string `json:"channel"`
			} `json:"items"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &kitchen); err != nil {
			t.Fatal(err)
		}
		if len(kitchen.Items) != 2 {
			t.Fatalf("legacy kitchen filter omitted pickup: %s", w.Body.String())
		}
		for _, item := range kitchen.Items {
			if item.Channel != "recojo" {
				t.Fatal("legacy kitchen channel leaked")
			}
		}
	}
	w := call("/v1/admin/pos/orders?page=1&pageSize=20", "", api.listPOSOrders)
	var pos struct {
		Items []struct {
			ID      string `json:"id"`
			Channel string `json:"channel"`
		} `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &pos); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range pos.Items {
		if item.ID == legacyID {
			found = true
			if item.Channel != "recojo" {
				t.Fatal("POS has separate legacy pickup")
			}
		}
	}
	if !found {
		t.Fatal("legacy pickup lost from POS")
	}
	var stored string
	if err := pool.QueryRow(ctx, `SELECT channel FROM orders WHERE id=$1`, legacyID).Scan(&stored); err != nil || stored != "mostrador" {
		t.Fatalf("history rewritten: %q %v", stored, err)
	}
	var productID string
	if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price,quantity_control) VALUES($1,'PICKUP-TEST','Plato para llevar',10,'none') RETURNING id`, s.OrganizationID).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	for _, alias := range []string{"mostrador", "recojo"} {
		w = call("/v1/admin/orders", fmt.Sprintf(`{"channel":%q,"items":[{"productId":%q,"qty":1}]}`, alias, productID), api.createOrder)
		var created order
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
			t.Fatal(err)
		}
		if created.Channel != "recojo" || created.TableID != "" || created.DeliveryFee != "0.00" {
			t.Fatalf("new pickup: %+v", created)
		}
		if err := pool.QueryRow(ctx, `SELECT channel FROM orders WHERE id=$1`, created.ID).Scan(&stored); err != nil || stored != "recojo" {
			t.Fatalf("new channel not canonical: %q %v", stored, err)
		}
	}
}
