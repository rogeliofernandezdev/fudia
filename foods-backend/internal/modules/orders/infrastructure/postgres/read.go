package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/orders/application"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/jackc/pgx/v5"
	"strings"
)

// listOrderSQL pone primero los pedidos abiertos y luego los más recientes. Debe
// coincidir literalmente con el índice de expresión orders_list_order_idx
// (migración 000109); TestListOrderMatchesIndex lo comprueba.
const listOrderSQL = `CASE WHEN status='cancelado' OR completed_at IS NOT NULL OR (status='entregado' AND channel<>'salon') THEN 1 ELSE 0 END, created_at DESC`

func (r *Orders) List(ctx context.Context, s scope, f application.Filter) (application.Page, error) {
	out := application.Page{Page: f.Page, PageSize: f.Size}
	page, size, q, channel, status := f.Page, f.Size, strings.TrimSpace(f.Query), f.Channel, f.Status
	// scopeWhere es búsqueda + canal; el estado se suma aparte para saber qué estados existen en ese alcance.
	scopeWhere := `organization_id=$1 AND location_id=$2 AND ($3='' OR code ILIKE '%'||$3||'%' OR customer_name ILIKE '%'||$3||'%' OR customer_phone ILIKE '%'||$3||'%') AND ($4='' OR channel=$4 OR ($4 IN ('recojo','mostrador') AND channel IN ('recojo','mostrador')) OR ($4='salon' AND channel='whatsapp'))`
	where := scopeWhere + ` AND ($5='' OR ($5='abiertos' AND status<>'cancelado' AND (status<>'entregado' OR (channel='salon' AND completed_at IS NULL))) OR status=$5)`
	// Estados con al menos un pedido para la búsqueda y el canal actuales: una sonda EXISTS por estado del
	// catálogo usa el índice (organización, local, estado) sin recorrer todo el historial.
	catalog := make([]string, 0, len(domain.Statuses))
	for _, option := range domain.Statuses {
		catalog = append(catalog, option["value"])
	}
	// Las cinco lecturas son independientes y van en un pgx.Batch: una ida y vuelta
	// en vez de cinco (Pedidos refresca el listado cada 10 s) y la misma instantánea.
	batch := &pgx.Batch{}
	batch.Queue(`SELECT count(*) FROM orders WHERE `+where, s.OrganizationID, s.LocationID, q, channel, status)
	batch.Queue(`SELECT `+orderColumns+` FROM orders WHERE `+where+` ORDER BY `+listOrderSQL+` LIMIT $6 OFFSET $7`, s.OrganizationID, s.LocationID, q, channel, status, size, (page-1)*size)
	batch.Queue(`SELECT st FROM unnest($5::text[]) st WHERE EXISTS(SELECT 1 FROM orders WHERE `+scopeWhere+` AND status=st)`, s.OrganizationID, s.LocationID, q, channel, catalog)
	batch.Queue(`SELECT channel, count(*) FROM orders WHERE organization_id=$1 AND location_id=$2 AND status<>'cancelado' AND (status<>'entregado' OR (channel='salon' AND completed_at IS NULL)) GROUP BY channel`, s.OrganizationID, s.LocationID)
	batch.Queue(`
		SELECT p.currency_symbol
		FROM locations l
		JOIN organization_fiscal_profiles p
		  ON p.id=l.fiscal_profile_id AND p.organization_id=l.organization_id AND p.active
		WHERE l.id=$1 AND l.organization_id=$2 AND l.active
	`, s.LocationID, s.OrganizationID)
	results := r.db(ctx).SendBatch(ctx, batch)
	defer results.Close()

	var total int
	if err := results.QueryRow().Scan(&total); err != nil {
		return out, err
	}
	items := []domain.Order{}
	rows, err := results.Query()
	if err != nil {
		return out, err
	}
	for rows.Next() {
		o, scanErr := scanOrder(rows)
		if scanErr != nil {
			rows.Close()
			return out, scanErr
		}
		items = append(items, o)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return out, err
	}
	present := map[string]bool{}
	statusRows, err := results.Query()
	if err != nil {
		return out, err
	}
	for statusRows.Next() {
		var st string
		if err = statusRows.Scan(&st); err != nil {
			statusRows.Close()
			return out, err
		}
		present[st] = true
	}
	statusRows.Close()
	if err = statusRows.Err(); err != nil {
		return out, err
	}
	out.PresentStatuses = present
	channelCounts := map[string]int{}
	countRows, err := results.Query()
	if err != nil {
		return out, err
	}
	for countRows.Next() {
		var ch string
		var n int
		if err = countRows.Scan(&ch, &n); err != nil {
			countRows.Close()
			return out, err
		}
		channelCounts[domain.ServiceChannel(ch)] += n
	}
	countRows.Close()
	if err = countRows.Err(); err != nil {
		return out, err
	}
	var currencySymbol string
	if err := results.QueryRow().Scan(&currencySymbol); err != nil {
		return out, err
	}
	out.Items = items
	out.Total = total
	out.ChannelCounts = channelCounts
	out.CurrencySymbol = currencySymbol
	return out, nil
}
func (r *Orders) Floor(ctx context.Context, s scope) ([]domain.FloorTable, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT t.id,t.name,t.zone,t.seats,
		  o.id,COALESCE(o.code,''),COALESCE(o.channel,''),COALESCE(o.status,''),COALESCE(o.customer_id::text,''),COALESCE(o.customer_name,''),COALESCE(o.customer_phone,''),COALESCE(o.address,''),COALESCE(o.reference,''),COALESCE(o.table_id::text,''),'',COALESCE(o.notes,''),COALESCE(o.subtotal::text,'0'),COALESCE(o.delivery_fee::text,'0'),COALESCE(o.total::text,'0'),COALESCE(to_char(o.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),''),COALESCE(to_char(o.updated_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),''),
		  COALESCE(item_totals.item_count,0),COALESCE(payment_totals.paid,0)::text,COALESCE(to_char(o.completed_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),''),
		  COALESCE(o.waiter_id::text,''),COALESCE(waiter.full_name,''),COALESCE(payment_totals.collectors,'{}'::text[]),COALESCE(to_char(o.bill_closed_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),'')
		FROM tables t
		LEFT JOIN orders o
		  ON o.table_id=t.id
		 AND o.organization_id=t.organization_id
		 AND o.location_id=t.location_id
		 AND o.status<>'cancelado' AND (o.status<>'entregado' OR (o.channel='salon' AND o.completed_at IS NULL))
		LEFT JOIN users waiter ON waiter.id=o.waiter_id AND waiter.organization_id=o.organization_id
		LEFT JOIN LATERAL (
		  SELECT COALESCE(sum(i.qty)::int,0) AS item_count
		  FROM order_items i
		  WHERE i.order_id=o.id AND i.organization_id=o.organization_id
		) item_totals ON o.id IS NOT NULL
		LEFT JOIN LATERAL (
		  SELECT COALESCE(sum(p.amount-COALESCE(refunds.refunded,0)),0) AS paid,
		    array_agg(DISTINCT collector.full_name ORDER BY collector.full_name) FILTER (WHERE collector.id IS NOT NULL) AS collectors
		  FROM payments p
		  JOIN users collector ON collector.id=p.created_by AND collector.organization_id=p.organization_id
		  LEFT JOIN LATERAL (
		    SELECT COALESCE(sum(pr.amount),0) AS refunded
		    FROM payment_refunds pr
		    WHERE pr.payment_id=p.id
		      AND pr.organization_id=p.organization_id
		      AND pr.location_id=p.location_id
		  ) refunds ON true
		  WHERE p.order_id=o.id
		    AND p.organization_id=o.organization_id
		    AND p.location_id=o.location_id
		) payment_totals ON o.id IS NOT NULL
		WHERE t.organization_id=$1 AND t.location_id=$2 AND t.active AND t.service_type='table'
		ORDER BY t.zone,t.name
	`, s.OrganizationID, s.LocationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.FloorTable{}
	for rows.Next() {
		var ft domain.FloorTable
		var o domain.Order
		var oid *string
		var paidText string
		if err = rows.Scan(
			&ft.ID, &ft.Name, &ft.Zone, &ft.Seats,
			&oid, &o.Code, &o.Channel, &o.Status, &o.CustomerID, &o.CustomerName, &o.CustomerPhone,
			&o.Address, &o.Reference, &o.TableID, &o.TableName, &o.Notes, &o.Subtotal, &o.DeliveryFee,
			&o.Total, &o.CreatedAt, &o.UpdatedAt, &o.ItemCount, &paidText, &o.CompletedAt, &o.WaiterID, &o.WaiterName, &o.CollectedByNames, &o.BillClosedAt,
		); err != nil {
			return nil, err
		}
		if oid != nil {
			o.ID = *oid
			// A table becomes occupied when its active salon order is opened.
			// Keep this timestamp stable across edits, kitchen rounds and payments.
			ft.OccupiedSince = o.CreatedAt
			ft.PaidText = paidText
			ft.Order = &o
		}
		items = append(items, ft)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
