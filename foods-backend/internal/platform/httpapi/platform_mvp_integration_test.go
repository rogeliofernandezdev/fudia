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

func TestPlatformOnboardingCreatesOperationalTenant(t *testing.T){
	pool:=integrationPool(t)
	actor:=seedInventoryScope(t,pool)
	api:=New(pool)
	nonce:=time.Now().UnixNano()
	email:=fmt.Sprintf("owner-%d@example.test",nonce)
	password:="OwnerPass123"
	taxID:=fmt.Sprintf("%011d",nonce%100000000000)
	body:=[]byte(fmt.Sprintf(`{
		"legalName":"MVP Restaurant SAC",
		"tradeName":"MVP Restaurant",
		"taxId":%q,
		"timezone":"America/Lima",
		"country":"PE",
		"currency":"PEN",
		"currencyPosition":"before",
		"taxName":"IGV",
		"taxRate":"0.18",
		"taxIncluded":false,
		"locationName":"Local principal",
		"locationCode":"PRINCIPAL",
		"address":"Dirección de prueba",
		"adminName":"Propietario MVP",
		"adminEmail":%q,
		"adminPassword":%q
	}`,taxID,email,password))
	req:=httptest.NewRequest("POST","/v1/platform/organizations",bytes.NewReader(body))
	req=req.WithContext(context.WithValue(req.Context(),scopeKey{},actor))
	rec:=httptest.NewRecorder()
	api.onboardTenant(rec,req)
	if rec.Code!=201{t.Fatalf("onboarding: %d %s",rec.Code,rec.Body.String())}
	var created struct{OrganizationID,LocationID,AdministratorID string}
	if err:=json.Unmarshal(rec.Body.Bytes(),&created);err!=nil{t.Fatal(err)}
	if created.OrganizationID==""||created.LocationID==""||created.AdministratorID==""{t.Fatalf("incomplete onboarding response: %#v",created)}

	var activeMVP,inactiveFuture int
	if err:=pool.QueryRow(context.Background(),`SELECT count(*) FROM organization_modules WHERE organization_id=$1 AND active`,created.OrganizationID).Scan(&activeMVP);err!=nil{t.Fatal(err)}
	if activeMVP!=len(mvpModuleKeys){t.Fatalf("expected %d MVP modules active, got %d",len(mvpModuleKeys),activeMVP)}
	if err:=pool.QueryRow(context.Background(),`SELECT count(*) FROM organization_modules WHERE organization_id=$1 AND module_key IN ('facturacion','integraciones','crm','bi') AND NOT active`,created.OrganizationID).Scan(&inactiveFuture);err!=nil{t.Fatal(err)}
	if inactiveFuture!=4{t.Fatalf("future modules must start inactive, got %d",inactiveFuture)}

	var adminRoles,defaults int
	if err:=pool.QueryRow(context.Background(),`SELECT count(*) FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE ur.user_id=$1 AND ur.location_id=$2 AND r.system_key='administrator' AND r.permissions=ARRAY['*']::text[] AND r.menu_access=ARRAY['*']::text[]`,created.AdministratorID,created.LocationID).Scan(&adminRoles);err!=nil{t.Fatal(err)}
	if adminRoles!=1{t.Fatalf("tenant owner did not receive administrator role")}
	if err:=pool.QueryRow(context.Background(),`SELECT count(*) FROM roles WHERE organization_id=$1 AND system_key IS NOT NULL`,created.OrganizationID).Scan(&defaults);err!=nil{t.Fatal(err)}
	if defaults!=len(defaultOrganizationRoles){t.Fatalf("expected %d predefined roles, got %d",len(defaultOrganizationRoles),defaults)}

	loginReq:=httptest.NewRequest("POST","/v1/auth/login",bytes.NewReader([]byte(fmt.Sprintf(`{"email":%q,"password":%q}`,email,password))))
	loginRec:=httptest.NewRecorder()
	api.login(loginRec,loginReq)
	if loginRec.Code!=200||!strings.Contains(loginRec.Header().Get("Set-Cookie"),"foods_session="){t.Fatalf("new tenant administrator cannot login: %d %s",loginRec.Code,loginRec.Body.String())}
}

func TestReservationsPersistCapacityAndSchedule(t *testing.T){
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool)
	api:=New(pool)
	nonce:=time.Now().UnixNano()
	var tableID string
	if err:=pool.QueryRow(context.Background(),`
		INSERT INTO tables(organization_id,location_id,name,seats,zone,active,qr_token,qr_enabled)
		VALUES($1,$2,$3,4,'Principal',true,encode(gen_random_bytes(16),'hex'),false)
		RETURNING id
	`,s.OrganizationID,s.LocationID,fmt.Sprintf("Mesa reserva %d",nonce)).Scan(&tableID);err!=nil{t.Fatal(err)}
	starts:=time.Now().UTC().Add(24*time.Hour).Truncate(time.Minute).Format(time.RFC3339)
	create:=func(guests int)*httptest.ResponseRecorder{
		body:=[]byte(fmt.Sprintf(`{"customerName":"Familia Test","customerPhone":"999999999","startsAt":%q,"guests":%d,"tableId":%q,"notes":"Ventana"}`,starts,guests,tableID))
		req:=httptest.NewRequest("POST","/v1/admin/reservations",bytes.NewReader(body))
		req=req.WithContext(context.WithValue(req.Context(),scopeKey{},s))
		rec:=httptest.NewRecorder();api.createReservation(rec,req);return rec
	}
	tooMany:=create(5)
	if tooMany.Code!=400||!strings.Contains(tooMany.Body.String(),"table_capacity"){t.Fatalf("capacity must be enforced: %d %s",tooMany.Code,tooMany.Body.String())}
	ok:=create(4)
	if ok.Code!=201{t.Fatalf("create reservation: %d %s",ok.Code,ok.Body.String())}
	var item reservationView
	if err:=json.Unmarshal(ok.Body.Bytes(),&item);err!=nil{t.Fatal(err)}
	if item.TableID==nil||*item.TableID!=tableID||item.Status!="pending"{t.Fatalf("unexpected reservation: %#v",item)}
	conflict:=create(2)
	if conflict.Code!=400||!strings.Contains(conflict.Body.String(),"reservation_conflict"){t.Fatalf("same table/time must conflict: %d %s",conflict.Code,conflict.Body.String())}

	statusReq:=httptest.NewRequest("PATCH","/v1/admin/reservations/"+item.ID+"/status",bytes.NewReader([]byte(`{"status":"confirmed"}`)))
	statusReq.SetPathValue("id",item.ID)
	statusReq=statusReq.WithContext(context.WithValue(statusReq.Context(),scopeKey{},s))
	statusRec:=httptest.NewRecorder();api.updateReservationStatus(statusRec,statusReq)
	if statusRec.Code!=204{t.Fatalf("confirm reservation: %d %s",statusRec.Code,statusRec.Body.String())}

	listReq:=httptest.NewRequest("GET","/v1/admin/reservations?status=confirmed&page=1&pageSize=20",nil)
	listReq=listReq.WithContext(context.WithValue(listReq.Context(),scopeKey{},s))
	listRec:=httptest.NewRecorder();api.listReservations(listRec,listReq)
	if listRec.Code!=200||!strings.Contains(listRec.Body.String(),item.ID){t.Fatalf("confirmed reservation not listed: %d %s",listRec.Code,listRec.Body.String())}
}
