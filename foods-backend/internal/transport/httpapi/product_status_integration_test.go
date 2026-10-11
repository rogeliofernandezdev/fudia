package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func productStatusRequest(s scope, id, body string) *http.Request {
	req := httptest.NewRequest("PATCH", "/v1/admin/products/"+id+"/status", strings.NewReader(body))
	req.SetPathValue("id", id)
	return req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
}

func TestProductReactivationPreservesCommercialDataAndAudits(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	categoryID := seedRetailCategory(t, pool, s)
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO products(organization_id,category_id,sku,name,description,price,product_type,quantity_control,image_url,prep_minutes,allergens,featured,cost_price,available_from,available_until,available_days,available_until_time)
		VALUES($1,$2,'STATUS-1','Agua mineral','Botella 500 ml',4.50,'retail','inventory','/uploads/products/agua.png',3,ARRAY['leche'],true,2.10,'2026-01-01','2026-12-31',ARRAY[1,2],'18:00')
		RETURNING id`, s.OrganizationID, categoryID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	// Exercise the existing deactivate action before using the new activate action.
	deactivateReq := httptest.NewRequest("DELETE", "/v1/admin/products/"+id, nil)
	deactivateReq.SetPathValue("id", id)
	deactivateReq = deactivateReq.WithContext(context.WithValue(deactivateReq.Context(), scopeKey{}, s))
	deactivated := httptest.NewRecorder()
	api.deactivateProduct(deactivated, deactivateReq)
	if deactivated.Code != http.StatusNoContent {
		t.Fatalf("deactivate: %d %s", deactivated.Code, deactivated.Body.String())
	}
	var before, after string
	if err := pool.QueryRow(ctx, `SELECT (to_jsonb(p)-'active'-'updated_at')::text FROM products p WHERE id=$1`, id).Scan(&before); err != nil {
		t.Fatal(err)
	}
	activated := httptest.NewRecorder()
	api.updateProductStatus(activated, productStatusRequest(s, id, `{"active":true}`))
	if activated.Code != http.StatusNoContent {
		t.Fatalf("activate: %d %s", activated.Code, activated.Body.String())
	}
	var active bool
	if err := pool.QueryRow(ctx, `SELECT active,(to_jsonb(p)-'active'-'updated_at')::text FROM products p WHERE id=$1`, id).Scan(&active, &after); err != nil {
		t.Fatal(err)
	}
	if !active || before != after {
		t.Fatalf("reactivation must change only state/timestamp: active=%v before=%s after=%s", active, before, after)
	}
	var auditCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE organization_id=$1 AND user_id=$2 AND entity_id=$3 AND action='product.activated'`, s.OrganizationID, s.UserID, id).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one activation audit entry, got %d", auditCount)
	}
	// Repeated activation is safe; false also updates only the requested state.
	for _, body := range []string{`{"active":true}`, `{"active":false}`} {
		rec := httptest.NewRecorder()
		api.updateProductStatus(rec, productStatusRequest(s, id, body))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status %s: %d %s", body, rec.Code, rec.Body.String())
		}
	}
	if err := pool.QueryRow(ctx, `SELECT active FROM products WHERE id=$1`, id).Scan(&active); err != nil || active {
		t.Fatalf("expected final inactive state, got active=%v err=%v", active, err)
	}
}

func TestProductStatusEnforcesOrganizationAndPermission(t *testing.T) {
	pool := integrationPool(t)
	owner := seedInventoryScope(t, pool)
	other := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	var id string
	if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price,active) VALUES($1,'STATUS-2','Producto inactivo',5,false) RETURNING id`, owner.OrganizationID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	foreign := httptest.NewRecorder()
	api.updateProductStatus(foreign, productStatusRequest(other, id, `{"active":true}`))
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("another organization must not activate the product: %d %s", foreign.Code, foreign.Body.String())
	}
	protected := api.requirePermission("menu.manage", http.HandlerFunc(api.updateProductStatus))
	denied := httptest.NewRecorder()
	protected.ServeHTTP(denied, productStatusRequest(owner, id, `{"active":true}`))
	if denied.Code != http.StatusForbidden {
		t.Fatalf("read-only user must not activate the product: %d %s", denied.Code, denied.Body.String())
	}
	var active bool
	if err := pool.QueryRow(ctx, `SELECT active FROM products WHERE id=$1`, id).Scan(&active); err != nil || active {
		t.Fatalf("rejected requests changed the product: active=%v err=%v", active, err)
	}
	var roleID string
	if err := pool.QueryRow(ctx, `INSERT INTO roles(organization_id,name,permissions) VALUES($1,'Gestor de carta',ARRAY['menu.manage']) RETURNING id`, owner.OrganizationID).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, owner.UserID, roleID, owner.LocationID); err != nil {
		t.Fatal(err)
	}
	allowed := httptest.NewRecorder()
	protected.ServeHTTP(allowed, productStatusRequest(owner, id, `{"active":true}`))
	if allowed.Code != http.StatusNoContent {
		t.Fatalf("menu manager should activate the product: %d %s", allowed.Code, allowed.Body.String())
	}
}
