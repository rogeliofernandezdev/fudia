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
	t.Cleanup(func(){
		_,_=pool.Exec(context.Background(),`DELETE FROM concierge_order_requests WHERE organization_id=$1 AND table_id=$2`,s.OrganizationID,tableID)
		_,_=pool.Exec(context.Background(),`DELETE FROM orders WHERE organization_id=$1 AND table_id=$2`,s.OrganizationID,tableID)
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

	orderBody:=[]byte(fmt.Sprintf(`{
	  "customerPhone":"51999999999",
	  "conversationId":"conv-%d",
	  "items":[{"productId":%q,"qty":2,"unitPrice":0.01,"name":"precio inventado","note":"sin cebolla","selections":[]}]
	}`,nonce,productID))
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
	if created.Total!="55.00"{
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

	startReq:=httptest.NewRequest("PATCH","/v1/admin/kitchen/tickets/"+created.ID+"/status",bytes.NewReader([]byte(`{"status":"preparando"}`)))
	startReq.SetPathValue("id",created.ID)
	startReq=startReq.WithContext(context.WithValue(startReq.Context(),scopeKey{},s))
	startRec:=httptest.NewRecorder()
	api.updateKitchenTicketStatus(startRec,startReq)
	if startRec.Code!=204{t.Fatalf("kitchen start: %d %s",startRec.Code,startRec.Body.String())}

	readyReq:=httptest.NewRequest("PATCH","/v1/admin/kitchen/tickets/"+created.ID+"/status",bytes.NewReader([]byte(`{"status":"listo"}`)))
	readyReq.SetPathValue("id",created.ID)
	readyReq=readyReq.WithContext(context.WithValue(readyReq.Context(),scopeKey{},s))
	readyRec:=httptest.NewRecorder()
	api.updateKitchenTicketStatus(readyRec,readyReq)
	if readyRec.Code!=204{t.Fatalf("kitchen ready: %d %s",readyRec.Code,readyRec.Body.String())}

	deliverReq:=httptest.NewRequest("PATCH","/v1/admin/orders/"+created.ID+"/status",bytes.NewReader([]byte(`{"status":"entregado"}`)))
	deliverReq.SetPathValue("id",created.ID)
	deliverReq=deliverReq.WithContext(context.WithValue(deliverReq.Context(),scopeKey{},s))
	deliverRec:=httptest.NewRecorder()
	api.updateOrderStatus(deliverRec,deliverReq)
	if deliverRec.Code!=409||!strings.Contains(deliverRec.Body.String(),"payment_required_before_delivery"){
		t.Fatalf("unpaid QR table order must not release table: %d %s",deliverRec.Code,deliverRec.Body.String())
	}
}
