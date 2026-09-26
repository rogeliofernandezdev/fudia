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

func TestConciergeHandoffLifecycle(t *testing.T) {
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool)
	api:=New(pool)
	ctx:=context.Background()
	nonce:=time.Now().UnixNano()
	t.Setenv("FUDIA_CONCIERGE_API_KEY","concierge-test-key")

	var tableID,qrToken string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO tables(
		  organization_id,location_id,name,seats,zone,active,qr_token,qr_enabled
		)
		VALUES($1,$2,$3,4,'Principal',true,encode(gen_random_bytes(16),'hex'),true)
		RETURNING id,qr_token
	`,s.OrganizationID,s.LocationID,fmt.Sprintf("Mesa Handoff %d",nonce)).
		Scan(&tableID,&qrToken);err!=nil{t.Fatal(err)}
	if _,err:=pool.Exec(ctx,`
		INSERT INTO organization_modules(organization_id,module_key,active)
		VALUES($1,'whatsapp_bot',true)
		ON CONFLICT(organization_id,module_key)
		DO UPDATE SET active=true,updated_at=now()
	`,s.OrganizationID);err!=nil{t.Fatal(err)}
	t.Cleanup(func(){
		_,_=pool.Exec(context.Background(),`DELETE FROM concierge_handoffs WHERE organization_id=$1 AND table_id=$2`,s.OrganizationID,tableID)
		_,_=pool.Exec(context.Background(),`DELETE FROM tables WHERE id=$1 AND organization_id=$2`,tableID,s.OrganizationID)
	})

	conversationID:=fmt.Sprintf("handoff-%d",nonce)
	body:=[]byte(fmt.Sprintf(`{
	  "conversationId":%q,
	  "customerPhone":"51999999999",
	  "reason":"Necesito ayuda con mi pedido"
	}`,conversationID))
	request:=httptest.NewRequest("POST","/v1/integrations/concierge/"+qrToken+"/handoffs",bytes.NewReader(body))
	request.Header.Set("X-Fudia-Concierge-Key","concierge-test-key")
	rec:=httptest.NewRecorder()
	api.Routes().ServeHTTP(rec,request)
	if rec.Code!=200{t.Fatalf("request handoff: %d %s",rec.Code,rec.Body.String())}

	var created conciergeHandoffView
	if err:=json.Unmarshal(rec.Body.Bytes(),&created);err!=nil{t.Fatal(err)}
	if created.ID==""||created.TableID!=tableID||created.Status!="pending"||created.Reason!="Necesito ayuda con mi pedido"{
		t.Fatalf("unexpected handoff: %#v",created)
	}

	repeat:=httptest.NewRequest("POST","/v1/integrations/concierge/"+qrToken+"/handoffs",bytes.NewReader(body))
	repeat.Header.Set("X-Fudia-Concierge-Key","concierge-test-key")
	repeatRec:=httptest.NewRecorder()
	api.Routes().ServeHTTP(repeatRec,repeat)
	if repeatRec.Code!=200{t.Fatalf("repeat handoff: %d %s",repeatRec.Code,repeatRec.Body.String())}
	var repeated conciergeHandoffView
	if err:=json.Unmarshal(repeatRec.Body.Bytes(),&repeated);err!=nil{t.Fatal(err)}
	if repeated.ID!=created.ID{t.Fatalf("handoff must be idempotent while pending: %s != %s",repeated.ID,created.ID)}

	listReq:=httptest.NewRequest("GET","/v1/operations/concierge-handoffs?status=pending",nil)
	listReq=listReq.WithContext(context.WithValue(listReq.Context(),scopeKey{},s))
	listRec:=httptest.NewRecorder()
	api.listOperationalConciergeHandoffs(listRec,listReq)
	if listRec.Code!=200{t.Fatalf("list handoffs: %d %s",listRec.Code,listRec.Body.String())}
	var listed struct{Items []conciergeHandoffView `json:"items"`;Total int `json:"total"`}
	if err:=json.Unmarshal(listRec.Body.Bytes(),&listed);err!=nil{t.Fatal(err)}
	found:=false
	for _,item:=range listed.Items{if item.ID==created.ID{found=true}}
	if !found{t.Fatalf("pending handoff not returned: %#v",listed)}

	resolveReq:=httptest.NewRequest("PATCH","/v1/operations/concierge-handoffs/"+created.ID+"/resolve",nil)
	resolveReq.SetPathValue("id",created.ID)
	resolveReq=resolveReq.WithContext(context.WithValue(resolveReq.Context(),scopeKey{},s))
	resolveRec:=httptest.NewRecorder()
	api.resolveOperationalConciergeHandoff(resolveRec,resolveReq)
	if resolveRec.Code!=204{t.Fatalf("resolve handoff: %d %s",resolveRec.Code,resolveRec.Body.String())}

	statusReq:=httptest.NewRequest("GET","/v1/integrations/concierge/"+qrToken+"/handoffs/"+conversationID,nil)
	statusReq.Header.Set("X-Fudia-Concierge-Key","concierge-test-key")
	statusRec:=httptest.NewRecorder()
	api.Routes().ServeHTTP(statusRec,statusReq)
	if statusRec.Code!=200{t.Fatalf("handoff status: %d %s",statusRec.Code,statusRec.Body.String())}
	var resolved conciergeHandoffView
	if err:=json.Unmarshal(statusRec.Body.Bytes(),&resolved);err!=nil{t.Fatal(err)}
	if resolved.Status!="resolved"||resolved.ResolvedAt==nil||resolved.ResolvedByName==nil{
		t.Fatalf("unexpected resolved handoff: %#v",resolved)
	}
}
