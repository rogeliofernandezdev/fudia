package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPurchaseReceiptSupportsPartialReceivingAndTracksOrder(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	api := New(pool)

	var supplierID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO suppliers(organization_id,name)
		VALUES($1,'Proveedor recepción')
		RETURNING id`, s.OrganizationID).Scan(&supplierID); err != nil {
		t.Fatal(err)
	}

	var inventoryItemID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO inventory_items(organization_id,sku,name,unit,minimum_stock,active)
		VALUES($1,$2,'Harina','kg',0,true)
		RETURNING id`, s.OrganizationID, fmt.Sprintf("HAR-%d", time.Now().UnixNano())).Scan(&inventoryItemID); err != nil {
		t.Fatal(err)
	}

	var presentationID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO inventory_presentations(organization_id,inventory_item_id,presentation_type,units_per_presentation)
		VALUES($1,$2,'unit',1)
		RETURNING id`, s.OrganizationID, inventoryItemID).Scan(&presentationID); err != nil {
		t.Fatal(err)
	}

	number := "OC-PARCIAL"
	var orderID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO purchase_orders(organization_id,location_id,supplier_id,number,status,total,approved_at)
		VALUES($1,$2,$3,$4,'approved',20,now())
		RETURNING id`, s.OrganizationID, s.LocationID, supplierID, number).Scan(&orderID); err != nil {
		t.Fatal(err)
	}

	var lineID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO purchase_order_items(
			organization_id,purchase_order_id,inventory_item_id,presentation_id,
			quantity,presentation_type,units_per_presentation,unit_cost
		)
		VALUES($1,$2,$3,$4,10,'unit',1,2)
		RETURNING id`, s.OrganizationID, orderID, inventoryItemID, presentationID).Scan(&lineID); err != nil {
		t.Fatal(err)
	}

	var initialBalances int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM stock_balances
		WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,
		s.OrganizationID, s.LocationID, inventoryItemID).Scan(&initialBalances); err != nil {
		t.Fatal(err)
	}
	if initialBalances != 0 {
		t.Fatalf("saving and approving an order must not create stock, got %d balances", initialBalances)
	}

	firstBody := []byte(fmt.Sprintf(`{"items":[{"purchaseOrderItemId":%q,"quantity":4}],"notes":"primer despacho"}`, lineID))
	firstReq := httptest.NewRequest("POST", "/v1/admin/purchase-orders/"+orderID+"/receive", bytes.NewReader(firstBody))
	firstReq.SetPathValue("id", orderID)
	firstReq = firstReq.WithContext(context.WithValue(firstReq.Context(), scopeKey{}, s))
	firstRec := httptest.NewRecorder()
	api.receivePurchaseOrder(firstRec, firstReq)
	if firstRec.Code != 201 {
		t.Fatalf("expected partial receipt 201, got %d body=%s", firstRec.Code, firstRec.Body.String())
	}
	var firstResult struct {
		Code string `json:"code"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(firstRec.Body.Bytes(), &firstResult); err != nil {
		t.Fatal(err)
	}
	if firstResult.Status != "partially_received" || !strings.HasPrefix(firstResult.Code, "REC-") {
		t.Fatalf("unexpected partial receipt result: %#v", firstResult)
	}

	var status string
	var received, balance float64
	if err := pool.QueryRow(ctx, `
		SELECT status FROM purchase_orders WHERE id=$1 AND organization_id=$2`,
		orderID, s.OrganizationID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT received_quantity::float8
		FROM purchase_order_items WHERE id=$1 AND organization_id=$2`,
		lineID, s.OrganizationID).Scan(&received); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT quantity::float8 FROM stock_balances
		WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,
		s.OrganizationID, s.LocationID, inventoryItemID).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	if status != "partially_received" || received != 4 || balance != 4 {
		t.Fatalf("unexpected partial state status=%q received=%v balance=%v", status, received, balance)
	}

	var receiptCount, movementCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM purchase_receipts
		WHERE organization_id=$1 AND purchase_order_id=$2`,
		s.OrganizationID, orderID).Scan(&receiptCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM stock_movements
		WHERE organization_id=$1 AND inventory_item_id=$2
		  AND movement_type='entry' AND source_type='purchase_receipt'
		  AND quantity_delta=4 AND balance_after=4`,
		s.OrganizationID, inventoryItemID).Scan(&movementCount); err != nil {
		t.Fatal(err)
	}
	if receiptCount != 1 || movementCount != 1 {
		t.Fatalf("expected one receipt and one stock movement, receipts=%d movements=%d", receiptCount, movementCount)
	}

	listReq := httptest.NewRequest("GET", "/v1/admin/inventory/movements?inventoryItemId="+inventoryItemID, nil)
	listReq = listReq.WithContext(context.WithValue(listReq.Context(), scopeKey{}, s))
	listRec := httptest.NewRecorder()
	api.listInventoryMovements(listRec, listReq)
	if listRec.Code != 200 {
		t.Fatalf("expected kardex 200, got %d body=%s", listRec.Code, listRec.Body.String())
	}
	var kardex struct {
		Items []inventoryMovementView `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &kardex); err != nil {
		t.Fatal(err)
	}
	if len(kardex.Items) != 1 || !strings.Contains(kardex.Items[0].SourceReference, number) || !strings.Contains(kardex.Items[0].SourceReference, "REC-") {
		t.Fatalf("receipt must be linked to its order in kardex: %#v", kardex.Items)
	}

	overBody := []byte(fmt.Sprintf(`{"items":[{"purchaseOrderItemId":%q,"quantity":7}]}`, lineID))
	overReq := httptest.NewRequest("POST", "/v1/admin/purchase-orders/"+orderID+"/receive", bytes.NewReader(overBody))
	overReq.SetPathValue("id", orderID)
	overReq = overReq.WithContext(context.WithValue(overReq.Context(), scopeKey{}, s))
	overRec := httptest.NewRecorder()
	api.receivePurchaseOrder(overRec, overReq)
	if overRec.Code != 409 || !strings.Contains(overRec.Body.String(), "purchase_receipt_exceeds_pending") {
		t.Fatalf("expected over-receipt 409, got %d body=%s", overRec.Code, overRec.Body.String())
	}
	if err := pool.QueryRow(ctx, `
		SELECT quantity::float8 FROM stock_balances
		WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,
		s.OrganizationID, s.LocationID, inventoryItemID).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	if balance != 4 {
		t.Fatalf("rejected receipt must not alter stock, got %v", balance)
	}

	finalBody := []byte(fmt.Sprintf(`{"items":[{"purchaseOrderItemId":%q,"quantity":6}]}`, lineID))
	finalReq := httptest.NewRequest("POST", "/v1/admin/purchase-orders/"+orderID+"/receive", bytes.NewReader(finalBody))
	finalReq.SetPathValue("id", orderID)
	finalReq = finalReq.WithContext(context.WithValue(finalReq.Context(), scopeKey{}, s))
	finalRec := httptest.NewRecorder()
	api.receivePurchaseOrder(finalRec, finalReq)
	if finalRec.Code != 201 {
		t.Fatalf("expected final receipt 201, got %d body=%s", finalRec.Code, finalRec.Body.String())
	}
	if err := pool.QueryRow(ctx, `
		SELECT po.status,poi.received_quantity::float8,sb.quantity::float8
		FROM purchase_orders po
		JOIN purchase_order_items poi ON poi.purchase_order_id=po.id AND poi.organization_id=po.organization_id
		JOIN stock_balances sb ON sb.inventory_item_id=poi.inventory_item_id AND sb.organization_id=po.organization_id AND sb.location_id=po.location_id
		WHERE po.id=$1 AND po.organization_id=$2`,
		orderID, s.OrganizationID).Scan(&status, &received, &balance); err != nil {
		t.Fatal(err)
	}
	if status != "received" || received != 10 || balance != 10 {
		t.Fatalf("unexpected completed state status=%q received=%v balance=%v", status, received, balance)
	}
}

func TestPurchaseItemCreationStartsAtZeroWithoutMovement(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	api := New(pool)

	ingredientName := fmt.Sprintf("Pimienta %d", time.Now().UnixNano())
	ingredientBody := []byte(fmt.Sprintf(`{"newIngredient":{"name":%q},"unit":"kg","presentationType":"box","unitsPerPresentation":5,"minimumStock":2}`, ingredientName))
	ingredientReq := httptest.NewRequest("POST", "/v1/admin/purchase-inventory-items", bytes.NewReader(ingredientBody))
	ingredientReq = ingredientReq.WithContext(context.WithValue(ingredientReq.Context(), scopeKey{}, s))
	ingredientRec := httptest.NewRecorder()
	api.createPurchaseInventoryItem(ingredientRec, ingredientReq)
	if ingredientRec.Code != 201 {
		t.Fatalf("expected ingredient creation 201, got %d body=%s", ingredientRec.Code, ingredientRec.Body.String())
	}
	var ingredient inventoryProductOption
	if err := json.Unmarshal(ingredientRec.Body.Bytes(), &ingredient); err != nil {
		t.Fatal(err)
	}
	if ingredient.ProductID != nil || ingredient.Kind != "ingredient" || ingredient.Quantity != "0" {
		t.Fatalf("ingredient must remain non-vendible at stock zero: %#v", ingredient)
	}
	var balance, movements float64
	if err := pool.QueryRow(ctx, `
		SELECT quantity::float8 FROM stock_balances
		WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,
		s.OrganizationID, s.LocationID, ingredient.ID).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT count(*)::float8 FROM stock_movements
		WHERE organization_id=$1 AND inventory_item_id=$2`,
		s.OrganizationID, ingredient.ID).Scan(&movements); err != nil {
		t.Fatal(err)
	}
	if balance != 0 || movements != 0 || len(ingredient.Presentations) != 2 {
		t.Fatalf("new ingredient must have zero stock, no movement and reusable presentations, balance=%v movements=%v presentations=%d", balance, movements, len(ingredient.Presentations))
	}

	categoryID := seedRetailCategory(t, pool, s)
	productName := fmt.Sprintf("Agua compra %d", time.Now().UnixNano())
	productBody := []byte(fmt.Sprintf(`{"newProduct":{"name":%q,"price":"4.50","categoryId":%q,"description":"500 ml"},"unit":"botella","presentationType":"package","unitsPerPresentation":12,"minimumStock":3}`, productName, categoryID))
	productReq := httptest.NewRequest("POST", "/v1/admin/purchase-inventory-items", bytes.NewReader(productBody))
	productReq = productReq.WithContext(context.WithValue(productReq.Context(), scopeKey{}, s))
	productRec := httptest.NewRecorder()
	api.createPurchaseInventoryItem(productRec, productReq)
	if productRec.Code != 201 {
		t.Fatalf("expected sellable product creation 201, got %d body=%s", productRec.Code, productRec.Body.String())
	}
	var product inventoryProductOption
	if err := json.Unmarshal(productRec.Body.Bytes(), &product); err != nil {
		t.Fatal(err)
	}
	if product.ProductID == nil || product.Kind != "product" || product.Quantity != "0" {
		t.Fatalf("sellable purchase item must link Product at zero stock: %#v", product)
	}
	var productType, control string
	if err := pool.QueryRow(ctx, `
		SELECT product_type,quantity_control FROM products
		WHERE id=$1 AND organization_id=$2`, *product.ProductID, s.OrganizationID).Scan(&productType, &control); err != nil {
		t.Fatal(err)
	}
	if productType != "retail" || control != "inventory" {
		t.Fatalf("unexpected sellable product classification type=%q control=%q", productType, control)
	}
}
