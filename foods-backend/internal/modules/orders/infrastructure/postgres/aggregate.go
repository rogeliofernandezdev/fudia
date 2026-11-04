package postgres

import (
	"context"
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/jackc/pgx/v5"
)

const orderColumns = `id,code,channel,status,COALESCE(customer_id::text,''),customer_name,customer_phone,address,reference,COALESCE(table_id::text,''),COALESCE((SELECT name FROM tables t WHERE t.id=orders.table_id),''),notes,subtotal::text,delivery_fee::text,total::text,to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),COALESCE(to_char(completed_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),''),
COALESCE(waiter_id::text,''),COALESCE((SELECT u.full_name FROM users u WHERE u.id=orders.waiter_id AND u.organization_id=orders.organization_id),''),
ARRAY(SELECT DISTINCT u.full_name FROM payments p JOIN users u ON u.id=p.created_by AND u.organization_id=p.organization_id WHERE p.order_id=orders.id AND p.organization_id=orders.organization_id AND p.location_id=orders.location_id ORDER BY u.full_name),COALESCE(to_char(bill_closed_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),''),
delivery_latitude::float8,delivery_longitude::float8,delivery_distance_km::float8`
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

func scanOrder(row pgx.Row) (domain.Order, error) {
	var o domain.Order
	err := row.Scan(&o.ID, &o.Code, &o.Channel, &o.Status, &o.CustomerID, &o.CustomerName, &o.CustomerPhone, &o.Address, &o.Reference, &o.TableID, &o.TableName, &o.Notes, &o.Subtotal, &o.DeliveryFee, &o.Total, &o.CreatedAt, &o.UpdatedAt, &o.CompletedAt, &o.WaiterID, &o.WaiterName, &o.CollectedByNames, &o.BillClosedAt, &o.DeliveryLatitude, &o.DeliveryLongitude, &o.DeliveryDistanceKm)
	o.Origin = domain.Origin(o.Channel)
	o.Channel = domain.NormalizeChannel(o.Channel)
	o.AccountState = "open"
	if o.BillClosedAt != "" {
		o.AccountState = "awaiting_payment"
	}
	return o, err
}
func (r *Orders) QuantityUsage(ctx context.Context, s scope, orderID string) (map[string]float64, error) {
	usage := map[string]float64{}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT product_id::text,qty::float8
		FROM order_items
		WHERE order_id=$1 AND organization_id=$2 AND item_type='product' AND product_id IS NOT NULL
		UNION ALL
		SELECT s.option_product_id::text,oi.qty::float8
		FROM order_item_combo_selections s
		JOIN order_items oi ON oi.id=s.order_item_id AND oi.organization_id=s.organization_id
		WHERE oi.order_id=$1 AND oi.organization_id=$2`, orderID, s.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var productID string
		var qty float64
		if err := rows.Scan(&productID, &qty); err != nil {
			return nil, err
		}
		usage[productID] += qty
	}
	return usage, rows.Err()
}
// insertItemSQL inserta una línea con sus selecciones de combo, modificadores y
// líneas de servicio en una sola sentencia. Las líneas de servicio son una por
// selección de combo o, sin selecciones, una para el producto.
const insertItemSQL = `
	WITH item AS (
		INSERT INTO order_items(organization_id,order_id,product_id,name,qty,unit_price,note,item_type,kitchen_round_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,'')::uuid)
		RETURNING id
	), selections AS (
		INSERT INTO order_item_combo_selections(organization_id,order_item_id,group_id,group_name,option_product_id,option_name,surcharge)
		SELECT $1,item.id,x.group_id,x.group_name,x.product_id,x.name,x.surcharge
		FROM item CROSS JOIN ROWS FROM (jsonb_to_recordset($10::jsonb) AS (group_id uuid,group_name text,product_id uuid,name text,surcharge numeric)) WITH ORDINALITY AS x(group_id,group_name,product_id,name,surcharge,position)
		ORDER BY x.position
	), modifiers AS (
		INSERT INTO order_item_modifier_selections(organization_id,order_item_id,group_id,group_name,option_id,option_name,surcharge)
		SELECT $1,item.id,x.group_id,x.group_name,x.option_id,x.name,x.surcharge
		FROM item CROSS JOIN ROWS FROM (jsonb_to_recordset($11::jsonb) AS (group_id uuid,group_name text,option_id uuid,name text,surcharge numeric)) WITH ORDINALITY AS x(group_id,group_name,option_id,name,surcharge,position)
		ORDER BY x.position
	)
	INSERT INTO order_service_items(organization_id,order_id,order_item_id,product_id,name,qty,destination,status)
	SELECT $1,$2,item.id,NULLIF(t.product_id,'')::uuid,t.name,$5,COALESCE(p.service_destination,'kitchen'),
	       CASE WHEN o.status='nuevo' THEN 'nuevo' WHEN COALESCE(p.service_destination,'kitchen')='direct' THEN 'listo' ELSE 'confirmado' END
	FROM item
	CROSS JOIN ROWS FROM (jsonb_to_recordset($12::jsonb) AS (product_id text,name text)) WITH ORDINALITY AS t(product_id,name,position)
	JOIN orders o ON o.id=$2 AND o.organization_id=$1
	LEFT JOIN products p ON p.id=NULLIF(t.product_id,'')::uuid AND p.organization_id=$1
	ORDER BY t.position`

type serviceTarget struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
}
type selectionRow struct {
	GroupID   string  `json:"group_id"`
	GroupName string  `json:"group_name"`
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Surcharge float64 `json:"surcharge"`
}
type modifierRow struct {
	GroupID   string  `json:"group_id"`
	GroupName string  `json:"group_name"`
	OptionID  string  `json:"option_id"`
	Name      string  `json:"name"`
	Surcharge float64 `json:"surcharge"`
}

// InsertItems envía todas las líneas en un único pgx.Batch (una ida y vuelta),
// en el mismo orden en que llegaron.
func (r *Orders) InsertItems(ctx context.Context, s scope, orderID, roundID string, items []domain.PreparedOrderItem) error {
	if len(items) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, it := range items {
		selections := make([]selectionRow, len(it.Selections))
		targets := make([]serviceTarget, 0, len(it.Selections)+1)
		for i, sel := range it.Selections {
			selections[i] = selectionRow{sel.GroupID, sel.GroupName, sel.ProductID, sel.Name, sel.Surcharge}
			targets = append(targets, serviceTarget{sel.ProductID, sel.Name})
		}
		if len(it.Selections) == 0 {
			productID := ""
			if it.ProductID != nil {
				productID = *it.ProductID
			}
			targets = append(targets, serviceTarget{productID, it.Name})
		}
		modifiers := make([]modifierRow, len(it.Modifiers))
		for i, m := range it.Modifiers {
			modifiers[i] = modifierRow{m.GroupID, m.GroupName, m.OptionID, m.Name, m.Surcharge}
		}
		rawSelections, err := json.Marshal(selections)
		if err != nil {
			return err
		}
		rawModifiers, err := json.Marshal(modifiers)
		if err != nil {
			return err
		}
		rawTargets, err := json.Marshal(targets)
		if err != nil {
			return err
		}
		batch.Queue(insertItemSQL, s.OrganizationID, orderID, it.ProductID, it.Name, it.Qty, it.UnitPrice, it.Note, it.ItemType, roundID, rawSelections, rawModifiers, rawTargets)
	}
	results := r.db(ctx).SendBatch(ctx, batch)
	for range batch.Len() {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return err
		}
	}
	return results.Close()
}
func (r *Orders) insertServiceItem(ctx context.Context, s scope, orderID, itemID, productID, name string, qty float64) error {
	_, err := r.db(ctx).Exec(ctx, `INSERT INTO order_service_items(organization_id,order_id,order_item_id,product_id,name,qty,destination,status)
 SELECT $1,$2,$3,NULLIF($4,'')::uuid,$5,$6,COALESCE(p.service_destination,'kitchen'),
 CASE WHEN o.status='nuevo' THEN 'nuevo' WHEN COALESCE(p.service_destination,'kitchen')='direct' THEN 'listo' ELSE 'confirmado' END
 FROM orders o LEFT JOIN products p ON p.id=NULLIF($4,'')::uuid AND p.organization_id=$1
 WHERE o.id=$2 AND o.organization_id=$1`, s.OrganizationID, orderID, itemID, productID, name, qty)
	return err
}
func (r *Orders) SyncServiceStatus(ctx context.Context, s scope, id string) error {
	_, err := r.db(ctx).Exec(ctx, `UPDATE orders o SET status=CASE
 WHEN EXISTS(SELECT 1 FROM order_service_items i WHERE i.order_id=o.id AND i.organization_id=o.organization_id AND i.status='preparando') THEN 'preparando'
 WHEN EXISTS(SELECT 1 FROM order_service_items i WHERE i.order_id=o.id AND i.organization_id=o.organization_id AND i.status='confirmado') THEN 'confirmado'
 WHEN EXISTS(SELECT 1 FROM order_service_items i WHERE i.order_id=o.id AND i.organization_id=o.organization_id AND i.status='nuevo') THEN 'nuevo'
 WHEN EXISTS(SELECT 1 FROM order_service_items i WHERE i.order_id=o.id AND i.organization_id=o.organization_id AND i.status='listo') THEN 'listo'
 ELSE 'entregado' END,updated_at=now()
 WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3 AND o.completed_at IS NULL AND o.status NOT IN ('cancelado','en_camino')
 AND EXISTS(SELECT 1 FROM order_service_items i WHERE i.order_id=o.id AND i.organization_id=o.organization_id)`, id, s.OrganizationID, s.LocationID)
	return err
}
func (r *Orders) CompleteDeliveredAccount(ctx context.Context, s scope, orderID string) (bool, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE orders o
		SET completed_at=now(),updated_at=now()
		WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3
		  AND o.status='entregado' AND o.completed_at IS NULL
 AND (o.channel<>'salon' OR o.bill_closed_at IS NOT NULL)
 AND NOT EXISTS(SELECT 1 FROM order_service_items si WHERE si.order_id=o.id AND si.organization_id=o.organization_id AND si.status<>'entregado')
		  AND `+netPaidSQL+` >= o.total
	`, orderID, s.OrganizationID, s.LocationID)
	return tag.RowsAffected() > 0, err
}
func (r *Orders) Record(ctx context.Context, s scope, id string) (domain.Order, error) {
	v, err := scanOrder(r.db(ctx).QueryRow(ctx, `SELECT `+orderColumns+` FROM orders WHERE id=$1 AND organization_id=$2 AND location_id=$3`, id, s.OrganizationID, s.LocationID))
	return v, orderError(err)
}
