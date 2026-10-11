package httpapi

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Current operational balances include unfinished orders from earlier days.
// Daily financial indicators remain separate in the dashboard response.
type dashboardOperations struct {
	PendingBalance      string  `json:"pendingBalance"`
	UnpaidOrders        int     `json:"unpaidOrders"`
	PartialOrders       int     `json:"partialOrders"`
	TablesTotal         int     `json:"tablesTotal"`
	TablesOccupied      int     `json:"tablesOccupied"`
	KitchenConfirmed    int     `json:"kitchenConfirmed"`
	KitchenPreparing    int     `json:"kitchenPreparing"`
	ReadyOrders         int     `json:"readyOrders"`
	DeliveryPending     int     `json:"deliveryPending"`
	DeliveryInTransit   int     `json:"deliveryInTransit"`
	ActiveCashRegisters int     `json:"activeCashRegisters"`
	OpenCashShifts      int     `json:"openCashShifts"`
	CashBalance         *string `json:"cashBalance"`
	SoldOutProducts     int     `json:"soldOutProducts"`
}

func loadDashboardOperations(ctx context.Context, tx pgx.Tx, s scope, canSeeExpected bool) (dashboardOperations, string, error) {
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
