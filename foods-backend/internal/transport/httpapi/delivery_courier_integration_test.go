package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Delivery Fudia con motorizado: la central escribe el nombre antes de obtener
// el enlace; cambiarlo invalida el anterior sin perder el avance. Liberar antes
// de recoger lo ofrece a las demás centrales. Con el pedido listo y sin recoger
// hay alertas (la de 8 min también al restaurante) y nunca se libera solo. Si el
// motorizado no puede entregar, Fudia pide al cliente que se comunique y la
// central registra la entrega fallida con sus montos.
func TestDeliveryFudiaCourierFlow(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE locations SET district_code='150101',latitude=-12.046374,longitude=-77.042793 WHERE id=$1`, s.LocationID); err != nil {
		t.Fatal(err)
	}
	prefix := fmt.Sprintf("517%08d", time.Now().UnixNano()%100000000)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM organization_delivery_settings WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM delivery_dispatches WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM delivery_partners WHERE whatsapp_phone LIKE $1`, prefix+"%")
	})
	d := api.handlers.Delivery
	call := func(handler func(http.ResponseWriter, *http.Request), method, body string, path map[string]string, subject scope) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/", bytes.NewReader([]byte(body)))
		for key, value := range path {
			req.SetPathValue(key, value)
		}
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, subject))
		rec := httptest.NewRecorder()
		handler(rec, req)
		return rec
	}
	decode := func(rec *httptest.ResponseRecorder, out any) {
		t.Helper()
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("decode %s: %v", rec.Body.String(), err)
		}
	}
	partner := func(name string, index int) string {
		rec := call(d.CreatePartner, "POST", fmt.Sprintf(`{"name":%q,"whatsappPhone":"%s%d","coverageCodes":["150101"]}`, name, prefix, index), nil, s)
		if rec.Code != 201 {
			t.Fatalf("create partner: %d %s", rec.Code, rec.Body.String())
		}
		var out struct{ ID string }
		decode(rec, &out)
		return operatorSession(t, d, s, out.ID, fmt.Sprintf("op-%s-%d@courier.test", prefix, index))
	}
	northPanel, southPanel := partner("Motos Norte", 1), partner("Motos Sur", 2)
	if _, err := pool.Exec(ctx, `UPDATE delivery_partners SET active=false WHERE whatsapp_phone NOT LIKE $1`, prefix+"%"); err != nil {
		t.Fatal(err)
	}
	if rec := call(d.SetOrganizationDelivery, "PUT", `{"enabled":true}`, map[string]string{"id": s.OrganizationID}, s); rec.Code != 200 {
		t.Fatalf("enable delivery: %d %s", rec.Code, rec.Body.String())
	}

	type panelOrder struct {
		Token, Stage, OrderStatus, CourierName, CourierToken, PickupAlert, IssueReportedAt, Total string
		PickupWaitSeconds, FailAvailableInSeconds                                                 int
		PickupAlertAcknowledged                                                                   bool
	}
	type failed struct {
		OrderCode, CourierName, FoodAmount, CourierReimbursement, PartnerShare, FudiaShare string
		FudiaShareEligible                                                                 bool
		ValidationNotes                                                                    []string
	}
	panel := func(token string) ([]panelOrder, []failed) {
		t.Helper()
		rec := panelCall(d, d.Panel, "GET", "", token)
		if rec.Code != 200 {
			t.Fatalf("panel: %d %s", rec.Code, rec.Body.String())
		}
		var out struct {
			Orders           []panelOrder
			FailedDeliveries []failed
		}
		decode(rec, &out)
		return out.Orders, out.FailedDeliveries
	}
	act := func(panelToken, action, token string) string {
		t.Helper()
		rec := panelCall(d, d.PanelAction, "POST", fmt.Sprintf(`{"action":%q,"token":%q}`, action, token), panelToken)
		if rec.Code != 200 {
			t.Fatalf("%s: %d %s", action, rec.Code, rec.Body.String())
		}
		var out struct{ Outcome string }
		decode(rec, &out)
		return out.Outcome
	}
	assign := func(panelToken, offer, name string) (int, string, string) {
		t.Helper()
		rec := panelCall(d, d.PanelCourier, "POST", fmt.Sprintf(`{"token":%q,"courierName":%q}`, offer, name), panelToken)
		var out struct{ Outcome, CourierToken string }
		if rec.Code == 200 {
			decode(rec, &out)
		}
		return rec.Code, out.Outcome, out.CourierToken
	}
	courier := func(token string) (int, panelOrder) {
		t.Helper()
		rec := call(d.Courier, "GET", "", map[string]string{"token": token}, scope{})
		var out struct {
			PartnerName, CourierName string
			Order                    panelOrder
		}
		if rec.Code == 200 {
			decode(rec, &out)
		}
		return rec.Code, out.Order
	}
	courierAct := func(token, action string) (int, string) {
		t.Helper()
		rec := call(d.CourierAction, "POST", fmt.Sprintf(`{"action":%q}`, action), map[string]string{"token": token}, scope{})
		var out struct{ Outcome string }
		if rec.Code == 200 {
			decode(rec, &out)
		}
		return rec.Code, out.Outcome
	}
	state := func(orderID string) (dispatch, order, total string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT d.status,o.status,o.total::text FROM delivery_dispatches d JOIN orders o ON o.id=d.order_id WHERE o.id=$1`, orderID).Scan(&dispatch, &order, &total); err != nil {
			t.Fatal(err)
		}
		return dispatch, order, total
	}
	count := func(query, orderID string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, query, orderID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	var productID string
	if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control) VALUES($1,'FUDIA-COURIER','Arroz chaufa',30,true,'prepared','none') RETURNING id`, s.OrganizationID).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	rec := call(api.createOrder, "POST", fmt.Sprintf(`{"channel":"delivery","customerName":"Ana","customerPhone":"999123456","address":"Av. Perú 100","deliveryFee":5,"sendToKitchen":true,"items":[{"productId":%q,"qty":2}]}`, productID), nil, s)
	if rec.Code != 201 {
		t.Fatalf("create delivery: %d %s", rec.Code, rec.Body.String())
	}
	var created struct{ ID string }
	decode(rec, &created)
	orderID := created.ID
	// El cliente pidió por el chat: puede recibir el aviso para comunicarse.
	if _, err := pool.Exec(ctx, `UPDATE orders SET delivery_latitude=-12.121900,delivery_longitude=-77.029700 WHERE id=$1;`, orderID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO concierge_delivery_requests(organization_id,location_id,request_id,conversation_id,order_id) VALUES($1,$2,'courier-test','web:courier-test',$3)`, s.OrganizationID, s.LocationID, orderID); err != nil {
		t.Fatal(err)
	}
	kitchen := func(status string) {
		t.Helper()
		if rec := call(api.updateKitchenTicketStatus, "PATCH", fmt.Sprintf(`{"status":%q}`, status), map[string]string{"id": orderID}, s); rec.Code != 204 {
			t.Fatalf("kitchen %s: %d %s", status, rec.Code, rec.Body.String())
		}
	}
	kitchen("preparando")
	orders, _ := panel(northPanel)
	if len(orders) != 1 || orders[0].Stage != "offer" {
		t.Fatalf("offer: %#v", orders)
	}
	northOffer := orders[0].Token
	if got := act(northPanel, "accept", northOffer); got != "accepted" {
		t.Fatalf("accept: %s", got)
	}

	// Sin nombre válido no hay enlace; otra central no puede asignar en una oferta ajena.
	if code, _, _ := assign(northPanel, northOffer, " L "); code != 400 {
		t.Fatalf("short courier name: %d", code)
	}
	if _, outcome, _ := assign(southPanel, northOffer, "Luis Pérez"); outcome != "not_found" {
		t.Fatalf("foreign central assigned: %s", outcome)
	}
	tokenShape := regexp.MustCompile(`^[a-f0-9]{32}$`)
	_, outcome, luis := assign(northPanel, northOffer, "Luis Pérez")
	if outcome != "courier_assigned" || !tokenShape.MatchString(luis) {
		t.Fatalf("assign: %s %q", outcome, luis)
	}
	// Doble clic con el mismo nombre conserva el enlace.
	if _, _, again := assign(northPanel, northOffer, "  Luis   Pérez "); again != luis {
		t.Fatalf("same name must keep the link: %s %s", again, luis)
	}
	if orders, _ = panel(northPanel); orders[0].CourierName != "Luis Pérez" || orders[0].CourierToken != luis {
		t.Fatalf("courier in panel: %#v", orders)
	}
	if code, view := courier(luis); code != 200 || view.Stage != "accepted" || view.Token != "" || view.CourierToken != "" || view.Total != "65.00" {
		t.Fatalf("courier view: %d %#v", code, view)
	}
	// Cambiar el nombre genera otro enlace y el anterior deja de abrir; el avance se mantiene.
	_, _, mario := assign(northPanel, northOffer, "Mario Díaz")
	if mario == luis || !tokenShape.MatchString(mario) {
		t.Fatalf("new courier link: %q", mario)
	}
	if code, _ := courier(luis); code != 404 {
		t.Fatalf("old courier link still opens: %d", code)
	}
	if code, _ := courierAct(luis, "pickup"); code != 404 {
		t.Fatalf("old courier link still acts: %d", code)
	}
	if dispatch, _, _ := state(orderID); dispatch != "accepted" {
		t.Fatalf("courier change altered progress: %s", dispatch)
	}

	// El motorizado no puede continuar antes de recoger: la central libera y se ofrece a las demás.
	if got := act(northPanel, "release", northOffer); got != "released" {
		t.Fatalf("release: %s", got)
	}
	if got := act(northPanel, "release", northOffer); got != "taken" {
		t.Fatalf("double release: %s", got)
	}
	if code, _ := courier(mario); code != 404 {
		t.Fatalf("released courier link still opens: %d", code)
	}
	if dispatch, _, total := state(orderID); dispatch != "searching" || total != "65.00" {
		t.Fatalf("after release: %s %s", dispatch, total)
	}
	if orders, _ = panel(northPanel); len(orders) != 0 {
		t.Fatalf("released order offered again to the same central: %#v", orders)
	}
	orders, _ = panel(southPanel)
	if len(orders) != 1 || orders[0].Stage != "offer" {
		t.Fatalf("release must offer to the other centrals: %#v", orders)
	}
	southOffer := orders[0].Token
	if got := act(southPanel, "accept", southOffer); got != "accepted" {
		t.Fatalf("south accept: %s", got)
	}
	_, _, rosa := assign(southPanel, southOffer, "Rosa Quispe")

	// Pedido listo y el motorizado no llega: alertas sin liberar por tiempo.
	kitchen("listo")
	if code, got := courierAct(rosa, "pickup"); code != 200 || got != "payment_pending" {
		t.Fatalf("pickup before payment: %d %s", code, got)
	}
	if _, err := pool.Exec(ctx, `UPDATE delivery_dispatches SET ready_notified_at=now()-interval '4 minutes',courier_assigned_at=now()-interval '4 minutes' WHERE order_id=$1`, orderID); err != nil {
		t.Fatal(err)
	}
	if orders, _ = panel(southPanel); orders[0].PickupAlert != "warning" || orders[0].PickupAlertAcknowledged {
		t.Fatalf("3 minute alert: %#v", orders)
	}
	if err := api.dispatcher.Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	delayed := `SELECT count(*) FROM staff_notices WHERE order_id=$1 AND kind='delivery_pickup_delayed'`
	if n := count(delayed, orderID); n != 0 {
		t.Fatalf("restaurant alerted before 8 minutes: %d", n)
	}
	if _, err := pool.Exec(ctx, `UPDATE delivery_dispatches SET ready_notified_at=now()-interval '9 minutes',courier_assigned_at=now()-interval '9 minutes' WHERE order_id=$1`, orderID); err != nil {
		t.Fatal(err)
	}
	if orders, _ = panel(southPanel); orders[0].PickupAlert != "critical" || orders[0].PickupWaitSeconds < 540 {
		t.Fatalf("critical alert: %#v", orders)
	}
	for range 2 {
		if err := api.dispatcher.Sweep(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if n := count(delayed, orderID); n != 1 {
		t.Fatalf("restaurant critical alert once: %d", n)
	}
	if dispatch, _, _ := state(orderID); dispatch != "accepted" {
		t.Fatalf("time must not release the order: %s", dispatch)
	}
	// La opción «Delivery» del restaurante ve la misma alerta, la central y el motorizado.
	type boardOrder struct {
		ID, Code, DispatchStatus, PartnerName, CourierName, PickupAlert, IssueReportedAt string
		FailedDelivery                                                                   bool
	}
	board := func(subject scope) []boardOrder {
		t.Helper()
		rec := call(d.DeliveryBoard, "GET", "", nil, subject)
		if rec.Code != 200 {
			t.Fatalf("board: %d %s", rec.Code, rec.Body.String())
		}
		var out struct{ Items []boardOrder }
		decode(rec, &out)
		return out.Items
	}
	if items := board(s); len(items) != 1 || items[0].ID != orderID || items[0].DispatchStatus != "accepted" ||
		items[0].PartnerName != "Motos Sur" || items[0].CourierName != "Rosa Quispe" || items[0].PickupAlert != "critical" {
		t.Fatalf("board with alert: %#v", items)
	}
	// El detalle consulta un pedido sin depender del historial reciente del tablero.
	readDispatch := func(subject scope) *httptest.ResponseRecorder {
		return call(d.OrderDispatch, "GET", "", map[string]string{"id": orderID}, subject)
	}
	if rec := readDispatch(s); rec.Code != http.StatusOK {
		t.Fatalf("order dispatch: %d %s", rec.Code, rec.Body.String())
	} else {
		var detail struct {
			PartnerName, PartnerPhone, CourierName, PickupAlert string
			PickupWaitSeconds                                   int
		}
		decode(rec, &detail)
		if detail.PartnerName != "Motos Sur" || detail.PartnerPhone != prefix+"2" || detail.CourierName != "Rosa Quispe" || detail.PickupAlert != "critical" || detail.PickupWaitSeconds < 540 {
			t.Fatalf("order dispatch tracking: %#v", detail)
		}
	}
	foreign := s
	foreign.LocationID = "00000000-0000-0000-0000-000000000001"
	if rec := readDispatch(foreign); rec.Code != http.StatusNotFound {
		t.Fatalf("order dispatch location isolation: %d %s", rec.Code, rec.Body.String())
	}
	foreign = s
	foreign.OrganizationID = "00000000-0000-0000-0000-000000000001"
	if rec := readDispatch(foreign); rec.Code != http.StatusNotFound {
		t.Fatalf("order dispatch organization isolation: %d %s", rec.Code, rec.Body.String())
	}
	if got := act(southPanel, "keep", southOffer); got != "kept" {
		t.Fatalf("keep: %s", got)
	}
	if orders, _ = panel(southPanel); !orders[0].PickupAlertAcknowledged || orders[0].PickupAlert != "critical" {
		t.Fatalf("kept alert: %#v", orders)
	}
	// Caja confirma el pago del motorizado: con pagos ya no se libera desde el panel.
	grantCashManagementForPaymentTest(t, api, s)
	var registerID string
	if err := pool.QueryRow(ctx, `INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,'Caja motorizados',$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&registerID); err != nil {
		t.Fatal(err)
	}
	if rec := call(api.openCashShift, "POST", fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":0}`, registerID), nil, s); rec.Code != 201 {
		t.Fatalf("open cash: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(api.createPayment, "POST", fmt.Sprintf(`{"orderId":%q,"method":"cash","amount":60}`, orderID), nil, s); rec.Code != 201 {
		t.Fatalf("courier payment: %d %s", rec.Code, rec.Body.String())
	}
	if got := act(southPanel, "release", southOffer); got != "payment_conflict" {
		t.Fatalf("release after payment: %s", got)
	}
	// Otro motorizado: enlace nuevo y la espera vuelve a contar desde el cambio.
	_, _, pedro := assign(southPanel, southOffer, "Pedro Ramos")
	if orders, _ = panel(southPanel); orders[0].PickupAlert != "" || orders[0].PickupWaitSeconds > 60 || orders[0].CourierToken != pedro {
		t.Fatalf("courier change resets the wait: %#v", orders)
	}
	if code, _ := courier(rosa); code != 404 {
		t.Fatalf("replaced courier link: %d", code)
	}
	if code, got := courierAct(pedro, "pickup"); code != 200 || got != "picked_up" {
		t.Fatalf("courier pickup: %d %s", code, got)
	}
	if got := act(southPanel, "release", southOffer); got != "already_picked_up" {
		t.Fatalf("release after pickup: %s", got)
	}
	if dispatch, order, _ := state(orderID); dispatch != "picked_up" || order != "en_camino" {
		t.Fatalf("after pickup: %s %s", dispatch, order)
	}
	// Preparado y en camino: nadie puede cancelarlo.
	if rec := call(api.updateOrderStatus, "PATCH", `{"status":"cancelado"}`, map[string]string{"id": orderID}, s); rec.Code != 409 || !strings.Contains(rec.Body.String(), "delivery_in_progress_not_cancellable") {
		t.Fatalf("cancel on the way: %d %s", rec.Code, rec.Body.String())
	}
	// Recojo y entrega son del repartidor: caja solo cobra.
	for _, status := range []string{"en_camino", "entregado"} {
		if rec := call(api.updateOrderStatus, "PATCH", fmt.Sprintf(`{"status":%q}`, status), map[string]string{"id": orderID}, s); rec.Code != 409 || !strings.Contains(rec.Body.String(), "delivery_handled_by_partner") {
			t.Fatalf("staff %s on a Fudia delivery: %d %s", status, rec.Code, rec.Body.String())
		}
	}

	// Ubicación del motorizado desde su enlace.
	locate := func(token, body string) int {
		return call(d.CourierLocation, "POST", body, map[string]string{"token": token}, scope{}).Code
	}
	if code := locate(pedro, `{"latitude":123,"longitude":-77}`); code != 400 {
		t.Fatalf("invalid location: %d", code)
	}
	if code := locate(rosa, `{"latitude":-12.12,"longitude":-77.03}`); code != 404 {
		t.Fatalf("location with a replaced link: %d", code)
	}
	if code := locate(pedro, `{"latitude":-12.1215,"longitude":-77.0299}`); code != 204 {
		t.Fatalf("location: %d", code)
	}

	// El cliente no aparece: «No pude entregar» avisa a la central y al chat del cliente.
	if _, got := courierAct(pedro, "not_delivered"); got != "issue_reported" {
		t.Fatalf("not delivered: %s", got)
	}
	if _, got := courierAct(pedro, "not_delivered"); got != "issue_reported" {
		t.Fatalf("repeat not delivered: %s", got)
	}
	contact := `SELECT count(*) FROM customer_notices WHERE order_id=$1 AND kind='delivery_contact_request'`
	if n := count(contact, orderID); n != 1 {
		t.Fatalf("customer contact notice: %d", n)
	}
	if orders, _ = panel(southPanel); orders[0].IssueReportedAt == "" || orders[0].FailAvailableInSeconds <= 0 {
		t.Fatalf("issue in panel: %#v", orders)
	}
	if got := act(southPanel, "fail", southOffer); got != "too_soon" {
		t.Fatalf("fail before the customer can answer: %s", got)
	}
	if _, err := pool.Exec(ctx, `UPDATE delivery_dispatches SET delivery_issue_reported_at=now()-interval '11 minutes' WHERE order_id=$1`, orderID); err != nil {
		t.Fatal(err)
	}
	if orders, _ = panel(southPanel); orders[0].FailAvailableInSeconds != 0 {
		t.Fatalf("fail must be available: %#v", orders)
	}
	if got := act(southPanel, "fail", southOffer); got != "delivery_failed" {
		t.Fatalf("fail: %s", got)
	}
	if got := act(southPanel, "fail", southOffer); got != "cancelled" {
		t.Fatalf("double fail: %s", got)
	}
	// El restaurante ya cobró la comida: su pedido se cierra; el reparto termina sin estados nuevos.
	if dispatch, order, total := state(orderID); dispatch != "cancelled" || order != "entregado" || total != "60.00" {
		t.Fatalf("after failed delivery: %s %s %s", dispatch, order, total)
	}
	if code, _ := courier(pedro); code != 404 {
		t.Fatalf("courier link after failed delivery: %d", code)
	}
	orders, failures := panel(southPanel)
	if len(orders) != 0 || len(failures) != 1 {
		t.Fatalf("failed delivery in panel: %#v %#v", orders, failures)
	}
	if items := board(s); len(items) != 1 || items[0].DispatchStatus != "cancelled" || !items[0].FailedDelivery {
		t.Fatalf("board after failed delivery: %#v", items)
	}
	if _, err := pool.Exec(ctx, `UPDATE orders SET created_at=now()-interval '30 days' WHERE id=$1`, orderID); err != nil {
		t.Fatal(err)
	}
	if rec := readDispatch(s); rec.Code != http.StatusOK {
		t.Fatalf("historical order dispatch: %d %s", rec.Code, rec.Body.String())
	} else {
		var detail struct{ FailedDelivery bool }
		decode(rec, &detail)
		if !detail.FailedDelivery {
			t.Fatalf("historical failed delivery: %#v", detail)
		}
	}
	partnersRec := call(d.LocationPartners, "GET", "", nil, s)
	var located struct {
		Enabled      bool
		DistrictName string
		Items        []struct{ Name, WhatsappPhone string }
	}
	decode(partnersRec, &located)
	if partnersRec.Code != 200 || !located.Enabled || located.DistrictName == "" || len(located.Items) != 2 {
		t.Fatalf("location partners: %d %s", partnersRec.Code, partnersRec.Body.String())
	}
	failedRec := call(d.LocationFailedDeliveries, "GET", "", nil, s)
	var localFailed struct {
		Items []struct{ OrderCode, PartnerName, FudiaShare string }
	}
	decode(failedRec, &localFailed)
	if failedRec.Code != 200 || len(localFailed.Items) != 1 || localFailed.Items[0].PartnerName != "Motos Sur" || localFailed.Items[0].FudiaShare != "30.00" {
		t.Fatalf("location failed deliveries: %d %s", failedRec.Code, failedRec.Body.String())
	}
	// Otro local de la misma empresa no ve el delivery ni sus incidencias.
	other := s
	other.LocationID = "00000000-0000-0000-0000-000000000000"
	if items := board(other); len(items) != 0 {
		t.Fatalf("board leaked to another location: %#v", items)
	}
	if rec := call(d.LocationFailedDeliveries, "GET", "", nil, other); rec.Code != 200 || strings.Contains(rec.Body.String(), "Motos Sur") {
		t.Fatalf("failed deliveries leaked: %d %s", rec.Code, rec.Body.String())
	}
	f := failures[0]
	if f.CourierName != "Pedro Ramos" || f.FoodAmount != "60.00" || f.CourierReimbursement != "60.00" ||
		f.PartnerShare != "30.00" || f.FudiaShare != "30.00" || !f.FudiaShareEligible || len(f.ValidationNotes) != 0 {
		t.Fatalf("failed delivery amounts: %#v", f)
	}
}
