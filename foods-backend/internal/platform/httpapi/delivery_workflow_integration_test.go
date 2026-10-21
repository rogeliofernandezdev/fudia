package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestManualDeliveryKitchenPaymentAndDispatch(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	var productID string
	if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control) VALUES($1,'DELIVERY-TEST','Plato delivery',25,true,'prepared','none') RETURNING id`, s.OrganizationID).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	request := func(method, path string, body []byte, id string, subject scope) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		if id != "" {
			req.SetPathValue("id", id)
		}
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, subject))
		rec := httptest.NewRecorder()
		switch {
		case path == "/v1/admin/orders" && method == "POST":
			api.createOrder(rec, req)
		case strings.Contains(path, "kitchen/tickets") && method == "PATCH":
			api.updateKitchenTicketStatus(rec, req)
		case strings.Contains(path, "kitchen/tickets"):
			api.listKitchenTickets(rec, req)
		case strings.HasSuffix(path, "/status"):
			api.updateOrderStatus(rec, req)
		case path == "/v1/admin/payments":
			api.createPayment(rec, req)
		case path == "/v1/admin/cash-shifts":
			api.openCashShift(rec, req)
		case strings.HasPrefix(path, "/v1/admin/pos/orders"):
			api.getPOSOrder(rec, req)
		case strings.Contains(path, "?"):
			api.listOrders(rec, req)
		default:
			api.getOrder(rec, req)
		}
		return rec
	}
	body := []byte(fmt.Sprintf(`{"channel":"delivery","customerName":"Ana","customerPhone":"+51 999123456","address":"Av. Perú 100","reference":"Puerta azul","deliveryFee":5,"notes":"Tocar el timbre","sendToKitchen":true,"items":[{"productId":%q,"qty":2,"unitPrice":1,"note":"Sin cebolla"}]}`, productID))
	createdRec := request("POST", "/v1/admin/orders", body, "", s)
	if createdRec.Code != 201 {
		t.Fatalf("create delivery: %d %s", createdRec.Code, createdRec.Body.String())
	}
	var created order
	if err := json.Unmarshal(createdRec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Channel != "delivery" || created.Status != "confirmado" || created.TableID != "" || created.Total != "55.00" || created.Subtotal != "50.00" || created.Address != "Av. Perú 100" {
		t.Fatalf("unexpected delivery: %#v", created)
	}
	list := request("GET", "/v1/admin/orders?channel=delivery&status=abiertos", nil, "", s)
	if list.Code != 200 || !strings.Contains(list.Body.String(), created.ID) {
		t.Fatalf("delivery list: %d %s", list.Code, list.Body.String())
	}
	kitchen := request("GET", "/v1/admin/kitchen/tickets?channel=delivery", nil, "", s)
	if kitchen.Code != 200 || !strings.Contains(kitchen.Body.String(), created.ID) {
		t.Fatalf("delivery KDS: %d %s", kitchen.Code, kitchen.Body.String())
	}
	foreign := seedInventoryScope(t, pool)
	hidden := request("GET", "/v1/admin/orders/"+created.ID, nil, created.ID, foreign)
	if hidden.Code != 404 {
		t.Fatalf("foreign delivery visible: %d %s", hidden.Code, hidden.Body.String())
	}
	var secondLocation string
	if err := pool.QueryRow(ctx, `INSERT INTO locations(organization_id,name,code,address) VALUES($1,'Otro local','DELIVERY-OTHER','') RETURNING id`, s.OrganizationID).Scan(&secondLocation); err != nil {
		t.Fatal(err)
	}
	otherLocal := s
	otherLocal.LocationID = secondLocation
	hidden = request("GET", "/v1/admin/orders/"+created.ID, nil, created.ID, otherLocal)
	if hidden.Code != 404 {
		t.Fatalf("foreign local delivery visible: %d %s", hidden.Code, hidden.Body.String())
	}
	for _, status := range []string{"preparando", "listo"} {
		rec := request("PATCH", "/v1/admin/kitchen/tickets/"+created.ID+"/status", []byte(fmt.Sprintf(`{"status":%q}`, status)), created.ID, s)
		if rec.Code != 204 {
			t.Fatalf("KDS %s: %d %s", status, rec.Code, rec.Body.String())
		}
	}
	dispatch := request("PATCH", "/v1/admin/orders/"+created.ID+"/status", []byte(`{"status":"en_camino"}`), created.ID, s)
	if dispatch.Code != 200 {
		t.Fatalf("dispatch: %d %s", dispatch.Code, dispatch.Body.String())
	}
	var registerID string
	if err := pool.QueryRow(ctx, `INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,'Caja delivery',$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&registerID); err != nil {
		t.Fatal(err)
	}
	shift := request("POST", "/v1/admin/cash-shifts", []byte(fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":0}`, registerID)), "", s)
	if shift.Code != 201 {
		t.Fatalf("cash shift: %d %s", shift.Code, shift.Body.String())
	}
	paid := request("POST", "/v1/admin/payments", []byte(fmt.Sprintf(`{"orderId":%q,"method":"card","amount":55}`, created.ID)), "", s)
	if paid.Code != 201 {
		t.Fatalf("delivery payment: %d %s", paid.Code, paid.Body.String())
	}
	delivered := request("PATCH", "/v1/admin/orders/"+created.ID+"/status", []byte(`{"status":"entregado"}`), created.ID, s)
	if delivered.Code != 200 {
		t.Fatalf("delivery completion: %d %s", delivered.Code, delivered.Body.String())
	}
	var final order
	if err := json.Unmarshal(delivered.Body.Bytes(), &final); err != nil {
		t.Fatal(err)
	}
	if final.PaymentStatus != "paid" || final.Status != "entregado" || final.CompletedAt == "" {
		t.Fatalf("unexpected final delivery: %#v", final)
	}
	detail := request("GET", "/v1/admin/pos/orders/"+created.ID, nil, created.ID, s)
	if detail.Code != 200 || !strings.Contains(detail.Body.String(), "Sin cebolla") {
		t.Fatalf("POS delivery detail: %d %s", detail.Code, detail.Body.String())
	}
}
