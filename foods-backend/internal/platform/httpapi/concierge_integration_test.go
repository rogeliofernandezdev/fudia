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

func TestConciergeQRCodeMenuOrderAndKitchenWorkflow(t *testing.T) {
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool)
	api:=New(pool)
	ctx:=context.Background()
	nonce:=time.Now().UnixNano()
	t.Setenv("FUDIA_CONCIERGE_API_KEY","concierge-test-key")

	var productID string
	productName:=fmt.Sprintf("Plato Concierge %d",nonce)
	if err:=pool.QueryRow(ctx,`
		INSERT INTO products(
		  organization_id,sku,name,description,price,active,product_type,quantity_control
		)
		VALUES($1,$2,$3,'Preparado para Concierge',27.50,true,'prepared','none')
		RETURNING id
	`,s.OrganizationID,fmt.Sprintf("CON-%d",nonce),productName).Scan(&productID);err!=nil{
		t.Fatal(err)
	}


	var comboID,optionAID,optionBID,groupID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO products(
		  organization_id,sku,name,description,price,active,product_type,quantity_control
		)
		VALUES($1,$2,'Combo Conversacional','Combo para prueba',20,true,'prepared','none')
		RETURNING id
	`,s.OrganizationID,fmt.Sprintf("CCB-%d",nonce)).Scan(&comboID);err!=nil{t.Fatal(err)}
	if _,err:=pool.Exec(ctx,`INSERT INTO menu_combos(product_id,organization_id) VALUES($1,$2)`,comboID,s.OrganizationID);err!=nil{t.Fatal(err)}
	if err:=pool.QueryRow(ctx,`
		INSERT INTO products(organization_id,sku,name,description,price,active,product_type,quantity_control)
		VALUES($1,$2,'Papas clásicas','Opción del combo',8,true,'prepared','none')
		RETURNING id
	`,s.OrganizationID,fmt.Sprintf("OPA-%d",nonce)).Scan(&optionAID);err!=nil{t.Fatal(err)}
	if err:=pool.QueryRow(ctx,`
		INSERT INTO products(organization_id,sku,name,description,price,active,product_type,quantity_control)
		VALUES($1,$2,'Papas especiales','Opción con recargo',11,true,'prepared','none')
		RETURNING id
	`,s.OrganizationID,fmt.Sprintf("OPB-%d",nonce)).Scan(&optionBID);err!=nil{t.Fatal(err)}
	if err:=pool.QueryRow(ctx,`
		INSERT INTO menu_combo_groups(
		  organization_id,combo_product_id,name,required,min_selections,max_selections,sort_order
		)
		VALUES($1,$2,'Acompañamiento',true,1,1,0)
		RETURNING id
	`,s.OrganizationID,comboID).Scan(&groupID);err!=nil{t.Fatal(err)}
	if _,err:=pool.Exec(ctx,`
		INSERT INTO menu_combo_options(
		  organization_id,group_id,option_product_id,surcharge,sort_order
		)
		VALUES($1,$2,$3,0,0),($1,$2,$4,3,1)
	`,s.OrganizationID,groupID,optionAID,optionBID);err!=nil{t.Fatal(err)}

	var tableID,qrToken string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO tables(
		  organization_id,location_id,name,seats,zone,active,qr_token,qr_enabled
		)
		VALUES($1,$2,$3,4,'Principal',true,encode(gen_random_bytes(16),'hex'),true)
		RETURNING id,qr_token
	`,s.OrganizationID,s.LocationID,fmt.Sprintf("Mesa Concierge %d",nonce)).
		Scan(&tableID,&qrToken);err!=nil{
		t.Fatal(err)
	}
	if _,err:=pool.Exec(ctx,`
		INSERT INTO organization_modules(organization_id,module_key,active)
		VALUES($1,'whatsapp_bot',true)
		ON CONFLICT(organization_id,module_key)
		DO UPDATE SET active=true,updated_at=now()
	`,s.OrganizationID);err!=nil{
		t.Fatal(err)
	}
	t.Cleanup(func(){
		_,_=pool.Exec(context.Background(),`DELETE FROM concierge_order_requests WHERE organization_id=$1 AND table_id=$2`,s.OrganizationID,tableID)
		_,_=pool.Exec(context.Background(),`DELETE FROM orders WHERE organization_id=$1 AND table_id=$2`,s.OrganizationID,tableID)
		_,_=pool.Exec(context.Background(),`DELETE FROM menu_combo_groups WHERE organization_id=$1 AND combo_product_id=$2`,s.OrganizationID,comboID)
		_,_=pool.Exec(context.Background(),`DELETE FROM menu_combos WHERE organization_id=$1 AND product_id=$2`,s.OrganizationID,comboID)
		_,_=pool.Exec(context.Background(),`DELETE FROM tables WHERE id=$1 AND organization_id=$2`,tableID,s.OrganizationID)
	})

	unauthorizedReq:=httptest.NewRequest("GET","/v1/integrations/concierge/"+qrToken+"/menu?q=Concierge",nil)
	unauthorizedRec:=httptest.NewRecorder()
	api.Routes().ServeHTTP(unauthorizedRec,unauthorizedReq)
	if unauthorizedRec.Code!=401||!strings.Contains(unauthorizedRec.Body.String(),"concierge_unauthorized"){
		t.Fatalf("concierge API must require service credential: %d %s",unauthorizedRec.Code,unauthorizedRec.Body.String())
	}

	menuReq:=httptest.NewRequest("GET","/v1/integrations/concierge/"+qrToken+"/menu?q=Concierge",nil)
	menuReq.Header.Set("X-Fudia-Concierge-Key","concierge-test-key")
	menuRec:=httptest.NewRecorder()
	api.Routes().ServeHTTP(menuRec,menuReq)
	if menuRec.Code!=200{
		t.Fatalf("menu: %d %s",menuRec.Code,menuRec.Body.String())
	}
	var menu struct{
		Items []conciergeMenuItem `json:"items"`
		CurrencySymbol string `json:"currencySymbol"`
	}
	if err:=json.Unmarshal(menuRec.Body.Bytes(),&menu);err!=nil{t.Fatal(err)}
	if len(menu.Items)!=1||menu.Items[0].ProductID!=productID||menu.Items[0].Price!="27.50"||menu.Items[0].Status!="available"{
		t.Fatalf("unexpected concierge menu: %#v",menu)
	}


	comboReq:=httptest.NewRequest("GET","/v1/integrations/concierge/"+qrToken+"/combos/"+comboID,nil)
	comboReq.Header.Set("X-Fudia-Concierge-Key","concierge-test-key")
	comboRec:=httptest.NewRecorder()
	api.Routes().ServeHTTP(comboRec,comboReq)
	if comboRec.Code!=200{
		t.Fatalf("combo detail: %d %s",comboRec.Code,comboRec.Body.String())
	}
	var combo conciergeComboDetail
	if err:=json.Unmarshal(comboRec.Body.Bytes(),&combo);err!=nil{t.Fatal(err)}
	if combo.ID!=comboID||len(combo.Groups)!=1||len(combo.Groups[0].Options)!=2{
		t.Fatalf("unexpected concierge combo: %#v",combo)
	}
	if combo.Groups[0].Options[1].ProductID!=optionBID||combo.Groups[0].Options[1].Surcharge!="3.00"||!combo.Groups[0].Options[1].Available{
		t.Fatalf("unexpected combo option: %#v",combo.Groups[0].Options[1])
	}

	orderBody:=[]byte(fmt.Sprintf(`{
	  "customerPhone":"51999999999",
	  "conversationId":"conv-session-%d",
	  "requestId":"round-1-%d",
	  "items":[
	    {"productId":%q,"qty":2,"unitPrice":0.01,"name":"precio inventado","note":"sin cebolla","selections":[]},
	    {"productId":%q,"qty":1,"unitPrice":0.01,"name":"combo inventado","note":"","selections":[{"groupId":%q,"productId":%q}]}
	  ]
	}`,nonce,nonce,productID,comboID,groupID,optionBID))
	orderReq:=httptest.NewRequest("POST","/v1/integrations/concierge/"+qrToken+"/orders",bytes.NewReader(orderBody))
	orderReq.Header.Set("X-Fudia-Concierge-Key","concierge-test-key")
	orderRec:=httptest.NewRecorder()
	api.Routes().ServeHTTP(orderRec,orderReq)
	if orderRec.Code!=201{
		t.Fatalf("create concierge order: %d %s",orderRec.Code,orderRec.Body.String())
	}
	var created order
	if err:=json.Unmarshal(orderRec.Body.Bytes(),&created);err!=nil{t.Fatal(err)}
	if created.Channel!="whatsapp"||created.Status!="confirmado"||created.TableID!=tableID{
		t.Fatalf("unexpected concierge order: %#v",created)
	}
	if created.Total!="78.00"{
		t.Fatalf("backend catalog price must win, got %s",created.Total)
	}

	retryReq:=httptest.NewRequest("POST","/v1/integrations/concierge/"+qrToken+"/orders",bytes.NewReader(orderBody))
	retryReq.Header.Set("X-Fudia-Concierge-Key","concierge-test-key")
	retryRec:=httptest.NewRecorder()
	api.Routes().ServeHTTP(retryRec,retryReq)
	if retryRec.Code!=200{
		t.Fatalf("idempotent retry must recover the order: %d %s",retryRec.Code,retryRec.Body.String())
	}
	var retried order
	if err:=json.Unmarshal(retryRec.Body.Bytes(),&retried);err!=nil{t.Fatal(err)}
	if retried.ID!=created.ID||retried.Code!=created.Code{
		t.Fatalf("retry created a different order: created=%s retried=%s",created.ID,retried.ID)
	}
	var orderCount int
	if err:=pool.QueryRow(ctx,`
		SELECT count(*) FROM orders
		WHERE organization_id=$1 AND location_id=$2 AND table_id=$3
	`,s.OrganizationID,s.LocationID,tableID).Scan(&orderCount);err!=nil{t.Fatal(err)}
	if orderCount!=1{
		t.Fatalf("idempotent retry duplicated the order, count=%d",orderCount)
	}

	extraBody:=[]byte(fmt.Sprintf(`{
	  "customerPhone":"51999999999",
	  "conversationId":"conv-session-%d",
	  "requestId":"round-2-%d",
	  "items":[{"productId":%q,"qty":1,"unitPrice":999,"name":"otro precio falso","note":"","selections":[]}]
	}`,nonce,nonce,productID))
	extraReq:=httptest.NewRequest("POST","/v1/integrations/concierge/"+qrToken+"/orders",bytes.NewReader(extraBody))
	extraReq.Header.Set("X-Fudia-Concierge-Key","concierge-test-key")
	extraRec:=httptest.NewRecorder()
	api.Routes().ServeHTTP(extraRec,extraReq)
	if extraRec.Code!=200{
		t.Fatalf("second concierge round must append to its open order: %d %s",extraRec.Code,extraRec.Body.String())
	}
	var expanded order
	if err:=json.Unmarshal(extraRec.Body.Bytes(),&expanded);err!=nil{t.Fatal(err)}
	if expanded.ID!=created.ID||expanded.Total!="105.50"{
		t.Fatalf("unexpected expanded concierge order: %#v",expanded)
	}
	if err:=pool.QueryRow(ctx,`
		SELECT count(*) FROM orders
		WHERE organization_id=$1 AND location_id=$2 AND table_id=$3
	`,s.OrganizationID,s.LocationID,tableID).Scan(&orderCount);err!=nil{t.Fatal(err)}
	if orderCount!=1{
		t.Fatalf("second Concierge round must reuse the open order, count=%d",orderCount)
	}
	var requestCount int
	if err:=pool.QueryRow(ctx,`
		SELECT count(*) FROM concierge_order_requests
		WHERE organization_id=$1 AND location_id=$2 AND order_id=$3
	`,s.OrganizationID,s.LocationID,created.ID).Scan(&requestCount);err!=nil{t.Fatal(err)}
	if requestCount!=2{
		t.Fatalf("expected one idempotency record per Concierge round, got %d",requestCount)
	}
	var appendAuditCount int
	if err:=pool.QueryRow(ctx,`
		SELECT count(*) FROM audit_log
		WHERE organization_id=$1 AND entity_id=$2
		  AND action='concierge.order.items_added' AND user_id IS NULL
	`,s.OrganizationID,created.ID).Scan(&appendAuditCount);err!=nil{t.Fatal(err)}
	if appendAuditCount!=1{
		t.Fatalf("expected one Concierge append audit event, got %d",appendAuditCount)
	}

	var createdBy *string
	if err:=pool.QueryRow(ctx,`
		SELECT created_by::text
		FROM orders
		WHERE id=$1 AND organization_id=$2
	`,created.ID,s.OrganizationID).Scan(&createdBy);err!=nil{t.Fatal(err)}
	if createdBy!=nil{
		t.Fatalf("concierge order must not impersonate a staff user: %v",createdBy)
	}

	var auditCount int
	if err:=pool.QueryRow(ctx,`
		SELECT count(*)
		FROM audit_log
		WHERE organization_id=$1 AND entity_id=$2
		  AND action='concierge.order.created' AND user_id IS NULL
	`,s.OrganizationID,created.ID).Scan(&auditCount);err!=nil{t.Fatal(err)}
	if auditCount!=1{
		t.Fatalf("expected one concierge audit event, got %d",auditCount)
	}

	kitchenReq:=httptest.NewRequest("GET","/v1/admin/kitchen/tickets?channel=whatsapp",nil)
	kitchenReq=kitchenReq.WithContext(context.WithValue(kitchenReq.Context(),scopeKey{},s))
	kitchenRec:=httptest.NewRecorder()
	api.listKitchenTickets(kitchenRec,kitchenReq)
	if kitchenRec.Code!=200||!strings.Contains(kitchenRec.Body.String(),created.ID){
		t.Fatalf("concierge order must reach KDS: %d %s",kitchenRec.Code,kitchenRec.Body.String())
	}
	var kitchen struct{ Items []kitchenTicket `json:"items"` }
	if err:=json.Unmarshal(kitchenRec.Body.Bytes(),&kitchen);err!=nil{t.Fatal(err)}
	processedRounds:=0
	for _,ticket:=range kitchen.Items{
		if ticket.OrderID!=created.ID{continue}
		processedRounds++
		startReq:=httptest.NewRequest("PATCH","/v1/admin/kitchen/tickets/"+ticket.ID+"/status",bytes.NewReader([]byte(`{"status":"preparando"}`)))
		startReq.SetPathValue("id",ticket.ID)
		startReq=startReq.WithContext(context.WithValue(startReq.Context(),scopeKey{},s))
		startRec:=httptest.NewRecorder()
		api.updateKitchenTicketStatus(startRec,startReq)
		if startRec.Code!=204{t.Fatalf("kitchen round start: %d %s",startRec.Code,startRec.Body.String())}

		readyReq:=httptest.NewRequest("PATCH","/v1/admin/kitchen/tickets/"+ticket.ID+"/status",bytes.NewReader([]byte(`{"status":"listo"}`)))
		readyReq.SetPathValue("id",ticket.ID)
		readyReq=readyReq.WithContext(context.WithValue(readyReq.Context(),scopeKey{},s))
		readyRec:=httptest.NewRecorder()
		api.updateKitchenTicketStatus(readyRec,readyReq)
		if readyRec.Code!=204{t.Fatalf("kitchen round ready: %d %s",readyRec.Code,readyRec.Body.String())}
	}
	if processedRounds!=2{t.Fatalf("expected two independent kitchen rounds, got %d",processedRounds)}

	billReq:=httptest.NewRequest("POST","/v1/integrations/concierge/"+qrToken+"/bill",bytes.NewReader([]byte(fmt.Sprintf(`{"conversationId":"conv-session-%d","customerPhone":"51999999999"}`,nonce))))
	billReq.Header.Set("X-Fudia-Concierge-Key","concierge-test-key")
	billRec:=httptest.NewRecorder()
	api.Routes().ServeHTTP(billRec,billReq)
	if billRec.Code!=200||!strings.Contains(billRec.Body.String(),`"total":"105.50"`){
		t.Fatalf("bill must return backend total: %d %s",billRec.Code,billRec.Body.String())
	}

	deliverReq:=httptest.NewRequest("PATCH","/v1/admin/orders/"+created.ID+"/status",bytes.NewReader([]byte(`{"status":"entregado"}`)))
	deliverReq.SetPathValue("id",created.ID)
	deliverReq=deliverReq.WithContext(context.WithValue(deliverReq.Context(),scopeKey{},s))
	deliverRec:=httptest.NewRecorder()
	api.updateOrderStatus(deliverRec,deliverReq)
	if deliverRec.Code!=409||!strings.Contains(deliverRec.Body.String(),"payment_required_before_delivery"){
		t.Fatalf("unpaid QR table order must not release table: %d %s",deliverRec.Code,deliverRec.Body.String())
	}
}


func TestConciergeAppendsToConfirmedSalonOrderWithoutImpersonatingStaff(t *testing.T) {
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool)
	api:=New(pool)
	ctx:=context.Background()
	nonce:=time.Now().UnixNano()
	t.Setenv("FUDIA_CONCIERGE_API_KEY","concierge-test-key")

	var baseProductID,addedProductID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO products(
		  organization_id,sku,name,description,price,active,product_type,quantity_control
		)
		VALUES($1,$2,'Agua de mesa','Pedido inicial',10,true,'retail','none')
		RETURNING id
	`,s.OrganizationID,fmt.Sprintf("SAL-BASE-%d",nonce)).Scan(&baseProductID);err!=nil{t.Fatal(err)}
	if err:=pool.QueryRow(ctx,`
		INSERT INTO products(
		  organization_id,sku,name,description,price,active,product_type,quantity_control
		)
		VALUES($1,$2,'Postre Concierge','Ronda adicional',12.50,true,'prepared','none')
		RETURNING id
	`,s.OrganizationID,fmt.Sprintf("SAL-ADD-%d",nonce)).Scan(&addedProductID);err!=nil{t.Fatal(err)}

	var tableID,qrToken string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO tables(
		  organization_id,location_id,name,seats,zone,active,qr_token,qr_enabled
		)
		VALUES($1,$2,$3,4,'Principal',true,encode(gen_random_bytes(16),'hex'),true)
		RETURNING id,qr_token
	`,s.OrganizationID,s.LocationID,fmt.Sprintf("Mesa salón Concierge %d",nonce)).
		Scan(&tableID,&qrToken);err!=nil{t.Fatal(err)}
	if _,err:=pool.Exec(ctx,`
		INSERT INTO organization_modules(organization_id,module_key,active)
		VALUES($1,'whatsapp_bot',true)
		ON CONFLICT(organization_id,module_key)
		DO UPDATE SET active=true,updated_at=now()
	`,s.OrganizationID);err!=nil{t.Fatal(err)}

	var orderID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO orders(
		  organization_id,location_id,channel,status,table_id,
		  subtotal,delivery_fee,total,created_by
		)
		VALUES($1,$2,'salon','confirmado',$3,10,0,10,$4)
		RETURNING id
	`,s.OrganizationID,s.LocationID,tableID,s.UserID).Scan(&orderID);err!=nil{t.Fatal(err)}
	if _,err:=pool.Exec(ctx,`
		INSERT INTO order_items(
		  organization_id,order_id,product_id,name,qty,unit_price,note,item_type
		)
		VALUES($1,$2,$3,'Agua de mesa',1,10,'','product')
	`,s.OrganizationID,orderID,baseProductID);err!=nil{t.Fatal(err)}

	body:=[]byte(fmt.Sprintf(`{
	  "customerPhone":"51999999999",
	  "conversationId":"salon-session-%d",
	  "requestId":"salon-round-%d",
	  "items":[{"productId":%q,"qty":1,"note":"","selections":[]}]
	}`,nonce,nonce,addedProductID))
	req:=httptest.NewRequest("POST","/v1/integrations/concierge/"+qrToken+"/orders",bytes.NewReader(body))
	req.Header.Set("X-Fudia-Concierge-Key","concierge-test-key")
	rec:=httptest.NewRecorder()
	api.Routes().ServeHTTP(rec,req)
	if rec.Code!=200{
		t.Fatalf("Concierge must append to a confirmed salon order: %d %s",rec.Code,rec.Body.String())
	}
	var expanded order
	if err:=json.Unmarshal(rec.Body.Bytes(),&expanded);err!=nil{t.Fatal(err)}
	if expanded.ID!=orderID||expanded.Channel!="salon"||expanded.Total!="22.50"{
		t.Fatalf("unexpected expanded salon order: %#v",expanded)
	}

	var orderCount,requestCount,auditCount int
	if err:=pool.QueryRow(ctx,`
		SELECT count(*) FROM orders
		WHERE organization_id=$1 AND location_id=$2 AND table_id=$3
	`,s.OrganizationID,s.LocationID,tableID).Scan(&orderCount);err!=nil{t.Fatal(err)}
	if orderCount!=1{t.Fatalf("Concierge duplicated the salon order, count=%d",orderCount)}
	if err:=pool.QueryRow(ctx,`
		SELECT count(*) FROM concierge_order_requests
		WHERE organization_id=$1 AND location_id=$2 AND order_id=$3
	`,s.OrganizationID,s.LocationID,orderID).Scan(&requestCount);err!=nil{t.Fatal(err)}
	if requestCount!=1{t.Fatalf("expected one Concierge request linked to salon order, got %d",requestCount)}
	if err:=pool.QueryRow(ctx,`
		SELECT count(*) FROM audit_log
		WHERE organization_id=$1 AND entity_id=$2
		  AND action='concierge.order.items_added'
		  AND metadata->>'existingChannel'='salon'
	`,s.OrganizationID,orderID).Scan(&auditCount);err!=nil{t.Fatal(err)}
	if auditCount!=1{t.Fatalf("expected Concierge append audit on salon order, got %d",auditCount)}

	var createdBy *string
	if err:=pool.QueryRow(ctx,`
		SELECT created_by::text FROM orders
		WHERE id=$1 AND organization_id=$2
	`,orderID,s.OrganizationID).Scan(&createdBy);err!=nil{t.Fatal(err)}
	if createdBy==nil||*createdBy!=s.UserID{
		t.Fatalf("Concierge must preserve original staff ownership, got %v",createdBy)
	}
}
