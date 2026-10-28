package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Requests go through the real cookie, authorization middleware and transactions.
func TestCashCollaboratorsCollectWithTemporaryAuthority(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO organization_modules(organization_id,module_key,active) VALUES($1,'pos',true) ON CONFLICT(organization_id,module_key) DO UPDATE SET active=true`, s.OrganizationID)
	role := func(name, key string, permissions []string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO roles(organization_id,name,system_key,permissions,menu_access) VALUES($1,$2,NULLIF($3,''),$4,ARRAY['pedidos']) RETURNING id`, s.OrganizationID, name, key, permissions).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	cashierRole := role("Cajero de prueba", "cashier", []string{"cash.manage", "cash.read", "orders.read"})
	waiterRole := role("Mozo de prueba", "waiter", []string{"orders.read", "orders.manage", "menu.read"})
	administratorRole := role("Administrador de empresa", "administrator", []string{"cash.manage", "cash.read", "orders.read"})
	exec(`INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, s.UserID, cashierRole, s.LocationID)
	user := func(name, roleID, location string, active bool) scope {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO users(organization_id,email,full_name,password_hash,active) VALUES($1,$2,$3,'test',$4) RETURNING id`, s.OrganizationID, fmt.Sprintf("team-%d@example.test", time.Now().UnixNano()), name, active).Scan(&id); err != nil {
			t.Fatal(err)
		}
		exec(`INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, id, roleID, location)
		return scope{OrganizationID: s.OrganizationID, LocationID: location, UserID: id, Name: name}
	}
	waiter := user("Piter", waiterRole, s.LocationID, true)
	otherCashier := user("Segundo cajero", cashierRole, s.LocationID, true)
	administrator := user("Administradora", administratorRole, s.LocationID, true)
	// Having another eligible role must not make an administrator assignable.
	exec(`INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, administrator.UserID, waiterRole, s.LocationID)
	inactive := user("Inactivo", waiterRole, s.LocationID, false)
	var otherLocation string
	if err := pool.QueryRow(ctx, `INSERT INTO locations(organization_id,name,code) VALUES($1,'Otro local',$2) RETURNING id`, s.OrganizationID, fmt.Sprintf("team-%d", time.Now().UnixNano())).Scan(&otherLocation); err != nil {
		t.Fatal(err)
	}
	foreign := user("Mozo de otro local", waiterRole, otherLocation, true)
	cashierCall := companySession(t, api, s)
	waiterCall := companySession(t, api, waiter)
	otherCall := companySession(t, api, otherCashier)
	expect := func(rec *httptest.ResponseRecorder, status int) {
		t.Helper()
		if rec.Code != status {
			t.Fatalf("want %d, got %d: %s", status, rec.Code, rec.Body.String())
		}
	}
	open := func(call func(string, string, any) *httptest.ResponseRecorder, name string) string {
		t.Helper()
		var register string
		if err := pool.QueryRow(ctx, `INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,$3,$4) RETURNING id`, s.OrganizationID, s.LocationID, name, s.UserID).Scan(&register); err != nil {
			t.Fatal(err)
		}
		rec := call("POST", "/v1/admin/cash-shifts", map[string]any{"cashRegisterId": register, "openingAmount": 0})
		expect(rec, 201)
		var result struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result.ID
	}
	shift := open(cashierCall, "Caja principal")
	secondShift := open(otherCall, "Caja secundaria")
	assign := func(id string) *httptest.ResponseRecorder {
		return cashierCall("POST", "/v1/admin/cash-shifts/"+shift+"/users", map[string]string{"userId": id})
	}

	t.Run("responsable interno no aparece ni puede retirarse como colaborador", func(t *testing.T) {
		for _, call := range []func(string, string, any) *httptest.ResponseRecorder{cashierCall, companySession(t, api, administrator)} {
			listed := call("GET", "/v1/admin/cash-shifts/"+shift+"/users", nil)
			expect(listed, 200)
			var team struct {
				Items []struct {
					UserID string `json:"userId"`
				} `json:"items"`
			}
			if err := json.Unmarshal(listed.Body.Bytes(), &team); err != nil {
				t.Fatal(err)
			}
			for _, person := range team.Items {
				if person.UserID == s.UserID {
					t.Fatal("responsible cashier exposed as collaborator")
				}
			}
			denied := call("DELETE", "/v1/admin/cash-shifts/"+shift+"/users/"+s.UserID, nil)
			expect(denied, 409)
			if !strings.Contains(denied.Body.String(), "cash_shift_owner_not_removable") {
				t.Fatal(denied.Body.String())
			}
		}
		var active bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cash_shift_users WHERE shift_id=$1 AND user_id=$2 AND unassigned_at IS NULL)`, shift, s.UserID).Scan(&active); err != nil || !active {
			t.Fatalf("responsible association lost: %v %v", active, err)
		}
		expect(cashierCall("GET", "/v1/admin/pos/access", nil), 200)
	})
	t.Run("selector y restricciones de asignación", func(t *testing.T) {
		rec := cashierCall("GET", "/v1/admin/cash-users/options", nil)
		expect(rec, 200)
		for _, excluded := range []string{s.UserID, administrator.UserID, inactive.UserID, foreign.UserID} {
			if strings.Contains(rec.Body.String(), excluded) {
				t.Fatalf("ineligible user in options: %s", rec.Body.String())
			}
		}
		expect(assign(s.UserID), 400)
		for _, id := range []string{administrator.UserID, inactive.UserID, foreign.UserID} {
			expect(assign(id), 404)
		}
		expect(assign(otherCashier.UserID), 409)
		expect(assign(waiter.UserID), 201)
		expect(assign(waiter.UserID), 409)
		expect(otherCall("POST", "/v1/admin/cash-shifts/"+secondShift+"/users", map[string]string{"userId": waiter.UserID}), 409)
	})
	order := func(total int) string {
		t.Helper()
		var id, tableID string
		if err := pool.QueryRow(ctx, `INSERT INTO tables(organization_id,location_id,name,seats,zone,qr_token) VALUES($1,$2,$3,4,'Principal',encode(gen_random_bytes(16),'hex')) RETURNING id`, s.OrganizationID, s.LocationID, fmt.Sprintf("Mesa equipo %d", time.Now().UnixNano())).Scan(&tableID); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `INSERT INTO orders(organization_id,location_id,code,channel,status,total,created_by,table_id,waiter_id,bill_closed_at) VALUES($1,$2,$3,'salon','entregado',$4,$5,$6,$7,now()) RETURNING id`, s.OrganizationID, s.LocationID, fmt.Sprintf("PED-TEAM-%d", time.Now().UnixNano()), total, s.UserID, tableID, waiter.UserID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	pay := func(call func(string, string, any) *httptest.ResponseRecorder, id string, amount int) *httptest.ResponseRecorder {
		return call("POST", "/v1/admin/payments", map[string]any{"orderId": id, "method": "cash", "amount": amount})
	}
	var paymentID string
	t.Run("mozo cobra sin adquirir cash.manage ni administrar caja", func(t *testing.T) {
		workContext := waiterCall("GET", "/v1/admin/context", nil)
		expect(workContext, 200)
		var accessContext struct {
			CashCollection bool     `json:"cashCollection"`
			Permissions    []string `json:"permissions"`
		}
		if err := json.Unmarshal(workContext.Body.Bytes(), &accessContext); err != nil || !accessContext.CashCollection || containsString(accessContext.Permissions, "cash.manage") {
			t.Fatalf("wrong session capability: %s %v", workContext.Body.String(), err)
		}
		access := waiterCall("GET", "/v1/admin/pos/access", nil)
		expect(access, 200)
		if !strings.Contains(access.Body.String(), shift) || strings.Contains(access.Body.String(), "openingAmount") || strings.Contains(access.Body.String(), "expectedAmount") {
			t.Fatalf("unsafe access projection: %s", access.Body.String())
		}
		rec := pay(waiterCall, order(30), 10)
		expect(rec, 201)
		var payment struct{ ID, ShiftID, CreatedByName string }
		if err := json.Unmarshal(rec.Body.Bytes(), &payment); err != nil {
			t.Fatal(err)
		}
		paymentID = payment.ID
		if payment.ShiftID != shift || payment.CreatedByName != waiter.Name {
			t.Fatalf("wrong payment attribution: %s", rec.Body.String())
		}
		var granted bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE ur.user_id=$1 AND 'cash.manage'=ANY(r.permissions))`, waiter.UserID).Scan(&granted); err != nil || granted {
			t.Fatalf("cash.manage granted: %v %v", granted, err)
		}
		denied := []struct {
			method, path string
			body         any
		}{
			{"POST", "/v1/admin/cash-registers", map[string]any{"name": "No autorizada"}},
			{"PATCH", "/v1/admin/cash-registers/" + shift, map[string]any{"name": "No autorizada"}},
			{"POST", "/v1/admin/cash-shifts", map[string]any{}},
			{"POST", "/v1/admin/cash-shifts/" + shift + "/close", map[string]any{"countedAmount": 0}},
			{"POST", "/v1/admin/cash-shifts/" + shift + "/movements", map[string]any{"movementType": "income", "amount": 5, "reason": "No autorizado"}},
			{"POST", "/v1/admin/cash-shifts/" + shift + "/users", map[string]any{"userId": otherCashier.UserID}},
			{"DELETE", "/v1/admin/cash-shifts/" + shift + "/users/" + s.UserID, nil},
			{"POST", "/v1/admin/cash-shifts/" + shift + "/operations", map[string]any{"operationType": "transfer", "targetShiftId": secondShift, "amount": 1, "reason": "No autorizado"}},
			{"POST", "/v1/admin/payments/" + paymentID + "/refund", map[string]any{"amount": 1, "reason": "No autorizado"}},
			{"GET", "/v1/admin/cash-shifts/" + shift + "/report", nil},
			{"GET", "/v1/admin/cash-shifts/current", nil},
		}
		for _, request := range denied {
			expect(waiterCall(request.method, request.path, request.body), 403)
		}
		expect(waiterCall("GET", "/v1/admin/payment-methods/operational?usage=sales", nil), 200)
	})
	t.Run("solo cobra la atencion vinculada automaticamente al enviar comanda", func(t *testing.T) {
		otherWaiter := user("Otra moza", waiterRole, s.LocationID, true)
		expect(assign(otherWaiter.UserID), 201)
		otherWaiterCall := companySession(t, api, otherWaiter)
		var tableID, productID string
		if err := pool.QueryRow(ctx, `INSERT INTO tables(organization_id,location_id,name,seats,zone,qr_token) VALUES($1,$2,'Mesa de comanda',4,'Principal',encode(gen_random_bytes(16),'hex')) RETURNING id`, s.OrganizationID, s.LocationID).Scan(&tableID); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price,product_type,quantity_control,service_destination) VALUES($1,$2,'Plato directo',20,'prepared','none','direct') RETURNING id`, s.OrganizationID, fmt.Sprintf("team-plate-%d", time.Now().UnixNano())).Scan(&productID); err != nil {
			t.Fatal(err)
		}
		created := waiterCall("POST", "/v1/admin/orders", map[string]any{"channel": "salon", "tableId": tableID, "sendToKitchen": true, "items": []map[string]any{{"productId": productID, "qty": 1}}})
		expect(created, 201)
		var linked struct {
			ID       string `json:"id"`
			WaiterID string `json:"waiterId"`
		}
		if err := json.Unmarshal(created.Body.Bytes(), &linked); err != nil || linked.WaiterID != waiter.UserID {
			t.Fatalf("command did not bind its waiter: %s %v", created.Body.String(), err)
		}
		var serviceID string
		if err := pool.QueryRow(ctx, `SELECT id FROM order_service_items WHERE order_id=$1`, linked.ID).Scan(&serviceID); err != nil {
			t.Fatal(err)
		}
		expect(waiterCall("PATCH", "/v1/admin/orders/"+linked.ID+"/service-items/"+serviceID+"/deliver", nil), 200)
		expect(waiterCall("POST", "/v1/admin/orders/"+linked.ID+"/bill/close", nil), 200)
		for _, request := range []struct {
			path string
			body any
		}{
			{"/v1/admin/payments", map[string]any{"orderId": linked.ID, "method": "cash", "amount": 10}},
			{"/v1/operations/payments/batch", map[string]any{"orderId": linked.ID, "payments": []map[string]any{{"method": "cash", "amount": 5}, {"method": "card", "amount": 5}}}},
		} {
			denied := otherWaiterCall("POST", request.path, request.body)
			expect(denied, 403)
			if !strings.Contains(denied.Body.String(), "cash_collection_order_forbidden") {
				t.Fatal(denied.Body.String())
			}
		}
		var count int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM payments WHERE order_id=$1`, linked.ID).Scan(&count); err != nil || count != 0 {
			t.Fatalf("denied collection wrote payments: %d %v", count, err)
		}
		expect(otherWaiterCall("GET", "/v1/admin/pos/orders/"+linked.ID, nil), 404)
		listed := otherWaiterCall("GET", "/v1/admin/pos/orders?paymentStatus=all", nil)
		expect(listed, 200)
		if strings.Contains(listed.Body.String(), linked.ID) {
			t.Fatal("another waiter's table appeared in POS")
		}
		expect(pay(waiterCall, linked.ID, 5), 201)
		expect(waiterCall("POST", "/v1/operations/payments/batch", map[string]any{"orderId": linked.ID, "payments": []map[string]any{{"method": "cash", "amount": 5}, {"method": "card", "amount": 10}}}), 201)
		for _, change := range []string{"waiter_id=NULL", "channel='recojo',table_id=NULL,status='listo',bill_closed_at=NULL"} {
			id := order(12)
			exec("UPDATE orders SET "+change+" WHERE id=$1", id)
			expect(pay(waiterCall, id, 12), 403)
			expect(pay(cashierCall, id, 12), 201)
		}
	})
	t.Run("cobro dividido y lectura limitada al turno", func(t *testing.T) {
		id := order(25)
		expect(waiterCall("POST", "/v1/operations/payments/batch", map[string]any{"orderId": id, "payments": []map[string]any{{"method": "cash", "amount": 10}, {"method": "card", "amount": 15}}}), 201)
		var count int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM payments WHERE order_id=$1 AND shift_id=$2 AND created_by=$3`, id, shift, waiter.UserID).Scan(&count); err != nil || count != 2 {
			t.Fatalf("batch attribution: %d %v", count, err)
		}
		oldOrder := order(20)
		expect(pay(otherCall, oldOrder, 20), 201)
		expect(waiterCall("GET", "/v1/admin/pos/orders/"+oldOrder, nil), 404)
		expect(cashierCall("GET", "/v1/admin/pos/orders/"+oldOrder, nil), 200)
		listed := waiterCall("GET", "/v1/admin/pos/orders?paymentStatus=all", nil)
		expect(listed, 200)
		if strings.Contains(listed.Body.String(), oldOrder) {
			t.Fatal("temporary operator can see another shift's sales history")
		}
	})
	t.Run("cobros simultáneos consistentes y atribuidos a cada usuario", func(t *testing.T) {
		id := order(100)
		var wg sync.WaitGroup
		results := make(chan *httptest.ResponseRecorder, 2)
		start := make(chan struct{})
		for _, call := range []func(string, string, any) *httptest.ResponseRecorder{cashierCall, waiterCall} {
			wg.Add(1)
			go func(call func(string, string, any) *httptest.ResponseRecorder) {
				defer wg.Done()
				<-start
				results <- pay(call, id, 50)
			}(call)
		}
		close(start)
		wg.Wait()
		close(results)
		for rec := range results {
			expect(rec, 201)
		}
		var count, authors int
		var total float64
		if err := pool.QueryRow(ctx, `SELECT count(*),count(DISTINCT created_by),sum(amount)::float8 FROM payments WHERE order_id=$1 AND shift_id=$2`, id, shift).Scan(&count, &authors, &total); err != nil || count != 2 || authors != 2 || total != 100 {
			t.Fatalf("inconsistent payment: count=%d authors=%d total=%v err=%v", count, authors, total, err)
		}
		id = order(50)
		results = make(chan *httptest.ResponseRecorder, 2)
		start = make(chan struct{})
		for _, call := range []func(string, string, any) *httptest.ResponseRecorder{cashierCall, waiterCall} {
			wg.Add(1)
			go func(call func(string, string, any) *httptest.ResponseRecorder) {
				defer wg.Done()
				<-start
				results <- pay(call, id, 50)
			}(call)
		}
		close(start)
		wg.Wait()
		close(results)
		success, conflict := 0, 0
		for rec := range results {
			if rec.Code == 201 {
				success++
			} else if rec.Code == 409 {
				conflict++
			} else {
				t.Fatalf("unexpected result: %d %s", rec.Code, rec.Body.String())
			}
		}
		if success != 1 || conflict != 1 {
			t.Fatalf("double payment: success=%d conflict=%d", success, conflict)
		}
	})
	t.Run("retiro revoca de inmediato y conserva historial", func(t *testing.T) {
		expect(cashierCall("DELETE", "/v1/admin/cash-shifts/"+shift+"/users/"+waiter.UserID, nil), 204)
		expect(pay(waiterCall, order(10), 10), 403)
		workContext := waiterCall("GET", "/v1/admin/context", nil)
		expect(workContext, 200)
		if !strings.Contains(workContext.Body.String(), `"cashCollection":false`) {
			t.Fatal(workContext.Body.String())
		}
		rec := waiterCall("GET", "/v1/admin/pos/access", nil)
		expect(rec, 200)
		if !strings.Contains(rec.Body.String(), `"shift":null`) {
			t.Fatal(rec.Body.String())
		}
		var assignments, payments int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM cash_shift_users WHERE shift_id=$1 AND user_id=$2 AND unassigned_at IS NOT NULL AND assigned_by=$3`, shift, waiter.UserID, s.UserID).Scan(&assignments); err != nil || assignments != 1 {
			t.Fatalf("assignment audit lost: %d %v", assignments, err)
		}
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM payments WHERE id=$1 AND created_by=$2 AND shift_id=$3`, paymentID, waiter.UserID, shift).Scan(&payments); err != nil || payments != 1 {
			t.Fatalf("payment audit lost: %d %v", payments, err)
		}
	})
	t.Run("solo el abridor puede cerrar incluso ante otro cajero o administrador", func(t *testing.T) {
		for _, call := range []func(string, string, any) *httptest.ResponseRecorder{otherCall, companySession(t, api, administrator)} {
			for _, prefix := range []string{"/v1/admin", "/v1/operations"} {
				rec := call("POST", prefix+"/cash-shifts/"+shift+"/close", map[string]any{"counts": []map[string]any{{"denomination": 10, "quantity": 1}}})
				expect(rec, 403)
				if !strings.Contains(rec.Body.String(), "cash_shift_owner_required") {
					t.Fatal(rec.Body.String())
				}
			}
		}
		var status string
		var counts, reports int
		if err := pool.QueryRow(ctx, `SELECT status,(SELECT count(*) FROM cash_count_lines WHERE shift_id=cs.id),(SELECT count(*) FROM cash_shift_reports WHERE shift_id=cs.id) FROM cash_shifts cs WHERE id=$1`, shift).Scan(&status, &counts, &reports); err != nil || status != "open" || counts != 0 || reports != 0 {
			t.Fatalf("unauthorized closure wrote data: %s %d %d %v", status, counts, reports, err)
		}
	})
	t.Run("cierre revoca todos los colaboradores sin cambiar responsable", func(t *testing.T) {
		expect(assign(waiter.UserID), 201)
		var expected float64
		if err := pool.QueryRow(ctx, `SELECT COALESCE(sum(amount),0)::float8 FROM cash_movements WHERE shift_id=$1 AND movement_type='income'`, shift).Scan(&expected); err != nil {
			t.Fatal(err)
		}
		// Queue closure before an already-authorized payment. The shared shift
		// lock must wait, recheck closure and reject without a deadlock or write.
		gate, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer gate.Rollback(ctx)
		if _, err := gate.Exec(ctx, `SELECT id FROM cash_shifts WHERE id=$1 FOR UPDATE`, shift); err != nil {
			t.Fatal(err)
		}
		waitForLock := func(pattern string) {
			t.Helper()
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				var waiting bool
				if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE wait_event_type='Lock' AND query LIKE $1)`, pattern).Scan(&waiting); err != nil {
					t.Fatal(err)
				}
				if waiting {
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
			t.Fatal("request did not reach expected row lock")
		}
		closure := make(chan *httptest.ResponseRecorder, 1)
		go func() {
			closure <- cashierCall("POST", "/v1/admin/cash-shifts/"+shift+"/close", map[string]any{"countedAmount": expected})
		}()
		waitForLock("%cs.opening_amount::float8%FOR UPDATE%")
		lateOrder := order(10)
		latePayment := make(chan *httptest.ResponseRecorder, 1)
		go func() { latePayment <- pay(waiterCall, lateOrder, 10) }()
		waitForLock("%WITH active_collection_shift%")
		if err := gate.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		expect(<-closure, 200)
		expect(<-latePayment, 409)
		expect(pay(waiterCall, order(10), 10), 403)
		expect(assign(waiter.UserID), 409)
		var opener, closer string
		var active int
		if err := pool.QueryRow(ctx, `SELECT opened_by::text,closed_by::text,(SELECT count(*) FROM cash_shift_users WHERE shift_id=cs.id AND unassigned_at IS NULL) FROM cash_shifts cs WHERE id=$1`, shift).Scan(&opener, &closer, &active); err != nil || opener != s.UserID || closer != opener || active != 0 {
			t.Fatalf("closure changed ownership or kept authority: %s %d %v", opener, active, err)
		}
	})
}
