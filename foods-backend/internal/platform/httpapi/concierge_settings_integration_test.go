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

func TestConciergeSettingsControlPublicQRCode(t *testing.T) {
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool)
	api:=New(pool)
	ctx:=context.Background()
	nonce:=time.Now().UnixNano()

	var tableID,token string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO tables(
		  organization_id,location_id,name,seats,zone,active,qr_token,qr_enabled
		)
		VALUES($1,$2,$3,4,'Principal',true,encode(gen_random_bytes(16),'hex'),true)
		RETURNING id,qr_token
	`,s.OrganizationID,s.LocationID,fmt.Sprintf("Mesa settings %d",nonce)).Scan(&tableID,&token);err!=nil{
		t.Fatal(err)
	}
	t.Cleanup(func(){
		_,_=pool.Exec(context.Background(),`DELETE FROM tables WHERE id=$1 AND organization_id=$2`,tableID,s.OrganizationID)
	})

	publicReq:=httptest.NewRequest("GET","/v1/public/tables/"+token,nil)
	publicReq.SetPathValue("token",token)
	publicRec:=httptest.NewRecorder()
	api.getTableByQR(publicRec,publicReq)
	if publicRec.Code!=200{t.Fatalf("public table before settings: %d %s",publicRec.Code,publicRec.Body.String())}
	var before struct{
		ConciergeEnabled bool `json:"conciergeEnabled"`
		WhatsAppPhone string `json:"whatsappPhone"`
	}
	if err:=json.Unmarshal(publicRec.Body.Bytes(),&before);err!=nil{t.Fatal(err)}
	if before.ConciergeEnabled||before.WhatsAppPhone!=""{
		t.Fatalf("concierge must start disabled: %#v",before)
	}

	invalidReq:=httptest.NewRequest("PATCH","/v1/admin/concierge-settings",bytes.NewReader([]byte(`{"active":true,"whatsappPhone":"999"}`)))
	invalidReq=invalidReq.WithContext(context.WithValue(invalidReq.Context(),scopeKey{},s))
	invalidRec:=httptest.NewRecorder()
	api.updateConciergeSettings(invalidRec,invalidReq)
	if invalidRec.Code!=400||!strings.Contains(invalidRec.Body.String(),"invalid_whatsapp_phone"){
		t.Fatalf("invalid phone must be rejected: %d %s",invalidRec.Code,invalidRec.Body.String())
	}

	saveReq:=httptest.NewRequest("PATCH","/v1/admin/concierge-settings",bytes.NewReader([]byte(`{"active":true,"whatsappPhone":"+51987654321"}`)))
	saveReq=saveReq.WithContext(context.WithValue(saveReq.Context(),scopeKey{},s))
	saveRec:=httptest.NewRecorder()
	api.updateConciergeSettings(saveRec,saveReq)
	if saveRec.Code!=200{t.Fatalf("save concierge settings: %d %s",saveRec.Code,saveRec.Body.String())}

	publicAfterReq:=httptest.NewRequest("GET","/v1/public/tables/"+token,nil)
	publicAfterReq.SetPathValue("token",token)
	publicAfterRec:=httptest.NewRecorder()
	api.getTableByQR(publicAfterRec,publicAfterReq)
	if publicAfterRec.Code!=200{t.Fatalf("public table after settings: %d %s",publicAfterRec.Code,publicAfterRec.Body.String())}
	var after struct{
		ConciergeEnabled bool `json:"conciergeEnabled"`
		WhatsAppPhone string `json:"whatsappPhone"`
	}
	if err:=json.Unmarshal(publicAfterRec.Body.Bytes(),&after);err!=nil{t.Fatal(err)}
	if !after.ConciergeEnabled||after.WhatsAppPhone!="+51987654321"{
		t.Fatalf("public QR must expose enabled Concierge config: %#v",after)
	}
}
