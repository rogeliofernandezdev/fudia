package httpapi

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no configurado")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Fatalf("ping integration database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func seedInventoryScope(t *testing.T, pool *pgxpool.Pool) scope {
	t.Helper()
	ctx := context.Background()
	nonce := time.Now().UnixNano()
	taxID := fmt.Sprintf("%011d", nonce%100000000000)
	var organizationID, locationID, userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO organizations(legal_name,trade_name,tax_id)
		VALUES($1,$1,$2)
		RETURNING id`, fmt.Sprintf("Inventory Test %d", nonce), taxID).Scan(&organizationID); err != nil {
		t.Fatalf("seed organization: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO locations(organization_id,name,code,address)
		VALUES($1,'Principal',$2,'')
		RETURNING id`, organizationID, fmt.Sprintf("T%d", nonce%1000000)).Scan(&locationID); err != nil {
		t.Fatalf("seed location: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users(organization_id,email,full_name,password_hash)
		VALUES($1,$2,'Inventory Test','test')
		RETURNING id`, organizationID, fmt.Sprintf("inventory-%d@example.test", nonce)).Scan(&userID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	s := scope{OrganizationID: organizationID, LocationID: locationID, UserID: userID, Name: "Inventory Test"}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM stock_movements WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM inventory_entries WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM order_item_combo_selections WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM order_items WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM orders WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM product_availability WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM stock_balances WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM inventory_items WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM products WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM audit_log WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM organizations WHERE id=$1`, organizationID)
	})
	return s
}

func TestCreateInventoryEntryRollsBackNewProductOnLateFailure(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)

	// Fuerza un error tardío: el UUID es válido pero no corresponde a un usuario.
	badScope := s
	badScope.UserID = "00000000-0000-0000-0000-000000000001"
	name := fmt.Sprintf("Atomic Cola %d", time.Now().UnixNano())
	body := []byte(fmt.Sprintf(`{"newProduct":{"name":%q,"price":"5.00"},"quantity":12,"unit":"botella","minimumStock":2,"note":"primera entrada"}`, name))
	req := httptest.NewRequest("POST", "/v1/admin/inventory/entries", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, badScope))
	rec := httptest.NewRecorder()

	api.createInventoryEntry(rec, req)
	if rec.Code < 500 {
		t.Fatalf("expected late database failure, got status %d body=%s", rec.Code, rec.Body.String())
	}

	var productCount, itemCount, balanceCount int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM products WHERE organization_id=$1 AND name=$2`, s.OrganizationID, name).Scan(&productCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM inventory_items WHERE organization_id=$1 AND name=$2`, s.OrganizationID, name).Scan(&itemCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*)
		FROM stock_balances sb
		JOIN inventory_items ii ON ii.id=sb.inventory_item_id AND ii.organization_id=sb.organization_id
		WHERE sb.organization_id=$1 AND ii.name=$2`, s.OrganizationID, name).Scan(&balanceCount); err != nil {
		t.Fatal(err)
	}
	if productCount != 0 || itemCount != 0 || balanceCount != 0 {
		t.Fatalf("rollback left residues product=%d item=%d balance=%d", productCount, itemCount, balanceCount)
	}
}

func TestConcurrentInventorySalesCannotGoNegative(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()

	var productID, inventoryItemID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO products(organization_id,sku,name,price,quantity_control)
		VALUES($1,$2,'Última botella',5,'inventory')
		RETURNING id`, s.OrganizationID, fmt.Sprintf("P-%d", time.Now().UnixNano())).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO inventory_items(organization_id,sku,name,unit,minimum_stock,product_id)
		VALUES($1,$2,'Última botella','botella',0,$3)
		RETURNING id`, s.OrganizationID, fmt.Sprintf("I-%d", time.Now().UnixNano()), productID).Scan(&inventoryItemID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity)
		VALUES($1,$2,$3,1)`, s.OrganizationID, s.LocationID, inventoryItemID); err != nil {
		t.Fatal(err)
	}

	orderIDs := make([]string, 2)
	for i := range orderIDs {
		if err := pool.QueryRow(ctx, `
			INSERT INTO orders(organization_id,location_id,channel,created_by)
			VALUES($1,$2,'mostrador',$3)
			RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&orderIDs[i]); err != nil {
			t.Fatal(err)
		}
	}

	api := New(pool)
	type result struct {
		err *orderPreparationError
	}
	results := make(chan result, 2)
	var start sync.WaitGroup
	start.Add(1)
	var workers sync.WaitGroup
	workers.Add(2)
	for _, orderID := range orderIDs {
		orderID := orderID
		go func() {
			defer workers.Done()
			start.Wait()
			tx, err := pool.Begin(ctx)
			if err != nil {
				results <- result{err: &orderPreparationError{Status: 500, Code: "begin_failed", Message: err.Error()}}
				return
			}
			defer tx.Rollback(ctx)
			quantityErr := api.applyOrderQuantityDelta(ctx, tx, s, orderID, map[string]float64{productID: 1}, "sale")
			if quantityErr == nil {
				if err := tx.Commit(ctx); err != nil {
					quantityErr = &orderPreparationError{Status: 500, Code: "commit_failed", Message: err.Error()}
				}
			}
			results <- result{err: quantityErr}
		}()
	}
	start.Done()
	workers.Wait()
	close(results)

	successes := 0
	insufficient := 0
	for result := range results {
		if result.err == nil {
			successes++
			continue
		}
		if result.err.Code == "insufficient_stock" {
			insufficient++
			continue
		}
		t.Fatalf("unexpected concurrent sale error: %#v", result.err)
	}
	if successes != 1 || insufficient != 1 {
		t.Fatalf("expected one sale and one insufficient_stock, got successes=%d insufficient=%d", successes, insufficient)
	}

	var balance float64
	if err := pool.QueryRow(ctx, `
		SELECT quantity::float8 FROM stock_balances
		WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,
		s.OrganizationID, s.LocationID, inventoryItemID).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	if balance != 0 {
		t.Fatalf("expected final balance 0, got %v", balance)
	}
	var movementCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM stock_movements
		WHERE organization_id=$1 AND product_id=$2 AND movement_type='sale'`, s.OrganizationID, productID).Scan(&movementCount); err != nil {
		t.Fatal(err)
	}
	if movementCount != 1 {
		t.Fatalf("expected one sale movement, got %d", movementCount)
	}
}
