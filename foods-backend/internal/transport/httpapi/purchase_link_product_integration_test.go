package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

// Un producto que vendía con cupo del día se vincula al inventario sin duplicarse:
// conserva su ficha e historial, pasa a Inventario físico y empieza en cero.
func TestPurchaseLinksExistingPortionProductWithoutDuplicating(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	api := New(pool)
	nonce := time.Now().UnixNano()

	var categoryID, productID string
	if err := pool.QueryRow(ctx, `INSERT INTO menu_categories(organization_id,name,product_scope) VALUES($1,$2,'prepared') RETURNING id`,
		s.OrganizationID, fmt.Sprintf("Bebidas %d", nonce)).Scan(&categoryID); err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("Inca Kola vinculable %d", nonce)
	if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,category_id,sku,name,price,quantity_control) VALUES($1,$2,$3,$4,5,'portions') RETURNING id`,
		s.OrganizationID, categoryID, fmt.Sprintf("LINK-%d", nonce), name).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO product_availability(organization_id,location_id,product_id,business_date,portion_quantity,sold_quantity) VALUES($1,$2,$3,current_date,24,5)`,
		s.OrganizationID, s.LocationID, productID); err != nil {
		t.Fatal(err)
	}

	listed := func() bool {
		req := httptest.NewRequest("GET", "/v1/admin/purchase-linkable-products?q="+fmt.Sprint(nonce), nil)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		api.listLinkableProducts(rec, req)
		if rec.Code != 200 {
			t.Fatalf("linkable list: %d %s", rec.Code, rec.Body.String())
		}
		var out struct {
			Items []struct {
				ProductID string `json:"productId"`
			} `json:"items"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		for _, item := range out.Items {
			if item.ProductID == productID {
				return true
			}
		}
		return false
	}
	if !listed() {
		t.Fatal("a portions product must be offered for linking")
	}

	body := []byte(fmt.Sprintf(`{"productId":%q,"unit":"botella","presentations":[{"presentationType":"unit","unitsPerPresentation":1,"isDefault":true}],"minimumStock":6}`, productID))
	req := httptest.NewRequest("POST", "/v1/admin/purchase-inventory-items", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.createPurchaseInventoryItem(rec, req)
	if rec.Code != 201 {
		t.Fatalf("expected link 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var linked inventoryProductOption
	if err := json.Unmarshal(rec.Body.Bytes(), &linked); err != nil {
		t.Fatal(err)
	}
	if linked.ProductID == nil || *linked.ProductID != productID || linked.Quantity != "0" {
		t.Fatalf("link must reuse the product at zero stock: %#v", linked)
	}

	var products int
	var control, productType, scope string
	var movements, availability int
	if err := pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM products WHERE organization_id=$1 AND name=$2),
		       p.quantity_control,p.product_type,c.product_scope,
		       (SELECT count(*) FROM stock_movements WHERE organization_id=$1 AND product_id=$3),
		       (SELECT count(*) FROM product_availability WHERE organization_id=$1 AND product_id=$3)
		FROM products p JOIN menu_categories c ON c.id=p.category_id
		WHERE p.id=$3`, s.OrganizationID, name, productID).Scan(&products, &control, &productType, &scope, &movements, &availability); err != nil {
		t.Fatal(err)
	}
	if products != 1 || control != "inventory" || productType != "retail" || scope != "both" || movements != 0 || availability != 1 {
		t.Fatalf("unexpected linked state products=%d control=%s type=%s scope=%s movements=%d availability=%d", products, control, productType, scope, movements, availability)
	}
	if listed() {
		t.Fatal("a linked product must leave the linkable list")
	}
}
