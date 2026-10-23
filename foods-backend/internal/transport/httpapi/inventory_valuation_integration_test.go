package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// scopeForInventory carries the seeded scope with its pool for item fixtures.
type scopeForInventory struct {
	scope
	pool *pgxpool.Pool
}

func inventoryValuationItem(t *testing.T, s scopeForInventory, name string, quantity, average float64) string {
	t.Helper()
	var id string
	if err := s.pool.QueryRow(context.Background(), `
		INSERT INTO inventory_items(organization_id,sku,name,unit,minimum_stock,active)
		VALUES($1,$2,$3,'kg',0,true) RETURNING id`,
		s.OrganizationID, fmt.Sprintf("VAL-%d", time.Now().UnixNano()), name).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(context.Background(), `
		INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity,average_unit_cost)
		VALUES($1,$2,$3,$4,$5)`, s.OrganizationID, s.LocationID, id, quantity, average); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestInventoryEntryAdjustmentBringsItsCostIntoTheAverage(t *testing.T) {
	pool := integrationPool(t)
	s := scopeForInventory{seedInventoryScope(t, pool), pool}
	api := New(pool)
	ctx := context.Background()
	itemID := inventoryValuationItem(t, s, "Harina", 10, 2)

	adjust := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/v1/admin/inventory/adjustments", bytes.NewReader([]byte(body)))
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s.scope))
		rec := httptest.NewRecorder()
		api.createInventoryAdjustment(rec, req)
		return rec
	}
	if rec := adjust(fmt.Sprintf(`{"inventoryItemId":%q,"movementType":"entry","reason":"surplus_adjustment","quantity":10,"unitCost":4}`, itemID)); rec.Code != 201 {
		t.Fatalf("entry adjustment with cost: %d %s", rec.Code, rec.Body.String())
	}
	var average, unitCost, valueDelta, balanceValue float64
	if err := pool.QueryRow(ctx, `SELECT average_unit_cost::float8 FROM stock_balances WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`, s.OrganizationID, s.LocationID, itemID).Scan(&average); err != nil || average != 3 {
		t.Fatalf("average must blend 10 kg at 2 with 10 kg at 4: %v %v", average, err)
	}
	if err := pool.QueryRow(ctx, `SELECT unit_cost::float8,value_delta::float8,balance_value_after::float8 FROM stock_movements WHERE organization_id=$1 AND inventory_item_id=$2`, s.OrganizationID, itemID).Scan(&unitCost, &valueDelta, &balanceValue); err != nil || unitCost != 4 || valueDelta != 40 || balanceValue != 60 {
		t.Fatalf("Kárdex must value the entry at its cost: cost=%v delta=%v balance=%v err=%v", unitCost, valueDelta, balanceValue, err)
	}

	withoutAverage := inventoryValuationItem(t, s, "Sal", 0, 0)
	if rec := adjust(fmt.Sprintf(`{"inventoryItemId":%q,"movementType":"entry","reason":"surplus_adjustment","quantity":1}`, withoutAverage)); rec.Code != 400 || !bytes.Contains(rec.Body.Bytes(), []byte("unit_cost_required")) {
		t.Fatalf("an item without average needs a cost: %d %s", rec.Code, rec.Body.String())
	}
}

func TestInventorySearchTreatsWildcardsLiterally(t *testing.T) {
	pool := integrationPool(t)
	s := scopeForInventory{seedInventoryScope(t, pool), pool}
	api := New(pool)
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	literal := inventoryValuationItem(t, s, "Cacao 50% "+suffix, 1, 1)
	inventoryValuationItem(t, s, "Cacao 500 "+suffix, 1, 1)

	req := httptest.NewRequest("GET", "/v1/admin/inventory/products?q="+url.QueryEscape("50% "+suffix), nil)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s.scope))
	rec := httptest.NewRecorder()
	api.listInventoryProducts(rec, req)
	var page struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil || rec.Code != 200 {
		t.Fatalf("search: %d %s %v", rec.Code, rec.Body.String(), err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != literal {
		t.Fatalf("%% must not act as a wildcard: %#v", page.Items)
	}
}

func TestKardexKeepsTheOrderOfMovementsFromOneTransaction(t *testing.T) {
	pool := integrationPool(t)
	s := scopeForInventory{seedInventoryScope(t, pool), pool}
	api := New(pool)
	ctx := context.Background()
	itemID := inventoryValuationItem(t, s, "Aceite", 0, 0)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	// Both rows share now(); only the registration sequence can order them.
	for _, balance := range []int{1, 2, 3} {
		if _, err = tx.Exec(ctx, `
			INSERT INTO stock_movements(organization_id,location_id,inventory_item_id,movement_type,quantity_delta,balance_after,source_type,source_id)
			VALUES($1,$2,$3,'inventory_adjustment',1,$4,'inventory_adjustment',gen_random_uuid())`,
			s.OrganizationID, s.LocationID, itemID, balance); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/v1/admin/inventory/movements?inventoryItemId="+itemID, nil)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s.scope))
	rec := httptest.NewRecorder()
	api.listInventoryMovements(rec, req)
	var page struct {
		Items []struct {
			BalanceAfter string `json:"balanceAfter"`
		} `json:"items"`
	}
	if err = json.Unmarshal(rec.Body.Bytes(), &page); err != nil || len(page.Items) != 3 {
		t.Fatalf("movements: %d %s %v", rec.Code, rec.Body.String(), err)
	}
	for index, want := range []string{"3.000", "2.000", "1.000"} {
		if page.Items[index].BalanceAfter != want {
			t.Fatalf("Kárdex must list the latest registration first: %#v", page.Items)
		}
	}
}
