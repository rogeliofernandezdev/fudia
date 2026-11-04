package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Venta por mostrador: el tipo de atención vive en la mesa, el QR decide el
// flujo en un solo lugar y los pedidos de mostrador son independientes, van a
// Cocina y a la bandeja de caja, descuentan inventario y se cobran sin cuenta
// de mesa. La mesa tradicional conserva su cuenta abierta.
func TestCounterServiceEndToEnd(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	foreign := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	nonce := time.Now().UnixNano()
	t.Setenv("FUDIA_CONCIERGE_API_KEY", "concierge-test-key")
	grantCashManagementForPaymentTest(t, api, s)

	for _, owner := range []scope{s, foreign} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO organization_modules(organization_id,module_key,active) VALUES($1,'whatsapp_bot',true)
			ON CONFLICT(organization_id,module_key) DO UPDATE SET active=true,updated_at=now()
		`, owner.OrganizationID); err != nil {
			t.Fatal(err)
		}
	}
	var kitchenProduct, directProduct string
	if err := pool.QueryRow(ctx, `
		INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control,service_destination)
		VALUES($1,$2,'Ceviche mostrador',25,true,'prepared','portions','kitchen') RETURNING id
	`, s.OrganizationID, fmt.Sprintf("CNT-K-%d", nonce)).Scan(&kitchenProduct); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control,service_destination)
		VALUES($1,$2,'Gaseosa mostrador',5,true,'retail','none','direct') RETURNING id
	`, s.OrganizationID, fmt.Sprintf("CNT-D-%d", nonce)).Scan(&directProduct); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO product_availability(organization_id,location_id,product_id,business_date,portion_quantity)
		SELECT $1,$2,$3,(now() AT TIME ZONE l.timezone)::date,40 FROM locations l WHERE l.id=$2
	`, s.OrganizationID, s.LocationID, kitchenProduct); err != nil {
		t.Fatal(err)
	}
	sold := func() float64 {
		t.Helper()
		var value float64
		if err := pool.QueryRow(ctx, `SELECT sold_quantity::float8 FROM product_availability WHERE organization_id=$1 AND location_id=$2 AND product_id=$3`, s.OrganizationID, s.LocationID, kitchenProduct).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}

	admin := func(owner scope, method, target, body string, handler http.HandlerFunc, pathValues ...string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, target, strings.NewReader(body))
		for i := 0; i+1 < len(pathValues); i += 2 {
			r.SetPathValue(pathValues[i], pathValues[i+1])
		}
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, owner))
		w := httptest.NewRecorder()
		handler(w, r)
		return w
	}
	type tableView struct {
		ID          string `json:"id"`
		QrToken     string `json:"qrToken"`
		ServiceType string `json:"serviceType"`
	}
	createTable := func(owner scope, body string) tableView {
		t.Helper()
		w := admin(owner, "POST", "/v1/admin/tables", body, api.createTable)
		var out tableView
		if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &out) != nil {
			t.Fatalf("create table: %d %s", w.Code, w.Body.String())
		}
		return out
	}

	// 1. Mesa tradicional: sin tipo explícito queda como mesa.
	table := createTable(s, fmt.Sprintf(`{"name":"Mesa CNT %d","seats":4}`, nonce))
	if table.ServiceType != "table" {
		t.Fatalf("a table defaults to table service: %+v", table)
	}
	// 2. Mostrador con la casilla «Venta por mostrador».
	counter := createTable(s, fmt.Sprintf(`{"name":"Mostrador %d","serviceType":"counter"}`, nonce))
	if counter.ServiceType != "counter" {
		t.Fatalf("counter not registered: %+v", counter)
	}
	if w := admin(s, "POST", "/v1/admin/tables", `{"name":"Barra X","serviceType":"bar"}`, api.createTable); w.Code != 400 || !strings.Contains(w.Body.String(), "invalid_service_type") {
		t.Fatalf("unknown service type: %d %s", w.Code, w.Body.String())
	}
	foreignCounter := createTable(foreign, fmt.Sprintf(`{"name":"Mostrador ajeno %d","serviceType":"counter"}`, nonce))

	// 3. Editar el tipo: ida y vuelta; omitirlo conserva el actual.
	patch := func(id, body string) *httptest.ResponseRecorder {
		return admin(s, "PATCH", "/v1/admin/tables/"+id, body, api.updateTable, "id", id)
	}
	if w := patch(counter.ID, fmt.Sprintf(`{"name":"Mostrador %d","serviceType":"table"}`, nonce)); w.Code != 200 || !strings.Contains(w.Body.String(), `"serviceType":"table"`) {
		t.Fatalf("counter to table: %d %s", w.Code, w.Body.String())
	}
	if w := patch(counter.ID, fmt.Sprintf(`{"name":"Mostrador %d","serviceType":"counter"}`, nonce)); w.Code != 200 || !strings.Contains(w.Body.String(), `"serviceType":"counter"`) {
		t.Fatalf("table to counter: %d %s", w.Code, w.Body.String())
	}
	if w := patch(counter.ID, fmt.Sprintf(`{"name":"Mostrador %d"}`, nonce)); w.Code != 200 || !strings.Contains(w.Body.String(), `"serviceType":"counter"`) {
		t.Fatalf("omitted type must keep counter: %d %s", w.Code, w.Body.String())
	}

	// 4 y 5. El QR de cada tipo existe y el contexto público informa el tipo.
	for _, item := range []tableView{table, counter} {
		r := httptest.NewRequest("GET", "/v1/public/tables/"+item.QrToken, nil)
		r.SetPathValue("token", item.QrToken)
		w := httptest.NewRecorder()
		api.getTableByQR(w, r)
		if len(item.QrToken) != 32 || w.Code != 200 || !strings.Contains(w.Body.String(), `"serviceType":"`+item.ServiceType+`"`) {
			t.Fatalf("public QR for %s: %d %s", item.ServiceType, w.Code, w.Body.String())
		}
	}

	concierge := func(token, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/v1/integrations/concierge/"+token+path, strings.NewReader(body))
		req.Header.Set("X-Fudia-Concierge-Key", "concierge-test-key")
		rec := httptest.NewRecorder()
		api.Routes().ServeHTTP(rec, req)
		return rec
	}
	orderBody := func(name, phone, conversation, requestID string, items string) string {
		return fmt.Sprintf(`{"customerName":%q,"customerPhone":%q,"conversationId":%q,"requestId":%q,"notes":"Sin ají","items":%s}`, name, phone, conversation, requestID, items)
	}
	kitchenItems := fmt.Sprintf(`[{"productId":%q,"qty":2,"note":"bien frío"},{"productId":%q,"qty":1}]`, kitchenProduct, directProduct)
	decode := func(rec *httptest.ResponseRecorder) order {
		t.Helper()
		var out order
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode order: %v %s", err, rec.Body.String())
		}
		return out
	}

	// 6. QR de mesa: se conserva la cuenta abierta y la segunda ronda se suma.
	tableConversation := fmt.Sprintf("tbl-%d", nonce)
	rec := concierge(table.QrToken, "/orders", orderBody("Lucía", "51911111111", tableConversation, "tbl-r1", fmt.Sprintf(`[{"productId":%q,"qty":1}]`, kitchenProduct)))
	tableOrder := decode(rec)
	if rec.Code != 201 || tableOrder.Channel != "whatsapp" || tableOrder.TableID != table.ID || tableOrder.Origin != "standard" {
		t.Fatalf("table QR flow changed: %d %+v", rec.Code, tableOrder)
	}
	rec = concierge(table.QrToken, "/orders", orderBody("Lucía", "51911111111", tableConversation, "tbl-r2", fmt.Sprintf(`[{"productId":%q,"qty":1}]`, directProduct)))
	if second := decode(rec); rec.Code != 200 || second.ID != tableOrder.ID || second.Total != "30.00" {
		t.Fatalf("second round must join the open account: %d %+v", rec.Code, second)
	}
	if w := patch(table.ID, fmt.Sprintf(`{"name":"Mesa CNT %d","seats":4,"serviceType":"counter"}`, nonce)); w.Code != 409 || !strings.Contains(w.Body.String(), "table_in_use") {
		t.Fatalf("a table with an open account cannot become a counter: %d %s", w.Code, w.Body.String())
	}
	if rec = concierge(table.QrToken, "/bill", fmt.Sprintf(`{"conversationId":%q,"requestId":"bill-1"}`, tableConversation)); rec.Code != 200 {
		t.Fatalf("table bill: %d %s", rec.Code, rec.Body.String())
	}

	// 7. QR de mostrador: pedido independiente, sin mesa, guardado hasta que caja lo envía.
	soldBefore := sold()
	rec = concierge(counter.QrToken, "/orders", orderBody("Ana", "+51 922 222 222", "cnt-ana", "cnt-ana-1", kitchenItems))
	first := decode(rec)
	if rec.Code != 201 || first.Channel != "recojo" || first.Origin != "counter" || first.TableID != "" ||
		first.Status != "nuevo" || first.CustomerName != "Ana" || first.CustomerID == "" || first.Total != "55.00" || first.Notes != "Sin ají" || !strings.HasPrefix(first.Code, "PED-") {
		t.Fatalf("counter QR order: %d %+v", rec.Code, first)
	}
	var stored string
	var tableID *string
	if err := pool.QueryRow(ctx, `SELECT channel,table_id::text FROM orders WHERE id=$1`, first.ID).Scan(&stored, &tableID); err != nil || stored != "mostrador" || tableID != nil {
		t.Fatalf("counter order must be stored as mostrador without table: %q %v %v", stored, tableID, err)
	}
	// 13. Inventario: el pedido descuenta las porciones al registrarse.
	if got := sold(); got != soldBefore+2 {
		t.Fatalf("counter order must consume portions: before=%v after=%v", soldBefore, got)
	}
	// 15. Reintento con el mismo requestId: mismo pedido, sin duplicar consumo.
	rec = concierge(counter.QrToken, "/orders", orderBody("Ana", "+51 922 222 222", "cnt-ana", "cnt-ana-1", kitchenItems))
	if retry := decode(rec); rec.Code != 200 || retry.ID != first.ID || sold() != soldBefore+2 {
		t.Fatalf("duplicate counter order: %d %+v sold=%v", rec.Code, retry, sold())
	}
	// 8. Otro cliente con el mismo QR: pedido distinto; el mostrador no se ocupa.
	rec = concierge(counter.QrToken, "/orders", orderBody("Beto", "51933333333", "cnt-beto", "cnt-beto-1", fmt.Sprintf(`[{"productId":%q,"qty":1}]`, kitchenProduct)))
	beto := decode(rec)
	if rec.Code != 201 || beto.ID == first.ID || beto.CustomerName != "Beto" || beto.CustomerID == first.CustomerID {
		t.Fatalf("second customer must get an independent order: %d %+v", rec.Code, beto)
	}
	// El mismo cliente puede pedir de nuevo: otro requestId es otro pedido.
	rec = concierge(counter.QrToken, "/orders", orderBody("Ana", "51922222222", "cnt-ana", "cnt-ana-2", fmt.Sprintf(`[{"productId":%q,"qty":1}]`, directProduct)))
	if again := decode(rec); rec.Code != 201 || again.ID == first.ID || again.CustomerID != first.CustomerID {
		t.Fatalf("a new request from the same customer is a new order with the same customer: %d %+v", rec.Code, again)
	}
	if rec = concierge(counter.QrToken, "/orders", orderBody("", "51944444444", "cnt-x", "cnt-x-1", kitchenItems)); rec.Code != 400 {
		t.Fatalf("a counter order needs the customer's name: %d %s", rec.Code, rec.Body.String())
	}
	if rec = concierge(counter.QrToken, "/bill", `{"conversationId":"cnt-ana","requestId":"bill-cnt"}`); rec.Code != 409 || !strings.Contains(rec.Body.String(), "counter_has_no_bill") {
		t.Fatalf("a counter has no open bill: %d %s", rec.Code, rec.Body.String())
	}
	// El mostrador no aparece en el mapa de Salón ni queda ocupado.
	w := admin(s, "GET", "/v1/admin/orders/floor", "", api.getOrdersFloor)
	if w.Code != 200 || strings.Contains(w.Body.String(), counter.ID) || !strings.Contains(w.Body.String(), table.ID) {
		t.Fatalf("salon floor must list tables only: %d %s", w.Code, w.Body.String())
	}
	if w := patch(counter.ID, fmt.Sprintf(`{"name":"Mostrador %d","active":false,"serviceType":"counter"}`, nonce)); w.Code != 200 {
		t.Fatalf("open counter orders never occupy the counter: %d %s", w.Code, w.Body.String())
	}
	if w := patch(counter.ID, fmt.Sprintf(`{"name":"Mostrador %d","active":true,"serviceType":"counter"}`, nonce)); w.Code != 200 {
		t.Fatalf("reactivate counter: %d %s", w.Code, w.Body.String())
	}

	// 9. Pedidos simultáneos con el mismo QR: todos independientes; el mismo
	// requestId enviado a la vez crea un solo pedido.
	var wg sync.WaitGroup
	ids := make(chan string, 8)
	codes := make(chan int, 8)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := concierge(counter.QrToken, "/orders", orderBody(fmt.Sprintf("Cliente %d", i), fmt.Sprintf("5195555550%d", i), fmt.Sprintf("cnt-par-%d", i), fmt.Sprintf("cnt-par-%d", i), fmt.Sprintf(`[{"productId":%q,"qty":1}]`, directProduct)))
			if r.Code >= 300 {
				t.Logf("parallel %d: %d %s", i, r.Code, r.Body.String())
			}
			codes <- r.Code
			var out order
			_ = json.Unmarshal(r.Body.Bytes(), &out)
			ids <- out.ID
		}(i)
	}
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := concierge(counter.QrToken, "/orders", orderBody("Doble", "51966666666", "cnt-dup", "cnt-dup-1", fmt.Sprintf(`[{"productId":%q,"qty":1}]`, directProduct)))
			codes <- r.Code
			var out order
			_ = json.Unmarshal(r.Body.Bytes(), &out)
			ids <- out.ID
		}()
	}
	wg.Wait()
	close(ids)
	close(codes)
	for code := range codes {
		if code != 200 && code != 201 {
			t.Fatalf("simultaneous counter order failed: %d", code)
		}
	}
	unique := map[string]bool{}
	for id := range ids {
		unique[id] = true
	}
	if len(unique) != 7 {
		t.Fatalf("six customers and one duplicated request must create 7 orders, got %d", len(unique))
	}

	// 16. Clientes: prerregistro sin duplicados por número, asociado a sus pedidos.
	var anaCustomers int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM customers WHERE organization_id=$1 AND regexp_replace(phone,'\D','','g')='51922222222'`, s.OrganizationID).Scan(&anaCustomers); err != nil || anaCustomers != 1 {
		t.Fatalf("customer must be pre-registered once: %d %v", anaCustomers, err)
	}
	rec = concierge(counter.QrToken, "/customer", `{"customerPhone":"+51 922 222 222"}`)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"known":true`) || !strings.Contains(rec.Body.String(), `"customerName":"Ana"`) || !strings.Contains(rec.Body.String(), `"marketingConsent":"pending"`) {
		t.Fatalf("customer profile: %d %s", rec.Code, rec.Body.String())
	}
	if rec = concierge(counter.QrToken, "/customer", `{"customerPhone":"51900000001"}`); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"known":false`) {
		t.Fatalf("new customer: %d %s", rec.Code, rec.Body.String())
	}

	// 17. Consentimiento promocional: una sola pregunta, respuesta con evidencia, retiro.
	consent := func(token, requestID, decision, evidence string) *httptest.ResponseRecorder {
		return concierge(token, "/marketing-consent", fmt.Sprintf(`{"customerPhone":"51922222222","conversationId":"cnt-ana","requestId":%q,"decision":%q,"evidence":%q}`, requestID, decision, evidence))
	}
	if rec = consent(counter.QrToken, "c-ask", "asked", "¿Te gustaría recibir ofertas…?"); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"marketingConsent":"asked"`) {
		t.Fatalf("ask consent: %d %s", rec.Code, rec.Body.String())
	}
	if rec = consent(counter.QrToken, "c-ask-2", "asked", "otra vez"); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"marketingConsent":"asked"`) {
		t.Fatalf("second ask must be ignored: %d %s", rec.Code, rec.Body.String())
	}
	if rec = consent(counter.QrToken, "c-yes", "granted", "Sí, quiero recibirlos"); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"marketingConsent":"granted"`) {
		t.Fatalf("grant consent: %d %s", rec.Code, rec.Body.String())
	}
	if rec = consent(counter.QrToken, "c-yes", "granted", "Sí, quiero recibirlos"); rec.Code != 200 {
		t.Fatalf("idempotent consent: %d %s", rec.Code, rec.Body.String())
	}
	var granted bool
	var events int
	if err := pool.QueryRow(ctx, `
		SELECT c.marketing_consent,(SELECT count(*) FROM customer_marketing_consents e WHERE e.customer_id=c.id AND e.organization_id=c.organization_id)
		FROM customers c WHERE c.id=$1
	`, first.CustomerID).Scan(&granted, &events); err != nil || !granted || events != 2 {
		t.Fatalf("consent evidence: granted=%v events=%d err=%v", granted, events, err)
	}
	if rec = consent(counter.QrToken, "c-off", "withdrawn", "Ya no quiero promociones"); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"marketingConsent":"withdrawn"`) {
		t.Fatalf("withdraw consent: %d %s", rec.Code, rec.Body.String())
	}
	var evidence, source string
	if err := pool.QueryRow(ctx, `SELECT c.marketing_consent,e.evidence,e.source FROM customers c JOIN customer_marketing_consents e ON e.customer_id=c.id AND e.request_id='c-off' WHERE c.id=$1`, first.CustomerID).Scan(&granted, &evidence, &source); err != nil || granted || evidence != "Ya no quiero promociones" || source != "whatsapp" {
		t.Fatalf("withdrawal must disable campaigns with evidence: %v %q %q %v", granted, evidence, source, err)
	}
	if rec = concierge(counter.QrToken, "/marketing-consent", `{"customerPhone":"51900000001","requestId":"c-unknown","decision":"granted"}`); rec.Code != 404 {
		t.Fatalf("consent for a non-customer: %d %s", rec.Code, rec.Body.String())
	}
	// Clientes: activar el interruptor deja evidencia del autor; se ve el último estado.
	w = admin(s, "PATCH", "/v1/admin/customers/"+first.CustomerID, `{"customerType":"person","displayName":"Ana","phone":"+51922222222","preferredChannel":"whatsapp","marketingConsent":true}`, api.updateCustomer, "id", first.CustomerID)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"marketingConsentStatus":"granted"`) || !strings.Contains(w.Body.String(), `"marketingConsentUpdatedAt":"20`) {
		t.Fatalf("admin consent: %d %s", w.Code, w.Body.String())
	}
	var adminEvents int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM customer_marketing_consents WHERE customer_id=$1 AND source='admin' AND decision='granted' AND created_by=$2`, first.CustomerID, s.UserID).Scan(&adminEvents); err != nil || adminEvents != 1 {
		t.Fatalf("admin consent evidence: %d %v", adminEvents, err)
	}
	if rec = concierge(counter.QrToken, "/customer", `{"customerPhone":"51922222222"}`); !strings.Contains(rec.Body.String(), `"marketingConsent":"granted"`) {
		t.Fatalf("profile after admin change: %s", rec.Body.String())
	}

	// 10. Cajero: venta por mostrador sin mesa desde Pedidos; un mostrador no abre cuenta de mesa.
	w = admin(s, "POST", "/v1/admin/orders", fmt.Sprintf(`{"channel":"mostrador","customerName":"Rosa","notes":"Para llevar","sendToKitchen":true,"items":[{"productId":%q,"qty":1}]}`, kitchenProduct), api.createOrder)
	var cashier order
	if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &cashier) != nil || cashier.Origin != "counter" || cashier.Channel != "recojo" || cashier.TableID != "" || cashier.Status != "confirmado" {
		t.Fatalf("cashier counter order: %d %s", w.Code, w.Body.String())
	}
	if err := pool.QueryRow(ctx, `SELECT channel FROM orders WHERE id=$1`, cashier.ID).Scan(&stored); err != nil || stored != "mostrador" {
		t.Fatalf("cashier counter sale stored as %q: %v", stored, err)
	}
	if w = admin(s, "POST", "/v1/admin/orders", fmt.Sprintf(`{"channel":"salon","tableId":%q,"items":[{"productId":%q,"qty":1}]}`, counter.ID, directProduct), api.createOrder); w.Code != 409 || !strings.Contains(w.Body.String(), "counter_not_table") {
		t.Fatalf("a counter cannot open a salon account: %d %s", w.Code, w.Body.String())
	}
	if w = admin(s, "POST", "/v1/admin/orders", fmt.Sprintf(`{"channel":"mostrador","tableId":%q,"items":[{"productId":%q,"qty":1}]}`, counter.ID, directProduct), api.createOrder); w.Code != 400 || !strings.Contains(w.Body.String(), "table_not_allowed") {
		t.Fatalf("a counter sale never takes a table: %d %s", w.Code, w.Body.String())
	}
	w = admin(s, "GET", "/v1/admin/orders?pageSize=50", "", api.listOrders)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"manualChannelOptions":[{"label":"Mostrador","value":"mostrador"}`) {
		t.Fatalf("manual channel options: %d %s", w.Code, w.Body.String()[:min(400, len(w.Body.String()))])
	}

	// 11 y 12. Guardada por el bot, Cocina no la ve hasta que caja pulsa «Enviar comanda»;
	// Sin preparación nunca entra al KDS.
	w = admin(s, "GET", "/v1/admin/kitchen/tickets?destination=kitchen", "", api.listKitchenTickets)
	if strings.Contains(w.Body.String(), first.ID) {
		t.Fatalf("an unsent counter order must not reach the kitchen: %s", w.Body.String())
	}
	sendBotOrder(t, api, s, first.ID)
	w = admin(s, "GET", "/v1/admin/kitchen/tickets?destination=kitchen", "", api.listKitchenTickets)
	var kitchen struct {
		Items []struct {
			OrderID string `json:"orderId"`
			Origin  string `json:"origin"`
			Items   []struct {
				Name string `json:"name"`
			} `json:"items"`
		} `json:"items"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &kitchen) != nil {
		t.Fatalf("kitchen: %d %s", w.Code, w.Body.String())
	}
	found := false
	for _, ticket := range kitchen.Items {
		if ticket.OrderID != first.ID {
			continue
		}
		found = true
		if ticket.Origin != "counter" {
			t.Fatalf("kitchen ticket must keep the counter origin: %+v", ticket)
		}
		for _, item := range ticket.Items {
			if item.Name == "Gaseosa mostrador" {
				t.Fatalf("direct products never reach the KDS: %+v", ticket)
			}
		}
	}
	if !found {
		t.Fatalf("counter order missing from the kitchen: %s", w.Body.String())
	}
	var directStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM order_service_items WHERE order_id=$1 AND destination='direct'`, first.ID).Scan(&directStatus); err != nil || directStatus != "listo" {
		t.Fatalf("direct product waits for delivery: %q %v", directStatus, err)
	}

	// 14. Caja: la bandeja muestra el pedido de mostrador y se cobra antes de prepararlo.
	w = admin(s, "GET", "/v1/admin/pos/orders?pageSize=50", "", api.listPOSOrders)
	var tray struct {
		Items []struct {
			ID            string `json:"id"`
			Code          string `json:"code"`
			Origin        string `json:"origin"`
			Status        string `json:"status"`
			CustomerName  string `json:"customerName"`
			ItemsText     string `json:"itemsText"`
			PaymentStatus string `json:"paymentStatus"`
			Total         string `json:"total"`
		} `json:"items"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &tray) != nil {
		t.Fatalf("POS tray: %d %s", w.Code, w.Body.String())
	}
	inTray := false
	for _, item := range tray.Items {
		if item.ID == first.ID {
			inTray = true
			if item.Origin != "counter" || item.CustomerName != "Ana" || item.Status != "confirmado" || item.PaymentStatus != "pending" || item.Total != "55.00" ||
				!strings.Contains(item.ItemsText, "2 × Ceviche mostrador") || !strings.Contains(item.ItemsText, "1 × Gaseosa mostrador") || item.Code != first.Code {
				t.Fatalf("counter order in the tray: %+v", item)
			}
		}
	}
	if !inTray {
		t.Fatal("counter order missing from the cashier tray")
	}
	var registerID string
	if err := pool.QueryRow(ctx, `INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,'Caja mostrador',$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&registerID); err != nil {
		t.Fatal(err)
	}
	if w = admin(s, "POST", "/v1/admin/cash-shifts", fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":50}`, registerID), api.openCashShift); w.Code != 201 {
		t.Fatalf("open shift: %d %s", w.Code, w.Body.String())
	}
	payment := httptest.NewRequest("POST", "/v1/admin/payments", bytes.NewReader([]byte(fmt.Sprintf(`{"orderId":%q,"method":"cash","amount":55}`, first.ID))))
	payment = payment.WithContext(context.WithValue(payment.Context(), scopeKey{}, s))
	paid := httptest.NewRecorder()
	api.createPayment(paid, payment)
	if paid.Code != 201 {
		t.Fatalf("pay counter order before preparation: %d %s", paid.Code, paid.Body.String())
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM orders WHERE id=$1`, first.ID).Scan(&status); err != nil || status != "confirmado" {
		t.Fatalf("payment must not change preparation: %q %v", status, err)
	}

	// 13. Cancelar un pedido de mostrador sin preparar revierte las porciones.
	beforeCancel := sold()
	if w = admin(s, "PATCH", "/v1/admin/orders/"+beto.ID+"/status", `{"status":"cancelado"}`, api.updateOrderStatus, "id", beto.ID); w.Code != 200 {
		t.Fatalf("cancel counter order: %d %s", w.Code, w.Body.String())
	}
	if got := sold(); got != beforeCancel-1 {
		t.Fatalf("cancellation must revert portions: before=%v after=%v", beforeCancel, got)
	}

	// 18. Aislamiento: el QR ajeno no ve clientes ni pedidos de esta empresa.
	if rec = concierge(foreignCounter.QrToken, "/customer", `{"customerPhone":"51922222222"}`); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"known":false`) {
		t.Fatalf("customers leak across companies: %d %s", rec.Code, rec.Body.String())
	}
	if rec = concierge(foreignCounter.QrToken, "/marketing-consent", `{"customerPhone":"51922222222","requestId":"c-foreign","decision":"granted"}`); rec.Code != 404 {
		t.Fatalf("consent leaks across companies: %d %s", rec.Code, rec.Body.String())
	}
	if rec = concierge(foreignCounter.QrToken, "/orders", orderBody("Ana", "51922222222", "cnt-ana", "cnt-ana-1", kitchenItems)); rec.Code < 400 {
		t.Fatalf("a foreign QR cannot order another company's products: %d %s", rec.Code, rec.Body.String())
	}
	w = admin(foreign, "GET", "/v1/admin/pos/orders?pageSize=50", "", api.listPOSOrders)
	if w.Code != 200 || strings.Contains(w.Body.String(), first.ID) {
		t.Fatalf("cashier tray leaks across companies: %d %s", w.Code, w.Body.String())
	}
	var foreignOrders int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE organization_id=$1`, foreign.OrganizationID).Scan(&foreignOrders); err != nil || foreignOrders != 0 {
		t.Fatalf("foreign company got orders: %d %v", foreignOrders, err)
	}
}
