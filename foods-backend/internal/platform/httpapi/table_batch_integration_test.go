package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func postTableBatch(api *API, s scope, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/v1/admin/tables/batch", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.createTablesBatch(rec, req)
	return rec
}

func TestTableBatchPersistsOrderedTablesAndAtomicAudit(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO zones(organization_id,location_id,name) VALUES($1,$2,'Principal')`, s.OrganizationID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM zones WHERE organization_id=$1`, s.OrganizationID) })
	items := make([]tableBatchItem, 50)
	for i := range items {
		items[i] = tableBatchItem{Name: fmt.Sprintf(" Mesa %02d ", 50-i), Seats: 4, Zone: " Principal ", QrEnabled: i%2 == 0}
	}
	body, _ := json.Marshal(map[string]any{"items": items})
	started := time.Now()
	rec := postTableBatch(New(pool), s, string(body))
	t.Logf("50 tables + QR + audit: %s", time.Since(started))
	if rec.Code != 201 {
		t.Fatalf("batch: %d %s", rec.Code, rec.Body.String())
	}
	var result struct {
		Items []table `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil || len(result.Items) != 50 {
		t.Fatalf("batch result: %s err=%v", rec.Body.String(), err)
	}
	seen := map[string]bool{}
	for i, item := range result.Items {
		if item.ID == "" || item.Name != strings.TrimSpace(items[i].Name) || item.Zone != "Principal" || item.Seats != 4 || !item.Active || item.QrEnabled != items[i].QrEnabled || len(item.QrToken) != 32 || seen[item.QrToken] {
			t.Fatalf("unexpected item %d: %+v", i, item)
		}
		seen[item.QrToken] = true
	}
	var tables, audits, auditedIDs int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM tables WHERE organization_id=$1 AND location_id=$2`, s.OrganizationID, s.LocationID).Scan(&tables); err != nil || tables != 50 {
		t.Fatalf("persisted count=%d err=%v", tables, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*),COALESCE(sum(jsonb_array_length(metadata->'tableIds')),0)::integer FROM audit_log WHERE organization_id=$1 AND location_id=$2 AND user_id=$3 AND action='table.batch_created' AND entity_id IS NULL`, s.OrganizationID, s.LocationID, s.UserID).Scan(&audits, &auditedIDs); err != nil || audits != 1 || auditedIDs != 50 {
		t.Fatalf("audit count=%d ids=%d err=%v", audits, auditedIDs, err)
	}
}

func TestTableBatchRejectsInvalidZonesAndConflictsWithoutPartialWrites(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	var otherLocal string
	if err := pool.QueryRow(ctx, `INSERT INTO locations(organization_id,name,code,address) VALUES($1,'Otro','OTHER','') RETURNING id`, s.OrganizationID).Scan(&otherLocal); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO zones(organization_id,location_id,name,active) VALUES($1,$2,'Inactiva',false),($1,$3,'Otro local',true)`, s.OrganizationID, s.LocationID, otherLocal); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM zones WHERE organization_id=$1`, s.OrganizationID) })
	api := New(pool)
	for _, zone := range []string{"No existe", "Inactiva", "Otro local"} {
		rec := postTableBatch(api, s, fmt.Sprintf(`{"items":[{"name":"Válida"},{"name":"Inválida","zone":%q}]}`, zone))
		if rec.Code != 400 || !strings.Contains(rec.Body.String(), "invalid_zone") {
			t.Fatalf("zone %s: %d %s", zone, rec.Code, rec.Body.String())
		}
	}
	if rec := postTableBatch(api, s, `{"items":[{"name":"Existente"}]}`); rec.Code != 201 {
		t.Fatalf("seed batch: %d %s", rec.Code, rec.Body.String())
	}
	rec := postTableBatch(api, s, `{"items":[{"name":"Nueva"},{"name":"Existente"}]}`)
	if rec.Code != 409 || !strings.Contains(rec.Body.String(), "table_conflict") {
		t.Fatalf("conflict: %d %s", rec.Code, rec.Body.String())
	}
	var count, audits int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM tables WHERE organization_id=$1`, s.OrganizationID).Scan(&count)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE organization_id=$1 AND action='table.batch_created'`, s.OrganizationID).Scan(&audits)
	if count != 1 || audits != 1 {
		t.Fatalf("partial batch/audit saved: tables=%d audits=%d", count, audits)
	}
	otherScope := s
	otherScope.LocationID = otherLocal
	if rec := postTableBatch(api, otherScope, `{"items":[{"name":"Existente","zone":"Otro local"}]}`); rec.Code != 201 {
		t.Fatalf("same name in different local: %d %s", rec.Code, rec.Body.String())
	}
}

func TestTableBatchAuditFailureRollsBackTables(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	// Abort only this test tenant's audit; tests run against temporary PostgreSQL.
	function := fmt.Sprintf("batch_audit_guard_%d", time.Now().UnixNano())
	query := fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.organization_id='%s'::uuid AND NEW.action='table.batch_created' THEN RAISE EXCEPTION 'test audit failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER %s BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION %s()`, function, s.OrganizationID, function, function)
	if _, err := pool.Exec(ctx, query); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON audit_log; DROP FUNCTION IF EXISTS %s()", function, function))
	})
	rec := postTableBatch(New(pool), s, `{"items":[{"name":"Rollback 1"},{"name":"Rollback 2"}]}`)
	if rec.Code != 503 || !strings.Contains(rec.Body.String(), "tables_unavailable") {
		t.Fatalf("audit failure: %d %s", rec.Code, rec.Body.String())
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM tables WHERE organization_id=$1`, s.OrganizationID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("tables committed without audit: count=%d err=%v", count, err)
	}
}

func TestTableBatchConcurrentDuplicateIsAtomic(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	var wg sync.WaitGroup
	responses := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			responses <- postTableBatch(api, s, `{"items":[{"name":"Concurrente 1"},{"name":"Concurrente 2"}]}`).Code
		}()
	}
	wg.Wait()
	close(responses)
	statuses := map[int]int{}
	for status := range responses {
		statuses[status]++
	}
	var count, audits int
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM tables WHERE organization_id=$1`, s.OrganizationID).Scan(&count)
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_log WHERE organization_id=$1 AND action='table.batch_created'`, s.OrganizationID).Scan(&audits)
	if statuses[201] != 1 || statuses[409] != 1 || count != 2 || audits != 1 {
		t.Fatalf("concurrent batch: statuses=%v tables=%d audits=%d", statuses, count, audits)
	}
}

func TestTableBatchRequiresPermissionOnRealRoute(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	var roleID string
	if err := pool.QueryRow(ctx, `INSERT INTO roles(organization_id,name,permissions) VALUES($1,'Batch reader',ARRAY['tables.read']) RETURNING id`, s.OrganizationID).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, s.UserID, roleID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	token := "table-batch-test-" + s.UserID
	hash := sha256.Sum256([]byte(token))
	if _, err := pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,organization_id,location_id,expires_at) VALUES($1,$2,$3,$4,now()+interval '1 hour')`, hash[:], s.UserID, s.OrganizationID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, hash[:]) })
	routes := New(pool).Routes()
	call := func(authenticated bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/v1/admin/tables/batch", strings.NewReader(`{"items":[{"name":"Con permiso"}]}`))
		if authenticated {
			req.AddCookie(&http.Cookie{Name: "foods_session", Value: token})
		}
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		return rec
	}
	if rec := call(false); rec.Code != 401 {
		t.Fatalf("anonymous batch: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(true); rec.Code != 403 {
		t.Fatalf("read-only role: %d %s", rec.Code, rec.Body.String())
	}
	if _, err := pool.Exec(ctx, `UPDATE roles SET permissions=ARRAY['tables.read','tables.manage'] WHERE id=$1`, roleID); err != nil {
		t.Fatal(err)
	}
	if rec := call(true); rec.Code != 201 {
		t.Fatalf("delegated table permission: %d %s", rec.Code, rec.Body.String())
	}
}

func TestTableBatchCancelledDuringAuditDoesNotCommit(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	function := fmt.Sprintf("batch_audit_delay_%d", time.Now().UnixNano())
	query := fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.organization_id='%s'::uuid AND NEW.action='table.batch_created' THEN PERFORM pg_sleep(2); END IF; RETURN NEW; END $$; CREATE TRIGGER %s BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION %s()`, function, s.OrganizationID, function, function)
	if _, err := pool.Exec(ctx, query); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON audit_log; DROP FUNCTION IF EXISTS %s()", function, function))
	})
	requestCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest("POST", "/v1/admin/tables/batch", strings.NewReader(`{"items":[{"name":"Timeout"}]}`))
	req = req.WithContext(context.WithValue(requestCtx, scopeKey{}, s))
	rec := httptest.NewRecorder()
	New(pool).createTablesBatch(rec, req)
	if rec.Code != 503 {
		t.Fatalf("cancelled batch: %d %s", rec.Code, rec.Body.String())
	}
	var tables, audits int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM tables WHERE organization_id=$1),(SELECT count(*) FROM audit_log WHERE organization_id=$1 AND action='table.batch_created')`, s.OrganizationID).Scan(&tables, &audits); err != nil || tables != 0 || audits != 0 {
		t.Fatalf("cancelled batch committed: tables=%d audits=%d err=%v", tables, audits, err)
	}
}
