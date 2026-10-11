package httpapi

import (
	"errors"
	"github.com/jackc/pgx/v5"
	"net/http"
)

func (a *API) dashboard(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	period, ok := dashboardPeriodByKey(r.URL.Query().Get("period"))
	if !ok {
		fail(w, 400, "invalid_period", "El periodo del resumen no es válido.")
		return
	}
	canSeeExpected := a.canSeeCashExpected(r, s)
	tx, err := a.db.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		fail(w, 503, "dashboard_unavailable", "No pudimos calcular el resumen operativo.")
		return
	}
	defer tx.Rollback(r.Context())
	var salesNet, averageTicket string
	var paidOrders, openOrders, critical, purchases, reservationsToday, kitchenPending int

	err = tx.QueryRow(r.Context(), `
		WITH loc AS (
		  SELECT timezone
		  FROM locations
		  WHERE id=$2 AND organization_id=$1
		),
		today_payments AS (
		  SELECT p.order_id,p.amount
		  FROM payments p
		  CROSS JOIN loc
		  WHERE p.organization_id=$1 AND p.location_id=$2
		    AND (p.created_at AT TIME ZONE loc.timezone)::date BETWEEN (now() AT TIME ZONE loc.timezone)::date-($3::int-1) AND (now() AT TIME ZONE loc.timezone)::date
		),
		today_refunds AS (
		  SELECT pr.amount
		  FROM payment_refunds pr
		  CROSS JOIN loc
		  WHERE pr.organization_id=$1 AND pr.location_id=$2
		    AND (pr.created_at AT TIME ZONE loc.timezone)::date BETWEEN (now() AT TIME ZONE loc.timezone)::date-($3::int-1) AND (now() AT TIME ZONE loc.timezone)::date
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
		    AND EXISTS(SELECT 1 FROM today_payments p WHERE p.order_id=o.id)
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
		   CROSS JOIN loc
		   WHERE r.organization_id=$1 AND r.location_id=$2
		     AND r.status IN ('pending','confirmed')
		     AND (r.starts_at AT TIME ZONE loc.timezone)::date=(now() AT TIME ZONE loc.timezone)::date),
		  (SELECT count(*) FROM orders
		   WHERE organization_id=$1 AND location_id=$2
		     AND status IN ('confirmado','preparando'))
		FROM sales CROSS JOIN paid
	`, s.OrganizationID, s.LocationID, period.Days).Scan(
		&salesNet, &paidOrders, &averageTicket, &openOrders, &critical, &purchases, &reservationsToday, &kitchenPending,
	)
	if err != nil {
		fail(w, 503, "dashboard_unavailable", "No pudimos calcular el resumen operativo.")
		return
	}

	hourly := []map[string]any{}
	rows, err := tx.Query(r.Context(), `
		SELECT hour,COALESCE(sum(value),0)::text
		FROM (
		  SELECT extract(hour FROM p.created_at AT TIME ZONE l.timezone)::int AS hour,p.amount AS value
		  FROM payments p JOIN locations l ON l.id=p.location_id AND l.organization_id=p.organization_id
		  WHERE p.organization_id=$1 AND p.location_id=$2
		    AND (p.created_at AT TIME ZONE l.timezone)::date=(now() AT TIME ZONE l.timezone)::date
		  UNION ALL
		  SELECT extract(hour FROM pr.created_at AT TIME ZONE l.timezone)::int AS hour,-pr.amount AS value
		  FROM payment_refunds pr JOIN locations l ON l.id=pr.location_id AND l.organization_id=pr.organization_id
		  WHERE pr.organization_id=$1 AND pr.location_id=$2
		    AND (pr.created_at AT TIME ZONE l.timezone)::date=(now() AT TIME ZONE l.timezone)::date
		) movements
		GROUP BY hour ORDER BY hour
	`, s.OrganizationID, s.LocationID)
	if err != nil {
		fail(w, 503, "dashboard_unavailable", "No pudimos cargar las ventas por hora.")
		return
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
		fail(w, 503, "dashboard_unavailable", "No pudimos cargar las ventas por hora.")
		return
	}

	topProducts := []map[string]any{}
	productRows, err := tx.Query(r.Context(), `
		SELECT oi.name,sum(oi.qty)::text,sum(oi.qty*oi.unit_price)::text
		FROM order_items oi
		JOIN orders o ON o.id=oi.order_id AND o.organization_id=oi.organization_id
		WHERE o.organization_id=$1 AND o.location_id=$2 AND o.status<>'cancelado'
		  AND (`+netPaidSQL+`) >= o.total
		  AND EXISTS(
		    SELECT 1 FROM payments today_payment
		    JOIN locations l ON l.id=today_payment.location_id AND l.organization_id=today_payment.organization_id
		    WHERE today_payment.order_id=o.id AND today_payment.organization_id=o.organization_id AND today_payment.location_id=o.location_id
		      AND (today_payment.created_at AT TIME ZONE l.timezone)::date
		          BETWEEN (now() AT TIME ZONE l.timezone)::date-($3::int-1) AND (now() AT TIME ZONE l.timezone)::date
		  )
		GROUP BY oi.name ORDER BY sum(oi.qty) DESC,oi.name LIMIT 5
	`, s.OrganizationID, s.LocationID, period.Days)
	if err != nil {
		fail(w, 503, "dashboard_unavailable", "No pudimos cargar los productos vendidos.")
		return
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
		fail(w, 503, "dashboard_unavailable", "No pudimos cargar los productos vendidos.")
		return
	}
	operations, businessDate, err := loadDashboardOperations(r.Context(), tx, s, canSeeExpected)
	if err != nil {
		fail(w, 503, "dashboard_unavailable", "No pudimos calcular el estado del restaurante.")
		return
	}
	insights, err := loadDashboardInsights(r.Context(), tx, s, period)
	if err != nil {
		fail(w, 503, "dashboard_unavailable", "No pudimos calcular las tendencias de ventas.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "dashboard_unavailable", "No pudimos completar el resumen operativo.")
		return
	}
	writeJSON(w, 200, map[string]any{
		"salesNet": salesNet, "paidOrders": paidOrders, "averageTicket": averageTicket, "openOrders": openOrders,
		"criticalStock": critical, "purchasesToApprove": purchases, "reservationsToday": reservationsToday,
		"kitchenPending": kitchenPending, "hourlySales": hourly, "topProducts": topProducts,
		"operations": operations, "businessDate": businessDate,
		"period": insights.Period, "previous": insights.Previous, "trend": insights.Trend,
		"salesByChannel": insights.SalesByChannel, "salesByPaymentMethod": insights.SalesByPaymentMethod,
		"salesByCategory": insights.SalesByCategory,
	})
}

func (a *API) getContext(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var organizationName, locationName, locationCountry, locationTimezone string
	var platformAdmin bool
	var permissions, menuAccess []string
	err := a.db.QueryRow(r.Context(), `
		SELECT o.trade_name,l.name,p.country_code,l.timezone,u.platform_admin,
		       CASE WHEN u.platform_admin THEN ARRAY['*']::text[] ELSE COALESCE((
		         SELECT array_agg(DISTINCT permission ORDER BY permission)
		         FROM user_roles ur
		         JOIN roles ro ON ro.id=ur.role_id AND ro.organization_id=$1 AND ro.active
		         CROSS JOIN LATERAL unnest(ro.permissions) permission
		         WHERE ur.user_id=$3 AND ur.location_id=$2
		       ),ARRAY[]::text[]) END,
		       CASE WHEN u.platform_admin THEN ARRAY['*']::text[] ELSE COALESCE((
		         SELECT array_agg(DISTINCT access_key ORDER BY access_key)
		         FROM user_roles ur
		         JOIN roles ro ON ro.id=ur.role_id AND ro.organization_id=$1 AND ro.active
		         CROSS JOIN LATERAL unnest(ro.menu_access) access_key
		         WHERE ur.user_id=$3 AND ur.location_id=$2
		       ),ARRAY[]::text[]) END
		FROM organizations o
		JOIN locations l ON l.organization_id=o.id
		JOIN organization_fiscal_profiles p ON p.id=l.fiscal_profile_id AND p.organization_id=l.organization_id AND p.active
		JOIN users u ON u.id=$3 AND u.active AND (u.platform_admin OR u.organization_id=o.id)
		WHERE o.id=$1 AND l.id=$2 AND o.active AND l.active
	`, s.OrganizationID, s.LocationID, s.UserID).Scan(
		&organizationName, &locationName, &locationCountry, &locationTimezone, &platformAdmin,
		&permissions, &menuAccess,
	)
	if err != nil {
		fail(w, 503, "context_unavailable", "No pudimos cargar el contexto de trabajo.")
		return
	}
	modules := a.activeModules(s.OrganizationID)
	if modules == nil {
		fail(w, 503, "modules_unavailable", "No pudimos cargar los módulos de la empresa.")
		return
	}
	setupRequired := false
	if !platformAdmin {
		if err = a.db.QueryRow(r.Context(), `
			SELECT completed_at IS NULL
			FROM organization_operational_setup
			WHERE organization_id=$1
		`, s.OrganizationID).Scan(&setupRequired); errors.Is(err, pgx.ErrNoRows) {
			setupRequired = true
		} else if err != nil {
			fail(w, 503, "context_unavailable", "No pudimos cargar el estado inicial de la empresa.")
			return
		}
	}
	writeJSON(w, 200, map[string]any{
		"user":          map[string]any{"id": s.UserID, "name": s.Name, "platformAdmin": platformAdmin},
		"organization":  map[string]string{"id": s.OrganizationID, "name": organizationName},
		"location":      map[string]string{"id": s.LocationID, "name": locationName, "country": locationCountry, "timezone": locationTimezone},
		"modules":       modules,
		"menuAccess":    menuAccess,
		"permissions":   permissions,
		"setupRequired": setupRequired,
	})
}
