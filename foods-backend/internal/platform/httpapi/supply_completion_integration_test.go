package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func seedSupplyItem(t *testing.T,pool interface{
	QueryRow(context.Context,string,...any) interface{Scan(...any) error}
},s scope,name string)(string,string){
	t.Helper()
	return "",""
}

// TestSupplyCompletionReceiptIdempotencyValuationAndReturns validates the financial/physical
// invariant: one receipt changes stock once, weighted cost is preserved, and corrections/returns
// are bounded by what was actually received.
func TestSupplyCompletionReceiptIdempotencyValuationAndReturns(t *testing.T){
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool)
	ctx:=context.Background()
	api:=New(pool)
	nonce:=time.Now().UnixNano()

	var supplierID string
	if err:=pool.QueryRow(ctx,`INSERT INTO suppliers(organization_id,tax_id,name) VALUES($1,$2,$3) RETURNING id`,
		s.OrganizationID,fmt.Sprintf("%011d",nonce%100000000000),fmt.Sprintf("Proveedor valor %d",nonce)).Scan(&supplierID);err!=nil{t.Fatal(err)}
	var itemID string
	if err:=pool.QueryRow(ctx,`INSERT INTO inventory_items(organization_id,sku,name,unit,minimum_stock,active) VALUES($1,$2,$3,'und',0,true) RETURNING id`,
		s.OrganizationID,fmt.Sprintf("VAL-%d",nonce),fmt.Sprintf("Artículo valor %d",nonce)).Scan(&itemID);err!=nil{t.Fatal(err)}
	var presentationID string
	if err:=pool.QueryRow(ctx,`INSERT INTO inventory_presentations(organization_id,inventory_item_id,presentation_type,units_per_presentation) VALUES($1,$2,'unit',1) RETURNING id`,
		s.OrganizationID,itemID).Scan(&presentationID);err!=nil{t.Fatal(err)}
	if _,err:=pool.Exec(ctx,`INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity,average_unit_cost) VALUES($1,$2,$3,10,2)`,
		s.OrganizationID,s.LocationID,itemID);err!=nil{t.Fatal(err)}
	var orderID,lineID string
	if err:=pool.QueryRow(ctx,`INSERT INTO purchase_orders(organization_id,location_id,supplier_id,number,status,total,approved_at) VALUES($1,$2,$3,$4,'approved',20,now()) RETURNING id`,
		s.OrganizationID,s.LocationID,supplierID,fmt.Sprintf("OC-VAL-%d",nonce)).Scan(&orderID);err!=nil{t.Fatal(err)}
	if err:=pool.QueryRow(ctx,`INSERT INTO purchase_order_items(organization_id,purchase_order_id,inventory_item_id,presentation_id,quantity,presentation_type,units_per_presentation,unit_cost) VALUES($1,$2,$3,$4,5,'unit',1,4) RETURNING id`,
		s.OrganizationID,orderID,itemID,presentationID).Scan(&lineID);err!=nil{t.Fatal(err)}

	key:=fmt.Sprintf("receipt-%d",nonce)
	body:=[]byte(fmt.Sprintf(`{"idempotencyKey":%q,"items":[{"purchaseOrderItemId":%q,"quantity":5}],"notes":"recepción valorizada"}`,key,lineID))
	doReceipt:=func() *httptest.ResponseRecorder{
		req:=httptest.NewRequest("POST","/v1/admin/purchase-orders/"+orderID+"/receive",bytes.NewReader(body))
		req.SetPathValue("id",orderID);req=req.WithContext(context.WithValue(req.Context(),scopeKey{},s))
		rec:=httptest.NewRecorder();api.receivePurchaseOrder(rec,req);return rec
	}
	first:=doReceipt()
	if first.Code!=201{t.Fatalf("first receipt: %d %s",first.Code,first.Body.String())}
	var receiptResult struct{ID string `json:"id"`;Code string `json:"code"`}
	if err:=json.Unmarshal(first.Body.Bytes(),&receiptResult);err!=nil{t.Fatal(err)}
	replay:=doReceipt()
	if replay.Code!=200||!strings.Contains(replay.Body.String(),`"idempotent":true`){t.Fatalf("receipt replay: %d %s",replay.Code,replay.Body.String())}

	var qty,avg float64
	if err:=pool.QueryRow(ctx,`SELECT quantity::float8,average_unit_cost::float8 FROM stock_balances WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,
		s.OrganizationID,s.LocationID,itemID).Scan(&qty,&avg);err!=nil{t.Fatal(err)}
	if qty!=15||math.Abs(avg-2.6667)>0.00011{t.Fatalf("weighted balance qty=%v avg=%v",qty,avg)}
	var receiptCount,movementCount int
	if err:=pool.QueryRow(ctx,`SELECT count(*) FROM purchase_receipts WHERE organization_id=$1 AND purchase_order_id=$2`,s.OrganizationID,orderID).Scan(&receiptCount);err!=nil{t.Fatal(err)}
	if err:=pool.QueryRow(ctx,`SELECT count(*) FROM stock_movements WHERE organization_id=$1 AND source_type='purchase_receipt' AND source_id=$2 AND quantity_delta=5 AND unit_cost=4 AND value_delta=20`,
		s.OrganizationID,receiptResult.ID).Scan(&movementCount);err!=nil{t.Fatal(err)}
	if receiptCount!=1||movementCount!=1{t.Fatalf("idempotency traces receipts=%d movements=%d",receiptCount,movementCount)}

	var receiptItemID string
	if err:=pool.QueryRow(ctx,`SELECT id FROM purchase_receipt_items WHERE purchase_receipt_id=$1 AND organization_id=$2`,receiptResult.ID,s.OrganizationID).Scan(&receiptItemID);err!=nil{t.Fatal(err)}
	correctionKey:=fmt.Sprintf("corr-%d",nonce)
	correctionBody:=[]byte(fmt.Sprintf(`{"idempotencyKey":%q,"kind":"receipt_correction","reason":"cantidad ingresada por error","items":[{"purchaseReceiptItemId":%q,"quantity":2}]}`,correctionKey,receiptItemID))
	doReturn:=func(body []byte)*httptest.ResponseRecorder{
		req:=httptest.NewRequest("POST","/v1/admin/purchase-receipts/"+receiptResult.ID+"/returns",bytes.NewReader(body))
		req.SetPathValue("id",receiptResult.ID);req=req.WithContext(context.WithValue(req.Context(),scopeKey{},s))
		rec:=httptest.NewRecorder();api.createPurchaseReturn(rec,req);return rec
	}
	corr:=doReturn(correctionBody)
	if corr.Code!=201{t.Fatalf("correction: %d %s",corr.Code,corr.Body.String())}
	corrReplay:=doReturn(correctionBody)
	if corrReplay.Code!=200||!strings.Contains(corrReplay.Body.String(),`"idempotent":true`){t.Fatalf("correction replay: %d %s",corrReplay.Code,corrReplay.Body.String())}
	var received float64;var status string
	if err:=pool.QueryRow(ctx,`SELECT poi.received_quantity::float8,po.status FROM purchase_order_items poi JOIN purchase_orders po ON po.id=poi.purchase_order_id WHERE poi.id=$1`,lineID).Scan(&received,&status);err!=nil{t.Fatal(err)}
	if received!=3||status!="partially_received"{t.Fatalf("correction state received=%v status=%s",received,status)}
	if err:=pool.QueryRow(ctx,`SELECT quantity::float8 FROM stock_balances WHERE location_id=$1 AND inventory_item_id=$2`,s.LocationID,itemID).Scan(&qty);err!=nil{t.Fatal(err)}
	if qty!=13{t.Fatalf("stock after correction=%v",qty)}

	returnKey:=fmt.Sprintf("return-%d",nonce)
	returnBody:=[]byte(fmt.Sprintf(`{"idempotencyKey":%q,"kind":"supplier_return","reason":"mercadería defectuosa","items":[{"purchaseReceiptItemId":%q,"quantity":1}]}`,returnKey,receiptItemID))
	ret:=doReturn(returnBody)
	if ret.Code!=201{t.Fatalf("supplier return: %d %s",ret.Code,ret.Body.String())}
	if err:=pool.QueryRow(ctx,`SELECT quantity::float8 FROM stock_balances WHERE location_id=$1 AND inventory_item_id=$2`,s.LocationID,itemID).Scan(&qty);err!=nil{t.Fatal(err)}
	if qty!=12{t.Fatalf("stock after supplier return=%v",qty)}
	if err:=pool.QueryRow(ctx,`SELECT received_quantity::float8 FROM purchase_order_items WHERE id=$1`,lineID).Scan(&received);err!=nil{t.Fatal(err)}
	if received!=3{t.Fatalf("supplier return must not reopen PO received qty, got %v",received)}
}

func TestSupplyCompletionTransferIdempotencyAndLocalSettings(t *testing.T){
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool)
	ctx:=context.Background()
	api:=New(pool)
	nonce:=time.Now().UnixNano()

	var itemID string
	if err:=pool.QueryRow(ctx,`INSERT INTO inventory_items(organization_id,sku,name,unit,minimum_stock,active) VALUES($1,$2,$3,'und',0,true) RETURNING id`,
		s.OrganizationID,fmt.Sprintf("TRF-%d",nonce),fmt.Sprintf("Transferible %d",nonce)).Scan(&itemID);err!=nil{t.Fatal(err)}
	var loc2 string
	if err:=pool.QueryRow(ctx,`INSERT INTO locations(organization_id,name,code,address) VALUES($1,$2,$3,'') RETURNING id`,
		s.OrganizationID,fmt.Sprintf("Destino %d",nonce),fmt.Sprintf("D%d",nonce%1000000)).Scan(&loc2);err!=nil{t.Fatal(err)}
	if _,err:=pool.Exec(ctx,`INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity,average_unit_cost) VALUES($1,$2,$3,10,3),($1,$4,$3,5,5)`,
		s.OrganizationID,s.LocationID,itemID,loc2);err!=nil{t.Fatal(err)}

	updateSettings:=func(sc scope,min,reorder,opt float64){
		body:=[]byte(fmt.Sprintf(`{"minimumStock":%v,"reorderPoint":%v,"optimalStock":%v}`,min,reorder,opt))
		req:=httptest.NewRequest("PATCH","/v1/admin/inventory/items/"+itemID+"/settings",bytes.NewReader(body));req.SetPathValue("id",itemID)
		req=req.WithContext(context.WithValue(req.Context(),scopeKey{},sc));rec:=httptest.NewRecorder();api.updateInventoryLocationSettings(rec,req)
		if rec.Code!=204{t.Fatalf("settings: %d %s",rec.Code,rec.Body.String())}
	}
	updateSettings(s,2,4,8)
	s2:=s;s2.LocationID=loc2
	updateSettings(s2,7,9,12)
	var min1,min2 float64
	if err:=pool.QueryRow(ctx,`SELECT minimum_stock::float8 FROM inventory_location_settings WHERE location_id=$1 AND inventory_item_id=$2`,s.LocationID,itemID).Scan(&min1);err!=nil{t.Fatal(err)}
	if err:=pool.QueryRow(ctx,`SELECT minimum_stock::float8 FROM inventory_location_settings WHERE location_id=$1 AND inventory_item_id=$2`,loc2,itemID).Scan(&min2);err!=nil{t.Fatal(err)}
	if min1!=2||min2!=7{t.Fatalf("local minimum isolation min1=%v min2=%v",min1,min2)}

	key:=fmt.Sprintf("trf-key-%d",nonce)
	body:=[]byte(fmt.Sprintf(`{"idempotencyKey":%q,"toLocationId":%q,"items":[{"inventoryItemId":%q,"quantity":4}]}`,key,loc2,itemID))
	doTransfer:=func()*httptest.ResponseRecorder{
		req:=httptest.NewRequest("POST","/v1/admin/inventory/transfers",bytes.NewReader(body));req=req.WithContext(context.WithValue(req.Context(),scopeKey{},s))
		rec:=httptest.NewRecorder();api.createInventoryTransfer(rec,req);return rec
	}
	first:=doTransfer();if first.Code!=201{t.Fatalf("transfer: %d %s",first.Code,first.Body.String())}
	replay:=doTransfer();if replay.Code!=200||!strings.Contains(replay.Body.String(),`"idempotent":true`){t.Fatalf("transfer replay: %d %s",replay.Code,replay.Body.String())}
	var q1,c1,q2,c2 float64
	if err:=pool.QueryRow(ctx,`SELECT quantity::float8,average_unit_cost::float8 FROM stock_balances WHERE location_id=$1 AND inventory_item_id=$2`,s.LocationID,itemID).Scan(&q1,&c1);err!=nil{t.Fatal(err)}
	if err:=pool.QueryRow(ctx,`SELECT quantity::float8,average_unit_cost::float8 FROM stock_balances WHERE location_id=$1 AND inventory_item_id=$2`,loc2,itemID).Scan(&q2,&c2);err!=nil{t.Fatal(err)}
	if q1!=6||c1!=3||q2!=9||math.Abs(c2-4.1111)>0.00011{t.Fatalf("transfer balances source=%v/%v dest=%v/%v",q1,c1,q2,c2)}
	var transfers int
	if err:=pool.QueryRow(ctx,`SELECT count(*) FROM inventory_transfers WHERE organization_id=$1 AND from_location_id=$2 AND idempotency_key=$3`,s.OrganizationID,s.LocationID,key).Scan(&transfers);err!=nil{t.Fatal(err)}
	if transfers!=1{t.Fatalf("expected one transfer, got %d",transfers)}
}

func TestOptionalRecipesDoNotBlockBasePlanAndConsumeWhenEnabled(t *testing.T){
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool)
	ctx:=context.Background()
	api:=New(pool)
	nonce:=time.Now().UnixNano()

	var ingredientID string
	if err:=pool.QueryRow(ctx,`INSERT INTO inventory_items(organization_id,sku,name,unit,minimum_stock,active) VALUES($1,$2,$3,'kg',0,true) RETURNING id`,
		s.OrganizationID,fmt.Sprintf("REC-I-%d",nonce),fmt.Sprintf("Insumo receta %d",nonce)).Scan(&ingredientID);err!=nil{t.Fatal(err)}
	if _,err:=pool.Exec(ctx,`INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity,average_unit_cost) VALUES($1,$2,$3,10,2)`,s.OrganizationID,s.LocationID,ingredientID);err!=nil{t.Fatal(err)}
	var productID string
	if err:=pool.QueryRow(ctx,`INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control) VALUES($1,$2,$3,20,true,'prepared','none') RETURNING id`,
		s.OrganizationID,fmt.Sprintf("REC-P-%d",nonce),fmt.Sprintf("Plato receta %d",nonce)).Scan(&productID);err!=nil{t.Fatal(err)}
	var recipeID string
	if err:=pool.QueryRow(ctx,`INSERT INTO product_recipes(organization_id,product_id,yield_quantity,notes,active,created_by,updated_by) VALUES($1,$2,1,'',true,$3,$3) RETURNING id`,s.OrganizationID,productID,s.UserID).Scan(&recipeID);err!=nil{t.Fatal(err)}
	if _,err:=pool.Exec(ctx,`INSERT INTO product_recipe_items(organization_id,recipe_id,inventory_item_id,quantity,waste_percent) VALUES($1,$2,$3,2,0)`,s.OrganizationID,recipeID,ingredientID);err!=nil{t.Fatal(err)}

	setModule:=func(key string,active bool){
		if _,err:=pool.Exec(ctx,`INSERT INTO organization_modules(organization_id,module_key,active) VALUES($1,$2,$3) ON CONFLICT(organization_id,module_key) DO UPDATE SET active=EXCLUDED.active`,s.OrganizationID,key,active);err!=nil{t.Fatal(err)}
	}
	setModule("inventario",true);setModule("recetas",false)

	createOrder:=func(qty float64)*httptest.ResponseRecorder{
		body:=[]byte(fmt.Sprintf(`{"channel":"mostrador","items":[{"productId":%q,"name":"x","qty":%v,"unitPrice":1,"note":"","selections":[]}]}`,productID,qty))
		req:=httptest.NewRequest("POST","/v1/admin/orders",bytes.NewReader(body));req=req.WithContext(context.WithValue(req.Context(),scopeKey{},s))
		rec:=httptest.NewRecorder();api.createOrder(rec,req);return rec
	}
	base:=createOrder(2);if base.Code!=201{t.Fatalf("base plan order: %d %s",base.Code,base.Body.String())}
	var stock float64
	if err:=pool.QueryRow(ctx,`SELECT quantity::float8 FROM stock_balances WHERE location_id=$1 AND inventory_item_id=$2`,s.LocationID,ingredientID).Scan(&stock);err!=nil{t.Fatal(err)}
	if stock!=10{t.Fatalf("recipes disabled must not consume ingredients, stock=%v",stock)}

	setModule("recetas",true)
	controlled:=createOrder(2);if controlled.Code!=201{t.Fatalf("recipe order: %d %s",controlled.Code,controlled.Body.String())}
	var order order
	if err:=json.Unmarshal(controlled.Body.Bytes(),&order);err!=nil{t.Fatal(err)}
	if err:=pool.QueryRow(ctx,`SELECT quantity::float8 FROM stock_balances WHERE location_id=$1 AND inventory_item_id=$2`,s.LocationID,ingredientID).Scan(&stock);err!=nil{t.Fatal(err)}
	if stock!=6{t.Fatalf("recipe should consume 4 kg, stock=%v",stock)}

	cancel:=httptest.NewRequest("PATCH","/v1/admin/orders/"+order.ID+"/status",bytes.NewReader([]byte(`{"status":"cancelado"}`)));cancel.SetPathValue("id",order.ID);cancel=cancel.WithContext(context.WithValue(cancel.Context(),scopeKey{},s))
	cancelRec:=httptest.NewRecorder();api.updateOrderStatus(cancelRec,cancel)
	if cancelRec.Code!=200{t.Fatalf("cancel recipe order: %d %s",cancelRec.Code,cancelRec.Body.String())}
	if err:=pool.QueryRow(ctx,`SELECT quantity::float8 FROM stock_balances WHERE location_id=$1 AND inventory_item_id=$2`,s.LocationID,ingredientID).Scan(&stock);err!=nil{t.Fatal(err)}
	if stock!=10{t.Fatalf("recipe cancellation should restore stock, got %v",stock)}

	var consume,reversal int
	if err:=pool.QueryRow(ctx,`SELECT count(*) FILTER(WHERE movement_type='recipe_consumption'),count(*) FILTER(WHERE movement_type='recipe_reversal') FROM stock_movements WHERE organization_id=$1 AND source_type='order' AND source_id=$2`,s.OrganizationID,order.ID).Scan(&consume,&reversal);err!=nil{t.Fatal(err)}
	if consume!=1||reversal!=1{t.Fatalf("recipe kardex consume=%d reversal=%d",consume,reversal)}

	// If Inventory is disabled, Recipes must not become a hidden stock dependency.
	setModule("inventario",false)
	hidden:=createOrder(1);if hidden.Code!=201{t.Fatalf("inventory-disabled base order should still work: %d %s",hidden.Code,hidden.Body.String())}
	if err:=pool.QueryRow(ctx,`SELECT quantity::float8 FROM stock_balances WHERE location_id=$1 AND inventory_item_id=$2`,s.LocationID,ingredientID).Scan(&stock);err!=nil{t.Fatal(err)}
	if stock!=10{t.Fatalf("recipes must not consume when inventory module is disabled, stock=%v",stock)}
}

func TestOptionalRecipeInsufficientStockRollsBackOrder(t *testing.T){
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool);ctx:=context.Background();api:=New(pool);nonce:=time.Now().UnixNano()
	_,_=pool.Exec(ctx,`INSERT INTO organization_modules(organization_id,module_key,active) VALUES($1,'inventario',true),($1,'recetas',true) ON CONFLICT(organization_id,module_key) DO UPDATE SET active=true`,s.OrganizationID)
	var itemID,productID,recipeID string
	if err:=pool.QueryRow(ctx,`INSERT INTO inventory_items(organization_id,sku,name,unit,minimum_stock,active) VALUES($1,$2,$3,'kg',0,true) RETURNING id`,s.OrganizationID,fmt.Sprintf("LOW-I-%d",nonce),"Insumo escaso").Scan(&itemID);err!=nil{t.Fatal(err)}
	if _,err:=pool.Exec(ctx,`INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity,average_unit_cost) VALUES($1,$2,$3,3,2)`,s.OrganizationID,s.LocationID,itemID);err!=nil{t.Fatal(err)}
	if err:=pool.QueryRow(ctx,`INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control) VALUES($1,$2,$3,20,true,'prepared','none') RETURNING id`,s.OrganizationID,fmt.Sprintf("LOW-P-%d",nonce),"Plato escaso").Scan(&productID);err!=nil{t.Fatal(err)}
	if err:=pool.QueryRow(ctx,`INSERT INTO product_recipes(organization_id,product_id,yield_quantity,active) VALUES($1,$2,1,true) RETURNING id`,s.OrganizationID,productID).Scan(&recipeID);err!=nil{t.Fatal(err)}
	if _,err:=pool.Exec(ctx,`INSERT INTO product_recipe_items(organization_id,recipe_id,inventory_item_id,quantity,waste_percent) VALUES($1,$2,$3,2,0)`,s.OrganizationID,recipeID,itemID);err!=nil{t.Fatal(err)}
	var before int
	_ = pool.QueryRow(ctx,`SELECT count(*) FROM orders WHERE organization_id=$1 AND location_id=$2`,s.OrganizationID,s.LocationID).Scan(&before)
	body:=[]byte(fmt.Sprintf(`{"channel":"mostrador","items":[{"productId":%q,"qty":2,"unitPrice":1,"selections":[]}]}`,productID))
	req:=httptest.NewRequest("POST","/v1/admin/orders",bytes.NewReader(body));req=req.WithContext(context.WithValue(req.Context(),scopeKey{},s));rec:=httptest.NewRecorder();api.createOrder(rec,req)
	if rec.Code!=409||!strings.Contains(rec.Body.String(),"insufficient_recipe_stock"){t.Fatalf("expected insufficient_recipe_stock, got %d %s",rec.Code,rec.Body.String())}
	var after int;var stock float64
	_ = pool.QueryRow(ctx,`SELECT count(*) FROM orders WHERE organization_id=$1 AND location_id=$2`,s.OrganizationID,s.LocationID).Scan(&after)
	_ = pool.QueryRow(ctx,`SELECT quantity::float8 FROM stock_balances WHERE location_id=$1 AND inventory_item_id=$2`,s.LocationID,itemID).Scan(&stock)
	if after!=before||stock!=3{t.Fatalf("failed recipe order must rollback order/stock before=%d after=%d stock=%v",before,after,stock)}
}
