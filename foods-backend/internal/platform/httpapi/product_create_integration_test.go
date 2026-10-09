package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCreateProductWithInitialPortionsAndLaterAvailability(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE locations SET timezone='Pacific/Kiritimati' WHERE id=$1`, s.LocationID); err != nil {
		t.Fatal(err)
	}
	rec := postProduct(api, s, `{"name":"Ají de gallina","price":"20","quantityControl":"portions","initialPortionQuantity":15,"serviceDestination":"kitchen"}`)
	if rec.Code != 201 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created product
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	quota, sold, status := readDailyAvailability(t, pool, s, created.ID)
	if quota != 15 || sold != 0 || status != "available" {
		t.Fatalf("initial availability: %d %d %s", quota, sold, status)
	}
	var rows, audits int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM product_availability WHERE product_id=$1),(SELECT count(*) FROM audit_log WHERE entity_id=$1 AND action='product.created' AND (metadata->>'initialPortionQuantity')::int=15)`, created.ID).Scan(&rows, &audits); err != nil || rows != 1 || audits != 1 {
		t.Fatalf("only selected local/day and audited quantity: rows=%d audits=%d err=%v", rows, audits, err)
	}
	// Catalog and operational availability must both expose the newly saved product.
	for _, list := range []func(http.ResponseWriter, *http.Request){api.listProducts, api.listProductAvailability} {
		req := httptest.NewRequest("GET", "/?q=Aj%C3%AD", nil)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		result := httptest.NewRecorder()
		list(result, req)
		if result.Code != 200 || !strings.Contains(result.Body.String(), created.ID) {
			t.Fatalf("missing created product: %d %s", result.Code, result.Body.String())
		}
	}
	for _, body := range []string{`{"status":"available","portionQuantity":20,"reason":"Nueva producción"}`, `{"status":"sold_out","portionQuantity":null,"reason":"Pausa de producción"}`, `{"status":"available","portionQuantity":null,"reason":"Producción disponible"}`} {
		if result := updateAvailabilityRequest(t, api, s, created.ID, body); result.Code != 204 {
			t.Fatalf("later availability: %d %s", result.Code, result.Body.String())
		}
	}
	quota, sold, status = readDailyAvailability(t, pool, s, created.ID)
	if quota != 20 || sold != 0 || status != "available" {
		t.Fatalf("updated availability: %d %d %s", quota, sold, status)
	}
	// Commercial editing does not reset the daily quota.
	req := httptest.NewRequest("PATCH", "/", strings.NewReader(`{"name":"Ají de gallina actualizado","price":"22","quantityControl":"portions"}`))
	req.SetPathValue("id", created.ID)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	updated := httptest.NewRecorder()
	api.updateProduct(updated, req)
	if updated.Code != 200 {
		t.Fatalf("commercial update: %d %s", updated.Code, updated.Body.String())
	}
	quota, _, _ = readDailyAvailability(t, pool, s, created.ID)
	if quota != 20 {
		t.Fatal("commercial update overwrote availability")
	}
}

func TestCreateProductRollsBackWhenInitialAvailabilityOrAuditFails(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	for _, table := range []string{"product_availability", "audit_log"} {
		t.Run(table, func(t *testing.T) {
			s := seedInventoryScope(t, pool)
			function := fmt.Sprintf("product_create_failure_%d", time.Now().UnixNano())
			query := fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.organization_id='%s'::uuid THEN RAISE EXCEPTION 'test failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER %s BEFORE INSERT ON %s FOR EACH ROW EXECUTE FUNCTION %s()`, function, s.OrganizationID, function, table, function)
			if _, err := pool.Exec(ctx, query); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_, _ = pool.Exec(ctx, fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s; DROP FUNCTION IF EXISTS %s()", function, table, function))
			})
			rec := postProduct(New(pool), s, `{"name":"Fallido","price":"20","quantityControl":"portions","initialPortionQuantity":15}`)
			if rec.Code != 503 {
				t.Fatalf("failed save: %d %s", rec.Code, rec.Body.String())
			}
			var products, availability, audits int
			if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM products WHERE organization_id=$1),(SELECT count(*) FROM product_availability WHERE organization_id=$1),(SELECT count(*) FROM audit_log WHERE organization_id=$1)`, s.OrganizationID).Scan(&products, &availability, &audits); err != nil || products != 0 || availability != 0 || audits != 0 {
				t.Fatalf("partial save: products=%d availability=%d audits=%d err=%v", products, availability, audits, err)
			}
		})
	}
}

func TestCreateProductScopesInitialQuantityAndRequiresMenuManage(t *testing.T) {
	pool := integrationPool(t)
	owner, other := seedInventoryScope(t, pool), seedInventoryScope(t, pool)
	ctx := context.Background()
	// A foreign local cannot be used to initialize portions, even with a direct handler call.
	foreign := owner
	foreign.LocationID = other.LocationID
	if rec := postProduct(New(pool), foreign, `{"name":"Ají","price":"20","quantityControl":"portions","initialPortionQuantity":15}`); rec.Code != 503 {
		t.Fatalf("foreign local: %d %s", rec.Code, rec.Body.String())
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM products WHERE organization_id=$1`, owner.OrganizationID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("foreign local left catalog row: count=%d err=%v", count, err)
	}
	var roleID string
	if err := pool.QueryRow(ctx, `INSERT INTO roles(organization_id,name,permissions) VALUES($1,'Product reader',ARRAY['menu.read']) RETURNING id`, owner.OrganizationID).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, owner.UserID, roleID, owner.LocationID); err != nil {
		t.Fatal(err)
	}
	token := "product-create-test-" + owner.UserID
	hash := sha256.Sum256([]byte(token))
	if _, err := pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,organization_id,location_id,expires_at) VALUES($1,$2,$3,$4,now()+interval '1 hour')`, hash[:], owner.UserID, owner.OrganizationID, owner.LocationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, hash[:]) })
	routes := New(pool).Routes()
	call := func(authenticated bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/v1/admin/products", strings.NewReader(`{"name":"Ají","price":"20","quantityControl":"portions","initialPortionQuantity":15}`))
		if authenticated {
			req.AddCookie(&http.Cookie{Name: "foods_session", Value: token})
		}
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		return rec
	}
	if rec := call(false); rec.Code != 401 {
		t.Fatalf("anonymous: %d", rec.Code)
	}
	if rec := call(true); rec.Code != 403 {
		t.Fatalf("read-only: %d %s", rec.Code, rec.Body.String())
	}
	if _, err := pool.Exec(ctx, `UPDATE roles SET permissions=ARRAY['menu.read','menu.manage'] WHERE id=$1`, roleID); err != nil {
		t.Fatal(err)
	}
	if rec := call(true); rec.Code != 201 {
		t.Fatalf("authorized: %d %s", rec.Code, rec.Body.String())
	}
}

func TestCreateProductWithoutControlHasNoDailyQuantity(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	rec := postProduct(New(pool), s, `{"name":"Sin límite","price":"20"}`)
	if rec.Code != 201 {
		t.Fatalf("without portions: %d %s", rec.Code, rec.Body.String())
	}
	var quantityRows int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM product_availability WHERE organization_id=$1`, s.OrganizationID).Scan(&quantityRows); err != nil || quantityRows != 0 {
		t.Fatalf("uncontrolled product quantity: %d err=%v", quantityRows, err)
	}
}
