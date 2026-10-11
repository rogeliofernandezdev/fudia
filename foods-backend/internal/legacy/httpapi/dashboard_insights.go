package httpapi

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// dashboardPeriod define la ventana de ventas del dashboard en días locales del local.
// "today" se compara con el mismo día de la semana anterior; los demás periodos,
// con la ventana inmediatamente anterior de igual duración.
type dashboardPeriod struct {
	Key  string
	Days int
}

func dashboardPeriodByKey(key string) (dashboardPeriod, bool) {
	switch key {
	case "", "today":
		return dashboardPeriod{Key: "today", Days: 1}, true
	case "7d":
		return dashboardPeriod{Key: "7d", Days: 7}, true
	case "30d":
		return dashboardPeriod{Key: "30d", Days: 30}, true
	}
	return dashboardPeriod{}, false
}

// Desplazamientos en días respecto de hoy: [inicio, fin] de la ventana anterior.
func (p dashboardPeriod) previousOffsets() (int, int) {
	if p.Days == 1 {
		return 7, 7
	}
	return 2*p.Days - 1, p.Days
}

type dashboardPeriodView struct {
	Key  string `json:"key"`
	Days int    `json:"days"`
	From string `json:"from"`
	To   string `json:"to"`
}

type dashboardPreviousView struct {
	SalesNet      string `json:"salesNet"`
	PaidOrders    int    `json:"paidOrders"`
	AverageTicket string `json:"averageTicket"`
	From          string `json:"from"`
	To            string `json:"to"`
}

type dashboardTrendPoint struct {
	Key      string `json:"key"`
	Current  string `json:"current"`
	Previous string `json:"previous"`
	Orders   int    `json:"orders"`
}

type dashboardTrend struct {
	Granularity string                `json:"granularity"`
	Points      []dashboardTrendPoint `json:"points"`
}

type dashboardBreakdown struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Total string `json:"total"`
	Count int    `json:"count"`
}

type dashboardCategorySale struct {
	Name    string `json:"name"`
	Qty     string `json:"qty"`
	Revenue string `json:"revenue"`
}

type dashboardInsights struct {
	Period               dashboardPeriodView     `json:"period"`
	Previous             dashboardPreviousView   `json:"previous"`
	Trend                dashboardTrend          `json:"trend"`
	SalesByChannel       []dashboardBreakdown    `json:"salesByChannel"`
	SalesByPaymentMethod []dashboardBreakdown    `json:"salesByPaymentMethod"`
	SalesByCategory      []dashboardCategorySale `json:"salesByCategory"`
}

// Cobros y devoluciones del local con su fecha local, pedido, medio de pago y canal.
// Las devoluciones restan en el día, medio y canal del cobro que revierten.
// $3 = días hacia atrás que cubren la ventana actual y la anterior.
const dashboardMovementsSQL = `
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
	    AND (p.created_at AT TIME ZONE loc.timezone)::date>=loc.today-$3::int
	  UNION ALL
	  SELECT (pr.created_at AT TIME ZONE loc.timezone),-pr.amount,p.method,p.order_id,o.channel
	  FROM payment_refunds pr
	  JOIN payments p ON p.id=pr.payment_id AND p.organization_id=pr.organization_id AND p.location_id=pr.location_id
	  JOIN orders o ON o.id=p.order_id AND o.organization_id=p.organization_id
	  CROSS JOIN loc
	  WHERE pr.organization_id=$1 AND pr.location_id=$2
	    AND (pr.created_at AT TIME ZONE loc.timezone)::date>=loc.today-$3::int
	)`

// Pedidos no cancelados pagados por completo con algún cobro entre loc.today-$3 y loc.today-$4.
const dashboardPaidOrdersSQL = `
	SELECT o.id,o.total
	FROM orders o
	WHERE o.organization_id=$1 AND o.location_id=$2 AND o.status<>'cancelado'
	  AND (` + netPaidSQL + `) >= o.total
	  AND EXISTS(
	    SELECT 1 FROM payments window_payment
	    JOIN locations l ON l.id=window_payment.location_id AND l.organization_id=window_payment.organization_id
	    WHERE window_payment.order_id=o.id AND window_payment.organization_id=o.organization_id AND window_payment.location_id=o.location_id
	      AND (window_payment.created_at AT TIME ZONE l.timezone)::date
	          BETWEEN (now() AT TIME ZONE l.timezone)::date-$3::int AND (now() AT TIME ZONE l.timezone)::date-$4::int
	  )`

func loadDashboardInsights(ctx context.Context, tx pgx.Tx, s scope, period dashboardPeriod) (dashboardInsights, error) {
	out := dashboardInsights{
		Period:               dashboardPeriodView{Key: period.Key, Days: period.Days},
		Trend:                dashboardTrend{Granularity: "day", Points: []dashboardTrendPoint{}},
		SalesByChannel:       []dashboardBreakdown{},
		SalesByPaymentMethod: []dashboardBreakdown{},
		SalesByCategory:      []dashboardCategorySale{},
	}
	previousStart, previousEnd := period.previousOffsets()
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

	channelLabels := map[string]string{}
	for _, option := range orderChannels {
		channelLabels[option["value"]] = option["label"]
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
