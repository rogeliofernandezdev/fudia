package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/reporting/application"
	"github.com/foods-platform/foods-backend/internal/modules/reporting/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }
type scope = requestscope.Scope

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool} }

var _ application.Repository = (*Repository)(nil)

// localDayStartSQL es el instante en que empieza el día local (hoy - offset) del local $2.
// Comparar created_at con ese límite usa los índices (organization_id,location_id,created_at);
// convertir cada fila con (created_at AT TIME ZONE tz)::date obliga a leer todo el historial del local.
func localDayStartSQL(offset string) string {
	return `(SELECT (((now() AT TIME ZONE l.timezone)::date-(` + offset + `))::timestamp AT TIME ZONE l.timezone) FROM locations l WHERE l.id=$2 AND l.organization_id=$1)`
}

const netPaidSQL = `
	COALESCE((
	  SELECT sum(
	    p.amount
	    - COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)
	  )
	  FROM payments p
	  WHERE p.order_id=o.id AND p.organization_id=o.organization_id AND p.location_id=o.location_id
	),0)
`

var dashboardMovementsSQL = `
	WITH loc AS (
	  SELECT timezone,(now() AT TIME ZONE timezone)::date AS today
	  FROM locations
	  WHERE id=$2 AND organization_id=$1
	),
	movements AS (
	  SELECT (p.created_at AT TIME ZONE loc.timezone) AS local_at,p.amount,p.method,p.order_id,o.channel
	  FROM payments p
	  JOIN orders o ON o.id=p.order_id AND o.organization_id=p.organization_id
	  CROSS JOIN loc
	  WHERE p.organization_id=$1 AND p.location_id=$2
	    AND p.created_at>=` + localDayStartSQL("$3::int") + `
	  UNION ALL
	  SELECT (pr.created_at AT TIME ZONE loc.timezone),-pr.amount,p.method,p.order_id,o.channel
	  FROM payment_refunds pr
	  JOIN payments p ON p.id=pr.payment_id AND p.organization_id=pr.organization_id AND p.location_id=pr.location_id
	  JOIN orders o ON o.id=p.order_id AND o.organization_id=p.organization_id
	  CROSS JOIN loc
	  WHERE pr.organization_id=$1 AND pr.location_id=$2
	    AND pr.created_at>=` + localDayStartSQL("$3::int") + `
	)`
var dashboardPaidOrdersSQL = `
	SELECT o.id,o.total
	FROM orders o
	WHERE o.organization_id=$1 AND o.location_id=$2 AND o.status<>'cancelado'
	  AND (` + netPaidSQL + `) >= o.total
	  AND o.id IN (
	    SELECT window_payment.order_id FROM payments window_payment
	    WHERE window_payment.organization_id=$1 AND window_payment.location_id=$2
	      AND window_payment.created_at>=` + localDayStartSQL("$3::int") + `
	      AND window_payment.created_at<` + localDayStartSQL("$4::int-1") + `
	  )`

type dashboardOperations = domain.DashboardOperations
type dashboardPeriod = domain.DashboardPeriod
type dashboardPeriodView = domain.DashboardPeriodView
type dashboardPreviousView = domain.DashboardPreviousView
type dashboardTrendPoint = domain.DashboardTrendPoint
type dashboardTrend = domain.DashboardTrend
type dashboardBreakdown = domain.DashboardBreakdown
type dashboardCategorySale = domain.DashboardCategorySale
type dashboardInsights = domain.DashboardInsights

func loadDashboardOperations(ctx context.Context, tx database.Executor, s scope, canSeeExpected bool) (dashboardOperations, string, error) {
	var result dashboardOperations
	var businessDate string
	err := tx.QueryRow(ctx, `
		WITH loc AS (
		  SELECT timezone,(now() AT TIME ZONE timezone)::date AS day
		  FROM locations WHERE organization_id=$1 AND id=$2 AND active
		), active_orders AS (
		  SELECT o.*,(`+netPaidSQL+`) AS paid
		  FROM orders o WHERE o.organization_id=$1 AND o.location_id=$2
		    AND o.status<>'cancelado' AND o.completed_at IS NULL
		), cash AS (
		  SELECT cs.opening_amount + COALESCE((
		    SELECT sum(CASE cm.movement_type WHEN 'income' THEN cm.amount ELSE -cm.amount END)
		    FROM cash_movements cm
		    WHERE cm.organization_id=cs.organization_id AND cm.location_id=cs.location_id AND cm.shift_id=cs.id
		  ),0) AS balance,cr.blind_close
		  FROM cash_shifts cs
		  JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
		  WHERE cs.organization_id=$1 AND cs.location_id=$2 AND cs.status='open'
		), availability AS (
		  SELECT p.id,CASE
		    WHEN COALESCE(pa.manual_status,'available')='sold_out' THEN 'sold_out'
		    WHEN NOT ((p.available_from IS NULL OR now()>=p.available_from)
		      AND (p.available_until IS NULL OR now()<=p.available_until)
		      AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE loc.timezone)::integer=ANY(p.available_days))
		      AND (p.available_until_time IS NULL OR (now() AT TIME ZONE loc.timezone)::time<=p.available_until_time)) THEN 'unavailable'
		    WHEN p.quantity_control='portions' THEN CASE
		      WHEN pa.portion_quantity IS NULL OR pa.portion_quantity<=COALESCE(pa.sold_quantity,0) THEN 'sold_out'
		      WHEN pa.portion_quantity-COALESCE(pa.sold_quantity,0)<=3 THEN 'low' ELSE 'available' END
		    WHEN p.quantity_control='inventory' THEN CASE
		      WHEN COALESCE(sb.quantity,0)<=0 THEN 'sold_out'
		      WHEN sb.quantity<=3 OR (ii.minimum_stock>0 AND sb.quantity<=ii.minimum_stock) THEN 'low' ELSE 'available' END
		    ELSE 'available' END AS status
		  FROM products p CROSS JOIN loc
		  LEFT JOIN product_availability pa ON pa.product_id=p.id AND pa.organization_id=p.organization_id AND pa.location_id=$2 AND pa.business_date=loc.day
		  LEFT JOIN inventory_items ii ON ii.organization_id=p.organization_id AND ii.product_id=p.id AND ii.active
		  LEFT JOIN stock_balances sb ON sb.organization_id=p.organization_id AND sb.location_id=$2 AND sb.inventory_item_id=ii.id
		  WHERE p.organization_id=$1 AND p.active
		), blocked_combos AS (
		  SELECT DISTINCT g.combo_product_id FROM menu_combo_groups g
		  WHERE g.organization_id=$1 AND g.required AND (
		    SELECT count(*) FROM menu_combo_options opt
		    JOIN availability av ON av.id=opt.option_product_id AND av.status IN ('available','low')
		    WHERE opt.organization_id=g.organization_id AND opt.group_id=g.id
		  ) < GREATEST(g.min_selections,1)
		)
		SELECT loc.day::text,
		  COALESCE((SELECT sum(GREATEST(total-paid,0)) FROM active_orders),'0')::text,
		  (SELECT count(*) FROM active_orders WHERE total>paid AND paid<=0),
		  (SELECT count(*) FROM active_orders WHERE total>paid AND paid>0),
		  (SELECT count(*) FROM tables WHERE organization_id=$1 AND location_id=$2 AND active),
		  (SELECT count(*) FROM tables t WHERE t.organization_id=$1 AND t.location_id=$2 AND t.active
		    AND EXISTS(SELECT 1 FROM active_orders o WHERE o.channel='salon' AND o.table_id=t.id)),
		  (SELECT count(*) FROM active_orders WHERE status='confirmado'),
		  (SELECT count(*) FROM active_orders WHERE status='preparando'),
		  (SELECT count(*) FROM active_orders WHERE status='listo'),
		  (SELECT count(*) FROM active_orders WHERE channel='delivery' AND status IN ('nuevo','confirmado','preparando','listo')),
		  (SELECT count(*) FROM active_orders WHERE channel='delivery' AND status='en_camino'),
		  (SELECT count(*) FROM cash_registers WHERE organization_id=$1 AND location_id=$2 AND active),
		  (SELECT count(*) FROM cash),
		  CASE WHEN NOT $3 AND EXISTS(SELECT 1 FROM cash WHERE blind_close) THEN NULL ELSE COALESCE((SELECT sum(balance) FROM cash),0)::text END,
		  (SELECT count(DISTINCT av.id) FROM availability av WHERE av.status='sold_out'
		    OR (av.status='available' AND EXISTS(SELECT 1 FROM blocked_combos bc WHERE bc.combo_product_id=av.id)))
		FROM loc`, s.OrganizationID, s.LocationID, canSeeExpected).Scan(
		&businessDate, &result.PendingBalance, &result.UnpaidOrders, &result.PartialOrders,
		&result.TablesTotal, &result.TablesOccupied, &result.KitchenConfirmed, &result.KitchenPreparing,
		&result.ReadyOrders, &result.DeliveryPending, &result.DeliveryInTransit,
		&result.ActiveCashRegisters, &result.OpenCashShifts, &result.CashBalance, &result.SoldOutProducts,
	)
	return result, businessDate, err
}
func loadDashboardInsights(ctx context.Context, tx database.Executor, s scope, period dashboardPeriod, channelLabels map[string]string) (dashboardInsights, error) {
	out := dashboardInsights{
		Period:               dashboardPeriodView{Key: period.Key, Days: period.Days},
		Trend:                dashboardTrend{Granularity: "day", Points: []dashboardTrendPoint{}},
		SalesByChannel:       []dashboardBreakdown{},
		SalesByPaymentMethod: []dashboardBreakdown{},
		SalesByCategory:      []dashboardCategorySale{},
	}
	previousStart, previousEnd := period.PreviousOffsets()
	lookback := previousStart

	if err := tx.QueryRow(ctx, `
		SELECT to_char(today-($3::int-1),'YYYY-MM-DD'),to_char(today,'YYYY-MM-DD'),
		       to_char(today-$4::int,'YYYY-MM-DD'),to_char(today-$5::int,'YYYY-MM-DD')
		FROM (SELECT (now() AT TIME ZONE timezone)::date AS today FROM locations WHERE id=$2 AND organization_id=$1) loc
	`, s.OrganizationID, s.LocationID, period.Days, previousStart, previousEnd).Scan(
		&out.Period.From, &out.Period.To, &out.Previous.From, &out.Previous.To,
	); err != nil {
		return out, err
	}

	if err := tx.QueryRow(ctx, dashboardMovementsSQL+`
		SELECT COALESCE(sum(m.amount),0)::text
		FROM movements m CROSS JOIN loc
		WHERE m.local_at::date BETWEEN loc.today-$4::int AND loc.today-$5::int
	`, s.OrganizationID, s.LocationID, lookback, previousStart, previousEnd).Scan(&out.Previous.SalesNet); err != nil {
		return out, err
	}
	if err := tx.QueryRow(ctx, `
		SELECT count(*)::int,CASE WHEN count(*)>0 THEN (sum(total)/count(*))::text ELSE '0' END
		FROM (`+dashboardPaidOrdersSQL+`) paid
	`, s.OrganizationID, s.LocationID, previousStart, previousEnd).Scan(&out.Previous.PaidOrders, &out.Previous.AverageTicket); err != nil {
		return out, err
	}

	var rows pgx.Rows
	var err error
	if period.Days == 1 {
		out.Trend.Granularity = "hour"
		rows, err = tx.Query(ctx, dashboardMovementsSQL+`
			SELECT lpad(extract(hour FROM m.local_at)::int::text,2,'0'),
			       COALESCE(sum(m.amount) FILTER (WHERE m.local_at::date=loc.today),0)::text,
			       COALESCE(sum(m.amount) FILTER (WHERE m.local_at::date=loc.today-7),0)::text,
			       count(DISTINCT m.order_id) FILTER (WHERE m.amount>0 AND m.local_at::date=loc.today)::int
			FROM movements m CROSS JOIN loc
			WHERE m.local_at::date IN (loc.today,loc.today-7)
			GROUP BY 1
			ORDER BY 1
		`, s.OrganizationID, s.LocationID, lookback)
	} else {
		rows, err = tx.Query(ctx, dashboardMovementsSQL+`,
			days AS (
			  SELECT day::date AS day
			  FROM loc,generate_series(loc.today-($4::int-1),loc.today,interval '1 day') AS series(day)
			),
			daily AS (
			  SELECT m.local_at::date AS day,sum(m.amount) AS total,count(DISTINCT m.order_id) FILTER (WHERE m.amount>0) AS orders
			  FROM movements m
			  GROUP BY 1
			)
			SELECT to_char(days.day,'YYYY-MM-DD'),
			       COALESCE(current_day.total,0)::text,
			       COALESCE(previous_day.total,0)::text,
			       COALESCE(current_day.orders,0)::int
			FROM days
			LEFT JOIN daily current_day ON current_day.day=days.day
			LEFT JOIN daily previous_day ON previous_day.day=days.day-$4::int
			ORDER BY days.day
		`, s.OrganizationID, s.LocationID, lookback, period.Days)
	}
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var point dashboardTrendPoint
		if err = rows.Scan(&point.Key, &point.Current, &point.Previous, &point.Orders); err != nil {
			rows.Close()
			return out, err
		}
		out.Trend.Points = append(out.Trend.Points, point)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return out, err
	}

	breakdown := func(query string, labels map[string]string) ([]dashboardBreakdown, error) {
		items := []dashboardBreakdown{}
		rows, err := tx.Query(ctx, dashboardMovementsSQL+query, s.OrganizationID, s.LocationID, lookback, period.Days)
		if err != nil {
			return items, err
		}
		defer rows.Close()
		for rows.Next() {
			var item dashboardBreakdown
			if err = rows.Scan(&item.Value, &item.Label, &item.Total, &item.Count); err != nil {
				return items, err
			}
			if labels != nil && labels[item.Value] != "" {
				item.Label = labels[item.Value]
			}
			items = append(items, item)
		}
		return items, rows.Err()
	}
	if out.SalesByChannel, err = breakdown(`
		SELECT m.channel,m.channel,sum(m.amount)::text,count(*) FILTER (WHERE m.amount>0)::int
		FROM movements m CROSS JOIN loc
		WHERE m.local_at::date>loc.today-$4::int
		GROUP BY m.channel
		HAVING sum(m.amount)<>0
		ORDER BY sum(m.amount) DESC,m.channel
	`, channelLabels); err != nil {
		return out, err
	}
	if out.SalesByPaymentMethod, err = breakdown(`
		SELECT m.method,COALESCE(pm.name,m.method),sum(m.amount)::text,count(*) FILTER (WHERE m.amount>0)::int
		FROM movements m
		CROSS JOIN loc
		LEFT JOIN payment_methods pm ON pm.organization_id=$1 AND pm.code=m.method
		WHERE m.local_at::date>loc.today-$4::int
		GROUP BY m.method,pm.name
		HAVING sum(m.amount)<>0
		ORDER BY sum(m.amount) DESC,m.method
	`, nil); err != nil {
		return out, err
	}

	// Misma definición que los productos vendidos: pedidos pagados por completo con un cobro en el periodo.
	// Redondear solo el agregado de salida: la multiplicación conserva su precisión hasta sumar.
	rows, err = tx.Query(ctx, `
		SELECT COALESCE(mc.name,'Sin categoría'),sum(oi.qty)::text,round(sum(oi.qty*oi.unit_price),2)::text
		FROM order_items oi
		JOIN (`+dashboardPaidOrdersSQL+`) paid ON paid.id=oi.order_id
		LEFT JOIN products pr ON pr.id=oi.product_id AND pr.organization_id=oi.organization_id
		LEFT JOIN menu_categories mc ON mc.id=pr.category_id AND mc.organization_id=pr.organization_id
		WHERE oi.organization_id=$1
		GROUP BY COALESCE(mc.name,'Sin categoría')
		ORDER BY sum(oi.qty*oi.unit_price) DESC,1
	`, s.OrganizationID, s.LocationID, period.Days-1, 0)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item dashboardCategorySale
		if err = rows.Scan(&item.Name, &item.Qty, &item.Revenue); err != nil {
			rows.Close()
			return out, err
		}
		out.SalesByCategory = append(out.SalesByCategory, item)
	}
	rows.Close()
	return out, rows.Err()
}
func (r *Repository) Dashboard(ctx context.Context, s scope, period domain.DashboardPeriod, canSeeExpected bool, labels map[string]string) (domain.Dashboard, error) {
	tx := database.ExecutorFor(ctx, r.pool)
	// El resumen comparte el pool con la operación: una consulta degradada se corta en vez de retener la conexión.
	if _, err := tx.Exec(ctx, `SET LOCAL statement_timeout = '5s'`); err != nil {
		return domain.Dashboard{}, &application.ReportError{Stage: application.Summary, Cause: err}
	}
	var err error
	var salesNet, averageTicket string
	var paidOrders, openOrders, critical, purchases, reservationsToday, kitchenPending int

	err = tx.QueryRow(ctx, `
		WITH today_payments AS (
		  SELECT p.order_id,p.amount
		  FROM payments p
		  WHERE p.organization_id=$1 AND p.location_id=$2
		    AND p.created_at>=`+localDayStartSQL("$3::int-1")+` AND p.created_at<`+localDayStartSQL("-1")+`
		),
		today_refunds AS (
		  SELECT pr.amount
		  FROM payment_refunds pr
		  WHERE pr.organization_id=$1 AND pr.location_id=$2
		    AND pr.created_at>=`+localDayStartSQL("$3::int-1")+` AND pr.created_at<`+localDayStartSQL("-1")+`
		),
		sales AS (
		  SELECT
		    COALESCE((SELECT sum(amount) FROM today_payments),0)
		    - COALESCE((SELECT sum(amount) FROM today_refunds),0) AS net
		),
		paid AS (
		  SELECT count(*)::int AS orders,COALESCE(sum(o.total),0) AS total
		  FROM orders o
		  WHERE o.organization_id=$1 AND o.location_id=$2 AND o.status<>'cancelado'
		    AND (`+netPaidSQL+`) >= o.total
		    AND o.id IN (SELECT order_id FROM today_payments)
		)
		SELECT
		  sales.net::text,
		  paid.orders,
		  CASE WHEN paid.orders>0 THEN (paid.total/paid.orders)::text ELSE '0' END,
		  (SELECT count(*) FROM orders
		   WHERE organization_id=$1 AND location_id=$2
		     AND status<>'cancelado' AND (status<>'entregado' OR (channel='salon' AND completed_at IS NULL))),
		  (SELECT count(*)
		   FROM inventory_items i
		   LEFT JOIN stock_balances b
		     ON b.inventory_item_id=i.id
		    AND b.organization_id=i.organization_id
		    AND b.location_id=$2
		   WHERE i.organization_id=$1 AND i.active
		     AND COALESCE(b.quantity,0)<=i.minimum_stock),
		  (SELECT count(*) FROM purchase_orders
		   WHERE organization_id=$1 AND location_id=$2 AND status='pending_approval'),
		  (SELECT count(*)
		   FROM reservations r
		   WHERE r.organization_id=$1 AND r.location_id=$2
		     AND r.status IN ('pending','confirmed')
		     AND r.starts_at>=`+localDayStartSQL("0")+` AND r.starts_at<`+localDayStartSQL("-1")+`),
		  (SELECT count(*) FROM orders
		   WHERE organization_id=$1 AND location_id=$2
		     AND status IN ('confirmado','preparando'))
		FROM sales CROSS JOIN paid
	`, s.OrganizationID, s.LocationID, period.Days).Scan(
		&salesNet, &paidOrders, &averageTicket, &openOrders, &critical, &purchases, &reservationsToday, &kitchenPending,
	)
	if err != nil {
		return domain.Dashboard{}, &application.ReportError{Stage: application.Summary, Cause: err}
	}

	hourly := []map[string]any{}
	rows, err := tx.Query(ctx, `
		SELECT hour,COALESCE(sum(value),0)::text
		FROM (
		  SELECT extract(hour FROM p.created_at AT TIME ZONE l.timezone)::int AS hour,p.amount AS value
		  FROM payments p JOIN locations l ON l.id=p.location_id AND l.organization_id=p.organization_id
		  WHERE p.organization_id=$1 AND p.location_id=$2
		    AND p.created_at>=`+localDayStartSQL("0")+` AND p.created_at<`+localDayStartSQL("-1")+`
		  UNION ALL
		  SELECT extract(hour FROM pr.created_at AT TIME ZONE l.timezone)::int AS hour,-pr.amount AS value
		  FROM payment_refunds pr JOIN locations l ON l.id=pr.location_id AND l.organization_id=pr.organization_id
		  WHERE pr.organization_id=$1 AND pr.location_id=$2
		    AND pr.created_at>=`+localDayStartSQL("0")+` AND pr.created_at<`+localDayStartSQL("-1")+`
		) movements
		GROUP BY hour ORDER BY hour
	`, s.OrganizationID, s.LocationID)
	if err != nil {
		return domain.Dashboard{}, &application.ReportError{Stage: application.Hourly, Cause: err}
	}
	for rows.Next() {
		var hour int
		var total string
		if err = rows.Scan(&hour, &total); err != nil {
			break
		}
		hourly = append(hourly, map[string]any{"hour": hour, "total": total})
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return domain.Dashboard{}, &application.ReportError{Stage: application.Hourly, Cause: err}
	}

	topProducts := []map[string]any{}
	productRows, err := tx.Query(ctx, `
		SELECT oi.name,sum(oi.qty)::text,sum(oi.qty*oi.unit_price)::text
		FROM order_items oi
		JOIN orders o ON o.id=oi.order_id AND o.organization_id=oi.organization_id
		WHERE o.organization_id=$1 AND o.location_id=$2 AND o.status<>'cancelado'
		  AND (`+netPaidSQL+`) >= o.total
		  AND o.id IN (
		    SELECT today_payment.order_id FROM payments today_payment
		    WHERE today_payment.organization_id=$1 AND today_payment.location_id=$2
		      AND today_payment.created_at>=`+localDayStartSQL("$3::int-1")+` AND today_payment.created_at<`+localDayStartSQL("-1")+`
		  )
		GROUP BY oi.name ORDER BY sum(oi.qty) DESC,oi.name LIMIT 5
	`, s.OrganizationID, s.LocationID, period.Days)
	if err != nil {
		return domain.Dashboard{}, &application.ReportError{Stage: application.Products, Cause: err}
	}
	for productRows.Next() {
		var name, qty, revenue string
		if err = productRows.Scan(&name, &qty, &revenue); err != nil {
			break
		}
		topProducts = append(topProducts, map[string]any{"name": name, "qty": qty, "revenue": revenue})
	}
	if err == nil {
		err = productRows.Err()
	}
	productRows.Close()
	if err != nil {
		return domain.Dashboard{}, &application.ReportError{Stage: application.Products, Cause: err}
	}
	operations, businessDate, err := loadDashboardOperations(ctx, tx, s, canSeeExpected)
	if err != nil {
		return domain.Dashboard{}, &application.ReportError{Stage: application.Operations, Cause: err}
	}
	insights, err := loadDashboardInsights(ctx, tx, s, period, labels)
	if err != nil {
		return domain.Dashboard{}, &application.ReportError{Stage: application.Insights, Cause: err}
	}
	return domain.Dashboard{SalesNet: salesNet, PaidOrders: paidOrders, AverageTicket: averageTicket, OpenOrders: openOrders, CriticalStock: critical, PurchasesToApprove: purchases, ReservationsToday: reservationsToday, KitchenPending: kitchenPending, HourlySales: hourly, TopProducts: topProducts, Operations: operations, BusinessDate: businessDate, DashboardInsights: insights}, nil
}
