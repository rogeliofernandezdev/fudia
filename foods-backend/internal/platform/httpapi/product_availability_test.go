package httpapi

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func seedPortionAvailability(t *testing.T, pool *pgxpool.Pool, s scope, quota, sold int) string {
	t.Helper()
	ctx := context.Background()
	var productID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO products(organization_id,sku,name,price,quantity_control)
		VALUES($1,$2,$3,'12.00','portions')
		RETURNING id`,
		s.OrganizationID,
		fmt.Sprintf("PORT-%d", time.Now().UnixNano()),
		fmt.Sprintf("Producto porciones %d", time.Now().UnixNano()),
	).Scan(&productID); err != nil {
		t.Fatalf("seed portion product: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO product_availability(
			organization_id,location_id,product_id,business_date,
			portion_quantity,sold_quantity,manual_status,note,updated_by
		)
		SELECT $1,$2,$3,(now() AT TIME ZONE timezone)::date,$4,$5,'available','',$6
		FROM locations
		WHERE id=$2 AND organization_id=$1`,
		s.OrganizationID, s.LocationID, productID, quota, sold, s.UserID,
	); err != nil {
		t.Fatalf("seed product availability: %v", err)
	}
	return productID
}

func updateAvailabilityRequest(t *testing.T, api *API, s scope, productID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("PATCH", "/v1/admin/product-availability/"+productID, bytes.NewBufferString(body))
	req.SetPathValue("productId", productID)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.updateProductAvailability(rec, req)
	return rec
}

func readDailyAvailability(t *testing.T, pool *pgxpool.Pool, s scope, productID string) (int, int, string) {
	t.Helper()
	var quota, sold int
	var status string
	if err := pool.QueryRow(context.Background(), `
		SELECT portion_quantity,sold_quantity,manual_status
		FROM product_availability pa
		JOIN locations l ON l.id=pa.location_id AND l.organization_id=pa.organization_id
		WHERE pa.organization_id=$1 AND pa.location_id=$2 AND pa.product_id=$3
		  AND pa.business_date=(now() AT TIME ZONE l.timezone)::date`,
		s.OrganizationID, s.LocationID, productID,
	).Scan(&quota, &sold, &status); err != nil {
		t.Fatalf("read product availability: %v", err)
	}
	return quota, sold, status
}

func TestUpdateProductAvailabilityRejectsQuotaBelowSold(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	productID := seedPortionAvailability(t, pool, s, 10, 5)

	rec := updateAvailabilityRequest(t, api, s, productID, `{"status":"available","portionQuantity":4,"note":""}`)
	if rec.Code != 409 {
		t.Fatalf("expected 409, got %d body=%s", rec.Code, rec.Body.String())
	}
	quota, sold, status := readDailyAvailability(t, pool, s, productID)
	if quota != 10 || sold != 5 || status != "available" {
		t.Fatalf("availability changed after rejected quota: quota=%d sold=%d status=%q", quota, sold, status)
	}
}

func TestUpdateProductAvailabilityStatusOnlyPreservesQuota(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	productID := seedPortionAvailability(t, pool, s, 10, 5)

	rec := updateAvailabilityRequest(t, api, s, productID, `{"status":"sold_out","portionQuantity":null,"note":""}`)
	if rec.Code != 204 {
		t.Fatalf("expected sold_out 204, got %d body=%s", rec.Code, rec.Body.String())
	}
	quota, sold, status := readDailyAvailability(t, pool, s, productID)
	if quota != 10 || sold != 5 || status != "sold_out" {
		t.Fatalf("status update modified quota unexpectedly: quota=%d sold=%d status=%q", quota, sold, status)
	}

	rec = updateAvailabilityRequest(t, api, s, productID, `{"status":"available","portionQuantity":null,"note":""}`)
	if rec.Code != 204 {
		t.Fatalf("expected reactivate 204, got %d body=%s", rec.Code, rec.Body.String())
	}
	quota, sold, status = readDailyAvailability(t, pool, s, productID)
	if quota != 10 || sold != 5 || status != "available" {
		t.Fatalf("reactivation modified quota unexpectedly: quota=%d sold=%d status=%q", quota, sold, status)
	}
}

func TestUpdateProductAvailabilityAcceptsQuotaAtOrAboveSold(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	productID := seedPortionAvailability(t, pool, s, 10, 5)

	rec := updateAvailabilityRequest(t, api, s, productID, `{"status":"available","portionQuantity":7,"note":"Cupo ajustado"}`)
	if rec.Code != 204 {
		t.Fatalf("expected 204, got %d body=%s", rec.Code, rec.Body.String())
	}
	quota, sold, status := readDailyAvailability(t, pool, s, productID)
	if quota != 7 || sold != 5 || status != "available" {
		t.Fatalf("unexpected updated availability: quota=%d sold=%d status=%q", quota, sold, status)
	}
}
