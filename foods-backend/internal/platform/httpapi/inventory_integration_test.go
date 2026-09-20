package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
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
		_, _ = pool.Exec(context.Background(), `DELETE FROM inventory_presentations WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM inventory_items WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM products WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM audit_log WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM organizations WHERE id=$1`, organizationID)
	})
	return s
}

func TestInventoryEntryCreatesProductThenReusesIt(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	name := fmt.Sprintf("Agua mineral %d", time.Now().UnixNano())

	firstBody := []byte(fmt.Sprintf(`{"newProduct":{"name":%q,"price":"4.50"},"quantity":12,"unit":"botella","minimumStock":3,"note":"primera compra"}`, name))
	firstReq := httptest.NewRequest("POST", "/v1/admin/inventory/entries", bytes.NewReader(firstBody))
	firstReq = firstReq.WithContext(context.WithValue(firstReq.Context(), scopeKey{}, s))
	firstRec := httptest.NewRecorder()
	api.createInventoryEntry(firstRec, firstReq)
	if firstRec.Code != 201 {
		t.Fatalf("expected first entry 201, got %d body=%s", firstRec.Code, firstRec.Body.String())
	}

	var productID, productType, control string
	if err := pool.QueryRow(context.Background(), `
		SELECT id,product_type,quantity_control
		FROM products
		WHERE organization_id=$1 AND name=$2`, s.OrganizationID, name).Scan(&productID, &productType, &control); err != nil {
		t.Fatal(err)
	}
	if productType != "retail" {
		t.Fatalf("expected retail product type, got %q", productType)
	}
	if control != "inventory" {
		t.Fatalf("expected inventory control, got %q", control)
	}

	secondBody := []byte(fmt.Sprintf(`{"productId":%q,"quantity":5,"unit":"botella","minimumStock":3,"note":"reposición"}`, productID))
	secondReq := httptest.NewRequest("POST", "/v1/admin/inventory/entries", bytes.NewReader(secondBody))
	secondReq = secondReq.WithContext(context.WithValue(secondReq.Context(), scopeKey{}, s))
	secondRec := httptest.NewRecorder()
	api.createInventoryEntry(secondRec, secondReq)
	if secondRec.Code != 201 {
		t.Fatalf("expected second entry 201, got %d body=%s", secondRec.Code, secondRec.Body.String())
	}

	var productCount, entryCount, movementCount int
	var balance float64
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM products WHERE organization_id=$1 AND name=$2`, s.OrganizationID, name).Scan(&productCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM inventory_entries WHERE organization_id=$1 AND product_id=$2`, s.OrganizationID, productID).Scan(&entryCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM stock_movements WHERE organization_id=$1 AND product_id=$2 AND movement_type='entry'`, s.OrganizationID, productID).Scan(&movementCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT sb.quantity::float8
		FROM stock_balances sb
		JOIN inventory_items ii ON ii.organization_id=sb.organization_id AND ii.id=sb.inventory_item_id
		WHERE sb.organization_id=$1 AND sb.location_id=$2 AND ii.product_id=$3`,
		s.OrganizationID, s.LocationID, productID).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	if productCount != 1 || entryCount != 2 || movementCount != 2 || balance != 17 {
		t.Fatalf("unexpected inventory state products=%d entries=%d movements=%d balance=%v", productCount, entryCount, movementCount, balance)
	}
}

func TestInventoryPackageEntryConvertsToBaseUnits(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	name := fmt.Sprintf("Caja de agua %d", time.Now().UnixNano())

	body := []byte(fmt.Sprintf(`{"newProduct":{"name":%q,"price":"4.50"},"quantity":5,"unit":"botella","presentationType":"box","unitsPerPresentation":12,"minimumStock":6,"note":"5 cajas x 12"}`, name))
	req := httptest.NewRequest("POST", "/v1/admin/inventory/entries", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.createInventoryEntry(rec, req)
	if rec.Code != 201 {
		t.Fatalf("expected package entry 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	var productID string
	if err := pool.QueryRow(context.Background(), `
		SELECT id FROM products WHERE organization_id=$1 AND name=$2`,
		s.OrganizationID, name).Scan(&productID); err != nil {
		t.Fatal(err)
	}

	var entryQuantity, unitsPerPresentation, stockQuantity, balance, movementDelta float64
	var presentationType string
	if err := pool.QueryRow(context.Background(), `
		SELECT quantity::float8,presentation_type,units_per_presentation::float8,stock_quantity::float8
		FROM inventory_entries
		WHERE organization_id=$1 AND product_id=$2`,
		s.OrganizationID, productID).Scan(&entryQuantity, &presentationType, &unitsPerPresentation, &stockQuantity); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT sb.quantity::float8
		FROM stock_balances sb
		JOIN inventory_items ii ON ii.id=sb.inventory_item_id AND ii.organization_id=sb.organization_id
		WHERE sb.organization_id=$1 AND sb.location_id=$2 AND ii.product_id=$3`,
		s.OrganizationID, s.LocationID, productID).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT quantity_delta::float8
		FROM stock_movements
		WHERE organization_id=$1 AND product_id=$2 AND movement_type='entry'`,
		s.OrganizationID, productID).Scan(&movementDelta); err != nil {
		t.Fatal(err)
	}

	if entryQuantity != 5 || presentationType != "box" || unitsPerPresentation != 12 || stockQuantity != 60 {
		t.Fatalf("unexpected entry conversion quantity=%v type=%q factor=%v stock=%v", entryQuantity, presentationType, unitsPerPresentation, stockQuantity)
	}
	if balance != 60 || movementDelta != 60 {
		t.Fatalf("stock and kardex must use base units, balance=%v movement=%v", balance, movementDelta)
	}

	var presentationCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*)
		FROM inventory_presentations ip
		JOIN inventory_items ii ON ii.id=ip.inventory_item_id AND ii.organization_id=ip.organization_id
		WHERE ip.organization_id=$1 AND ii.product_id=$2 AND ip.active`,
		s.OrganizationID, productID).Scan(&presentationCount); err != nil {
		t.Fatal(err)
	}
	if presentationCount != 2 {
		t.Fatalf("expected reusable unit + box presentations, got %d", presentationCount)
	}
}

func TestKardexUsesReadableInventoryEntryReference(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	name := fmt.Sprintf("Kardex agua %d", time.Now().UnixNano())

	body := []byte(fmt.Sprintf(`{"newProduct":{"name":%q,"price":"4.50"},"quantity":12,"unit":"botella","minimumStock":3,"note":"recepción proveedor"}`, name))
	req := httptest.NewRequest("POST", "/v1/admin/inventory/entries", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.createInventoryEntry(rec, req)
	if rec.Code != 201 {
		t.Fatalf("expected entry 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest("GET", "/v1/admin/inventory/movements", nil)
	listReq = listReq.WithContext(context.WithValue(listReq.Context(), scopeKey{}, s))
	listRec := httptest.NewRecorder()
	api.listInventoryMovements(listRec, listReq)
	if listRec.Code != 200 {
		t.Fatalf("expected kardex 200, got %d body=%s", listRec.Code, listRec.Body.String())
	}

	var payload struct {
		Items []inventoryMovementView `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected one kardex movement, got %d", len(payload.Items))
	}
	item := payload.Items[0]
	if item.MovementType != "entry" || !strings.HasPrefix(item.SourceReference, "ENT-") {
		t.Fatalf("expected readable inventory entry reference, got type=%q reference=%q", item.MovementType, item.SourceReference)
	}
	if item.Note != "recepción proveedor" {
		t.Fatalf("expected traceability note, got %q", item.Note)
	}
}

func TestInventoryIngredientEntryDoesNotCreateProduct(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	name := fmt.Sprintf("Carne de res %d", time.Now().UnixNano())

	body := []byte(fmt.Sprintf(`{"newIngredient":{"name":%q},"quantity":15,"unit":"kg","minimumStock":2,"note":"recepción de insumo"}`, name))
	req := httptest.NewRequest("POST", "/v1/admin/inventory/entries", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.createInventoryEntry(rec, req)
	if rec.Code != 201 {
		t.Fatalf("expected ingredient entry 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	var result struct {
		InventoryItemID string  `json:"inventoryItemId"`
		ProductID       *string `json:"productId"`
		Kind            string  `json:"kind"`
		Balance         float64 `json:"balance"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.InventoryItemID == "" || result.ProductID != nil || result.Kind != "ingredient" || result.Balance != 15 {
		t.Fatalf("unexpected ingredient result: %#v", result)
	}

	var productCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM products WHERE organization_id=$1 AND name=$2`,
		s.OrganizationID, name).Scan(&productCount); err != nil {
		t.Fatal(err)
	}
	if productCount != 0 {
		t.Fatalf("ingredient must not create a sellable product, got %d", productCount)
	}

	var itemProductID *string
	var balance float64
	if err := pool.QueryRow(context.Background(), `
		SELECT ii.product_id::text,sb.quantity::float8
		FROM inventory_items ii
		JOIN stock_balances sb
		  ON sb.organization_id=ii.organization_id
		 AND sb.inventory_item_id=ii.id
		 AND sb.location_id=$2
		WHERE ii.id=$1 AND ii.organization_id=$3`,
		result.InventoryItemID, s.LocationID, s.OrganizationID).Scan(&itemProductID, &balance); err != nil {
		t.Fatal(err)
	}
	if itemProductID != nil || balance != 15 {
		t.Fatalf("expected unlinked ingredient with 15 kg, product=%v balance=%v", itemProductID, balance)
	}

	secondBody := []byte(fmt.Sprintf(`{"inventoryItemId":%q,"quantity":5,"unit":"kg","minimumStock":2}`, result.InventoryItemID))
	secondReq := httptest.NewRequest("POST", "/v1/admin/inventory/entries", bytes.NewReader(secondBody))
	secondReq = secondReq.WithContext(context.WithValue(secondReq.Context(), scopeKey{}, s))
	secondRec := httptest.NewRecorder()
	api.createInventoryEntry(secondRec, secondReq)
	if secondRec.Code != 201 {
		t.Fatalf("expected second ingredient entry 201, got %d body=%s", secondRec.Code, secondRec.Body.String())
	}

	if err := pool.QueryRow(context.Background(), `
		SELECT quantity::float8
		FROM stock_balances
		WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,
		s.OrganizationID, s.LocationID, result.InventoryItemID).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	if balance != 20 {
		t.Fatalf("expected reusable ingredient balance 20, got %v", balance)
	}

	var entryNullProducts, movementNullProducts int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM inventory_entries
		WHERE organization_id=$1 AND inventory_item_id=$2 AND product_id IS NULL`,
		s.OrganizationID, result.InventoryItemID).Scan(&entryNullProducts); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM stock_movements
		WHERE organization_id=$1 AND inventory_item_id=$2 AND product_id IS NULL`,
		s.OrganizationID, result.InventoryItemID).Scan(&movementNullProducts); err != nil {
		t.Fatal(err)
	}
	if entryNullProducts != 2 || movementNullProducts != 2 {
		t.Fatalf("ingredient traceability must remain product-less, entries=%d movements=%d", entryNullProducts, movementNullProducts)
	}

	listReq := httptest.NewRequest("GET", "/v1/admin/inventory/movements?inventoryItemId="+result.InventoryItemID, nil)
	listReq = listReq.WithContext(context.WithValue(listReq.Context(), scopeKey{}, s))
	listRec := httptest.NewRecorder()
	api.listInventoryMovements(listRec, listReq)
	if listRec.Code != 200 {
		t.Fatalf("expected ingredient kardex 200, got %d body=%s", listRec.Code, listRec.Body.String())
	}
	var movementsPayload struct {
		Items []inventoryMovementView `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &movementsPayload); err != nil {
		t.Fatal(err)
	}
	if len(movementsPayload.Items) != 2 || movementsPayload.Items[0].ItemName != name {
		t.Fatalf("expected ingredient in kardex, got %#v", movementsPayload.Items)
	}
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

func TestListInventoryProductsOnlyReturnsInventoryControlledProducts(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	nonce := time.Now().UnixNano()
	noneName := fmt.Sprintf("Ají de gallina %d", nonce)
	portionsName := fmt.Sprintf("Suspiro a la limeña %d", nonce)
	inventoryName := fmt.Sprintf("Agua mineral %d", nonce)

	if _, err := pool.Exec(ctx, `
		INSERT INTO products(organization_id,sku,name,price,quantity_control)
		VALUES
			($1,$2,$3,20,'none'),
			($1,$4,$5,12,'portions'),
			($1,$6,$7,4,'inventory')`,
		s.OrganizationID,
		fmt.Sprintf("NONE-%d", nonce), noneName,
		fmt.Sprintf("PORT-%d", nonce), portionsName,
		fmt.Sprintf("INV-%d", nonce), inventoryName,
	); err != nil {
		t.Fatal(err)
	}

	var inventoryProductID string
	if err := pool.QueryRow(ctx, `
		SELECT id
		FROM products
		WHERE organization_id=$1 AND name=$2`,
		s.OrganizationID, inventoryName).Scan(&inventoryProductID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO inventory_items(
			organization_id,sku,name,unit,minimum_stock,active,product_id
		)
		VALUES($1,$2,$3,'botella',0,true,$4)`,
		s.OrganizationID, fmt.Sprintf("ITEM-%d", nonce), inventoryName, inventoryProductID); err != nil {
		t.Fatal(err)
	}

	api := New(pool)
	req := httptest.NewRequest("GET", "/v1/admin/inventory/products", nil)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.listInventoryProducts(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, noneName) {
		t.Fatalf("product without quantity control must not be selectable: %s", body)
	}
	if strings.Contains(body, portionsName) {
		t.Fatalf("portion-controlled product must not be selectable: %s", body)
	}
	if !strings.Contains(body, inventoryName) {
		t.Fatalf("inventory-controlled product should be selectable: %s", body)
	}
}

func TestInventoryEntryRejectsProductWithoutInventoryControl(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	var productID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO products(organization_id,sku,name,price,quantity_control)
		VALUES($1,$2,'Ají de gallina',20,'none')
		RETURNING id`, s.OrganizationID, fmt.Sprintf("NONE-%d", time.Now().UnixNano())).Scan(&productID); err != nil {
		t.Fatal(err)
	}

	api := New(pool)
	body := []byte(fmt.Sprintf(`{"productId":%q,"quantity":3,"unit":"und","minimumStock":0}`, productID))
	req := httptest.NewRequest("POST", "/v1/admin/inventory/entries", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.createInventoryEntry(rec, req)

	if rec.Code != 409 {
		t.Fatalf("expected 409, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "quantity_control_conflict") {
		t.Fatalf("expected quantity_control_conflict, got body=%s", rec.Body.String())
	}

	var itemCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM inventory_items
		WHERE organization_id=$1 AND product_id=$2`, s.OrganizationID, productID).Scan(&itemCount); err != nil {
		t.Fatal(err)
	}
	if itemCount != 0 {
		t.Fatalf("rejected entry must not create inventory item, got %d", itemCount)
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
