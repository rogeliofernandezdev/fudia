package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Delivery y recojo de Concierge: tarifas por distancia en línea recta,
// cotización, pedido idempotente, tarifa cambiada y estado por conversación.
func TestConciergeDeliveryAndPickupOrders(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	nonce := time.Now().UnixNano()
	t.Setenv("FUDIA_CONCIERGE_API_KEY", "concierge-test-key")

	var productID, qrToken string
	if err := pool.QueryRow(ctx, `
		INSERT INTO products(organization_id,sku,name,description,price,active,product_type,quantity_control)
		VALUES($1,$2,'Ceviche delivery','Para llevar',30,true,'prepared','none')
		RETURNING id
	`, s.OrganizationID, fmt.Sprintf("DLV-%d", nonce)).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO tables(organization_id,location_id,name,seats,zone,active,qr_token,qr_enabled)
		VALUES($1,$2,$3,2,'Principal',true,encode(gen_random_bytes(16),'hex'),true)
		RETURNING qr_token
	`, s.OrganizationID, s.LocationID, fmt.Sprintf("Mesa Delivery %d", nonce)).Scan(&qrToken); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO organization_modules(organization_id,module_key,active)
		VALUES($1,'whatsapp_bot',true)
		ON CONFLICT(organization_id,module_key) DO UPDATE SET active=true,updated_at=now()
	`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM concierge_delivery_requests WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(bg, `DELETE FROM orders WHERE organization_id=$1 AND location_id=$2 AND channel IN ('delivery','recojo')`, s.OrganizationID, s.LocationID)
		_, _ = pool.Exec(bg, `DELETE FROM delivery_fee_tiers WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(bg, `DELETE FROM tables WHERE organization_id=$1 AND qr_token=$2`, s.OrganizationID, qrToken)
	})

	call := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/v1/integrations/concierge/"+qrToken+path, strings.NewReader(body))
		req.Header.Set("X-Fudia-Concierge-Key", "concierge-test-key")
		rec := httptest.NewRecorder()
		api.Routes().ServeHTTP(rec, req)
		return rec
	}

	// Sin ubicación del local ni tarifas: solo recojo.
	rec := call("GET", "/delivery-options", "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"deliveryAvailable":false`) || !strings.Contains(rec.Body.String(), `"pickupAvailable":true`) {
		t.Fatalf("delivery must be unavailable without setup: %d %s", rec.Code, rec.Body.String())
	}
	if rec = call("POST", "/delivery-quote", `{"latitude":-12.12,"longitude":-77.03}`); rec.Code != 409 || !strings.Contains(rec.Body.String(), "delivery_unavailable") {
		t.Fatalf("quote without setup: %d %s", rec.Code, rec.Body.String())
	}

	// Local en Plaza de Armas de Lima; rangos 2 km -> 5, 10 km -> 9.
	if _, err := pool.Exec(ctx, `UPDATE locations SET latitude=-12.045974,longitude=-77.030601 WHERE id=$1 AND organization_id=$2`, s.LocationID, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO delivery_fee_tiers(organization_id,location_id,max_distance_km,fee)
		VALUES($1,$2,2,5),($1,$2,10,9)
	`, s.OrganizationID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	rec = call("GET", "/delivery-options", "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"deliveryAvailable":true`) || !strings.Contains(rec.Body.String(), `"maxDistanceKm":"10.00"`) {
		t.Fatalf("delivery options: %d %s", rec.Code, rec.Body.String())
	}

	// Parque Kennedy (~8.4 km) cae en el rango de 10 km.
	rec = call("POST", "/delivery-quote", `{"latitude":-12.121948,"longitude":-77.02968}`)
	var quote struct {
		Covered    bool   `json:"covered"`
		Fee        string `json:"fee"`
		DistanceKm string `json:"distanceKm"`
	}
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &quote) != nil || !quote.Covered || quote.Fee != "9.00" {
		t.Fatalf("quote: %d %s", rec.Code, rec.Body.String())
	}
	// Punto de prueba 0.2° al sur (~22.24 km en línea recta), fuera de los 10 km.
	rec = call("POST", "/delivery-quote", `{"latitude":-12.245974,"longitude":-77.030601}`)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"covered":false`) {
		t.Fatalf("out of range quote: %d %s", rec.Code, rec.Body.String())
	}

	conversation := fmt.Sprintf("conv-delivery-%d", nonce)
	deliveryBody := func(requestID, fee string) string {
		return fmt.Sprintf(`{
		  "fulfillment":"delivery","customerName":"Ana","customerPhone":"51999999999",
		  "conversationId":%q,"requestId":%q,"address":"Av. Larco 123","reference":"Puerta azul",
		  "latitude":-12.121948,"longitude":-77.02968,"quotedFee":%q,
		  "items":[{"productId":%q,"qty":2,"unitPrice":0.01,"note":"","selections":[]}]
		}`, conversation, requestID, fee, productID)
	}
	if rec = call("POST", "/delivery-orders", deliveryBody(fmt.Sprintf("dlv-old-%d", nonce), "5.00")); rec.Code != 409 || !strings.Contains(rec.Body.String(), "delivery_fee_changed") {
		t.Fatalf("a stale quoted fee must be rejected: %d %s", rec.Code, rec.Body.String())
	}
	requestID := fmt.Sprintf("dlv-%d", nonce)
	rec = call("POST", "/delivery-orders", deliveryBody(requestID, "9.00"))
	var created order
	if rec.Code != 201 || json.Unmarshal(rec.Body.Bytes(), &created) != nil {
		t.Fatalf("create delivery order: %d %s", rec.Code, rec.Body.String())
	}
	if created.Channel != "delivery" || created.Status != "confirmado" || created.TableID != "" ||
		created.DeliveryFee != "9.00" || created.Total != "69.00" || created.Address != "Av. Larco 123" {
		t.Fatalf("unexpected delivery order: %#v", created)
	}
	// Reintento con la misma clave: mismo pedido aunque cambien las tarifas.
	if _, err := pool.Exec(ctx, `UPDATE delivery_fee_tiers SET fee=11 WHERE organization_id=$1 AND max_distance_km=10`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	rec = call("POST", "/delivery-orders", deliveryBody(requestID, "9.00"))
	var retried order
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &retried) != nil || retried.ID != created.ID {
		t.Fatalf("retry must return the same order: %d %s", rec.Code, rec.Body.String())
	}
	var distance float64
	if err := pool.QueryRow(ctx, `SELECT delivery_distance_km::float8 FROM orders WHERE id=$1`, created.ID).Scan(&distance); err != nil || distance < 8 || distance > 9 {
		t.Fatalf("delivery distance must be stored: %v %v", distance, err)
	}

	pickup := fmt.Sprintf(`{
	  "fulfillment":"pickup","customerName":"Ana","customerPhone":"51999999999",
	  "conversationId":%q,"requestId":"pick-%d","address":"ignorada",
	  "items":[{"productId":%q,"qty":1,"note":"","selections":[]}]
	}`, conversation, nonce, productID)
	var picked order
	if rec = call("POST", "/delivery-orders", pickup); rec.Code != 201 || json.Unmarshal(rec.Body.Bytes(), &picked) != nil {
		t.Fatalf("create pickup order: %d %s", rec.Code, rec.Body.String())
	}
	if picked.Channel != "recojo" || picked.DeliveryFee != "0.00" || picked.Total != "30.00" || picked.Address != "" {
		t.Fatalf("unexpected pickup order: %#v", picked)
	}

	rec = call("GET", "/orders/"+created.ID+"?conversationId="+conversation, "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"active":true`) || !strings.Contains(rec.Body.String(), `"channel":"delivery"`) {
		t.Fatalf("order status: %d %s", rec.Code, rec.Body.String())
	}
	if rec = call("GET", "/orders/"+created.ID+"?conversationId=otra-conversacion", ""); rec.Code != 404 {
		t.Fatalf("another conversation must not read the order: %d %s", rec.Code, rec.Body.String())
	}
	if _, err := pool.Exec(ctx, `UPDATE orders SET status='cancelado' WHERE id=$1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if rec = call("GET", "/orders/"+created.ID+"?conversationId="+conversation, ""); !strings.Contains(rec.Body.String(), `"active":false`) {
		t.Fatalf("cancelled delivery is no longer active: %d %s", rec.Code, rec.Body.String())
	}
}
