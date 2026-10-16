package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serviceSafetyCall(t *testing.T, s scope, id, body string, handler http.HandlerFunc, want int) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("PATCH", "/", strings.NewReader(body))
	r.SetPathValue("id", id)
	r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
	w := httptest.NewRecorder()
	handler(w, r)
	if w.Code != want {
		t.Fatalf("%s: got %d want %d: %s", id, w.Code, want, w.Body.String())
	}
	return w
}

func TestKitchenLegacyFallbackCannotMutateOtherStations(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = seedOrganizationRoles(ctx, tx, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) SELECT $1,id,$2 FROM roles WHERE organization_id=$3 AND system_key='cook'`, s.UserID, s.LocationID, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	token := "service-safety-" + s.UserID
	hash := sha256.Sum256([]byte(token))
	if _, err = pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,organization_id,location_id,expires_at) VALUES($1,$2,$3,$4,now()+interval '1 hour')`, hash[:], s.UserID, s.OrganizationID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, hash[:]) })
	routes := api.Routes()
	for _, destination := range []string{"bar", "direct", "kitchen"} {
		for _, round := range []bool{false, true} {
			for _, legacy := range []bool{false, true} {
				if legacy && destination != "kitchen" {
					continue
				}
				t.Run(fmt.Sprintf("%s/round=%t/legacy=%t", destination, round, legacy), func(t *testing.T) {
					var product string
					if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price,product_type,quantity_control,service_destination) VALUES($1,encode(gen_random_bytes(8),'hex'),'Prueba estación',10,'prepared','none',$2) RETURNING id`, s.OrganizationID, destination).Scan(&product); err != nil {
						t.Fatal(err)
					}
					w := serviceSafetyCall(t, s, "", fmt.Sprintf(`{"channel":"mostrador","sendToKitchen":true,"items":[{"productId":%q,"qty":1}]}`, product), api.createOrder, 201)
					var o order
					if err := json.Unmarshal(w.Body.Bytes(), &o); err != nil {
						t.Fatal(err)
					}
					id := o.ID
					if round {
						tx, err := pool.Begin(ctx)
						if err != nil {
							t.Fatal(err)
						}
						defer tx.Rollback(ctx)
						if err = ensureKitchenRoundForOrder(ctx, tx, s, o.ID, "confirmado"); err != nil {
							t.Fatal(err)
						}
						if err = tx.QueryRow(ctx, `SELECT id FROM order_kitchen_rounds WHERE order_id=$1`, o.ID).Scan(&id); err != nil {
							t.Fatal(err)
						}
						if err = tx.Commit(ctx); err != nil {
							t.Fatal(err)
						}
					}
					if legacy {
						if _, err := pool.Exec(ctx, `DELETE FROM order_service_items WHERE order_id=$1`, o.ID); err != nil {
							t.Fatal(err)
						}
					}
					patch := func(ticket string, want int) {
						t.Helper()
						r := httptest.NewRequest("PATCH", "/v1/admin/kitchen/tickets/"+ticket+"/status", strings.NewReader(`{"status":"preparando"}`))
						r.Header.Set("Authorization", "Bearer "+token)
						w := httptest.NewRecorder()
						routes.ServeHTTP(w, r)
						if w.Code != want {
							t.Fatalf("%s: got %d want %d: %s", ticket, w.Code, want, w.Body.String())
						}
					}
					if destination == "bar" {
						patch(id+"~bar", 403)
					}
					want := 404
					if destination == "kitchen" {
						want = 204
					}
					patch(id, want)
					var status string
					if err := pool.QueryRow(ctx, `SELECT status FROM orders WHERE id=$1`, o.ID).Scan(&status); err != nil {
						t.Fatal(err)
					}
					if want == 404 && status != o.Status {
						t.Fatalf("wrong station changed aggregate to %s", status)
					}
					if want == 204 && status != "preparando" {
						t.Fatalf("kitchen stopped working: %s", status)
					}
					if !legacy && want == 404 {
						// Delivered snapshots must not turn an old round into a legacy ticket.
						if _, err := pool.Exec(ctx, `UPDATE order_service_items SET status='entregado' WHERE order_id=$1`, o.ID); err != nil {
							t.Fatal(err)
						}
						patch(id, 404)
					}
				})
			}
		}
	}
}

func TestCancelUsesActualServiceProgress(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	for _, tc := range []struct {
		name, destination, taskStatus string
		want                          int
	}{
		{"unserved direct", "direct", "listo", 200},
		{"pending kitchen", "kitchen", "confirmado", 200},
		{"pending bar", "bar", "confirmado", 200},
		{"served direct", "direct", "entregado", 409},
		{"preparing kitchen", "kitchen", "preparando", 409},
		{"ready kitchen", "kitchen", "listo", 409},
		{"preparing bar", "bar", "preparando", 409},
		{"ready bar", "bar", "listo", 409},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var product, table string
			if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price,product_type,quantity_control,service_destination) VALUES($1,encode(gen_random_bytes(8),'hex'),'Prueba cancelación',10,'prepared','none',$2) RETURNING id`, s.OrganizationID, tc.destination).Scan(&product); err != nil {
				t.Fatal(err)
			}
			if err := pool.QueryRow(ctx, `INSERT INTO tables(organization_id,location_id,name,seats,qr_token) VALUES($1,$2,$3,4,encode(gen_random_bytes(16),'hex')) RETURNING id`, s.OrganizationID, s.LocationID, tc.name).Scan(&table); err != nil {
				t.Fatal(err)
			}
			w := serviceSafetyCall(t, s, "", fmt.Sprintf(`{"channel":"salon","tableId":%q,"sendToKitchen":true,"items":[{"productId":%q,"qty":1}]}`, table, product), api.createOrder, 201)
			var o order
			if err := json.Unmarshal(w.Body.Bytes(), &o); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `UPDATE order_service_items SET status=$2 WHERE order_id=$1`, o.ID, tc.taskStatus); err != nil {
				t.Fatal(err)
			}
			// Keep the aggregate stale to ensure the snapshots are authoritative.
			if tc.want == 409 {
				if _, err := pool.Exec(ctx, `UPDATE orders SET status='confirmado' WHERE id=$1`, o.ID); err != nil {
					t.Fatal(err)
				}
			}
			other := s
			other.UserID = "00000000-0000-0000-0000-000000000001"
			serviceSafetyCall(t, other, o.ID, `{"status":"cancelado"}`, api.updateOrderStatus, 403)
			other = s
			other.LocationID = "00000000-0000-0000-0000-000000000001"
			serviceSafetyCall(t, other, o.ID, `{"status":"cancelado"}`, api.updateOrderStatus, 404)
			serviceSafetyCall(t, s, o.ID, `{"status":"cancelado"}`, api.updateOrderStatus, tc.want)
			var status string
			if err := pool.QueryRow(ctx, `SELECT status FROM orders WHERE id=$1`, o.ID).Scan(&status); err != nil {
				t.Fatal(err)
			}
			if tc.want == 200 && status != "cancelado" {
				t.Fatal("cancellation not saved")
			}
			if tc.want == 409 && status != "confirmado" {
				t.Fatal("rejection modified order")
			}
			if tc.want == 200 {
				// A cancelled table can immediately accept a new account.
				serviceSafetyCall(t, s, "", fmt.Sprintf(`{"channel":"salon","tableId":%q,"sendToKitchen":true,"items":[{"productId":%q,"qty":1}]}`, table, product), api.createOrder, 201)
			}
		})
	}
}

func TestCancelDirectRestoresStockOnceAndPreservesGuards(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	category := seedRetailCategory(t, pool, s)
	serviceSafetyCall(t, s, "", fmt.Sprintf(`{"newProduct":{"name":"Gaseosa cancelación","price":"10","categoryId":%q},"quantity":12,"unit":"botella"}`, category), api.createInventoryEntry, 201)
	var product, register string
	if err := pool.QueryRow(ctx, `SELECT id FROM products WHERE organization_id=$1 AND name='Gaseosa cancelación'`, s.OrganizationID).Scan(&product); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,'Caja cancelación',$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&register); err != nil {
		t.Fatal(err)
	}
	serviceSafetyCall(t, s, "", fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":0}`, register), api.openCashShift, 201)
	for _, scenario := range []string{"unserved", "paid", "legacy ready", "completed", "in transit", "mixed pending", "previous delivery"} {
		t.Run(scenario, func(t *testing.T) {
			body := fmt.Sprintf(`{"channel":"mostrador","sendToKitchen":true,"items":[{"productId":%q,"qty":2}]}`, product)
			if scenario == "mixed pending" || scenario == "previous delivery" {
				var kitchen string
				if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price,product_type,quantity_control,service_destination) VALUES($1,encode(gen_random_bytes(8),'hex'),'Plato pendiente',10,'prepared','none','kitchen') RETURNING id`, s.OrganizationID).Scan(&kitchen); err != nil {
					t.Fatal(err)
				}
				body = fmt.Sprintf(`{"channel":"mostrador","sendToKitchen":true,"items":[{"productId":%q,"qty":1},{"productId":%q,"qty":1}]}`, product, kitchen)
			}
			balance := func() float64 {
				t.Helper()
				var qty float64
				if err := pool.QueryRow(ctx, `SELECT sb.quantity::float8 FROM stock_balances sb JOIN inventory_items ii ON ii.id=sb.inventory_item_id AND ii.organization_id=sb.organization_id WHERE sb.organization_id=$1 AND sb.location_id=$2 AND ii.product_id=$3`, s.OrganizationID, s.LocationID, product).Scan(&qty); err != nil {
					t.Fatal(err)
				}
				return qty
			}
			initial := balance()
			w := serviceSafetyCall(t, s, "", body, api.createOrder, 201)
			var o order
			if err := json.Unmarshal(w.Body.Bytes(), &o); err != nil {
				t.Fatal(err)
			}
			consumed := balance()
			if consumed >= initial {
				t.Fatal("stock was not consumed")
			}
			switch scenario {
			case "paid":
				serviceSafetyCall(t, s, "", fmt.Sprintf(`{"orderId":%q,"method":"card","amount":5}`, o.ID), api.createPayment, 201)
			case "legacy ready":
				if _, err := pool.Exec(ctx, `DELETE FROM order_service_items WHERE order_id=$1`, o.ID); err != nil {
					t.Fatal(err)
				}
			case "completed":
				if _, err := pool.Exec(ctx, `UPDATE orders SET completed_at=now() WHERE id=$1`, o.ID); err != nil {
					t.Fatal(err)
				}
			case "in transit":
				if _, err := pool.Exec(ctx, `UPDATE orders SET status='en_camino' WHERE id=$1`, o.ID); err != nil {
					t.Fatal(err)
				}
			case "previous delivery":
				if _, err := pool.Exec(ctx, `UPDATE order_service_items SET status='entregado' WHERE order_id=$1 AND destination='direct'`, o.ID); err != nil {
					t.Fatal(err)
				}
			}
			want := 409
			if scenario == "unserved" || scenario == "mixed pending" {
				want = 200
			}
			serviceSafetyCall(t, s, o.ID, `{"status":"cancelado"}`, api.updateOrderStatus, want)
			expected := consumed
			if want == 200 {
				expected = initial
			}
			if balance() != expected {
				t.Fatal("incorrect cancellation stock reversal")
			}
			serviceSafetyCall(t, s, o.ID, `{"status":"cancelado"}`, api.updateOrderStatus, 409)
			if balance() != expected {
				t.Fatal("retry changed stock")
			}
		})
	}
}
