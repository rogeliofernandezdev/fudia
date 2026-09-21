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
	"time"
)

func TestSalonKitchenPaymentDeliveryWorkflow(t *testing.T) {
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool)
	api:=New(pool)
	ctx:=context.Background()
	nonce:=time.Now().UnixNano()

	var productID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control)
		VALUES($1,$2,$3,25,true,'prepared','none')
		RETURNING id
	`,s.OrganizationID,fmt.Sprintf("WF-%d",nonce),fmt.Sprintf("Plato flujo %d",nonce)).Scan(&productID);err!=nil{t.Fatal(err)}

	var tableID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO tables(organization_id,location_id,name,seats,zone,active,qr_token,qr_enabled)
		VALUES($1,$2,$3,4,'Principal',true,encode(gen_random_bytes(16),'hex'),false)
		RETURNING id
	`,s.OrganizationID,s.LocationID,fmt.Sprintf("Mesa WF %d",nonce)).Scan(&tableID);err!=nil{t.Fatal(err)}

	defer func(){
		_,_=pool.Exec(context.Background(),`DELETE FROM orders WHERE organization_id=$1 AND table_id=$2`,s.OrganizationID,tableID)
		_,_=pool.Exec(context.Background(),`DELETE FROM tables WHERE id=$1 AND organization_id=$2`,tableID,s.OrganizationID)
	}()

	var registerID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO cash_registers(organization_id,location_id,name,created_by)
		VALUES($1,$2,$3,$4)
		RETURNING id
	`,s.OrganizationID,s.LocationID,fmt.Sprintf("Caja WF %d",nonce),s.UserID).Scan(&registerID);err!=nil{t.Fatal(err)}
	openReq:=httptest.NewRequest("POST","/v1/admin/cash-shifts",bytes.NewReader([]byte(fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":0}`,registerID))))
	openReq=openReq.WithContext(context.WithValue(openReq.Context(),scopeKey{},s))
	openRec:=httptest.NewRecorder()
	api.openCashShift(openRec,openReq)
	if openRec.Code!=201{t.Fatalf("open shift: %d %s",openRec.Code,openRec.Body.String())}

	createBody:=[]byte(fmt.Sprintf(`{
		"channel":"salon",
		"tableId":%q,
		"items":[{"productId":%q,"name":"ignorado","qty":1,"unitPrice":1,"note":"sin cebolla","selections":[]}],
		"sendToKitchen":true
	}`,tableID,productID))
	createReq:=httptest.NewRequest("POST","/v1/admin/orders",bytes.NewReader(createBody))
	createReq=createReq.WithContext(context.WithValue(createReq.Context(),scopeKey{},s))
	createRec:=httptest.NewRecorder()
	api.createOrder(createRec,createReq)
	if createRec.Code!=201{t.Fatalf("create order: %d %s",createRec.Code,createRec.Body.String())}
	var created order
	if err:=json.Unmarshal(createRec.Body.Bytes(),&created);err!=nil{t.Fatal(err)}
	if created.Status!="confirmado"{t.Fatalf("expected confirmed order sent to kitchen, got %q",created.Status)}
	if created.Total!="25.00"{t.Fatalf("server catalog price must win, got total %q",created.Total)}

	// Once a payment exists, the comanda amount/items cannot be edited.
	partialPayReq:=httptest.NewRequest("POST","/v1/admin/payments",bytes.NewReader([]byte(fmt.Sprintf(`{"orderId":%q,"method":"card","amount":5,"reference":"ANTICIPO"}`,created.ID))))
	partialPayReq=partialPayReq.WithContext(context.WithValue(partialPayReq.Context(),scopeKey{},s))
	partialPayRec:=httptest.NewRecorder()
	api.createPayment(partialPayRec,partialPayReq)
	if partialPayRec.Code!=201{t.Fatalf("partial payment: %d %s",partialPayRec.Code,partialPayRec.Body.String())}

	editBody:=[]byte(fmt.Sprintf(`{
		"customerName":"Cambio indebido",
		"customerPhone":"",
		"address":"",
		"reference":"",
		"notes":"",
		"deliveryFee":0,
		"items":[{"productId":%q,"name":"ignorado","qty":2,"unitPrice":1,"note":"","selections":[]}]
	}`,productID))
	editReq:=httptest.NewRequest("PATCH","/v1/admin/orders/"+created.ID,bytes.NewReader(editBody))
	editReq.SetPathValue("id",created.ID)
	editReq=editReq.WithContext(context.WithValue(editReq.Context(),scopeKey{},s))
	editRec:=httptest.NewRecorder()
	api.updateOrder(editRec,editReq)
	if editRec.Code!=409||!strings.Contains(editRec.Body.String(),"paid_order_not_editable"){
		t.Fatalf("paid comanda edit must be blocked: %d %s",editRec.Code,editRec.Body.String())
	}

	// Salón/Pedidos cannot start preparation; only Cocina can.
	genericPrepReq:=httptest.NewRequest("PATCH","/v1/admin/orders/"+created.ID+"/status",bytes.NewReader([]byte(`{"status":"preparando"}`)))
	genericPrepReq.SetPathValue("id",created.ID)
	genericPrepReq=genericPrepReq.WithContext(context.WithValue(genericPrepReq.Context(),scopeKey{},s))
	genericPrepRec:=httptest.NewRecorder()
	api.updateOrderStatus(genericPrepRec,genericPrepReq)
	if genericPrepRec.Code!=409||!strings.Contains(genericPrepRec.Body.String(),"kitchen_transition_required"){
		t.Fatalf("generic preparation must be blocked: %d %s",genericPrepRec.Code,genericPrepRec.Body.String())
	}

	kitchenStartReq:=httptest.NewRequest("PATCH","/v1/admin/kitchen/tickets/"+created.ID+"/status",bytes.NewReader([]byte(`{"status":"preparando"}`)))
	kitchenStartReq.SetPathValue("id",created.ID)
	kitchenStartReq=kitchenStartReq.WithContext(context.WithValue(kitchenStartReq.Context(),scopeKey{},s))
	kitchenStartRec:=httptest.NewRecorder()
	api.updateKitchenTicketStatus(kitchenStartRec,kitchenStartReq)
	if kitchenStartRec.Code!=204{t.Fatalf("kitchen start: %d %s",kitchenStartRec.Code,kitchenStartRec.Body.String())}

	kitchenReadyReq:=httptest.NewRequest("PATCH","/v1/admin/kitchen/tickets/"+created.ID+"/status",bytes.NewReader([]byte(`{"status":"listo"}`)))
	kitchenReadyReq.SetPathValue("id",created.ID)
	kitchenReadyReq=kitchenReadyReq.WithContext(context.WithValue(kitchenReadyReq.Context(),scopeKey{},s))
	kitchenReadyRec:=httptest.NewRecorder()
	api.updateKitchenTicketStatus(kitchenReadyRec,kitchenReadyReq)
	if kitchenReadyRec.Code!=204{t.Fatalf("kitchen ready: %d %s",kitchenReadyRec.Code,kitchenReadyRec.Body.String())}

	// A ready table with balance cannot be delivered/freed.
	unpaidDeliverReq:=httptest.NewRequest("PATCH","/v1/admin/orders/"+created.ID+"/status",bytes.NewReader([]byte(`{"status":"entregado"}`)))
	unpaidDeliverReq.SetPathValue("id",created.ID)
	unpaidDeliverReq=unpaidDeliverReq.WithContext(context.WithValue(unpaidDeliverReq.Context(),scopeKey{},s))
	unpaidDeliverRec:=httptest.NewRecorder()
	api.updateOrderStatus(unpaidDeliverRec,unpaidDeliverReq)
	if unpaidDeliverRec.Code!=409||!strings.Contains(unpaidDeliverRec.Body.String(),"payment_required_before_delivery"){
		t.Fatalf("unpaid salon delivery must be blocked: %d %s",unpaidDeliverRec.Code,unpaidDeliverRec.Body.String())
	}

	payReq:=httptest.NewRequest("POST","/v1/admin/payments",bytes.NewReader([]byte(fmt.Sprintf(`{"orderId":%q,"method":"card","amount":20,"reference":"SALDO"}`,created.ID))))
	payReq=payReq.WithContext(context.WithValue(payReq.Context(),scopeKey{},s))
	payRec:=httptest.NewRecorder()
	api.createPayment(payRec,payReq)
	if payRec.Code!=201{t.Fatalf("payment: %d %s",payRec.Code,payRec.Body.String())}

	// Paid orders cannot be cancelled until their payments are refunded.
	cancelReq:=httptest.NewRequest("PATCH","/v1/admin/orders/"+created.ID+"/status",bytes.NewReader([]byte(`{"status":"cancelado"}`)))
	cancelReq.SetPathValue("id",created.ID)
	cancelReq=cancelReq.WithContext(context.WithValue(cancelReq.Context(),scopeKey{},s))
	cancelRec:=httptest.NewRecorder()
	api.updateOrderStatus(cancelRec,cancelReq)
	if cancelRec.Code!=409||!strings.Contains(cancelRec.Body.String(),"refund_required_before_cancel"){
		t.Fatalf("paid cancellation must be blocked: %d %s",cancelRec.Code,cancelRec.Body.String())
	}

	deliverReq:=httptest.NewRequest("PATCH","/v1/admin/orders/"+created.ID+"/status",bytes.NewReader([]byte(`{"status":"entregado"}`)))
	deliverReq.SetPathValue("id",created.ID)
	deliverReq=deliverReq.WithContext(context.WithValue(deliverReq.Context(),scopeKey{},s))
	deliverRec:=httptest.NewRecorder()
	api.updateOrderStatus(deliverRec,deliverReq)
	if deliverRec.Code!=200{t.Fatalf("paid ready order should deliver: %d %s",deliverRec.Code,deliverRec.Body.String())}

	floorReq:=httptest.NewRequest("GET","/v1/admin/orders/floor",nil)
	floorReq=floorReq.WithContext(context.WithValue(floorReq.Context(),scopeKey{},s))
	floorRec:=httptest.NewRecorder()
	api.getOrdersFloor(floorRec,floorReq)
	if floorRec.Code!=200{t.Fatalf("floor: %d %s",floorRec.Code,floorRec.Body.String())}
	var floor struct{Items []struct{ID string `json:"id"`;Order *order `json:"order"`} `json:"items"`}
	if err:=json.Unmarshal(floorRec.Body.Bytes(),&floor);err!=nil{t.Fatal(err)}
	for _,item:=range floor.Items{
		if item.ID==tableID&&item.Order!=nil{t.Fatalf("table must be free after paid delivery, got order %#v",item.Order)}
	}
}


func TestSalonTableOpeningIsSerializedAndChannelScoped(t *testing.T) {
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool)
	api:=New(pool)
	ctx:=context.Background()
	nonce:=time.Now().UnixNano()

	var productID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control)
		VALUES($1,$2,$3,10,true,'prepared','none')
		RETURNING id
	`,s.OrganizationID,fmt.Sprintf("TABLE-%d",nonce),fmt.Sprintf("Plato mesa %d",nonce)).Scan(&productID);err!=nil{t.Fatal(err)}

	var tableID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO tables(organization_id,location_id,name,seats,zone,active,qr_token,qr_enabled)
		VALUES($1,$2,$3,4,'Principal',true,encode(gen_random_bytes(16),'hex'),false)
		RETURNING id
	`,s.OrganizationID,s.LocationID,fmt.Sprintf("Mesa Race %d",nonce)).Scan(&tableID);err!=nil{t.Fatal(err)}
	t.Cleanup(func(){
		_,_=pool.Exec(context.Background(),`DELETE FROM orders WHERE organization_id=$1 AND table_id=$2`,s.OrganizationID,tableID)
		_,_=pool.Exec(context.Background(),`DELETE FROM tables WHERE id=$1 AND organization_id=$2`,tableID,s.OrganizationID)
	})

	salonNoTableReq:=httptest.NewRequest("POST","/v1/admin/orders",bytes.NewReader([]byte(fmt.Sprintf(
		`{"channel":"salon","items":[{"productId":%q,"qty":1,"unitPrice":10,"selections":[]}]}`,productID))))
	salonNoTableReq=salonNoTableReq.WithContext(context.WithValue(salonNoTableReq.Context(),scopeKey{},s))
	salonNoTableRec:=httptest.NewRecorder()
	api.createOrder(salonNoTableRec,salonNoTableReq)
	if salonNoTableRec.Code!=400||!strings.Contains(salonNoTableRec.Body.String(),"table_required"){
		t.Fatalf("salon without table must be rejected: %d %s",salonNoTableRec.Code,salonNoTableRec.Body.String())
	}

	counterBody:=[]byte(fmt.Sprintf(
		`{"channel":"mostrador","tableId":%q,"items":[{"productId":%q,"qty":1,"unitPrice":10,"selections":[]}]}`,
		tableID,productID))
	counterReq:=httptest.NewRequest("POST","/v1/admin/orders",bytes.NewReader(counterBody))
	counterReq=counterReq.WithContext(context.WithValue(counterReq.Context(),scopeKey{},s))
	counterRec:=httptest.NewRecorder()
	api.createOrder(counterRec,counterReq)
	if counterRec.Code!=400||!strings.Contains(counterRec.Body.String(),"table_not_allowed"){
		t.Fatalf("non-salon table ownership must be rejected: %d %s",counterRec.Code,counterRec.Body.String())
	}

	body:=[]byte(fmt.Sprintf(
		`{"channel":"salon","tableId":%q,"items":[{"productId":%q,"qty":1,"unitPrice":10,"selections":[]}]}`,
		tableID,productID))
	codes:=make(chan int,2)
	bodies:=make(chan string,2)
	var wg sync.WaitGroup
	for i:=0;i<2;i++{
		wg.Add(1)
		go func(){
			defer wg.Done()
			req:=httptest.NewRequest("POST","/v1/admin/orders",bytes.NewReader(body))
			req=req.WithContext(context.WithValue(req.Context(),scopeKey{},s))
			rec:=httptest.NewRecorder()
			api.createOrder(rec,req)
			codes<-rec.Code
			bodies<-rec.Body.String()
		}()
	}
	wg.Wait()
	close(codes)
	close(bodies)

	successes,conflicts:=0,0
	for code:=range codes{
		if code==201{successes++}
		if code==409{conflicts++}
	}
	allBodies:=""
	for body:=range bodies{allBodies+=body}
	if successes!=1||conflicts!=1||!strings.Contains(allBodies,"table_occupied"){
		t.Fatalf("expected one opened table and one table_occupied conflict, success=%d conflict=%d bodies=%s",successes,conflicts,allBodies)
	}

	var openOrders int
	if err:=pool.QueryRow(ctx,`
		SELECT count(*)
		FROM orders
		WHERE organization_id=$1 AND table_id=$2 AND status NOT IN ('entregado','cancelado')
	`,s.OrganizationID,tableID).Scan(&openOrders);err!=nil{t.Fatal(err)}
	if openOrders!=1{t.Fatalf("expected exactly one open order for table, got %d",openOrders)}
}


func TestTablesAndZonesAreIsolatedByLocation(t *testing.T) {
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool)
	api:=New(pool)
	ctx:=context.Background()
	nonce:=time.Now().UnixNano()

	var secondLocationID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO locations(organization_id,name,code,address)
		VALUES($1,$2,$3,'')
		RETURNING id
	`,s.OrganizationID,fmt.Sprintf("Sucursal %d",nonce),fmt.Sprintf("S%d",nonce%1000000)).Scan(&secondLocationID);err!=nil{t.Fatal(err)}
	s2:=s
	s2.LocationID=secondLocationID

	tableName:=fmt.Sprintf("Mesa compartida %d",nonce)
	tableBody:=[]byte(fmt.Sprintf(`{"name":%q,"seats":4,"zone":"Terraza"}`,tableName))

	createTableFor:=func(current scope) table {
		req:=httptest.NewRequest("POST","/v1/admin/tables",bytes.NewReader(tableBody))
		req=req.WithContext(context.WithValue(req.Context(),scopeKey{},current))
		rec:=httptest.NewRecorder()
		api.createTable(rec,req)
		if rec.Code!=201{t.Fatalf("create table at %s: %d %s",current.LocationID,rec.Code,rec.Body.String())}
		var value table
		if err:=json.Unmarshal(rec.Body.Bytes(),&value);err!=nil{t.Fatal(err)}
		return value
	}
	firstTable:=createTableFor(s)
	secondTable:=createTableFor(s2)
	if firstTable.ID==secondTable.ID{t.Fatal("locations must have different physical table rows")}

	zoneName:=fmt.Sprintf("Terraza %d",nonce)
	createZoneFor:=func(current scope) {
		req:=httptest.NewRequest("POST","/v1/admin/zones",bytes.NewReader([]byte(fmt.Sprintf(`{"name":%q,"sortOrder":1}`,zoneName))))
		req=req.WithContext(context.WithValue(req.Context(),scopeKey{},current))
		rec:=httptest.NewRecorder()
		api.createZone(rec,req)
		if rec.Code!=201{t.Fatalf("create zone at %s: %d %s",current.LocationID,rec.Code,rec.Body.String())}
	}
	createZoneFor(s)
	createZoneFor(s2)

	t.Cleanup(func(){
		_,_=pool.Exec(context.Background(),`DELETE FROM zones WHERE organization_id=$1 AND name=$2`,s.OrganizationID,zoneName)
		_,_=pool.Exec(context.Background(),`DELETE FROM tables WHERE organization_id=$1 AND name=$2`,s.OrganizationID,tableName)
		_,_=pool.Exec(context.Background(),`DELETE FROM locations WHERE id=$1 AND organization_id=$2`,secondLocationID,s.OrganizationID)
	})

	assertTableList:=func(current scope,wantID string){
		req:=httptest.NewRequest("GET","/v1/admin/tables?page=1&pageSize=20",nil)
		req=req.WithContext(context.WithValue(req.Context(),scopeKey{},current))
		rec:=httptest.NewRecorder()
		api.listTables(rec,req)
		if rec.Code!=200{t.Fatalf("list tables: %d %s",rec.Code,rec.Body.String())}
		var body struct{Items []table `json:"items"`;Total int `json:"total"`}
		if err:=json.Unmarshal(rec.Body.Bytes(),&body);err!=nil{t.Fatal(err)}
		found:=false
		for _,item:=range body.Items{if item.ID==wantID{found=true}}
		if !found{t.Fatalf("location %s did not return its table %s: %#v",current.LocationID,wantID,body.Items)}
		for _,item:=range body.Items{
			if item.Name==tableName&&item.ID!=wantID{
				t.Fatalf("location %s leaked table %s from another site",current.LocationID,item.ID)
			}
		}
	}
	assertTableList(s,firstTable.ID)
	assertTableList(s2,secondTable.ID)

	qrReq:=httptest.NewRequest("GET","/v1/public/tables/"+secondTable.QrToken,nil)
	qrReq.SetPathValue("token",secondTable.QrToken)
	qrRec:=httptest.NewRecorder()
	api.getTableByQR(qrRec,qrReq)
	if qrRec.Code!=200{t.Fatalf("public QR: %d %s",qrRec.Code,qrRec.Body.String())}
	var qr struct{LocName string `json:"locationName"`}
	if err:=json.Unmarshal(qrRec.Body.Bytes(),&qr);err!=nil{t.Fatal(err)}
	if !strings.HasPrefix(qr.LocName,"Sucursal "){t.Fatalf("QR resolved wrong location: %#v",qr)}

	foreignTableBody:=[]byte(fmt.Sprintf(
		`{"channel":"salon","tableId":%q,"items":[{"productId":"00000000-0000-0000-0000-000000000001","qty":1,"unitPrice":1,"selections":[]}]}`,
		firstTable.ID))
	foreignReq:=httptest.NewRequest("POST","/v1/admin/orders",bytes.NewReader(foreignTableBody))
	foreignReq=foreignReq.WithContext(context.WithValue(foreignReq.Context(),scopeKey{},s2))
	foreignRec:=httptest.NewRecorder()
	api.createOrder(foreignRec,foreignReq)
	if foreignRec.Code!=400||!strings.Contains(foreignRec.Body.String(),"invalid_order"){
		t.Fatalf("cross-location table use must be rejected: %d %s",foreignRec.Code,foreignRec.Body.String())
	}
}
