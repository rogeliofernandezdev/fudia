package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func categoryStatusRequest(s scope, id, body string) *http.Request {
	req := httptest.NewRequest("PATCH", "/v1/admin/categories/"+id+"/status", strings.NewReader(body))
	req.SetPathValue("id", id)
	return req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
}

func TestCategoryReactivationPreservesDataAndReturnsToSelectors(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	id := seedRetailCategory(t, pool, s)
	if _, err := pool.Exec(ctx, `UPDATE menu_categories SET name='Bebidas',sort_order=7 WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	var productID string
	if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,category_id,sku,name,price,product_type,active) VALUES($1,$2,'CATEGORY-1','Agua',4,'retail',false) RETURNING id`, s.OrganizationID, id).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	deactivated := httptest.NewRecorder()
	api.deactivateCategory(deactivated, categoryStatusRequest(s, id, ""))
	if deactivated.Code != http.StatusNoContent {
		t.Fatalf("deactivate: %d %s", deactivated.Code, deactivated.Body.String())
	}
	var before, after string
	if err := pool.QueryRow(ctx, `SELECT (to_jsonb(c)-'active')::text FROM menu_categories c WHERE id=$1`, id).Scan(&before); err != nil {
		t.Fatal(err)
	}
	assertSelector := func(productType string, expected int) {
		t.Helper()
		req := httptest.NewRequest("GET", "/v1/admin/categories?productType="+productType, nil)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		api.listCategories(rec, req)
		var result struct {
			Items []category `json:"items"`
			Total int        `json:"total"`
		}
		if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &result) != nil || result.Total != expected || len(result.Items) != expected {
			t.Fatalf("selector %s expected %d categories: %d %s", productType, expected, rec.Code, rec.Body.String())
		}
		if expected == 1 && (result.Items[0].ID != id || !result.Items[0].Active) {
			t.Fatalf("expected reactivated category in selector: %+v", result.Items)
		}
	}
	assertSelector("retail", 0)
	activated := httptest.NewRecorder()
	api.updateCategoryStatus(activated, categoryStatusRequest(s, id, `{"active":true}`))
	if activated.Code != http.StatusNoContent {
		t.Fatalf("activate: %d %s", activated.Code, activated.Body.String())
	}
	var active bool
	if err := pool.QueryRow(ctx, `SELECT active,(to_jsonb(c)-'active')::text FROM menu_categories c WHERE id=$1`, id).Scan(&active, &after); err != nil {
		t.Fatal(err)
	}
	if !active || before != after {
		t.Fatalf("activation changed category data: active=%v before=%s after=%s", active, before, after)
	}
	var linkedCategory string
	if err := pool.QueryRow(ctx, `SELECT category_id::text,active FROM products WHERE id=$1`, productID).Scan(&linkedCategory, &active); err != nil || linkedCategory != id || active {
		t.Fatalf("activation must preserve associated product and its state: category=%s active=%v err=%v", linkedCategory, active, err)
	}
	assertSelector("retail", 1)
	assertSelector("prepared", 0)
	var auditCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE organization_id=$1 AND user_id=$2 AND entity_id=$3 AND action='category.activated'`, s.OrganizationID, s.UserID, id).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("expected activation audit: count=%d err=%v", auditCount, err)
	}
	repeated := httptest.NewRecorder()
	api.updateCategoryStatus(repeated, categoryStatusRequest(s, id, `{"active":true}`))
	if repeated.Code != http.StatusNoContent {
		t.Fatalf("repeated activation: %d %s", repeated.Code, repeated.Body.String())
	}
}

func TestCategoryStatusEnforcesOrganizationAndPermission(t *testing.T) {
	pool := integrationPool(t)
	owner := seedInventoryScope(t, pool)
	other := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	id := seedRetailCategory(t, pool, owner)
	if _, err := pool.Exec(ctx, `UPDATE menu_categories SET active=false WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	foreign := httptest.NewRecorder()
	api.updateCategoryStatus(foreign, categoryStatusRequest(other, id, `{"active":true}`))
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("foreign organization: %d %s", foreign.Code, foreign.Body.String())
	}
	protected := api.requirePermission("menu.manage", http.HandlerFunc(api.updateCategoryStatus))
	denied := httptest.NewRecorder()
	protected.ServeHTTP(denied, categoryStatusRequest(owner, id, `{"active":true}`))
	if denied.Code != http.StatusForbidden {
		t.Fatalf("read-only user: %d %s", denied.Code, denied.Body.String())
	}
	var active bool
	if err := pool.QueryRow(ctx, `SELECT active FROM menu_categories WHERE id=$1`, id).Scan(&active); err != nil || active {
		t.Fatalf("rejected requests changed state: active=%v err=%v", active, err)
	}
	var roleID string
	if err := pool.QueryRow(ctx, `INSERT INTO roles(organization_id,name,permissions) VALUES($1,'Gestor de categorías',ARRAY['menu.manage']) RETURNING id`, owner.OrganizationID).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, owner.UserID, roleID, owner.LocationID); err != nil {
		t.Fatal(err)
	}
	allowed := httptest.NewRecorder()
	protected.ServeHTTP(allowed, categoryStatusRequest(owner, id, `{"active":true}`))
	if allowed.Code != http.StatusNoContent {
		t.Fatalf("menu manager: %d %s", allowed.Code, allowed.Body.String())
	}
}

func TestCategoryStatusCannotDeactivateCategoryInUse(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	id := seedRetailCategory(t, pool, s)
	var productID string
	if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,category_id,sku,name,price,product_type) VALUES($1,$2,'CATEGORY-2','Agua',4,'retail') RETURNING id`, s.OrganizationID, id).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	blocked := httptest.NewRecorder()
	api.updateCategoryStatus(blocked, categoryStatusRequest(s, id, `{"active":false}`))
	if blocked.Code != http.StatusConflict || !strings.Contains(blocked.Body.String(), "category_in_use") || !strings.Contains(blocked.Body.String(), "La categoría se encuentra en uso") {
		t.Fatalf("must retain in-use guard: %d %s", blocked.Code, blocked.Body.String())
	}
	var active bool
	if err := pool.QueryRow(ctx, `SELECT active FROM menu_categories WHERE id=$1`, id).Scan(&active); err != nil || !active {
		t.Fatalf("blocked deactivation changed category: active=%v err=%v", active, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE products SET active=false WHERE id=$1`, productID); err != nil {
		t.Fatal(err)
	}
	deactivated := httptest.NewRecorder()
	api.updateCategoryStatus(deactivated, categoryStatusRequest(s, id, `{"active":false}`))
	if deactivated.Code != http.StatusNoContent {
		t.Fatalf("deactivation without active products: %d %s", deactivated.Code, deactivated.Body.String())
	}
}
