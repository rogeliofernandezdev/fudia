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

func availabilityHistoryRequest(api *API, s scope, productID, query string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "/v1/admin/product-availability/"+productID+"/history?"+query, nil)
	req.SetPathValue("productId", productID)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.listProductAvailabilityHistory(rec, req)
	return rec
}

func readAvailabilityHistory(t *testing.T, api *API, s scope, productID, query string) ([]availabilityHistoryItem, int) {
	t.Helper()
	rec := availabilityHistoryRequest(api, s, productID, query)
	if rec.Code != 200 {
		t.Fatalf("history: %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Items []availabilityHistoryItem
		Total int
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out.Items, out.Total
}

func TestAvailabilityReasonAndAtomicHistory(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	id := seedPortionAvailability(t, pool, s, 15, 5)
	ctx := context.Background()
	for _, reason := range []string{"", " \n\t ", strings.Repeat("á", 241)} {
		body, _ := json.Marshal(map[string]any{"status": "available", "portionQuantity": 20, "reason": reason})
		if rec := updateAvailabilityRequest(t, api, s, id, string(body)); rec.Code != 400 {
			t.Fatalf("invalid reason: %d %s", rec.Code, rec.Body.String())
		}
	}
	if rec := updateAvailabilityRequest(t, api, s, id, `{"status":"available","portionQuantity":20}`); rec.Code != 400 {
		t.Fatal("missing reason accepted")
	}
	if quota, _, _ := readDailyAvailability(t, pool, s, id); quota != 15 {
		t.Fatal("invalid reason changed quantity")
	}
	if _, total := readAvailabilityHistory(t, api, s, id, ""); total != 0 {
		t.Fatal("invalid reason left history")
	}
	if rec := updateAvailabilityRequest(t, api, s, id, `{"status":"available","portionQuantity":20,"reason":"  Producción adicional  "}`); rec.Code != 204 {
		t.Fatalf("save: %d %s", rec.Code, rec.Body.String())
	}
	items, total := readAvailabilityHistory(t, api, s, id, "")
	if total != 1 || len(items) != 1 {
		t.Fatalf("history length %d %d", total, len(items))
	}
	entry := items[0]
	if entry.Reason == nil || *entry.Reason != "Producción adicional" || entry.PreviousPortionQuantity == nil || *entry.PreviousPortionQuantity != 15 || entry.PortionQuantity == nil || *entry.PortionQuantity != 20 || entry.SoldQuantity == nil || *entry.SoldQuantity != 5 || entry.UserID == nil || *entry.UserID != s.UserID || entry.UserName != "Inventory Test" {
		t.Fatalf("incorrect snapshot: %+v", entry)
	}
	if _, err := time.Parse(time.RFC3339Nano, entry.CreatedAt); err != nil || !strings.HasSuffix(entry.CreatedAt, "Z") {
		t.Fatalf("UTC timestamp: %s", entry.CreatedAt)
	}
	// A replay of the confirmed value is a no-op, not another change.
	if rec := updateAvailabilityRequest(t, api, s, id, `{"status":"available","portionQuantity":20,"reason":"Reintento"}`); rec.Code != 204 {
		t.Fatal(rec.Body.String())
	}
	if _, total = readAvailabilityHistory(t, api, s, id, ""); total != 1 {
		t.Fatal("no-op duplicated history")
	}
	if _, err := pool.Exec(ctx, `UPDATE users SET full_name='Renamed User' WHERE id=$1`, s.UserID); err != nil {
		t.Fatal(err)
	}
	items, _ = readAvailabilityHistory(t, api, s, id, "")
	if items[0].UserName != "Inventory Test" {
		t.Fatal("historical author name changed")
	}
	function := fmt.Sprintf("availability_audit_failure_%d", time.Now().UnixNano())
	query := fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.organization_id='%s'::uuid AND NEW.action='product.availability_updated' THEN RAISE EXCEPTION 'test failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER %s BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION %s()`, function, s.OrganizationID, function, function)
	if _, err := pool.Exec(ctx, query); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON audit_log; DROP FUNCTION IF EXISTS %s()", function, function))
	})
	if rec := updateAvailabilityRequest(t, api, s, id, `{"status":"available","portionQuantity":30,"reason":"Producción adicional"}`); rec.Code != 503 {
		t.Fatalf("audit failure: %d %s", rec.Code, rec.Body.String())
	}
	if quota, _, _ := readDailyAvailability(t, pool, s, id); quota != 20 {
		t.Fatal("audit failure committed quantity")
	}
	if _, total = readAvailabilityHistory(t, api, s, id, ""); total != 1 {
		t.Fatal("audit failure left history")
	}
}

func TestAvailabilityHistoryIsolationPaginationAndLegacy(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	id := seedPortionAvailability(t, pool, s, 15, 5)
	for _, body := range []string{`{"status":"sold_out","reason":"Sin producción disponible"}`, `{"status":"available","reason":"Producción disponible"}`} {
		if rec := updateAvailabilityRequest(t, api, s, id, body); rec.Code != 204 {
			t.Fatalf("status: %d %s", rec.Code, rec.Body.String())
		}
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,created_at) VALUES($1,$2,$3,'product.availability_updated','product',$4,now()-interval '2 days')`, s.OrganizationID, s.LocationID, s.UserID, id); err != nil {
		t.Fatal(err)
	}
	items, total := readAvailabilityHistory(t, api, s, id, "page=1&pageSize=1")
	if total != 3 || len(items) != 1 || items[0].ManualStatus == nil || *items[0].ManualStatus != "available" || *items[0].PreviousManualStatus != "sold_out" || *items[0].PreviousPortionQuantity != 15 || *items[0].PortionQuantity != 15 {
		t.Fatalf("first page: %+v total=%d", items, total)
	}
	items, total = readAvailabilityHistory(t, api, s, id, "page=3&pageSize=1")
	if total != 3 || len(items) != 1 || items[0].Reason != nil || items[0].BusinessDate != nil || items[0].PreviousPortionQuantity != nil {
		t.Fatal("legacy details invented")
	}
	items, total = readAvailabilityHistory(t, api, s, id, "page=4&pageSize=1")
	if total != 3 || len(items) != 0 {
		t.Fatal("out of range page invalid")
	}
	second := s
	if err := pool.QueryRow(context.Background(), `INSERT INTO locations(organization_id,name,code) VALUES($1,'Second','HISTORY') RETURNING id`, s.OrganizationID).Scan(&second.LocationID); err != nil {
		t.Fatal(err)
	}
	if items, total = readAvailabilityHistory(t, api, second, id, ""); total != 0 || len(items) != 0 {
		t.Fatal("history crossed locations")
	}
	foreign := seedInventoryScope(t, pool)
	if rec := availabilityHistoryRequest(api, foreign, id, ""); rec.Code != 404 {
		t.Fatal("history crossed tenants")
	}
}

func TestAvailabilityConcurrentChangesHaveCompleteChain(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	id := seedPortionAvailability(t, pool, s, 15, 5)
	start := make(chan struct{})
	results := make(chan *httptest.ResponseRecorder, 2)
	for _, quota := range []int{20, 30} {
		go func(value int) {
			<-start
			results <- updateAvailabilityRequest(t, api, s, id, fmt.Sprintf(`{"status":"available","portionQuantity":%d,"reason":"Producción adicional"}`, value))
		}(quota)
	}
	close(start)
	for i := 0; i < 2; i++ {
		if rec := <-results; rec.Code != 204 {
			t.Fatalf("concurrent: %d %s", rec.Code, rec.Body.String())
		}
	}
	items, total := readAvailabilityHistory(t, api, s, id, "")
	if total != 2 || *items[1].PreviousPortionQuantity != 15 || *items[0].PreviousPortionQuantity != *items[1].PortionQuantity {
		t.Fatalf("incomplete history chain: %+v", items)
	}
	quota, sold, _ := readDailyAvailability(t, pool, s, id)
	if quota != *items[0].PortionQuantity || sold != 5 {
		t.Fatal("final state inconsistent with history")
	}
}

func TestAvailabilityHistoryRoutesRequireReadAndChangesRequireManage(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	id := seedPortionAvailability(t, pool, s, 15, 0)
	var roleID string
	if err := pool.QueryRow(ctx, `INSERT INTO roles(organization_id,name,permissions) VALUES($1,'Availability reader',ARRAY['menu.read']) RETURNING id`, s.OrganizationID).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, s.UserID, roleID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	token := "availability-history-test-" + s.UserID
	hash := sha256.Sum256([]byte(token))
	if _, err := pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,organization_id,location_id,expires_at) VALUES($1,$2,$3,$4,now()+interval '1 hour')`, hash[:], s.UserID, s.OrganizationID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, hash[:]) })
	routes := New(pool).Routes()
	call := func(method, path string, authenticated bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(`{"status":"sold_out","reason":"Sin producción"}`))
		if authenticated {
			req.AddCookie(&http.Cookie{Name: "foods_session", Value: token})
		}
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		return rec
	}
	for _, prefix := range []string{"/v1/admin/", "/v1/operations/"} {
		path := prefix + "product-availability/" + id
		if rec := call("GET", path+"/history", false); rec.Code != 401 {
			t.Fatalf("anonymous history: %d", rec.Code)
		}
		if rec := call("GET", path+"/history", true); rec.Code != 200 {
			t.Fatalf("reader history: %d %s", rec.Code, rec.Body.String())
		}
		if rec := call("PATCH", path, true); rec.Code != 403 {
			t.Fatalf("reader mutation: %d", rec.Code)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE roles SET permissions=ARRAY[]::text[] WHERE id=$1`, roleID); err != nil {
		t.Fatal(err)
	}
	if rec := call("GET", "/v1/admin/product-availability/"+id+"/history", true); rec.Code != 403 {
		t.Fatalf("without read: %d", rec.Code)
	}
}
