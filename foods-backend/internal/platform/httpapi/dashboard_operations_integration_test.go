package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type dashboardTestResponse struct {
	SalesNet          string              `json:"salesNet"`
	PaidOrders        int                 `json:"paidOrders"`
	AverageTicket     string              `json:"averageTicket"`
	OpenOrders        int                 `json:"openOrders"`
	ReservationsToday int                 `json:"reservationsToday"`
	BusinessDate      string              `json:"businessDate"`
	Operations        dashboardOperations `json:"operations"`
	HourlySales       []map[string]any    `json:"hourlySales"`
	TopProducts       []map[string]any    `json:"topProducts"`
	dashboardInsights
}

func readDashboardTest(t *testing.T, pool *pgxpool.Pool, s scope) dashboardTestResponse {
	t.Helper()
	return readDashboardPeriodTest(t, pool, s, "")
}

func readDashboardPeriodTest(t *testing.T, pool *pgxpool.Pool, s scope, period string) dashboardTestResponse {
	t.Helper()
	target := "/v1/admin/dashboard"
	if period != "" {
		target += "?period=" + period
	}
	req := httptest.NewRequest("GET", target, nil)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	New(pool).dashboard(rec, req)
	if rec.Code != 200 {
		t.Fatalf("dashboard: %d %s", rec.Code, rec.Body.String())
	}
	var result dashboardTestResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestDashboardConsolidatesRestaurantOperations(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	foreign := seedInventoryScope(t, pool)
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

	empty := readDashboardTest(t, pool, s)
	if empty.PaidOrders != 0 || empty.OpenOrders != 0 || empty.Operations.PendingBalance != "0" || empty.HourlySales == nil || empty.TopProducts == nil || empty.Trend.Points == nil || empty.Trend.Granularity != "hour" || empty.Period.Key != "today" || empty.Previous.SalesNet != "0" || empty.SalesByChannel == nil || empty.SalesByPaymentMethod == nil || empty.SalesByCategory == nil {
		t.Fatalf("empty dashboard: %+v", empty)
	}
	register := id(`INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,'Caja',$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID)
	shift := id(`INSERT INTO cash_shifts(organization_id,location_id,cash_register_id,opened_by,opening_amount,business_date) SELECT $1,$2,$3,$4,40,(now() AT TIME ZONE timezone)::date FROM locations WHERE id=$2 RETURNING id`, s.OrganizationID, s.LocationID, register, s.UserID)
	exec(`INSERT INTO cash_movements(organization_id,location_id,shift_id,movement_type,amount,reason,created_by) VALUES($1,$2,$3,'income',15,'Ingreso',$4),($1,$2,$3,'expense',5,'Salida',$4)`, s.OrganizationID, s.LocationID, shift, s.UserID)
	table := id(`INSERT INTO tables(organization_id,location_id,name,qr_token) VALUES($1,$2,'01',gen_random_uuid()::text) RETURNING id`, s.OrganizationID, s.LocationID)
	exec(`INSERT INTO tables(organization_id,location_id,name,qr_token) VALUES($1,$2,'02',gen_random_uuid()::text)`, s.OrganizationID, s.LocationID)
	order := func(channel, status string, total int, tableID any) string {
		return id(`INSERT INTO orders(organization_id,location_id,channel,status,total,table_id,created_by) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, s.OrganizationID, s.LocationID, channel, status, total, tableID, s.UserID)
	}
	full := order("mostrador", "entregado", 50, nil)
	exec(`UPDATE orders SET completed_at=now() WHERE id=$1`, full)
	partial := order("mostrador", "preparando", 50, nil)
	order("salon", "confirmado", 30, table)
	order("delivery", "listo", 10, nil)
	order("delivery", "en_camino", 20, nil)
	order("mostrador", "cancelado", 999, nil)
	closed := order("mostrador", "entregado", 999, nil)
	exec(`UPDATE orders SET completed_at=now() WHERE id=$1`, closed)
	exec(`INSERT INTO payments(organization_id,location_id,order_id,shift_id,method,amount,created_by) VALUES($1,$2,$3,$4,'card',50,$5),($1,$2,$6,$4,'card',20,$5)`, s.OrganizationID, s.LocationID, full, shift, s.UserID, partial)
	product := id(`INSERT INTO products(organization_id,sku,name,price) VALUES($1,'ALM','Almuerzo',25) RETURNING id`, s.OrganizationID)
	exec(`INSERT INTO order_items(organization_id,order_id,product_id,name,qty,unit_price) VALUES($1,$2,$3,'Almuerzo',2,25)`, s.OrganizationID, full, product)
	exec(`INSERT INTO orders(organization_id,location_id,channel,status,total,created_by) VALUES($1,$2,'delivery','preparando',999,$3)`, foreign.OrganizationID, foreign.LocationID, foreign.UserID)
	otherLocation := id(`INSERT INTO locations(organization_id,name,code) VALUES($1,'Otro','OTHER') RETURNING id`, s.OrganizationID)
	exec(`INSERT INTO orders(organization_id,location_id,channel,status,total,created_by) VALUES($1,$2,'delivery','preparando',999,$3)`, s.OrganizationID, otherLocation, s.UserID)
	exec(`INSERT INTO reservations(organization_id,location_id,customer_name,starts_at,guests,status) VALUES($1,$2,'Ana',now(),2,'confirmed'),($1,$2,'Luis',now()+interval '2 days',2,'pending'),($1,$2,'Eva',now(),2,'cancelled')`, s.OrganizationID, s.LocationID)
	result := readDashboardTest(t, pool, s)
	op := result.Operations
	if result.SalesNet != "70.00" || result.PaidOrders != 1 || result.AverageTicket != "50.0000000000000000" {
		t.Fatalf("financial indicators count partial payments as sales, not paid orders: %+v", result)
	}
	if op.PendingBalance != "90.00" || op.UnpaidOrders != 3 || op.PartialOrders != 1 || result.OpenOrders != 4 {
		t.Fatalf("pending orders: %+v", result)
	}
	if op.TablesTotal != 2 || op.TablesOccupied != 1 || op.KitchenConfirmed != 1 || op.KitchenPreparing != 1 || op.ReadyOrders != 1 || op.DeliveryPending != 1 || op.DeliveryInTransit != 1 || result.ReservationsToday != 1 {
		t.Fatalf("operational counts: %+v", result)
	}
	if op.OpenCashShifts != 1 || op.ActiveCashRegisters != 1 || op.CashBalance == nil || *op.CashBalance != "50.00" {
		t.Fatalf("cash must not include card payments: %+v", op)
	}
	if len(result.TopProducts) != 1 || result.TopProducts[0]["name"] != "Almuerzo" {
		t.Fatalf("paid products: %+v", result.TopProducts)
	}
	if len(result.Trend.Points) != 1 || result.Trend.Points[0].Current != "70.00" || result.Trend.Points[0].Previous != "0" || result.Trend.Points[0].Orders != 2 {
		t.Fatalf("hourly trend must compare today with the same day last week: %+v", result.Trend)
	}
	if result.Period.From != result.BusinessDate || result.Period.To != result.BusinessDate || result.Previous.SalesNet != "0" || result.Previous.PaidOrders != 0 {
		t.Fatalf("today period and previous window: %+v %+v", result.Period, result.Previous)
	}
	if len(result.SalesByChannel) != 1 || result.SalesByChannel[0].Value != "mostrador" || result.SalesByChannel[0].Label != "Mostrador" || result.SalesByChannel[0].Total != "70.00" || result.SalesByChannel[0].Count != 2 {
		t.Fatalf("sales by channel: %+v", result.SalesByChannel)
	}
	if len(result.SalesByPaymentMethod) != 1 || result.SalesByPaymentMethod[0].Value != "card" || result.SalesByPaymentMethod[0].Total != "70.00" || result.SalesByPaymentMethod[0].Label == "" {
		t.Fatalf("sales by payment method: %+v", result.SalesByPaymentMethod)
	}
	if len(result.SalesByCategory) != 1 || result.SalesByCategory[0].Name != "Sin categoría" || result.SalesByCategory[0].Revenue != "50.00" {
		t.Fatalf("sales by category only counts fully paid orders: %+v", result.SalesByCategory)
	}

	// Un cobro de hace tres días entra en 7 días pero no en hoy.
	older := order("salon", "entregado", 40, nil)
	exec(`UPDATE orders SET completed_at=now() WHERE id=$1`, older)
	exec(`INSERT INTO payments(organization_id,location_id,order_id,shift_id,method,amount,created_by,created_at) VALUES($1,$2,$3,$4,'card',40,$5,now()-interval '3 days')`, s.OrganizationID, s.LocationID, older, shift, s.UserID)
	week := readDashboardPeriodTest(t, pool, s, "7d")
	if week.Period.Key != "7d" || week.Period.Days != 7 || week.Trend.Granularity != "day" || len(week.Trend.Points) != 7 || week.SalesNet != "110.00" || week.PaidOrders != 2 {
		t.Fatalf("7-day period: %+v", week)
	}
	if week.Trend.Points[6].Current != "70.00" || week.Trend.Points[6].Key != week.BusinessDate {
		t.Fatalf("7-day trend must end today: %+v", week.Trend.Points)
	}
	if todayOnly := readDashboardTest(t, pool, s); todayOnly.SalesNet != "70.00" {
		t.Fatalf("older payments must not leak into today: %s", todayOnly.SalesNet)
	}
	req := httptest.NewRequest("GET", "/v1/admin/dashboard?period=year", nil)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	New(pool).dashboard(rec, req)
	if rec.Code != 400 {
		t.Fatalf("unknown period must be rejected, got %d", rec.Code)
	}

	exec(`UPDATE cash_registers SET blind_close=true WHERE id=$1`, register)
	if readDashboardTest(t, pool, s).Operations.CashBalance != nil {
		t.Fatal("blind cash leaked expected amount")
	}
	role := id(`INSERT INTO roles(organization_id,name,permissions) VALUES($1,'Control de caja',ARRAY['cash.expected.read']) RETURNING id`, s.OrganizationID)
	exec(`INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, s.UserID, role, s.LocationID)
	if readDashboardTest(t, pool, s).Operations.CashBalance == nil {
		t.Fatal("authorized cash amount hidden")
	}

	exec(`UPDATE locations SET timezone='Pacific/Kiritimati' WHERE id=$1`, s.LocationID)
	var expectedDate string
	if err := pool.QueryRow(ctx, `SELECT (now() AT TIME ZONE timezone)::date::text FROM locations WHERE id=$1`, s.LocationID).Scan(&expectedDate); err != nil {
		t.Fatal(err)
	}
	if readDashboardTest(t, pool, s).BusinessDate != expectedDate {
		t.Fatal("business date must use selected location timezone")
	}
}

func TestDashboardAvailabilityMatchesProductControlsAndCombos(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	id := func(query string, args ...any) string {
		t.Helper()
		var result string
		if err := pool.QueryRow(ctx, query, args...).Scan(&result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM menu_combo_options WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM menu_combo_groups WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM menu_combos WHERE organization_id=$1`, s.OrganizationID)
	})
	portion := id(`INSERT INTO products(organization_id,sku,name,price,quantity_control) VALUES($1,'ENT','Entrada',10,'portions') RETURNING id`, s.OrganizationID)
	manual := id(`INSERT INTO products(organization_id,sku,name,price) VALUES($1,'SEG','Segundo',10) RETURNING id`, s.OrganizationID)
	exec(`INSERT INTO products(organization_id,sku,name,price,quantity_control) VALUES($1,'INV','Inventario',10,'inventory')`, s.OrganizationID)
	exec(`INSERT INTO product_availability(organization_id,location_id,product_id,business_date,portion_quantity,sold_quantity,manual_status) SELECT $1,$2,$3,(now() AT TIME ZONE timezone)::date,10,10,'available' FROM locations WHERE id=$2`, s.OrganizationID, s.LocationID, portion)
	exec(`INSERT INTO product_availability(organization_id,location_id,product_id,business_date,manual_status) SELECT $1,$2,$3,(now() AT TIME ZONE timezone)::date,'sold_out' FROM locations WHERE id=$2`, s.OrganizationID, s.LocationID, manual)
	exec(`INSERT INTO products(organization_id,sku,name,price,available_from,quantity_control) VALUES($1,'MAN','Mañana',10,now()+interval '2 days','portions')`, s.OrganizationID)
	exec(`INSERT INTO products(organization_id,sku,name,price,active,quantity_control) VALUES($1,'INA','Inactivo',10,false,'portions')`, s.OrganizationID)
	combo := id(`INSERT INTO products(organization_id,sku,name,price) VALUES($1,'MEN','Menú',15) RETURNING id`, s.OrganizationID)
	exec(`INSERT INTO menu_combos(organization_id,product_id) VALUES($1,$2)`, s.OrganizationID, combo)
	group := id(`INSERT INTO menu_combo_groups(organization_id,combo_product_id,name,required,min_selections,max_selections) VALUES($1,$2,'Entrada',true,1,1) RETURNING id`, s.OrganizationID, combo)
	exec(`INSERT INTO menu_combo_options(organization_id,group_id,option_product_id) VALUES($1,$2,$3)`, s.OrganizationID, group, portion)
	if result := readDashboardTest(t, pool, s).Operations.SoldOutProducts; result != 4 {
		t.Fatalf("expected 4 sold-out products, got %d", result)
	}
	exec(`UPDATE product_availability SET sold_quantity=0 WHERE product_id=$1`, portion)
	if result := readDashboardTest(t, pool, s).Operations.SoldOutProducts; result != 2 {
		t.Fatalf("replenished portion and combo must be available, got %d", result)
	}
}
