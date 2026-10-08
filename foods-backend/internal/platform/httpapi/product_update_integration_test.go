package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestProductUpdatePersistsCommercialDataAndDestination(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	for i, destination := range []string{"kitchen", "bar", "direct"} {
		t.Run(destination, func(t *testing.T) {
			var id string
			sku := fmt.Sprintf("EDIT-%d", i)
			if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price,quantity_control) VALUES($1,$2,'Original',5,'none') RETURNING id`, s.OrganizationID, sku).Scan(&id); err != nil {
				t.Fatal(err)
			}
			active, featured, minutes := false, true, 8
			image, cost, from, until, cutoff := "/test-product.png", "2.50", "2026-01-01 00:00:00+00", "2030-12-31 00:00:00+00", "18:00:00"
			input := productInput{
				SKU: sku, Name: "Producto actualizado", Description: "Descripción actualizada", Price: "7.50",
				Active: &active, Featured: &featured, ProductType: "prepared", QuantityControl: "none",
				ServiceDestination: destination, ImageURL: &image, CostPrice: &cost, PrepMinutes: &minutes,
				Allergens: []string{"leche"}, AvailableFrom: &from, AvailableUntil: &until,
				AvailableDays: []int{1, 3}, AvailableUntilTime: &cutoff,
			}
			body, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest("PATCH", "/v1/admin/products/"+id, bytes.NewReader(body))
			req.SetPathValue("id", id)
			req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
			rec := httptest.NewRecorder()
			api.updateProduct(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("update %s: %d %s", destination, rec.Code, rec.Body.String())
			}
			var updated product
			if err = json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
				t.Fatal(err)
			}
			if updated.ID != id || updated.Name != input.Name || updated.Price != "7.50" || updated.ServiceDestination != destination {
				t.Fatalf("incorrect update response: %+v", updated)
			}
			// Check the committed values, not only the HTTP response.
			var stored productInput
			if err = pool.QueryRow(ctx, `SELECT sku,name,description,price::text,active,featured,product_type,quantity_control,service_destination,image_url,cost_price::text,prep_minutes,allergens,available_from::text,available_until::text,available_days,available_until_time::text FROM products WHERE organization_id=$1 AND id=$2`, s.OrganizationID, id).Scan(
				&stored.SKU, &stored.Name, &stored.Description, &stored.Price, &stored.Active, &stored.Featured,
				&stored.ProductType, &stored.QuantityControl, &stored.ServiceDestination, &stored.ImageURL,
				&stored.CostPrice, &stored.PrepMinutes, &stored.Allergens, &stored.AvailableFrom, &stored.AvailableUntil,
				&stored.AvailableDays, &stored.AvailableUntilTime,
			); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(stored, input) {
				actual, _ := json.Marshal(stored)
				t.Fatalf("commercial fields not persisted: got %s want %s", actual, body)
			}
			var audits int
			if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE organization_id=$1 AND user_id=$2 AND entity_id=$3 AND action='product.updated'`, s.OrganizationID, s.UserID, id).Scan(&audits); err != nil || audits != 1 {
				t.Fatalf("expected one update audit, got %d err=%v", audits, err)
			}
		})
	}
}

func TestProductUpdatePreservesOmittedDestinationAndTenantBoundary(t *testing.T) {
	pool := integrationPool(t)
	owner, other := seedInventoryScope(t, pool), seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	var id string
	if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price,product_type,quantity_control,service_destination) VALUES($1,'EDIT-LEGACY','Original',5,'retail','none','direct') RETURNING id`, owner.OrganizationID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	update := func(s scope, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("PATCH", "/v1/admin/products/"+id, bytes.NewBufferString(body))
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		api.updateProduct(rec, req)
		return rec
	}
	foreign := update(other, `{"name":"No permitido","price":"9.00","quantityControl":"none"}`)
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("foreign tenant update: %d %s", foreign.Code, foreign.Body.String())
	}
	invalid := update(owner, `{"name":"No permitido","price":"9.00","serviceDestination":"invalid","quantityControl":"none"}`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid destination: %d %s", invalid.Code, invalid.Body.String())
	}
	saved := update(owner, `{"name":"Nombre actualizado","price":"6.00","quantityControl":"none","allergens":[]}`)
	if saved.Code != http.StatusOK {
		t.Fatalf("update without destination: %d %s", saved.Code, saved.Body.String())
	}
	var name, destination, productType string
	if err := pool.QueryRow(ctx, `SELECT name,service_destination,product_type FROM products WHERE id=$1`, id).Scan(&name, &destination, &productType); err != nil {
		t.Fatal(err)
	}
	if name != "Nombre actualizado" || destination != "direct" || productType != "retail" {
		t.Fatalf("omitted fields were not preserved: %s %s %s", name, destination, productType)
	}
}
