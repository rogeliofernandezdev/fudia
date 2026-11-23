package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestSalesDateFiltersUseLocalDaysAndPagination(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE locations SET timezone='America/Lima' WHERE id=$1 AND organization_id=$2`, s.LocationID, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	// Lima 7 October runs from 05:00Z inclusive until 05:00Z on 8 October exclusive.
	for _, item := range []struct{ code, at string }{
		{"BEFORE", "2026-10-07T04:59:59Z"},
		{"START", "2026-10-07T05:00:00Z"},
		{"MIDDLE", "2026-10-07T17:00:00Z"},
		{"END", "2026-10-08T04:59:59.999999Z"},
		{"AFTER", "2026-10-08T05:00:00Z"},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO orders(organization_id,location_id,code,channel,status,total,created_by,created_at)
			VALUES($1,$2,$3,'mostrador','listo',0,$4,$5::timestamptz)`, s.OrganizationID, s.LocationID, item.code, s.UserID, item.at); err != nil {
			t.Fatal(err)
		}
	}
	type response struct {
		Items       []posOrderSummary `json:"items"`
		Total       int               `json:"total"`
		AmountTotal string            `json:"amountTotal"`
	}
	list := func(query string) response {
		t.Helper()
		r := httptest.NewRequest("GET", "/v1/admin/pos/orders?paymentStatus=paid&"+query, nil)
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
		w := httptest.NewRecorder()
		api.listPOSOrders(w, r)
		if w.Code != 200 {
			t.Fatalf("list: %d %s", w.Code, w.Body.String())
		}
		var out response
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	for _, tc := range []struct {
		query string
		codes []string
	}{
		{"from=2026-10-07&to=2026-10-07", []string{"END", "MIDDLE", "START"}},
		{"from=2026-10-07", []string{"AFTER", "END", "MIDDLE", "START"}},
		{"to=2026-10-07", []string{"END", "MIDDLE", "START", "BEFORE"}},
		{"", []string{"AFTER", "END", "MIDDLE", "START", "BEFORE"}},
		{"from=2026-10-09&to=2026-10-09", []string{}},
		{"from=2026-10-07&to=2026-10-07&q=MIDDLE", []string{"MIDDLE"}},
		{"from=" + url.QueryEscape(" 2026-10-07 ") + "&to=2026-10-07", []string{"END", "MIDDLE", "START"}},
	} {
		t.Run(tc.query, func(t *testing.T) {
			out := list(tc.query)
			codes := []string{}
			for _, item := range out.Items {
				codes = append(codes, item.Code)
			}
			if out.Total != len(tc.codes) || !reflect.DeepEqual(codes, tc.codes) {
				t.Fatalf("want %v, got %#v", tc.codes, out)
			}
		})
	}
	out := list("from=2026-10-07&to=2026-10-07&pageSize=1&page=2")
	if out.AmountTotal == "" {
		t.Fatal("the period total covers the whole filter, not only the visible page")
	}
	if out.Total != 3 || len(out.Items) != 1 || out.Items[0].Code != "MIDDLE" {
		t.Fatalf("filtered pagination: %#v", out)
	}
	// The bounds come from the location, not a fixed Peru/UTC offset.
	if _, err := pool.Exec(ctx, `UPDATE locations SET timezone='UTC' WHERE id=$1`, s.LocationID); err != nil {
		t.Fatal(err)
	}
	out = list("from=2026-10-07&to=2026-10-07")
	if out.Total != 3 || out.Items[0].Code != "MIDDLE" || out.Items[2].Code != "BEFORE" {
		t.Fatalf("UTC bounds: %#v", out)
	}
	// A daylight-saving day has 23 hours: do not add 24 hours to a UTC bound.
	if _, err := pool.Exec(ctx, `UPDATE locations SET timezone='America/New_York' WHERE id=$1`, s.LocationID); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ code, at string }{
		{"BEFORE", "2026-03-08T04:59:59Z"},
		{"START", "2026-03-08T05:00:00Z"},
		{"MIDDLE", "2026-03-08T17:00:00Z"},
		{"END", "2026-03-09T03:59:59.999999Z"},
		{"AFTER", "2026-03-09T04:00:00Z"},
	} {
		if _, err := pool.Exec(ctx, `UPDATE orders SET created_at=$1::timestamptz WHERE code=$2 AND organization_id=$3`, item.at, item.code, s.OrganizationID); err != nil {
			t.Fatal(err)
		}
	}
	out = list("from=2026-03-08&to=2026-03-08")
	if out.Total != 3 || out.Items[0].Code != "END" || out.Items[2].Code != "START" {
		t.Fatalf("daylight-saving bounds: %#v", out)
	}
}

// Ventas fecha cada pedido por su último cobro, como el Dashboard y Caja: un
// pedido registrado de noche y cobrado al día siguiente es venta del día del cobro.
func TestSalesDateFiltersUseLastPaymentDay(t *testing.T) {
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
	id := func(query string, args ...any) string {
		t.Helper()
		var result string
		if err := pool.QueryRow(ctx, query, args...).Scan(&result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	exec(`UPDATE locations SET timezone='America/Lima' WHERE id=$1 AND organization_id=$2`, s.LocationID, s.OrganizationID)
	register := id(`INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,'Caja',$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID)
	shift := id(`INSERT INTO cash_shifts(organization_id,location_id,cash_register_id,opened_by,opening_amount,business_date) VALUES($1,$2,$3,$4,0,'2026-11-10') RETURNING id`, s.OrganizationID, s.LocationID, register, s.UserID)
	order := func(code, createdAt string) string {
		return id(`INSERT INTO orders(organization_id,location_id,code,channel,status,total,created_by,created_at)
			VALUES($1,$2,$3,'mostrador','entregado',20,$4,$5::timestamptz) RETURNING id`, s.OrganizationID, s.LocationID, code, s.UserID, createdAt)
	}
	pay := func(orderID, amount, at string) {
		exec(`INSERT INTO payments(organization_id,location_id,order_id,shift_id,method,amount,created_by,created_at)
			VALUES($1,$2,$3,$4,'cash',$5::numeric,$6,$7::timestamptz)`, s.OrganizationID, s.LocationID, orderID, shift, amount, s.UserID, at)
	}
	// Lima: 10/11 23:44 registrado, 11/11 12:24 cobrado.
	pay(order("NIGHT", "2026-11-11T04:44:00Z"), "20", "2026-11-11T17:24:00Z")
	// 11/11 12:24 registrado, 12:36 cobrado.
	pay(order("NOON", "2026-11-11T17:24:00Z"), "20", "2026-11-11T17:36:00Z")
	// Pago parcial el 10/11 y saldo el 11/11: cuenta una vez, el día en que se terminó de pagar.
	split := order("SPLIT", "2026-11-10T20:00:00Z")
	pay(split, "5", "2026-11-10T20:05:00Z")
	pay(split, "15", "2026-11-11T16:00:00Z")
	// Registrado y cobrado el 10/11.
	pay(order("BEFORE", "2026-11-10T16:50:00Z"), "20", "2026-11-10T16:51:00Z")
	// Pedido sin cobros (total cero): usa su fecha de registro.
	exec(`INSERT INTO orders(organization_id,location_id,code,channel,status,total,created_by,created_at)
		VALUES($1,$2,'FREE','mostrador','entregado',0,$3,'2026-11-11T15:00:00Z')`, s.OrganizationID, s.LocationID, s.UserID)

	type response struct {
		Items       []posOrderSummary `json:"items"`
		Total       int               `json:"total"`
		AmountTotal string            `json:"amountTotal"`
	}
	list := func(query string) response {
		t.Helper()
		r := httptest.NewRequest("GET", "/v1/admin/pos/orders?paymentStatus=paid&"+query, nil)
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
		w := httptest.NewRecorder()
		api.listPOSOrders(w, r)
		if w.Code != 200 {
			t.Fatalf("list: %d %s", w.Code, w.Body.String())
		}
		var out response
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	codes := func(out response) []string {
		result := []string{}
		for _, item := range out.Items {
			result = append(result, item.Code)
		}
		return result
	}
	day := list("from=2026-11-11&to=2026-11-11")
	if want := []string{"NOON", "NIGHT", "SPLIT", "FREE"}; !reflect.DeepEqual(codes(day), want) || day.Total != 4 || day.AmountTotal != "60.00" {
		t.Fatalf("11/11 by last payment, newest payment first: want %v, got %v total=%d amount=%s", want, codes(day), day.Total, day.AmountTotal)
	}
	if paidAt := day.Items[1].PaidAt; paidAt == nil || *paidAt != "2026-11-11T17:24:00+00" {
		t.Fatalf("NIGHT paidAt is its last payment: %v", paidAt)
	}
	if day.Items[3].PaidAt != nil {
		t.Fatalf("an order without payments has no paidAt: %v", *day.Items[3].PaidAt)
	}
	previous := list("from=2026-11-10&to=2026-11-10")
	if want := []string{"BEFORE"}; !reflect.DeepEqual(codes(previous), want) || previous.AmountTotal != "20.00" {
		t.Fatalf("10/11 keeps only what finished paying that day: %v amount=%s", codes(previous), previous.AmountTotal)
	}
	if all := list(""); !reflect.DeepEqual(codes(all), []string{"NOON", "NIGHT", "SPLIT", "FREE", "BEFORE"}) {
		t.Fatalf("without a range, newest payment first: %v", codes(all))
	}
}
