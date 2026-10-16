package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestTableWaiterOwnershipAndIndependentPaymentAttribution(t *testing.T) {
	pool := integrationPool(t)
	owner := seedInventoryScope(t, pool)
	otherTenant := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	seedID := func(query string, args ...any) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, query, args...).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	_, err := pool.Exec(ctx, `UPDATE users SET full_name='Ana Mozo' WHERE id=$1`, owner.UserID)
	if err != nil {
		t.Fatal(err)
	}
	owner.Name = "Ana Mozo"
	other := owner
	other.UserID = seedID(`INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,'luis@test.local','Luis Mozo','test') RETURNING id`, owner.OrganizationID)
	other.Name = "Luis Mozo"
	cashier := owner
	cashier.UserID = seedID(`INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,'caja@test.local','Eva Cajera','test') RETURNING id`, owner.OrganizationID)
	cashier.Name = "Eva Cajera"
	otherLocation := owner
	otherLocation.LocationID = seedID(`INSERT INTO locations(organization_id,name,code) VALUES($1,'Otro local','OTHER') RETURNING id`, owner.OrganizationID)
	tableID := seedID(`INSERT INTO tables(organization_id,location_id,name,seats,zone,qr_token) VALUES($1,$2,'Mesa 04',4,'Principal',encode(gen_random_bytes(16),'hex')) RETURNING id`, owner.OrganizationID, owner.LocationID)
	productID := seedID(`INSERT INTO products(organization_id,sku,name,price,product_type,quantity_control) VALUES($1,'WAITER-TEST','Plato',25,'prepared','none') RETURNING id`, owner.OrganizationID)
	// Tables are not part of the shared inventory fixture cleanup.
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM payments WHERE organization_id=$1`, owner.OrganizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM orders WHERE organization_id=$1`, owner.OrganizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM tables WHERE id=$1`, tableID)
	})
	call := func(s scope, method, id, body string, handler http.HandlerFunc, expected int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "/", strings.NewReader(body))
		r.SetPathValue("id", id)
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
		w := httptest.NewRecorder()
		handler(w, r)
		if w.Code != expected {
			t.Fatalf("%s %s: got %d want %d: %s", method, id, w.Code, expected, w.Body.String())
		}
		return w
	}
	createBody := fmt.Sprintf(`{"channel":"salon","tableId":%q,"waiterId":%q,"items":[{"productId":%q,"qty":1}]}`, tableID, other.UserID, productID)
	w := call(owner, "POST", "", createBody, api.createOrder, 201)
	var created order
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.WaiterID != owner.UserID || created.WaiterName != owner.Name || len(created.CollectedByNames) != 0 {
		t.Fatalf("assignment must derive from authenticated user, not body: %#v", created)
	}
	updateBody := fmt.Sprintf(`{"customerName":"Familia","waiterId":%q,"items":[{"productId":%q,"qty":1}]}`, other.UserID, productID)
	for _, mutation := range []struct {
		body    string
		handler http.HandlerFunc
	}{
		{updateBody, api.updateOrder}, {`{"status":"confirmado"}`, api.updateOrderStatus}, {`{"status":"cancelado"}`, api.updateOrderStatus},
	} {
		w = call(other, "PATCH", created.ID, mutation.body, mutation.handler, 403)
		if !strings.Contains(w.Body.String(), "order_assigned_to_another_waiter") {
			t.Fatal(w.Body.String())
		}
	}
	for _, s := range []scope{otherTenant, otherLocation} {
		call(s, "GET", created.ID, "", api.getOrder, 404)
		call(s, "PATCH", created.ID, updateBody, api.updateOrder, 404)
	}
	w = call(other, "GET", created.ID, "", api.getOrder, 200)
	if !strings.Contains(w.Body.String(), owner.Name) {
		t.Fatal("another waiter must retain readable detail")
	}
	w = call(owner, "PATCH", created.ID, updateBody, api.updateOrder, 200)
	var edited order
	if err := json.Unmarshal(w.Body.Bytes(), &edited); err != nil {
		t.Fatal(err)
	}
	if edited.WaiterID != owner.UserID {
		t.Fatal("editing must not transfer ownership")
	}
	call(owner, "PATCH", created.ID, `{"status":"confirmado"}`, api.updateOrderStatus, 200)
	// A different kitchen operator keeps their own preparation workflow.
	call(other, "PATCH", created.ID, `{"status":"preparando"}`, api.updateKitchenTicketStatus, 204)
	call(other, "PATCH", created.ID, `{"status":"listo"}`, api.updateKitchenTicketStatus, 204)
	call(other, "PATCH", created.ID, `{"status":"entregado"}`, api.updateOrderStatus, 403)
	registerID := seedID(`INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,'Caja',$3) RETURNING id`, owner.OrganizationID, owner.LocationID, cashier.UserID)
	call(cashier, "POST", "", fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":0}`, registerID), api.openCashShift, 201)
	call(owner, "POST", created.ID, "", api.closeOrderBill, 409)
	call(owner, "PATCH", created.ID, `{"status":"entregado"}`, api.updateOrderStatus, 200)
	call(owner, "POST", created.ID, "", api.closeOrderBill, 200)
	for _, amount := range []int{10, 15} {
		w = call(cashier, "POST", "", fmt.Sprintf(`{"orderId":%q,"method":"card","amount":%d,"createdByName":"Luis Mozo"}`, created.ID, amount), api.createPayment, 201)
		var payment paymentView
		if err := json.Unmarshal(w.Body.Bytes(), &payment); err != nil {
			t.Fatal(err)
		}
		if payment.CreatedByName != cashier.Name {
			t.Fatalf("payment operator must derive from session: %#v", payment)
		}
		if amount == 10 {
			floor := call(other, "GET", created.ID, "", api.getOrdersFloor, 200)
			if !strings.Contains(floor.Body.String(), `"waiterName":"Ana Mozo"`) || !strings.Contains(floor.Body.String(), `"collectedByNames":["Eva Cajera"]`) {
				t.Fatalf("occupied table must identify waiter and collector: %s", floor.Body.String())
			}
		}
	}
	for _, handler := range []http.HandlerFunc{api.getOrder, api.getPOSOrder, api.listOrders} {
		w = call(other, "GET", created.ID, "", handler, 200)
		if !strings.Contains(w.Body.String(), `"waiterName":"Ana Mozo"`) || !strings.Contains(w.Body.String(), `"collectedByNames":["Eva Cajera"]`) {
			t.Fatalf("list/detail must distinguish waiter and collector: %s", w.Body.String())
		}
	}
	floor := call(other, "GET", created.ID, "", api.getOrdersFloor, 200)
	if !strings.Contains(floor.Body.String(), `"order":null`) {
		t.Fatalf("paid and delivered table must be free: %s", floor.Body.String())
	}
	// A new occupation assigns its own waiter rather than retaining the previous one.
	w = call(other, "POST", "", createBody, api.createOrder, 201)
	var next order
	if err := json.Unmarshal(w.Body.Bytes(), &next); err != nil {
		t.Fatal(err)
	}
	if next.ID == created.ID || next.WaiterID != other.UserID {
		t.Fatalf("new service must acquire its own waiter: %#v", next)
	}
	// The old paid order retains its original attribution after table reuse.
	w = call(owner, "GET", created.ID, "", api.getOrder, 200)
	var historical order
	if err := json.Unmarshal(w.Body.Bytes(), &historical); err != nil {
		t.Fatal(err)
	}
	if historical.WaiterID != owner.UserID || !reflect.DeepEqual(historical.CollectedByNames, []string{cashier.Name}) {
		t.Fatalf("lost historical attribution: %#v", historical)
	}
	_, err = pool.Exec(ctx, `UPDATE orders SET waiter_id=$1 WHERE id=$2`, otherTenant.UserID, next.ID)
	if err == nil {
		t.Fatal("database must reject a waiter from another company")
	}
}
