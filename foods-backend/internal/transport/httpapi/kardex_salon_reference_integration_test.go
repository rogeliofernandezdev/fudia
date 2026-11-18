package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

// El Kárdex muestra la referencia legible del pedido; el nombre de la mesa
// suele traer su propio prefijo («Mesa 1»), que no debe duplicarse.
func TestKardexDoesNotDuplicateTheMesaPrefixInSalonReferences(t *testing.T) {
	pool := integrationPool(t)
	s := scopeForInventory{seedInventoryScope(t, pool), pool}
	api := New(pool)
	ctx := context.Background()
	itemID := inventoryValuationItem(t, s, "Carne kardex", 0, 0)
	nonce := time.Now().UnixNano()

	cases := map[string]string{ // nombre de la mesa → motivo esperado
		fmt.Sprintf("Mesa K %d", nonce): fmt.Sprintf("Pedido · Mesa K %d", nonce),
		fmt.Sprintf("01 K %d", nonce):   fmt.Sprintf("Pedido · Mesa 01 K %d", nonce),
	}
	for tableName := range cases {
		var tableID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO tables(organization_id,location_id,name,seats,zone,qr_token)
			VALUES($1,$2,$3,2,'Principal',md5($3))
			RETURNING id`,
			s.OrganizationID, s.LocationID, tableName).Scan(&tableID); err != nil {
			t.Fatal(err)
		}
		var orderID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO orders(organization_id,location_id,channel,table_id,status)
			VALUES($1,$2,'salon',$3,'nuevo')
			RETURNING id`,
			s.OrganizationID, s.LocationID, tableID).Scan(&orderID); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO stock_movements(organization_id,location_id,inventory_item_id,movement_type,quantity_delta,balance_after,source_type,source_id)
			VALUES($1,$2,$3,'inventory_adjustment',-1,0,'order',$4)`,
			s.OrganizationID, s.LocationID, itemID, orderID); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest("GET", "/v1/admin/inventory/movements?inventoryItemId="+itemID, nil)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s.scope))
	rec := httptest.NewRecorder()
	api.listInventoryMovements(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected kardex 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var page struct {
		Items []struct {
			SourceReference string `json:"sourceReference"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != len(cases) {
		t.Fatalf("expected %d kardex movements, got %d: %#v", len(cases), len(page.Items), page.Items)
	}
	seen := map[string]bool{}
	for _, item := range page.Items {
		seen[item.SourceReference] = true
	}
	for _, want := range cases {
		if !seen[want] {
			t.Fatalf("expected kardex reference %q, got %v", want, seen)
		}
	}
}
