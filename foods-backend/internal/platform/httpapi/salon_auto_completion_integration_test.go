package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestSalonAutomaticCompletion(t *testing.T) {
	for _, scenario := range []string{"delivery_then_single", "delivery_then_batch", "premature_single", "premature_batch", "concurrent"} {
		t.Run(scenario, func(t *testing.T) {
			pool := integrationPool(t)
			s := seedInventoryScope(t, pool)
			api := New(pool)
			ctx := context.Background()
			request := func(handler http.HandlerFunc, method, path, body, id string) *httptest.ResponseRecorder {
				r := httptest.NewRequest(method, path, strings.NewReader(body))
				r.SetPathValue("id", id)
				r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
				w := httptest.NewRecorder()
				handler(w, r)
				return w
			}
			var registerID, tableID, orderID string
			if err := pool.QueryRow(ctx, `INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,'Auto cierre',$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&registerID); err != nil {
				t.Fatal(err)
			}
			w := request(api.openCashShift, "POST", "/v1/admin/cash-shifts", fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":0}`, registerID), "")
			if w.Code != 201 {
				t.Fatalf("open shift: %d %s", w.Code, w.Body.String())
			}
			if err := pool.QueryRow(ctx, `INSERT INTO tables(organization_id,location_id,name,seats,zone,qr_token) VALUES($1,$2,'Mesa auto',4,'Principal',encode(gen_random_bytes(16),'hex')) RETURNING id`, s.OrganizationID, s.LocationID).Scan(&tableID); err != nil {
				t.Fatal(err)
			}
			if err := pool.QueryRow(ctx, `INSERT INTO orders(organization_id,location_id,code,channel,status,total,table_id,created_by) VALUES($1,$2,'PED-AUTO','salon','listo',25,$3,$4) RETURNING id`, s.OrganizationID, s.LocationID, tableID, s.UserID).Scan(&orderID); err != nil {
				t.Fatal(err)
			}
			closeBill := request(api.closeOrderBill, "POST", "/v1/admin/orders/"+orderID+"/bill/close", "", orderID)
			if closeBill.Code != 409 || !strings.Contains(closeBill.Body.String(), "products_not_delivered") {
				t.Fatalf("close bill: %d %s", closeBill.Code, closeBill.Body.String())
			}
			assertClosed := func(want bool) {
				t.Helper()
				var closed bool
				if err := pool.QueryRow(ctx, `SELECT completed_at IS NOT NULL FROM orders WHERE id=$1`, orderID).Scan(&closed); err != nil {
					t.Fatal(err)
				}
				if closed != want {
					t.Fatalf("closed=%v, want %v", closed, want)
				}
			}
			deliver := func() *httptest.ResponseRecorder {
				return request(api.updateOrderStatus, "PATCH", "/v1/admin/orders/"+orderID+"/status", `{"status":"entregado"}`, orderID)
			}
			pay := func(amount int) *httptest.ResponseRecorder {
				if strings.Contains(scenario, "batch") {
					return request(api.createPaymentBatch, "POST", "/v1/admin/payments/batch", fmt.Sprintf(`{"orderId":%q,"payments":[{"method":"cash","amount":%d},{"method":"card","amount":%d}]}`, orderID, amount-1, 1), "")
				}
				return request(api.createPayment, "POST", "/v1/admin/payments", fmt.Sprintf(`{"orderId":%q,"method":"card","amount":%d}`, orderID, amount), "")
			}
			check := func(w *httptest.ResponseRecorder, status int) {
				t.Helper()
				if w.Code != status {
					t.Fatalf("response: %d want %d, %s", w.Code, status, w.Body.String())
				}
			}
			if strings.HasPrefix(scenario, "premature") {
				// A historical prematurely closed bill still cannot be charged.
				if _, err := pool.Exec(ctx, `UPDATE orders SET bill_closed_at=now() WHERE id=$1`, orderID); err != nil {
					t.Fatal(err)
				}
				check(pay(25), 409)
				assertClosed(false)
				var payments int
				if err := pool.QueryRow(ctx, `SELECT count(*) FROM payments WHERE order_id=$1`, orderID).Scan(&payments); err != nil || payments != 0 {
					t.Fatalf("premature payment must not persist: %d %v", payments, err)
				}
				if _, err := pool.Exec(ctx, `UPDATE orders SET bill_closed_at=NULL WHERE id=$1`, orderID); err != nil {
					t.Fatal(err)
				}
			}
			check(deliver(), 200)
			assertClosed(false)
			if scenario == "concurrent" {
				var responses [2]*httptest.ResponseRecorder
				var wg sync.WaitGroup
				wg.Add(2)
				go func() { defer wg.Done(); responses[0] = request(api.closeOrderBill, "POST", "/", "", orderID) }()
				go func() { defer wg.Done(); responses[1] = pay(25) }()
				wg.Wait()
				check(responses[0], 200)
				if responses[1].Code == 409 {
					check(pay(25), 201) // payment ran before the close acquired its lock
				} else {
					check(responses[1], 201)
				}
			} else {
				check(request(api.closeOrderBill, "POST", "/", "", orderID), 200)
				check(pay(10), 201)
				assertClosed(false)
				check(pay(15), 201)
			}
			assertClosed(true)
			var completedEvents int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE organization_id=$1 AND entity_id=$2 AND action='order.completed'`, s.OrganizationID, orderID).Scan(&completedEvents); err != nil {
				t.Fatal(err)
			}
			if completedEvents != 1 {
				t.Fatalf("completion must be recorded once, got %d", completedEvents)
			}
			check(pay(2), 409)
			// The same table may receive a new order immediately after closure.
			if _, err := pool.Exec(ctx, `INSERT INTO orders(organization_id,location_id,code,channel,status,table_id,created_by) VALUES($1,$2,'PED-NEXT','salon','nuevo',$3,$4)`, s.OrganizationID, s.LocationID, tableID, s.UserID); err != nil {
				t.Fatalf("table must be reusable: %v", err)
			}
		})
	}
}
