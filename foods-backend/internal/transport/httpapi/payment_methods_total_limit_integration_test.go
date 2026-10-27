package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestPaymentMethodsTotalIncludesInactiveAndIgnoresFilters(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE payment_methods SET active=false WHERE organization_id=$1 AND code='plin'`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"?q=Plin&status=inactive&page=1&pageSize=1", "?q=no-match&page=9&pageSize=1"} {
		req := httptest.NewRequest("GET", "/v1/admin/payment-methods"+query, nil)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		api.listPaymentMethodsAdmin(rec, req)
		var page struct {
			Total        int `json:"total"`
			CatalogTotal int `json:"catalogTotal"`
			CatalogLimit int `json:"catalogLimit"`
		}
		if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &page) != nil || page.CatalogTotal != 4 || page.CatalogLimit != 4 {
			t.Fatalf("catalog capacity depends on filters: %d %s", rec.Code, rec.Body.String())
		}
	}
	req := httptest.NewRequest("POST", "/v1/admin/payment-methods", strings.NewReader(`{"code":"voucher","name":"Vale","salesEnabled":true,"sortOrder":50}`))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.createPaymentMethod(rec, req)
	if rec.Code != 409 || !bytes.Contains(rec.Body.Bytes(), []byte("payment_method_limit")) {
		t.Fatalf("inactive entry freed a slot: %d %s", rec.Code, rec.Body.String())
	}
	var total int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM payment_methods WHERE organization_id=$1`, s.OrganizationID).Scan(&total); err != nil || total != 4 {
		t.Fatalf("rejected creation changed catalog: %d %v", total, err)
	}
}

func TestPaymentMethodsConcurrentCreationStopsAtFourTotal(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM payment_methods WHERE organization_id=$1 AND code IN ('yape','plin')`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan *httptest.ResponseRecorder, 3)
	var work sync.WaitGroup
	for i := 0; i < 3; i++ {
		work.Add(1)
		go func(i int) {
			defer work.Done()
			<-start
			req := httptest.NewRequest("POST", "/v1/admin/payment-methods", strings.NewReader(fmt.Sprintf(`{"code":"new_%d","name":"Medio %d","salesEnabled":true,"sortOrder":50}`, i, i)))
			req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
			rec := httptest.NewRecorder()
			api.createPaymentMethod(rec, req)
			results <- rec
		}(i)
	}
	close(start)
	work.Wait()
	close(results)
	created, rejected := 0, 0
	for rec := range results {
		switch rec.Code {
		case 201:
			created++
		case 409:
			if !bytes.Contains(rec.Body.Bytes(), []byte("payment_method_limit")) {
				t.Fatalf("unexpected conflict: %s", rec.Body.String())
			}
			rejected++
		default:
			t.Fatalf("concurrent creation: %d %s", rec.Code, rec.Body.String())
		}
	}
	var total int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM payment_methods WHERE organization_id=$1`, s.OrganizationID).Scan(&total); err != nil || total != 4 || created != 2 || rejected != 1 {
		t.Fatalf("total=%d created=%d rejected=%d error=%v", total, created, rejected, err)
	}
}
