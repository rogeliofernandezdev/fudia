package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	platformdomain "github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	platformpg "github.com/foods-platform/foods-backend/internal/modules/platformadmin/infrastructure/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPlatformOnboardingCreatesOperationalTenant(t *testing.T){
	pool:=integrationPool(t)
	actor:=seedInventoryScope(t,pool)
	api:=New(pool)
	nonce:=time.Now().UnixNano()
	seedOperationalWhatsAppChannel(t,pool,"PE",fmt.Sprint(nonce))
	email:=fmt.Sprintf("owner-%d@example.test",nonce)
	password:="OwnerPass123"
	taxID:=fmt.Sprintf("%011d",nonce%100000000000)
	planModules:=make([]string,0,len(mvpModuleKeys))
	for key:=range mvpModuleKeys{planModules=append(planModules,key)}
	var planID string
	if err:=pool.QueryRow(context.Background(),`
		INSERT INTO subscription_plans(code,name,description,currency,monthly_price,annual_price,trial_days,max_locations,max_users,module_keys,terms_version,active)
		VALUES($1,'MVP Test','Plan de prueba','PEN',99,990,14,3,20,$2,'test-v1',true)
		RETURNING id
	`,fmt.Sprintf("mvp-%d",nonce),planModules).Scan(&planID);err!=nil{t.Fatal(err)}
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
		"adminPassword":%q,
		"planId":%q,
		"billingCycle":"monthly",
		"termsAccepted":true
	}`,taxID,email,password,planID))
	req:=httptest.NewRequest("POST","/v1/platform/organizations",bytes.NewReader(body))
	req=req.WithContext(context.WithValue(req.Context(),scopeKey{},actor))
	rec:=httptest.NewRecorder()
	api.onboardTenant(rec,req)
	if rec.Code!=201{t.Fatalf("onboarding: %d %s",rec.Code,rec.Body.String())}
	var created struct{OrganizationID,FiscalProfileID,LocationID,AdministratorID string}
	if err:=json.Unmarshal(rec.Body.Bytes(),&created);err!=nil{t.Fatal(err)}
	if created.OrganizationID==""||created.LocationID==""||created.AdministratorID==""{t.Fatalf("incomplete onboarding response: %#v",created)}

	var activeMVP,inactiveFuture int
	if err:=pool.QueryRow(context.Background(),`SELECT count(*) FROM organization_modules WHERE organization_id=$1 AND active`,created.OrganizationID).Scan(&activeMVP);err!=nil{t.Fatal(err)}
	if activeMVP!=len(mvpModuleKeys){t.Fatalf("expected %d MVP modules active, got %d",len(mvpModuleKeys),activeMVP)}
	var subscriptionStatus,billingCycle,price string
	if err:=pool.QueryRow(context.Background(),`SELECT status,billing_cycle,price_amount::text FROM organization_subscriptions WHERE organization_id=$1 AND plan_id=$2`,created.OrganizationID,planID).Scan(&subscriptionStatus,&billingCycle,&price);err!=nil{t.Fatal(err)}
	if subscriptionStatus!="trial"||billingCycle!="monthly"||price!="99.00"{t.Fatalf("unexpected subscription: status=%s cycle=%s price=%s",subscriptionStatus,billingCycle,price)}
	if err:=pool.QueryRow(context.Background(),`SELECT count(*) FROM organization_modules WHERE organization_id=$1 AND module_key IN ('facturacion','integraciones','crm','bi') AND NOT active`,created.OrganizationID).Scan(&inactiveFuture);err!=nil{t.Fatal(err)}
	if inactiveFuture!=4{t.Fatalf("future modules must start inactive, got %d",inactiveFuture)}

	var adminRoles,defaults,wildcardRoles int
	if err:=pool.QueryRow(context.Background(),`
		SELECT count(*) FROM user_roles ur JOIN roles r ON r.id=ur.role_id
		WHERE ur.user_id=$1 AND ur.location_id=$2 AND r.system_key='administrator'
		  AND r.name='Administrador de empresa'
		  AND r.permissions @> ARRAY['users.manage','organizations.manage']::text[]
		  AND NOT ('*'=ANY(r.permissions)) AND NOT ('*'=ANY(r.menu_access))
	`,created.AdministratorID,created.LocationID).Scan(&adminRoles);err!=nil{t.Fatal(err)}
	if adminRoles!=1{t.Fatalf("tenant owner did not receive explicit company administrator role")}
	if err:=pool.QueryRow(context.Background(),`SELECT count(*) FROM roles WHERE organization_id=$1 AND ('*'=ANY(permissions) OR '*'=ANY(menu_access))`,created.OrganizationID).Scan(&wildcardRoles);err!=nil{t.Fatal(err)}
	if wildcardRoles!=0{t.Fatalf("tenant roles must never carry wildcard access, got %d",wildcardRoles)}
	moduleReq:=httptest.NewRequest("PATCH","/v1/admin/modules",bytes.NewReader([]byte(`{"key":"crm","active":true}`)))
	moduleReq=moduleReq.WithContext(context.WithValue(moduleReq.Context(),scopeKey{},scope{UserID:created.AdministratorID,OrganizationID:created.OrganizationID,LocationID:created.LocationID,Name:"Propietario MVP"}))
	moduleRec:=httptest.NewRecorder()
	api.requirePlatformAdmin(http.HandlerFunc(api.toggleModule)).ServeHTTP(moduleRec,moduleReq)
	if moduleRec.Code!=403||!strings.Contains(moduleRec.Body.String(),"platform_forbidden"){t.Fatalf("company administrator must not toggle modules: %d %s",moduleRec.Code,moduleRec.Body.String())}
	if err:=pool.QueryRow(context.Background(),`SELECT count(*) FROM roles WHERE organization_id=$1 AND system_key IS NOT NULL`,created.OrganizationID).Scan(&defaults);err!=nil{t.Fatal(err)}
	if defaults!=len(defaultOrganizationRoles){t.Fatalf("expected %d predefined roles, got %d",len(defaultOrganizationRoles),defaults)}
	// Roles are seeded in one statement; a retry returns the same ids in seed order without duplicating.
	retriedRoles,err:=platformpg.New(pool).SeedRoles(context.Background(),created.OrganizationID,defaultOrganizationRoles)
	if err!=nil{t.Fatal(err)}
	if len(retriedRoles)!=len(defaultOrganizationRoles){t.Fatalf("role retry returned %d ids",len(retriedRoles))}
	for i,seed:=range defaultOrganizationRoles{
		var name string
		if err:=pool.QueryRow(context.Background(),`SELECT name FROM roles WHERE id=$1 AND organization_id=$2`,retriedRoles[i],created.OrganizationID).Scan(&name);err!=nil||name!=seed.Name{t.Fatalf("role %d id out of order: %s err=%v",i,name,err)}
	}
	var retriedRoleCount int
	if err:=pool.QueryRow(context.Background(),`SELECT count(*) FROM roles WHERE organization_id=$1`,created.OrganizationID).Scan(&retriedRoleCount);err!=nil{t.Fatal(err)}
	if retriedRoleCount!=len(defaultOrganizationRoles){t.Fatalf("role retry duplicated roles: %d",retriedRoleCount)}
	var fiscalCountry,fiscalCurrency,locationProfile string
	if err:=pool.QueryRow(context.Background(),`SELECT fp.country_code,fp.currency,l.fiscal_profile_id::text FROM locations l JOIN organization_fiscal_profiles fp ON fp.id=l.fiscal_profile_id WHERE l.id=$1 AND fp.is_default`,created.LocationID).Scan(&fiscalCountry,&fiscalCurrency,&locationProfile);err!=nil{t.Fatal(err)}
	if fiscalCountry!="PE"||fiscalCurrency!="PEN"||locationProfile!=created.FiscalProfileID{t.Fatalf("batched tenant must link the configured default profile: %s %s %s",fiscalCountry,fiscalCurrency,locationProfile)}

	var paymentMethods,expenseCategories,cashRegisters,zones,setupPending int
	if err:=pool.QueryRow(context.Background(),`SELECT count(*) FROM payment_methods WHERE organization_id=$1 AND active`,created.OrganizationID).Scan(&paymentMethods);err!=nil{t.Fatal(err)}
	if paymentMethods!=4{t.Fatalf("expected 4 default payment methods, got %d",paymentMethods)}
	if err:=pool.QueryRow(context.Background(),`SELECT count(*) FROM expense_categories WHERE organization_id=$1 AND active`,created.OrganizationID).Scan(&expenseCategories);err!=nil{t.Fatal(err)}
	if expenseCategories!=6{t.Fatalf("expected 6 default expense categories, got %d",expenseCategories)}
	if err:=pool.QueryRow(context.Background(),`SELECT count(*) FROM cash_registers WHERE organization_id=$1 AND location_id=$2 AND active AND name='Caja principal'`,created.OrganizationID,created.LocationID).Scan(&cashRegisters);err!=nil{t.Fatal(err)}
	if cashRegisters!=1{t.Fatalf("expected Caja principal after onboarding, got %d",cashRegisters)}
	if err:=pool.QueryRow(context.Background(),`SELECT count(*) FROM zones WHERE organization_id=$1 AND location_id=$2 AND active AND name='Principal'`,created.OrganizationID,created.LocationID).Scan(&zones);err!=nil{t.Fatal(err)}
	if zones!=1{t.Fatalf("expected Principal zone after onboarding, got %d",zones)}
	if err:=pool.QueryRow(context.Background(),`SELECT count(*) FROM organization_operational_setup WHERE organization_id=$1 AND service_mode='dine_in' AND completed_at IS NULL`,created.OrganizationID).Scan(&setupPending);err!=nil{t.Fatal(err)}
	if setupPending!=1{t.Fatalf("expected pending restaurant setup in dine_in after onboarding, got %d",setupPending)}
	assertOnboardingRestaurantDefaults(t,pool,created.OrganizationID,created.LocationID)

	// Retrying the seed step must not duplicate anything nor overwrite a mode the tenant already changed.
	if _,err:=pool.Exec(context.Background(),`UPDATE organization_operational_setup SET service_mode='mixed' WHERE organization_id=$1`,created.OrganizationID);err!=nil{t.Fatal(err)}
	before:=onboardingSeedCounts(t,pool,created.OrganizationID,created.LocationID)
	for i:=0;i<2;i++{
		if err:=platformpg.New(pool).SeedOperationalDefaults(context.Background(),created.OrganizationID,created.LocationID,created.AdministratorID,platformdomain.NewTenantOperationalDefaults());err!=nil{t.Fatalf("retry %d: %v",i+1,err)}
	}
	if after:=onboardingSeedCounts(t,pool,created.OrganizationID,created.LocationID);after!=before{t.Fatalf("seed retry duplicated defaults: before=%v after=%v",before,after)}
	var retriedMode string
	if err:=pool.QueryRow(context.Background(),`SELECT service_mode FROM organization_operational_setup WHERE organization_id=$1`,created.OrganizationID).Scan(&retriedMode);err!=nil{t.Fatal(err)}
	if retriedMode!="mixed"{t.Fatalf("seed retry overwrote the service mode: %s",retriedMode)}
	if _,err:=pool.Exec(context.Background(),`UPDATE organization_operational_setup SET service_mode='dine_in' WHERE organization_id=$1`,created.OrganizationID);err!=nil{t.Fatal(err)}
	// The batched zone insert also creates the zone of the default tables when no template provides it.
	terrace:=platformdomain.NewTenantOperationalDefaults()
	terrace.TableZone="Terraza"
	terrace.Tables=[]platformdomain.DefaultTable{{Name:"Mesa T1",Seats:4,QREnabled:true}}
	if err:=platformpg.New(pool).SeedOperationalDefaults(context.Background(),created.OrganizationID,created.LocationID,created.AdministratorID,terrace);err!=nil{t.Fatal(err)}
	var terraceZones,terraceTables int
	if err:=pool.QueryRow(context.Background(),`SELECT (SELECT count(*) FROM zones WHERE organization_id=$1 AND location_id=$2 AND name='Terraza' AND active),(SELECT count(*) FROM tables WHERE organization_id=$1 AND location_id=$2 AND name='Mesa T1' AND zone='Terraza' AND seats=4)`,created.OrganizationID,created.LocationID).Scan(&terraceZones,&terraceTables);err!=nil{t.Fatal(err)}
	if terraceZones!=1||terraceTables!=1{t.Fatalf("expected non-template zone and its table, got zones=%d tables=%d",terraceZones,terraceTables)}
	for _,cleanup:=range []string{`DELETE FROM tables WHERE organization_id=$1 AND name='Mesa T1'`,`DELETE FROM zones WHERE organization_id=$1 AND name='Terraza'`}{
		if _,err:=pool.Exec(context.Background(),cleanup,created.OrganizationID);err!=nil{t.Fatal(err)}
	}

	var registerID string
	if err:=pool.QueryRow(context.Background(),`SELECT id FROM cash_registers WHERE organization_id=$1 AND location_id=$2 AND name='Caja principal'`,created.OrganizationID,created.LocationID).Scan(&registerID);err!=nil{t.Fatal(err)}
	openReq:=httptest.NewRequest("POST","/v1/admin/cash-shifts",bytes.NewReader([]byte(fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":0}`,registerID))))
	openReq=openReq.WithContext(context.WithValue(openReq.Context(),scopeKey{},scope{UserID:created.AdministratorID,OrganizationID:created.OrganizationID,LocationID:created.LocationID,Name:"Propietario MVP"}))
	openRec:=httptest.NewRecorder();api.openCashShift(openRec,openReq)
	if openRec.Code!=201{t.Fatalf("new tenant must be able to open its default cash register: %d %s",openRec.Code,openRec.Body.String())}

	permissionScope:=scope{UserID:created.AdministratorID,OrganizationID:created.OrganizationID,LocationID:created.LocationID,Name:"Propietario MVP"}
	permissionReq:=httptest.NewRequest("GET","/permission-check",nil)
	permissionReq=permissionReq.WithContext(context.WithValue(permissionReq.Context(),scopeKey{},permissionScope))
	permissionRec:=httptest.NewRecorder()
	api.requirePermission("dashboard.read",http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.WriteHeader(http.StatusNoContent)})).ServeHTTP(permissionRec,permissionReq)
	if permissionRec.Code!=http.StatusNoContent{t.Fatalf("company administrator permission middleware failed: %d %s",permissionRec.Code,permissionRec.Body.String())}
	deniedReq:=httptest.NewRequest("GET","/permission-check",nil)
	deniedReq=deniedReq.WithContext(context.WithValue(deniedReq.Context(),scopeKey{},permissionScope))
	deniedRec:=httptest.NewRecorder()
	api.requirePermission("nonexistent.permission",http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.WriteHeader(http.StatusNoContent)})).ServeHTTP(deniedRec,deniedReq)
	if deniedRec.Code!=http.StatusForbidden||!strings.Contains(deniedRec.Body.String(),"forbidden"){t.Fatalf("missing permission must return forbidden, got %d %s",deniedRec.Code,deniedRec.Body.String())}

	loginReq:=httptest.NewRequest("POST","/v1/auth/login",bytes.NewReader([]byte(fmt.Sprintf(`{"email":%q,"password":%q}`,email,password))))
	loginRec:=httptest.NewRecorder()
	api.login(loginRec,loginReq)
	if loginRec.Code!=200||!strings.Contains(loginRec.Header().Get("Set-Cookie"),"foods_session="){t.Fatalf("new tenant administrator cannot login: %d %s",loginRec.Code,loginRec.Body.String())}
}

// assertOnboardingRestaurantDefaults verifies the seed_operational_defaults preload:
// «General», «Mesa 1» and «Mesa 2» free in «Principal» for 2 people, and no demo data.
func assertOnboardingRestaurantDefaults(t *testing.T,pool *pgxpool.Pool,organizationID,locationID string){
	t.Helper()
	ctx:=context.Background()
	var categories int
	if err:=pool.QueryRow(ctx,`SELECT count(*) FROM menu_categories WHERE organization_id=$1 AND name='General' AND active AND product_scope='both'`,organizationID).Scan(&categories);err!=nil{t.Fatal(err)}
	if categories!=1{t.Fatalf("expected active General category for any product type, got %d",categories)}
	rows,err:=pool.Query(ctx,`SELECT name,seats,zone,active,qr_enabled,qr_token<>'' FROM tables WHERE organization_id=$1 AND location_id=$2 ORDER BY name`,organizationID,locationID)
	if err!=nil{t.Fatal(err)}
	defer rows.Close()
	var tables []string
	for rows.Next(){
		var name,zone string
		var seats int
		var active,qrEnabled,hasToken bool
		if err:=rows.Scan(&name,&seats,&zone,&active,&qrEnabled,&hasToken);err!=nil{t.Fatal(err)}
		if seats!=2||zone!="Principal"||!active||!qrEnabled||!hasToken{t.Fatalf("unexpected default table %s: seats=%d zone=%s active=%v qr=%v token=%v",name,seats,zone,active,qrEnabled,hasToken)}
		tables=append(tables,name)
	}
	if err:=rows.Err();err!=nil{t.Fatal(err)}
	if strings.Join(tables,",")!="Mesa 1,Mesa 2"{t.Fatalf("expected Mesa 1 and Mesa 2, got %v",tables)}
	var occupied,products,orders,shifts int
	if err:=pool.QueryRow(ctx,`
		SELECT
		  (SELECT count(*) FROM orders o JOIN tables t ON t.id=o.table_id WHERE t.organization_id=$1),
		  (SELECT count(*) FROM products WHERE organization_id=$1),
		  (SELECT count(*) FROM orders WHERE organization_id=$1),
		  (SELECT count(*) FROM cash_shifts WHERE organization_id=$1)
	`,organizationID).Scan(&occupied,&products,&orders,&shifts);err!=nil{t.Fatal(err)}
	if occupied!=0||products!=0||orders!=0||shifts!=0{t.Fatalf("onboarding must not create demo data: occupied=%d products=%d orders=%d shifts=%d",occupied,products,orders,shifts)}
}

type onboardingSeedCount struct{ExpenseCategories,Zones,CashRegisters,Categories,Tables,Setups int}

func onboardingSeedCounts(t *testing.T,pool *pgxpool.Pool,organizationID,locationID string)onboardingSeedCount{
	t.Helper()
	var c onboardingSeedCount
	if err:=pool.QueryRow(context.Background(),`
		SELECT
		  (SELECT count(*) FROM expense_categories WHERE organization_id=$1),
		  (SELECT count(*) FROM zones WHERE organization_id=$1 AND location_id=$2),
		  (SELECT count(*) FROM cash_registers WHERE organization_id=$1 AND location_id=$2),
		  (SELECT count(*) FROM menu_categories WHERE organization_id=$1),
		  (SELECT count(*) FROM tables WHERE organization_id=$1 AND location_id=$2),
		  (SELECT count(*) FROM organization_operational_setup WHERE organization_id=$1)
	`,organizationID,locationID).Scan(&c.ExpenseCategories,&c.Zones,&c.CashRegisters,&c.Categories,&c.Tables,&c.Setups);err!=nil{t.Fatal(err)}
	return c
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
	if item.TableID==nil||*item.TableID!=tableID||item.Status!="pending"||item.DurationMinutes!=90{t.Fatalf("unexpected reservation: %#v",item)}
	overlapStart:=time.Now().UTC().Add(24*time.Hour+30*time.Minute).Truncate(time.Minute).Format(time.RFC3339)
	overlapBody:=[]byte(fmt.Sprintf(`{"customerName":"Solapada","startsAt":%q,"guests":2,"durationMinutes":60,"tableId":%q}`,overlapStart,tableID))
	overlapReq:=httptest.NewRequest("POST","/v1/admin/reservations/",bytes.NewReader(overlapBody));overlapReq=overlapReq.WithContext(context.WithValue(overlapReq.Context(),scopeKey{},s))
	overlapRec:=httptest.NewRecorder();api.createReservation(overlapRec,overlapReq)
	if overlapRec.Code!=400||!strings.Contains(overlapRec.Body.String(),"reservation_conflict"){t.Fatalf("overlapping reservation must conflict: %d %s",overlapRec.Code,overlapRec.Body.String())}
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

func TestDefaultRolesOnlyExposeUsableMVPMenus(t *testing.T){
	required:=map[string]string{
		"dashboard":"dashboard.read","pos":"cash.read","pedidos":"orders.read","cocina":"orders.read",
		"mesas":"tables.read","caja":"cash.read","reservas":"reservations.read","productos":"menu.read",
		"disponibilidad":"menu.read","combos":"menu.read","recetas":"menu.read","inventario":"inventory.read",
		"kardex":"inventory.read","compras":"purchases.read","clientes":"customers.read","locales":"organizations.read",
		"fiscal":"organizations.read","usuarios":"users.read","costos":"expenses.read",
	}
	contains:=func(values []string,want string)bool{for _,value:=range values{if value==want||value=="*"{return true}};return false}
	for _,role:=range defaultOrganizationRoles{
		if contains(role.Permissions,"*")||contains(role.MenuAccess,"*"){t.Fatalf("tenant role %s must not contain wildcard access",role.Name)}
		if role.SystemKey=="administrator"{
			if role.Name!="Administrador de empresa"||!contains(role.Permissions,"users.manage")||!contains(role.Permissions,"organizations.manage"){t.Fatalf("invalid company administrator definition: %#v",role)}
			continue
		}
		for _,menu:=range role.MenuAccess{
			permission,ok:=required[menu]
			if !ok{t.Fatalf("default role %s exposes non-MVP or unmapped menu %s",role.Name,menu)}
			if !contains(role.Permissions,permission){t.Fatalf("default role %s exposes %s without %s",role.Name,menu,permission)}
		}
	}
}

func TestDashboardReflectsRealPaymentsAndRefunds(t *testing.T){
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool)
	api:=New(pool)
	ctx:=context.Background()
	grantCashManagementForPaymentTest(t,api,s)
	nonce:=time.Now().UnixNano()
	var registerID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO cash_registers(organization_id,location_id,name,created_by)
		VALUES($1,$2,$3,$4) RETURNING id
	`,s.OrganizationID,s.LocationID,fmt.Sprintf("Caja dashboard %d",nonce),s.UserID).Scan(&registerID);err!=nil{t.Fatal(err)}
	openReq:=httptest.NewRequest("POST","/v1/admin/cash-shifts",bytes.NewReader([]byte(fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":0}`,registerID))))
	openReq=openReq.WithContext(context.WithValue(openReq.Context(),scopeKey{},s))
	openRec:=httptest.NewRecorder();api.openCashShift(openRec,openReq)
	if openRec.Code!=201{t.Fatalf("open dashboard shift: %d %s",openRec.Code,openRec.Body.String())}

	var orderID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO orders(organization_id,location_id,code,channel,status,total,created_by)
		VALUES($1,$2,$3,'mostrador','listo',50,$4) RETURNING id
	`,s.OrganizationID,s.LocationID,fmt.Sprintf("PED-DASH-%d",nonce),s.UserID).Scan(&orderID);err!=nil{t.Fatal(err)}
	payReq:=httptest.NewRequest("POST","/v1/admin/payments",bytes.NewReader([]byte(fmt.Sprintf(`{"orderId":%q,"method":"card","amount":50,"reference":"DASH"}`,orderID))))
	payReq=payReq.WithContext(context.WithValue(payReq.Context(),scopeKey{},s))
	payRec:=httptest.NewRecorder();api.createPayment(payRec,payReq)
	if payRec.Code!=201{t.Fatalf("dashboard payment: %d %s",payRec.Code,payRec.Body.String())}
	var payment paymentView
	if err:=json.Unmarshal(payRec.Body.Bytes(),&payment);err!=nil{t.Fatal(err)}

	readDashboard:=func()(string,int){
		req:=httptest.NewRequest("GET","/v1/admin/dashboard",nil);req=req.WithContext(context.WithValue(req.Context(),scopeKey{},s))
		rec:=httptest.NewRecorder();api.dashboard(rec,req)
		if rec.Code!=200{t.Fatalf("dashboard: %d %s",rec.Code,rec.Body.String())}
		var body struct{SalesNet string `json:"salesNet"`;PaidOrders int `json:"paidOrders"`}
		if err:=json.Unmarshal(rec.Body.Bytes(),&body);err!=nil{t.Fatal(err)}
		return body.SalesNet,body.PaidOrders
	}
	sales,tickets:=readDashboard()
	if sales!="50.00"||tickets!=1{t.Fatalf("dashboard should reflect payment, sales=%s tickets=%d",sales,tickets)}

	refundReq:=httptest.NewRequest("POST","/v1/admin/payments/"+payment.ID+"/refund",bytes.NewReader([]byte(`{"amount":10,"reason":"Ajuste dashboard"}`)))
	refundReq.SetPathValue("id",payment.ID);refundReq=refundReq.WithContext(context.WithValue(refundReq.Context(),scopeKey{},s))
	refundRec:=httptest.NewRecorder();api.refundPayment(refundRec,refundReq)
	if refundRec.Code!=204{t.Fatalf("dashboard refund: %d %s",refundRec.Code,refundRec.Body.String())}
	sales,tickets=readDashboard()
	if sales!="40.00"||tickets!=0{t.Fatalf("dashboard should net refunds and exclude partially paid orders, sales=%s orders=%d",sales,tickets)}
}

func TestPlatformAdministratorIsUnique(t *testing.T){
	pool:=integrationPool(t)
	first:=seedInventoryScope(t,pool)
	second:=seedInventoryScope(t,pool)
	if _,err:=pool.Exec(context.Background(),`UPDATE users SET platform_admin=true WHERE id=$1`,first.UserID);err!=nil{
		t.Fatalf("enable first platform administrator: %v",err)
	}
	if _,err:=pool.Exec(context.Background(),`UPDATE users SET platform_admin=true WHERE id=$1`,second.UserID);err==nil{
		t.Fatal("expected database to reject a second platform administrator")
	}
}

func TestModuleAvailabilityProtectsTenantActivation(t *testing.T){
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool)
	api:=New(pool)

	if got:=moduleAvailability("reportes");got!=moduleAvailabilityReady{
		t.Fatalf("reportes must be ready, got %s",got)
	}
	if got:=moduleAvailability("carta_qr");got!=moduleAvailabilityDevelopment{
		t.Fatalf("carta_qr must be development, got %s",got)
	}
	if got:=moduleAvailability("whatsapp_bot");got!=moduleAvailabilityReady{
		t.Fatalf("whatsapp_bot must be ready, got %s",got)
	}
	if got:=moduleAvailability("crm");got!=moduleAvailabilityPlanned{
		t.Fatalf("crm must be planned, got %s",got)
	}

	req:=httptest.NewRequest("PATCH","/v1/admin/modules",bytes.NewReader([]byte(`{"key":"carta_qr","active":true}`)))
	req=req.WithContext(context.WithValue(req.Context(),scopeKey{},s))
	rec:=httptest.NewRecorder()
	api.toggleModule(rec,req)
	if rec.Code!=409||!strings.Contains(rec.Body.String(),"module_not_ready"){
		t.Fatalf("development module activation must be rejected: %d %s",rec.Code,rec.Body.String())
	}

	_,err:=pool.Exec(context.Background(),`
		INSERT INTO organization_modules(organization_id,module_key,active)
		VALUES($1,'crm',true)
		ON CONFLICT(organization_id,module_key) DO UPDATE SET active=true,updated_at=now()
	`,s.OrganizationID)
	if err!=nil{t.Fatal(err)}
	active:=api.activeModules(s.OrganizationID)
	if active["crm"]{
		t.Fatal("planned module must remain inactive even with stale active=true data")
	}

	conciergeReq:=httptest.NewRequest("PATCH","/v1/admin/modules",bytes.NewReader([]byte(`{"key":"whatsapp_bot","active":true}`)))
	conciergeReq=conciergeReq.WithContext(context.WithValue(conciergeReq.Context(),scopeKey{},s))
	conciergeRec:=httptest.NewRecorder()
	api.toggleModule(conciergeRec,conciergeReq)
	if conciergeRec.Code!=204{
		t.Fatalf("Fudia Concierge should be activable: %d %s",conciergeRec.Code,conciergeRec.Body.String())
	}

	readyReq:=httptest.NewRequest("PATCH","/v1/admin/modules",bytes.NewReader([]byte(`{"key":"reportes","active":true}`)))
	readyReq=readyReq.WithContext(context.WithValue(readyReq.Context(),scopeKey{},s))
	readyRec:=httptest.NewRecorder()
	api.toggleModule(readyRec,readyReq)
	if readyRec.Code!=204{
		t.Fatalf("ready module should be activable: %d %s",readyRec.Code,readyRec.Body.String())
	}
}
