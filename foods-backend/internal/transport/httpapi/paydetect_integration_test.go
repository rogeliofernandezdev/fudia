package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Flujo completo de Fudia Pay contra PostgreSQL real: canje del código del QR,
// reporte idempotente de notificaciones, revisión de Caja (asociar, confirmar,
// descartar), revocación y aislamiento entre restaurantes. Una notificación
// detectada nunca marca una venta pagada por sí sola: la confirmación exige
// un pago real registrado por el flujo de pagos.
func TestPayDetectPairingReportAndCashReview(t *testing.T) {
	pool := integrationPool(t)
	api := New(pool)
	s := seedInventoryScope(t, pool)
	other := seedInventoryScope(t, pool)
	handler := api.handlers.PayDetect

	t.Cleanup(func() {
		for _, statement := range []string{
			`DELETE FROM pay_pairing_codes WHERE organization_id=$1`,
			`DELETE FROM pay_devices WHERE organization_id=$1`,
		} {
			_, _ = pool.Exec(context.Background(), statement, s.OrganizationID)
			_, _ = pool.Exec(context.Background(), statement, other.OrganizationID)
		}
	})

	var deviceToken string
	call := func(method, path, id string, subject scope, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
		if id != "" {
			req.SetPathValue("id", id)
		}
		if subject.OrganizationID != "" {
			req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, subject))
		}
		if deviceToken != "" && strings.HasPrefix(path, "/v1/mobile/") {
			req.Header.Set("Authorization", "Bearer "+deviceToken)
		}
		rec := httptest.NewRecorder()
		switch {
		case path == "/v1/admin/pay-devices/pairing":
			handler.CreatePairing(rec, req)
		case path == "/v1/admin/pay-devices/pairing/status":
			handler.PairingStatus(rec, req)
		case path == "/v1/mobile/pair/scan":
			handler.ScanPairing(rec, req)
		case path == "/v1/mobile/pair":
			handler.Pair(rec, req)
		case path == "/v1/mobile/notifications":
			handler.DeviceAuth(http.HandlerFunc(handler.Report)).ServeHTTP(rec, req)
		case path == "/v1/mobile/session":
			handler.DeviceAuth(http.HandlerFunc(handler.DeviceSession)).ServeHTTP(rec, req)
		case path == "/v1/admin/pay-devices" && method == "GET":
			handler.Devices(rec, req)
		case path == "/v1/admin/pay-notifications":
			handler.Notifications(rec, req)
		default:
			if method == "PATCH" {
				handler.RenameDevice(rec, req)
				return rec
			}
			switch {
			case path == "/v1/admin/pay-devices/revoke":
				handler.RevokeDevice(rec, req)
			case path == "/v1/admin/pay-notifications/associate":
				handler.AssociateNotification(rec, req)
			default:
				handler.DismissNotification(rec, req)
			}
		}
		return rec
	}

	// 1. Caja genera el código temporal y el dispositivo lo canjea una sola vez.
	pairingRec := call("POST", "/v1/admin/pay-devices/pairing", "", s, "")
	if pairingRec.Code != 200 {
		t.Fatalf("create pairing: %d %s", pairingRec.Code, pairingRec.Body.String())
	}
	var draft struct {
		ID   string `json:"id"`
		Code string `json:"code"`
	}
	if err := json.Unmarshal(pairingRec.Body.Bytes(), &draft); err != nil {
		t.Fatal(err)
	}
	// Caja sigue el código como WhatsApp Web: esperando, escaneado y vinculado.
	pairingStatus := func(subject scope) (int, string, string, string) {
		t.Helper()
		rec := call("GET", "/v1/admin/pay-devices/pairing/status", draft.ID, subject, "")
		var out struct{ Status, Model, DeviceName string }
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out.Status, out.Model, out.DeviceName
	}
	if code, status, _, _ := pairingStatus(s); code != 200 || status != "pending" {
		t.Fatalf("pairing pending: %d %s", code, status)
	}
	if code, _, _, _ := pairingStatus(other); code != 400 {
		t.Fatalf("otro local no lee el código: %d", code)
	}
	if scan := call("POST", "/v1/mobile/pair/scan", "", scope{}, fmt.Sprintf(`{"code":%q,"model":"Samsung Galaxy A15"}`, draft.Code)); scan.Code != 204 {
		t.Fatalf("scan: %d %s", scan.Code, scan.Body.String())
	}
	if scan := call("POST", "/v1/mobile/pair/scan", "", scope{}, fmt.Sprintf(`{"code":%q,"model":"Otro"}`, draft.Code)); scan.Code != 204 {
		t.Fatalf("scan twice is idempotent: %d", scan.Code)
	}
	if code, status, model, _ := pairingStatus(s); code != 200 || status != "scanned" || model != "Samsung Galaxy A15" {
		t.Fatalf("pairing scanned: %d %s %q", code, status, model)
	}
	if bad := call("POST", "/v1/mobile/pair/scan", "", scope{}, `{"code":"abcdef"}`); bad.Code != 400 {
		t.Fatalf("scan with invalid code: %d", bad.Code)
	}
	pairRec := call("POST", "/v1/mobile/pair", "", scope{}, fmt.Sprintf(`{"code":%q,"platform":"ANDROID","model":"Samsung Galaxy A15"}`, draft.Code))
	if pairRec.Code != 200 {
		t.Fatalf("pair: %d %s", pairRec.Code, pairRec.Body.String())
	}
	var activation struct {
		Token            string `json:"token"`
		OrganizationName string `json:"organizationName"`
		LocationName     string `json:"locationName"`
		Device           struct {
			ID string `json:"id"`
		} `json:"device"`
	}
	if err := json.Unmarshal(pairRec.Body.Bytes(), &activation); err != nil {
		t.Fatal(err)
	}
	deviceToken = activation.Token
	if activation.Token == "" || activation.OrganizationName == "" || activation.LocationName == "" {
		t.Fatal("la activación no trajo credencial ni restaurante")
	}
	if code, status, _, name := pairingStatus(s); code != 200 || status != "linked" || name != "Samsung Galaxy A15" {
		t.Fatalf("pairing linked: %d %s %q", code, status, name)
	}
	if scan := call("POST", "/v1/mobile/pair/scan", "", scope{}, fmt.Sprintf(`{"code":%q}`, draft.Code)); scan.Code != 409 {
		t.Fatalf("scan a used code must be 409, got %d", scan.Code)
	}
	if reuse := call("POST", "/v1/mobile/pair", "", scope{}, fmt.Sprintf(`{"code":%q,"platform":"ANDROID"}`, draft.Code)); reuse.Code != 409 {
		t.Fatalf("reusar un código canjeado debe ser 409, fue %d", reuse.Code)
	}
	if invalid := call("POST", "/v1/mobile/pair", "", scope{}, `{"code":"abcdef","platform":"ANDROID"}`); invalid.Code != 400 {
		t.Fatalf("código inválido debe ser 400, fue %d", invalid.Code)
	}

	// 2. La sesión del dispositivo resuelve la tenencia desde la vinculación.
	sessionRec := call("GET", "/v1/mobile/session", "", scope{}, "")
	var session struct {
		Restaurant struct {
			Name string `json:"name"`
		} `json:"restaurant"`
	}
	if err := json.Unmarshal(sessionRec.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	if sessionRec.Code != 200 || session.Restaurant.Name == "" {
		t.Fatalf("device session: %d %s", sessionRec.Code, sessionRec.Body.String())
	}

	// 3. Reporte idempotente: el mismo evento vuelve como duplicate.
	report := func(events string) (int, int, int) {
		rec := call("POST", "/v1/mobile/notifications", "", scope{}, fmt.Sprintf(`{"events":[%s]}`, events))
		if rec.Code != 200 {
			t.Fatalf("report: %d %s", rec.Code, rec.Body.String())
		}
		var result struct {
			Registered int `json:"registered"`
			Duplicates int `json:"duplicates"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result.Registered, result.Duplicates, rec.Code
	}
	registered, _, _ := report(`{"provider":"yape","eventKey":"yape-op-9f81d2a","operationNumber":"0012345678","senderName":"Rosa Quispe","amount":62.5,"occurredAt":"2026-11-15T12:30:00Z"}`)
	if registered != 1 {
		t.Fatalf("el primer reporte debe registrar 1, fue %d", registered)
	}
	_, duplicates, _ := report(`{"provider":"yape","eventKey":"yape-op-9f81d2a","senderName":"Rosa Quispe","amount":62.5,"occurredAt":"2026-11-15T12:30:00Z"}`)
	if duplicates != 1 {
		t.Fatalf("el reenvío debe ser duplicado, fue %d", duplicates)
	}
	registeredTwo, _, _ := report(`{"provider":"plin","eventKey":"plin-op-b71c94e","senderName":"Ana Villanueva","amount":10,"occurredAt":"2026-11-15T12:35:00Z"}`)
	if registeredTwo != 1 {
		t.Fatal("un segundo pago legítimo con distinto eventKey debe registrarse")
	}
	var noticeCount int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM staff_notices WHERE organization_id=$1 AND location_id=$2 AND kind='pay_detected'`, s.OrganizationID, s.LocationID).Scan(&noticeCount); err != nil || noticeCount != 2 {
		t.Fatalf("each payment must publish one bell notice, including after duplicate report: count=%d err=%v", noticeCount, err)
	}
	// La corrección recupera los avisos perdidos, sin duplicarlos al reintentar.
	backfillSQL, err := os.ReadFile("../../../migrations/000136_pay_staff_notices.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(context.Background(), `DELETE FROM staff_notices WHERE organization_id=$1 AND kind='pay_detected'`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := tx.Exec(context.Background(), string(backfillSQL)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.QueryRow(context.Background(), `SELECT count(*) FROM staff_notices WHERE organization_id=$1 AND kind='pay_detected'`, s.OrganizationID).Scan(&noticeCount); err != nil || noticeCount != 2 {
		t.Fatalf("notice recovery must be idempotent: count=%d err=%v", noticeCount, err)
	}
	if err := tx.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ambiguous := call("POST", "/v1/mobile/notifications", "", scope{}, `{"events":[{"provider":"mercadopago","eventKey":"mp-12345678","amount":5,"occurredAt":"2026-11-15T12:36:00Z"}]}`); ambiguous.Code != 400 {
		t.Fatal("un proveedor desconocido no se registra")
	}

	// 4. Aislamiento: otro restaurante no ve las notificaciones ni puede ligarlas.
	if foreign := call("GET", "/v1/admin/pay-notifications", "", other, ""); foreign.Code != 200 {
		t.Fatalf("list foreign: %d", foreign.Code)
	} else {
		var page struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		}
		if err := json.Unmarshal(foreign.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if len(page.Items) != 0 {
			t.Fatal("un restaurante ajeno no debe ver pagos detectados ajenos")
		}
	}

	// 5. Caja lista, asocia a un pedido y confirma con un pago real registrado
	// con el flujo de pagos existente: el cajero tiene cash.manage, abre turno
	// y cobra por el mismo API.
	if _, err := pool.Exec(context.Background(), `
		WITH new_role AS (
			INSERT INTO roles(organization_id, name, permissions)
			VALUES($2, 'Cajero Fudia Pay', ARRAY['cash.manage','cash.read'])
			RETURNING id
		)
		INSERT INTO user_roles(user_id, role_id, location_id)
		SELECT $1, new_role.id, $3 FROM new_role
	`, s.UserID, s.OrganizationID, s.LocationID); err != nil {
		t.Fatalf("seed cajero: %v", err)
	}
	var registerID string
	for _, subject := range []scope{s, other} {
		request := httptest.NewRequest("GET", "/v1/admin/notices", nil)
		request = request.WithContext(context.WithValue(request.Context(), scopeKey{}, subject))
		response := httptest.NewRecorder()
		api.handlers.Notices.Feed(response, request)
		var feed noticeFeed
		if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &feed) != nil {
			t.Fatalf("payment bell feed: %d %s", response.Code, response.Body.String())
		}
		want := 0
		if subject.OrganizationID == s.OrganizationID {
			want = 2
		}
		if len(feed.Items) != want || feed.Unread != want {
			t.Fatalf("payment bell isolation: %+v, want %d", feed, want)
		}
	}
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO cash_registers(organization_id,location_id,name,created_by)
		VALUES($1,$2,'Caja Fudia Pay',$3) RETURNING id::text`,
		s.OrganizationID, s.LocationID, s.UserID).Scan(&registerID); err != nil {
		t.Fatalf("seed register: %v", err)
	}
	shiftRec := httptest.NewRecorder()
	shiftReq := httptest.NewRequest("POST", "/v1/admin/cash-shifts", bytes.NewReader([]byte(fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":0}`, registerID))))
	shiftReq = shiftReq.WithContext(context.WithValue(shiftReq.Context(), scopeKey{}, s))
	api.openCashShift(shiftRec, shiftReq)
	if shiftRec.Code != 201 {
		t.Fatalf("cash shift: %d %s", shiftRec.Code, shiftRec.Body.String())
	}
	var orderID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO orders(organization_id,location_id,code,channel,status,customer_name,total,created_by,created_at)
		VALUES($1,$2,'PED-PAYTEST','delivery','entregado','Rosa Quispe',62.5,$3,now()) RETURNING id::text`,
		s.OrganizationID, s.LocationID, s.UserID).Scan(&orderID); err != nil {
		t.Fatalf("seed order: %v", err)
	}
	var notificationID string
	if list := call("GET", "/v1/admin/pay-notifications", "", s, ""); list.Code != 200 {
		t.Fatalf("list: %d", list.Code)
	} else {
		var page struct {
			Items []struct {
				ID              string `json:"id"`
				Status          string `json:"status"`
				Provider        string `json:"provider"`
				OperationNumber string `json:"operationNumber"`
			} `json:"items"`
			Total int `json:"total"`
		}
		if err := json.Unmarshal(list.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if len(page.Items) != 2 || page.Total != 2 {
			t.Fatalf("el local debe ver 2 notificaciones, vio %d (total %d)", len(page.Items), page.Total)
		}
		for _, item := range page.Items {
			if item.Provider == "yape" {
				notificationID = item.ID
				if item.OperationNumber != "0012345678" {
					t.Fatalf("operation number lost: %q", item.OperationNumber)
				}
			} else if item.OperationNumber != "" {
				t.Fatal("a legacy event must not invent an operation number")
			}
		}
		if notificationID == "" {
			t.Fatal("Yape notification missing")
		}
	}
	// La lista de Caja pagina en el servidor: page y size, con el total de los filtros.
	pageOf := func(query string) (int, int) {
		t.Helper()
		req := httptest.NewRequest("GET", "/v1/admin/pay-notifications?"+query, nil)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		handler.Notifications(rec, req)
		var out struct {
			Items []json.RawMessage `json:"items"`
			Total int               `json:"total"`
		}
		if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &out) != nil {
			t.Fatalf("page %s: %d %s", query, rec.Code, rec.Body.String())
		}
		return len(out.Items), out.Total
	}
	if items, total := pageOf("page=1&size=10"); items != 2 || total != 2 {
		t.Fatalf("page 1: %d items, total %d", items, total)
	}
	if items, total := pageOf("page=2&size=10"); items != 0 || total != 2 {
		t.Fatalf("page 2 must be empty with the same total: %d items, total %d", items, total)
	}
	if empty := call("POST", "/v1/admin/pay-notifications/associate", notificationID, s, `{"orderId":""}`); empty.Code != 400 {
		t.Fatalf("asociar sin pedido debe ser 400, fue %d", empty.Code)
	}
	if missing := call("POST", "/v1/admin/pay-notifications/associate", notificationID, s, `{"orderId":"11111111-1111-4111-8111-111111111111"}`); missing.Code != 400 {
		t.Fatalf("un pedido inexistente no se puede ligar: %d", missing.Code)
	}
	for _, method := range []string{"card", "plin"} {
		wrong := call("POST", "/v1/admin/pay-notifications/associate", notificationID, s, fmt.Sprintf(`{"orderId":%q,"method":%q}`, orderID, method))
		if wrong.Code != 400 {
			t.Fatalf("changing the provider method must fail: %d %s", wrong.Code, wrong.Body.String())
		}
	}
	associatedRec := call("POST", "/v1/admin/pay-notifications/associate", notificationID, s, fmt.Sprintf(`{"orderId":%q,"method":"yape"}`, orderID))
	if associatedRec.Code != 200 {
		t.Fatalf("associate: %d %s", associatedRec.Code, associatedRec.Body.String())
	}
	var verified struct {
		Status    string `json:"status"`
		OrderID   string `json:"orderId"`
		PaymentID string `json:"paymentId"`
	}
	if err := json.Unmarshal(associatedRec.Body.Bytes(), &verified); err != nil {
		t.Fatal(err)
	}
	if verified.Status != "associated" || verified.OrderID != orderID || verified.PaymentID == "" {
		t.Fatalf("asociación incompleta: %+v", verified)
	}
	// La asociación registró el cobro real con el flujo de pagos existente.
	var paymentCount int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM payments WHERE order_id=$1 AND organization_id=$2`, orderID, s.OrganizationID).Scan(&paymentCount); err != nil {
		t.Fatal(err)
	}
	if paymentCount != 1 {
		t.Fatalf("la asociación debe registrar 1 cobro, hay %d", paymentCount)
	}
	var paymentMethod, paymentReference string
	if err := pool.QueryRow(context.Background(), `SELECT method, reference FROM payments WHERE id=$1`, verified.PaymentID).Scan(&paymentMethod, &paymentReference); err != nil {
		t.Fatal(err)
	}
	if paymentMethod != "yape" || paymentReference != "Fudia Pay · Yape · Operación 0012345678" {
		t.Fatalf("payment provenance: %q %q", paymentMethod, paymentReference)
	}
	againRec := call("POST", "/v1/admin/pay-notifications/associate", notificationID, s, fmt.Sprintf(`{"orderId":%q,"method":"card"}`, orderID))
	if againRec.Code != 409 {
		t.Fatalf("asociar el mismo movimiento dos veces debe ser 409, fue %d", againRec.Code)
	}

	// 6. Revocación: la credencial del dispositivo deja de servir al instante.
	var devices struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if list := call("GET", "/v1/admin/pay-devices", "", s, ""); list.Code != 200 {
		t.Fatalf("devices: %d", list.Code)
	} else if err := json.Unmarshal(list.Body.Bytes(), &devices); err != nil {
		t.Fatal(err)
	}
	if len(devices.Items) != 1 {
		t.Fatalf("el local debe ver 1 dispositivo, vio %d", len(devices.Items))
	}
	if revoked := call("POST", "/v1/admin/pay-devices/revoke", devices.Items[0].ID, s, ""); revoked.Code != 200 {
		t.Fatalf("revoke: %d", revoked.Code)
	}
	if blocked := call("POST", "/v1/mobile/notifications", "", scope{}, `{"events":[{"provider":"yape","eventKey":"yape-op-dead12","amount":5,"occurredAt":"2026-11-15T13:00:00Z"}]}`); blocked.Code != 401 {
		t.Fatalf("un dispositivo revocado debe recibir 401, fue %d", blocked.Code)
	}
}
