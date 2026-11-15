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

	deliverypg "github.com/foods-platform/foods-backend/internal/modules/delivery/infrastructure/postgres"
)

// Panel web del proveedor de Delivery Fudia: todas las centrales ven la oferta
// sin WhatsApp cuando cocina inicia la preparación; la primera que acepta se la
// queda, recoge y entrega; su ubicación llega al seguimiento del cliente solo
// mientras el reparto sigue en curso.
func TestDeliveryPartnerPanel(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE locations SET district_code='150101',latitude=-12.046374,longitude=-77.042793 WHERE id=$1`, s.LocationID); err != nil {
		t.Fatal(err)
	}
	prefix := fmt.Sprintf("518%08d", time.Now().UnixNano()%100000000)
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
	email := func(index int) string { return fmt.Sprintf("op-%s-%d@panel.test", prefix, index) }
	partner := func(name string, index int) (id, panel string) {
		rec := call(d.CreatePartner, "POST", fmt.Sprintf(`{"name":%q,"whatsappPhone":"%s%d","coverageCodes":["150101"]}`, name, prefix, index), nil, s)
		if rec.Code != 201 {
			t.Fatalf("create partner: %d %s", rec.Code, rec.Body.String())
		}
		var out struct{ ID string }
		decode(rec, &out)
		return out.ID, operatorSession(t, d, s, out.ID, email(index))
	}
	firstID, firstPanel := partner("Reparto Norte", 1)
	_, secondPanel := partner("Reparto Sur", 2)
	if firstPanel == secondPanel {
		t.Fatal("each operator needs its own session")
	}
	if _, err := pool.Exec(ctx, `UPDATE delivery_partners SET active=false WHERE whatsapp_phone NOT LIKE $1`, prefix+"%"); err != nil {
		t.Fatal(err)
	}
	if rec := call(d.SetOrganizationDelivery, "PUT", `{"enabled":true}`, map[string]string{"id": s.OrganizationID}, s); rec.Code != 200 {
		t.Fatalf("enable delivery: %d %s", rec.Code, rec.Body.String())
	}

	type panelOrder struct {
		Token, Stage, OrderCode, OrderStatus, CustomerName, Address, Subtotal, DeliveryFee, Total string
		PaymentConfirmed                                                                          bool
		ExpiresInSeconds                                                                          int
		LocationLatitude, Latitude                                                                *float64
		Items                                                                                     []struct {
			Quantity float64
			Name     string
			Note     string
		}
	}
	panel := func(token string) []panelOrder {
		t.Helper()
		rec := panelCall(d, d.Panel, "GET", "", token)
		if rec.Code != 200 {
			t.Fatalf("panel: %d %s", rec.Code, rec.Body.String())
		}
		var out struct {
			PartnerName string
			Active      bool
			Orders      []panelOrder
		}
		decode(rec, &out)
		if out.PartnerName == "" || !out.Active {
			t.Fatalf("panel header: %s", rec.Body.String())
		}
		return out.Orders
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
	status := func(orderID string) (dispatch, order string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT d.status,o.status FROM delivery_dispatches d JOIN orders o ON o.id=d.order_id WHERE o.id=$1`, orderID).Scan(&dispatch, &order); err != nil {
			t.Fatal(err)
		}
		return dispatch, order
	}

	// Sin sesión o con una sesión desconocida el panel exige iniciar sesión.
	for _, token := range []string{"", "nope", "00000000000000000000000000000000"} {
		if rec := panelCall(d, d.Panel, "GET", "", token); rec.Code != 401 {
			t.Fatalf("unknown session %q: %d %s", token, rec.Code, rec.Body.String())
		}
	}
	if orders := panel(firstPanel); len(orders) != 0 {
		t.Fatalf("empty panel expected: %#v", orders)
	}

	var productID string
	if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control) VALUES($1,'FUDIA-PANEL','Ceviche',30,true,'prepared','none') RETURNING id`, s.OrganizationID).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"channel":"delivery","customerName":"Ana","customerPhone":"999123456","address":"Av. Perú 100","reference":"Puerta azul","deliveryFee":5,"sendToKitchen":true,"items":[{"productId":%q,"qty":2,"note":"Sin ají"}]}`, productID)
	req := httptest.NewRequest("POST", "/v1/admin/orders", bytes.NewReader([]byte(body)))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.createOrder(rec, req)
	if rec.Code != 201 {
		t.Fatalf("create delivery: %d %s", rec.Code, rec.Body.String())
	}
	var created struct{ ID string }
	decode(rec, &created)
	orderID := created.ID
	if _, err := pool.Exec(ctx, `UPDATE orders SET delivery_latitude=-12.121900,delivery_longitude=-77.029700 WHERE id=$1`, orderID); err != nil {
		t.Fatal(err)
	}

	kitchen := func(status string) {
		t.Helper()
		if rec := call(api.updateKitchenTicketStatus, "PATCH", fmt.Sprintf(`{"status":%q}`, status), map[string]string{"id": orderID}, s); rec.Code != 204 {
			t.Fatalf("kitchen %s: %d %s", status, rec.Code, rec.Body.String())
		}
	}
	if orders := panel(firstPanel); len(orders) != 0 {
		t.Fatalf("offered before preparation: %#v", orders)
	}
	kitchen("preparando")
	// La oferta aparece en el panel sin WhatsApp, y verla inicia sus 3 minutos.
	orders := panel(firstPanel)
	if len(orders) != 1 || orders[0].Stage != "offer" || orders[0].CustomerName != "Ana" || orders[0].Total != "65.00" ||
		orders[0].Subtotal != "60.00" || orders[0].DeliveryFee != "5.00" || len(orders[0].Items) != 1 || orders[0].Items[0].Note != "Sin ají" ||
		orders[0].ExpiresInSeconds < 170 || orders[0].ExpiresInSeconds > 180 || orders[0].LocationLatitude == nil || orders[0].Latitude == nil {
		t.Fatalf("offer in panel: %#v", orders)
	}
	offer := orders[0].Token
	other := panel(secondPanel)
	if len(other) != 1 || other[0].Stage != "offer" || other[0].Token == offer {
		t.Fatalf("every partner receives its own offer: %#v", other)
	}
	// Otra central no puede responder la oferta aunque conozca su token.
	if got := act(secondPanel, "accept", offer); got != "not_found" {
		t.Fatalf("another partner accepted: %s", got)
	}
	if rec := panelCall(d, d.PanelAction, "POST", fmt.Sprintf(`{"action":"steal","token":%q}`, offer), firstPanel); rec.Code != 400 {
		t.Fatalf("unknown action: %d %s", rec.Code, rec.Body.String())
	}
	if got := act(firstPanel, "deliver", offer); got != "taken" {
		t.Fatalf("deliver before accepting: %s", got)
	}
	if got := act(firstPanel, "accept", offer); got != "accepted" {
		t.Fatalf("accept: %s", got)
	}
	if got := act(firstPanel, "accept", offer); got != "already_accepted" {
		t.Fatalf("double accept: %s", got)
	}
	// Aceptación exclusiva: la otra central deja de verla y ya no puede tomarla.
	if got := act(secondPanel, "accept", other[0].Token); got != "taken" {
		t.Fatalf("second partner took an accepted order: %s", got)
	}
	if late := panel(secondPanel); len(late) != 0 {
		t.Fatalf("accepted order still offered: %#v", late)
	}
	orders = panel(firstPanel)
	if len(orders) != 1 || orders[0].Stage != "accepted" || orders[0].Token != offer || orders[0].ExpiresInSeconds != 0 || orders[0].Total != "65.00" {
		t.Fatalf("accepted in panel: %#v", orders)
	}
	if got := act(firstPanel, "pickup", offer); got != "not_ready" {
		t.Fatalf("pickup before kitchen: %s", got)
	}
	if got := act(firstPanel, "deliver", offer); got != "not_picked_up" {
		t.Fatalf("deliver before pickup: %s", got)
	}

	// La ubicación (la comparte el motorizado desde su enlace) llega al seguimiento
	// del cliente mientras el reparto sigue en curso.
	if _, err := pool.Exec(ctx, `UPDATE delivery_dispatches SET courier_latitude=-12.06,courier_longitude=-77.03,courier_located_at=now() WHERE order_id=$1`, orderID); err != nil {
		t.Fatal(err)
	}
	tracking, found, err := api.dispatcher.DeliveryTracking(ctx, s, orderID)
	if err != nil || !found || tracking.DispatchStatus != "accepted" || tracking.PartnerName != "Reparto Norte" ||
		tracking.Origin == nil || tracking.Destination == nil || tracking.Courier == nil ||
		tracking.Courier.Latitude != -12.06 || tracking.Courier.Longitude != -77.03 || tracking.CourierLocatedAt == "" {
		t.Fatalf("tracking with courier: %#v %v %v", tracking, found, err)
	}
	// Otro local no puede leer el seguimiento.
	if _, found, _ := api.dispatcher.DeliveryTracking(ctx, scope{OrganizationID: s.OrganizationID, LocationID: "00000000-0000-0000-0000-000000000000"}, orderID); found {
		t.Fatal("tracking leaked to another location")
	}
	// Una ubicación antigua deja de mostrarse.
	if _, err := pool.Exec(ctx, `UPDATE delivery_dispatches SET courier_located_at=now()-interval '20 minutes' WHERE order_id=$1`, orderID); err != nil {
		t.Fatal(err)
	}
	if tracking, _, _ = api.dispatcher.DeliveryTracking(ctx, s, orderID); tracking.Courier != nil {
		t.Fatalf("stale location shown: %#v", tracking.Courier)
	}

	kitchen("listo")
	if orders = panel(firstPanel); len(orders) != 1 || orders[0].OrderStatus != "listo" || orders[0].PaymentConfirmed {
		t.Fatalf("ready in panel: %#v", orders)
	}
	assertPaymentBlocked := func() {
		t.Helper()
		if orders := panel(firstPanel); len(orders) != 1 || orders[0].PaymentConfirmed {
			t.Fatalf("payment must still be pending: %#v", orders)
		}
		if got := act(firstPanel, "pickup", offer); got != "payment_pending" {
			t.Fatalf("pickup without full payment: %s", got)
		}
		if dispatch, order := status(orderID); dispatch != "accepted" || order != "listo" {
			t.Fatalf("blocked pickup changed state: %s %s", dispatch, order)
		}
	}
	assertPaymentBlocked()
	// Caja registra pagos reales: ningún abono parcial permite recoger.
	grantCashManagementForPaymentTest(t, api, s)
	var registerID string
	if err := pool.QueryRow(ctx, `INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,'Caja de reparto',$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&registerID); err != nil {
		t.Fatal(err)
	}
	if rec := call(api.openCashShift, "POST", fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":0}`, registerID), nil, s); rec.Code != 201 {
		t.Fatalf("open cash: %d %s", rec.Code, rec.Body.String())
	}
	pay := func(amount int) string {
		t.Helper()
		rec := call(api.createPayment, "POST", fmt.Sprintf(`{"orderId":%q,"method":"card","amount":%d}`, orderID, amount), nil, s)
		if rec.Code != 201 {
			t.Fatalf("confirm payment: %d %s", rec.Code, rec.Body.String())
		}
		var payment struct{ ID string }
		decode(rec, &payment)
		return payment.ID
	}
	pay(20)
	assertPaymentBlocked()
	paymentID := pay(40)
	if orders := panel(firstPanel); len(orders) != 1 || !orders[0].PaymentConfirmed {
		t.Fatalf("full payment must enable pickup: %#v", orders)
	}
	// Una devolución previa a recoger vuelve a dejar saldo pendiente.
	if rec := call(api.refundPayment, "POST", `{"amount":15,"reason":"Corrección de cobro"}`, map[string]string{"id": paymentID}, s); rec.Code != 204 {
		t.Fatalf("refund: %d %s", rec.Code, rec.Body.String())
	}
	assertPaymentBlocked()
	pay(15)
	if orders := panel(firstPanel); len(orders) != 1 || !orders[0].PaymentConfirmed {
		t.Fatalf("payment restored: %#v", orders)
	}
	if got := act(firstPanel, "pickup", offer); got != "picked_up" {
		t.Fatalf("pickup: %s", got)
	}
	if dispatch, order := status(orderID); dispatch != "picked_up" || order != "en_camino" {
		t.Fatalf("after pickup: %s %s", dispatch, order)
	}
	if orders = panel(firstPanel); len(orders) != 1 || orders[0].Stage != "picked_up" {
		t.Fatalf("picked up in panel: %#v", orders)
	}
	if got := act(firstPanel, "deliver", offer); got != "delivered" {
		t.Fatalf("deliver: %s", got)
	}
	if got := act(firstPanel, "deliver", offer); got != "delivered" {
		t.Fatalf("double deliver: %s", got)
	}
	if dispatch, order := status(orderID); dispatch != "delivered" || order != "entregado" {
		t.Fatalf("after delivery: %s %s", dispatch, order)
	}
	// El envío sigue con la central después de entregar: no vuelve al pedido del local.
	if err := api.dispatcher.Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	var total, fee string
	if err := pool.QueryRow(ctx, `SELECT total::text,delivery_fee::text FROM orders WHERE id=$1`, orderID).Scan(&total, &fee); err != nil || total != "60.00" || fee != "0.00" {
		t.Fatalf("fee after delivery: %s %s %v", total, fee, err)
	}
	if orders = panel(firstPanel); len(orders) != 1 || orders[0].Stage != "delivered" {
		t.Fatalf("delivered in panel: %#v", orders)
	}
	tracking, found, err = api.dispatcher.DeliveryTracking(ctx, s, orderID)
	if err != nil || !found || tracking.DispatchStatus != "delivered" || tracking.Courier != nil || tracking.DeliveredAt == "" || tracking.PickedUpAt == "" {
		t.Fatalf("tracking after delivery hides the courier: %#v %v %v", tracking, found, err)
	}
	history := call(d.OrderDispatch, "GET", "", map[string]string{"id": orderID}, s)
	var dispatch struct{ Status, DeliveredAt string }
	decode(history, &dispatch)
	if history.Code != 200 || dispatch.Status != "delivered" || dispatch.DeliveredAt == "" {
		t.Fatalf("dispatch read model: %d %s", history.Code, history.Body.String())
	}

	// Cada acción del panel queda registrada con el operador que la hizo.
	var recorded int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM delivery_partner_operator_actions a JOIN delivery_partner_operators o ON o.id=a.operator_id WHERE o.email=$1 AND a.action='accept'`, email(1)).Scan(&recorded); err != nil || recorded == 0 {
		t.Fatalf("operator actions not recorded: %d %v", recorded, err)
	}
	operatorAccessRules(t, d, s, firstID, prefix)

	// La limpieza borra sesiones vencidas o revocadas hace más de 30 días y conserva la vigente.
	if _, err := pool.Exec(ctx, `
		INSERT INTO delivery_partner_sessions(token_hash,operator_id,expires_at,revoked_at)
		SELECT decode(md5(o.id::text||'old'),'hex'),o.id,now()-interval '40 days',NULL FROM delivery_partner_operators o WHERE o.email=$1
		UNION ALL
		SELECT decode(md5(o.id::text||'revoked'),'hex'),o.id,now()+interval '1 hour',now()-interval '31 days' FROM delivery_partner_operators o WHERE o.email=$1`, email(1)); err != nil {
		t.Fatal(err)
	}
	if purged, err := deliverypg.New(pool).PurgeOperatorSessions(ctx, time.Now().AddDate(0, 0, -30)); err != nil || purged < 2 {
		t.Fatalf("purge sessions: %d %v", purged, err)
	}
	if rec := panelCall(d, d.Panel, "GET", "", firstPanel); rec.Code != 200 {
		t.Fatalf("purge removed a live session: %d", rec.Code)
	}
}

type operatorAdministration interface {
	operatorAccess
	ListOperators(http.ResponseWriter, *http.Request)
	UpdateOperator(http.ResponseWriter, *http.Request)
	ResetOperatorPassword(http.ResponseWriter, *http.Request)
	PartnerLogout(http.ResponseWriter, *http.Request)
	PartnerSession(http.ResponseWriter, *http.Request)
	OperatorAuth(http.Handler) http.Handler
	Panel(http.ResponseWriter, *http.Request)
}

// operatorAccessRules cubre el ciclo del acceso: contraseña temporal obligatoria,
// correo único, bloqueo por intentos, cierre de sesión, desactivación y restablecimiento.
func operatorAccessRules(t *testing.T, d operatorAdministration, s scope, partnerID, prefix string) {
	t.Helper()
	call := func(handler func(http.ResponseWriter, *http.Request), method, body string, path map[string]string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		handler(rec, operatorRequest(method, body, path, s, ""))
		return rec
	}
	decode := func(rec *httptest.ResponseRecorder, out any) {
		t.Helper()
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("decode %s: %v", rec.Body.String(), err)
		}
	}
	mail := fmt.Sprintf(" Turno-%s@Panel.test ", prefix)
	normalized := strings.ToLower(strings.TrimSpace(mail))
	operatorID, temporary := createOperator(t, d, s, partnerID, "Lucía Ramos", mail)
	if rec := call(d.CreateOperator, "POST", fmt.Sprintf(`{"fullName":"Otra persona","email":%q}`, normalized), map[string]string{"id": partnerID}); rec.Code != 409 {
		t.Fatalf("duplicate operator email: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(d.CreateOperator, "POST", `{"fullName":"X","email":"no-es-correo"}`, map[string]string{"id": partnerID}); rec.Code != 400 {
		t.Fatalf("invalid operator: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(d.CreateOperator, "POST", `{"fullName":"Ana Pérez","email":"ana@panel.test"}`, map[string]string{"id": "00000000-0000-0000-0000-000000000000"}); rec.Code != 404 {
		t.Fatalf("operator of unknown partner: %d %s", rec.Code, rec.Body.String())
	}
	listed := call(d.ListOperators, "GET", "", map[string]string{"id": partnerID})
	var list struct {
		Items []struct {
			ID, Email          string
			MustChangePassword bool
		}
	}
	decode(listed, &list)
	found := false
	for _, item := range list.Items {
		found = found || (item.ID == operatorID && item.Email == normalized && item.MustChangePassword)
	}
	if listed.Code != 200 || !found || strings.Contains(listed.Body.String(), "assword_hash") || strings.Contains(listed.Body.String(), temporary) {
		t.Fatalf("operators list: %d %s", listed.Code, listed.Body.String())
	}

	// La contraseña temporal entra, pero el panel exige cambiarla antes de operar.
	login := operatorLogin(d, strings.ToUpper(normalized), temporary)
	session := sessionCookieOf(login)
	if login.Code != 200 || session == "" || !strings.Contains(login.Body.String(), `"mustChangePassword":true`) {
		t.Fatalf("temporary login: %d %s", login.Code, login.Body.String())
	}
	if rec := panelCall(d, d.Panel, "GET", "", session); rec.Code != 403 || !strings.Contains(rec.Body.String(), "password_change_required") {
		t.Fatalf("panel with temporary password: %d %s", rec.Code, rec.Body.String())
	}
	if rec := changeOperatorPassword(d, session, "incorrecta", "clave-nueva-1"); rec.Code != 400 {
		t.Fatalf("wrong current password: %d %s", rec.Code, rec.Body.String())
	}
	if rec := changeOperatorPassword(d, session, temporary, "corta"); rec.Code != 400 {
		t.Fatalf("short password: %d %s", rec.Code, rec.Body.String())
	}
	other := sessionCookieOf(operatorLogin(d, normalized, temporary))
	if rec := changeOperatorPassword(d, session, temporary, "clave-nueva-1"); rec.Code != 200 {
		t.Fatalf("change password: %d %s", rec.Code, rec.Body.String())
	}
	if rec := panelCall(d, d.Panel, "GET", "", session); rec.Code != 200 {
		t.Fatalf("panel after password change: %d %s", rec.Code, rec.Body.String())
	}
	if rec := panelCall(d, d.Panel, "GET", "", other); rec.Code != 401 {
		t.Fatalf("changing the password closes the other sessions: %d", rec.Code)
	}
	if rec := operatorLogin(d, normalized, temporary); rec.Code != 401 {
		t.Fatalf("old temporary password still works: %d", rec.Code)
	}

	// Cinco intentos fallidos seguidos bloquean incluso la contraseña correcta
	// (el intento con la temporal ya vencida fue el primero).
	for range 4 {
		if rec := operatorLogin(d, normalized, "no-es-la-clave"); rec.Code != 401 {
			t.Fatalf("wrong password: %d %s", rec.Code, rec.Body.String())
		}
	}
	if rec := operatorLogin(d, normalized, "clave-nueva-1"); rec.Code != 403 || !strings.Contains(rec.Body.String(), "delivery_operator_locked") {
		t.Fatalf("locked operator: %d %s", rec.Code, rec.Body.String())
	}
	if rec := operatorLogin(d, "nadie@panel.test", "clave-nueva-1"); rec.Code != 401 {
		t.Fatalf("unknown email: %d", rec.Code)
	}

	// Restablecer da otra contraseña temporal, quita el bloqueo y cierra sus sesiones.
	reset := call(d.ResetOperatorPassword, "POST", "", map[string]string{"id": partnerID, "operatorId": operatorID})
	var credentials struct{ TemporaryPassword string }
	decode(reset, &credentials)
	if reset.Code != 200 || len(credentials.TemporaryPassword) != 12 || credentials.TemporaryPassword == temporary {
		t.Fatalf("reset password: %d %s", reset.Code, reset.Body.String())
	}
	if rec := panelCall(d, d.Panel, "GET", "", session); rec.Code != 401 {
		t.Fatalf("reset keeps old session: %d", rec.Code)
	}
	login = operatorLogin(d, normalized, credentials.TemporaryPassword)
	session = sessionCookieOf(login)
	if login.Code != 200 || !strings.Contains(login.Body.String(), `"mustChangePassword":true`) {
		t.Fatalf("login after reset: %d %s", login.Code, login.Body.String())
	}
	if rec := call(d.ResetOperatorPassword, "POST", "", map[string]string{"id": "00000000-0000-0000-0000-000000000000", "operatorId": operatorID}); rec.Code != 404 {
		t.Fatalf("reset with another partner: %d", rec.Code)
	}

	// Cerrar sesión la revoca.
	logout := httptest.NewRecorder()
	d.PartnerLogout(logout, operatorRequest("POST", "", nil, scope{}, session))
	if logout.Code != 204 {
		t.Fatalf("logout: %d", logout.Code)
	}
	me := httptest.NewRecorder()
	d.OperatorAuth(http.HandlerFunc(d.PartnerSession)).ServeHTTP(me, operatorRequest("GET", "", nil, scope{}, session))
	if me.Code != 401 {
		t.Fatalf("session after logout: %d", me.Code)
	}

	// Desactivar cierra sus sesiones y le impide entrar.
	session = sessionCookieOf(operatorLogin(d, normalized, credentials.TemporaryPassword))
	if rec := call(d.UpdateOperator, "PATCH", fmt.Sprintf(`{"fullName":"Lucía Ramos","email":%q,"active":false}`, normalized), map[string]string{"id": partnerID, "operatorId": operatorID}); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"active":false`) {
		t.Fatalf("deactivate operator: %d %s", rec.Code, rec.Body.String())
	}
	if rec := panelCall(d, d.Panel, "GET", "", session); rec.Code != 401 {
		t.Fatalf("inactive operator keeps the session: %d", rec.Code)
	}
	if rec := operatorLogin(d, normalized, credentials.TemporaryPassword); rec.Code != 403 || !strings.Contains(rec.Body.String(), "delivery_operator_inactive") {
		t.Fatalf("inactive operator login: %d %s", rec.Code, rec.Body.String())
	}
}
