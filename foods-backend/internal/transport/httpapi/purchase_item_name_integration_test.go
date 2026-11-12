package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestPurchaseItemRejectsDuplicateNamesWithinCompany(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	other := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	call := func(actor scope, body string, entry bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/v1/admin/purchase-inventory-items", bytes.NewBufferString(body))
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, actor))
		rec := httptest.NewRecorder()
		if entry {
			api.createInventoryEntry(rec, req)
		} else {
			api.createPurchaseInventoryItem(rec, req)
		}
		return rec
	}
	ingredient := func(name string) string { return fmt.Sprintf(`{"newIngredient":{"name":%q},"unit":"kg"}`, name) }
	first := call(s, ingredient("Papa blanca"), false)
	if first.Code != 201 {
		t.Fatalf("initial creation: %d %s", first.Code, first.Body.String())
	}
	category := seedRetailCategory(t, pool, s)
	product := func(name string) string {
		return fmt.Sprintf(`{"newProduct":{"name":%q,"price":"4.50","categoryId":%q},"unit":"kg"}`, name, category)
	}
	assertDuplicate := func(rec *httptest.ResponseRecorder) {
		t.Helper()
		var response struct {
			Code string `json:"code"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if rec.Code != 409 || response.Code != "inventory_item_name_taken" {
			t.Fatalf("expected duplicate conflict, got %d %s", rec.Code, rec.Body.String())
		}
	}
	assertDuplicate(call(s, ingredient("  PAPA    BLANCA  "), false))
	assertDuplicate(call(s, product("Papa blanca"), false))
	assertDuplicate(call(s, `{"newIngredient":{"name":"PAPA BLANCA"},"unit":"kg","quantity":1}`, true))
	if _, err := pool.Exec(ctx, `UPDATE inventory_items SET active=false WHERE organization_id=$1 AND name='Papa blanca'`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	assertDuplicate(call(s, ingredient("Papa blanca"), false))
	// The catalog belongs to the company, even when the location changes.
	secondLocation := s
	if err := pool.QueryRow(ctx, `INSERT INTO locations(organization_id,name,code) VALUES($1,'Segundo','SECOND') RETURNING id`, s.OrganizationID).Scan(&secondLocation.LocationID); err != nil {
		t.Fatal(err)
	}
	assertDuplicate(call(secondLocation, ingredient("Papa blanca"), false))
	if rec := call(other, ingredient("Papa blanca"), false); rec.Code != 201 {
		t.Fatalf("another company may use the same name: %d %s", rec.Code, rec.Body.String())
	}
	// A product already in the menu is also an existing article, even if inactive.
	if _, err := pool.Exec(ctx, `INSERT INTO products(organization_id,sku,name,price,active) VALUES($1,'NAME-EXISTING','Agua mineral',5,false)`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	assertDuplicate(call(s, ingredient("AGUA  MINERAL"), false))
	assertDuplicate(call(s, product("Agua mineral"), false))
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inventory_items WHERE organization_id=$1 AND lower(name)='papa blanca'`, s.OrganizationID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate persisted: count=%d err=%v", count, err)
	}
	// Independent requests must not both pass the initial existence check.
	start := make(chan struct{})
	responses := make(chan *httptest.ResponseRecorder, 4)
	var group sync.WaitGroup
	for i := 0; i < 4; i++ {
		group.Add(1)
		go func() { defer group.Done(); <-start; responses <- call(s, ingredient("Azúcar concurrente"), false) }()
	}
	close(start)
	group.Wait()
	close(responses)
	created := 0
	for rec := range responses {
		if rec.Code == 201 {
			created++
		} else {
			assertDuplicate(rec)
		}
	}
	if created != 1 {
		t.Fatalf("concurrent requests created %d articles, want 1", created)
	}
}
