package postgres

import (
	"context"

	menucontracts "github.com/foods-platform/foods-backend/internal/modules/menu/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/jackc/pgx/v5"
)

// Las lecturas del detalle se encolan en un pgx.Batch: con la base remota cada
// consulta es una ida y vuelta, y el detalle de Salón se refresca cada 10 s.
// Un batch fuera de transacción corre en una transacción implícita, así que las
// líneas, combos, modificadores y cobros se leen de la misma instantánea.

const itemsSQL = `SELECT id,COALESCE(product_id::text,''),name,qty::text,unit_price::text,note,item_type FROM order_items WHERE order_id=$1 AND organization_id=$2 AND ($3='' OR kitchen_round_id::text=$3) ORDER BY created_at,id`

const itemSelectionsSQL = `
	SELECT s.order_item_id::text,s.group_id::text,s.group_name,s.option_product_id::text,s.option_name,s.surcharge::text
	FROM order_item_combo_selections s
	JOIN order_items i ON i.id=s.order_item_id AND i.organization_id=s.organization_id
	WHERE i.order_id=$1 AND i.organization_id=$2
	  AND ($3='' OR i.kitchen_round_id::text=$3)
	ORDER BY s.created_at,s.id`

const itemModifiersSQL = `
	SELECT m.order_item_id::text,m.group_id::text,m.group_name,
	       m.option_id::text,m.option_name,m.surcharge::text
	FROM order_item_modifier_selections m
	JOIN order_items i ON i.id=m.order_item_id AND i.organization_id=m.organization_id
	WHERE i.order_id=$1 AND i.organization_id=$2
	  AND ($3='' OR i.kitchen_round_id::text=$3)
	ORDER BY m.created_at,m.id`

const serviceItemsSQL = `SELECT id::text,order_item_id::text,COALESCE(product_id::text,''),name,qty::text,destination,status FROM order_service_items WHERE organization_id=$1 AND order_id=$2 ORDER BY created_at,id`

const orderNetPaidSQL = `
	SELECT COALESCE(sum(
	  p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)
	),0)::float8
	FROM payments p
	WHERE p.order_id=$1 AND p.organization_id=$2 AND p.location_id=$3`

var serviceStatusLabels = map[string]string{"nuevo": "Sin enviar", "confirmado": "Pendiente", "preparando": "Preparando", "listo": "Por entregar", "entregado": "Entregado"}

func queueItems(batch *pgx.Batch, s scope, orderID, roundID string) {
	batch.Queue(itemsSQL, orderID, s.OrganizationID, roundID)
	batch.Queue(itemSelectionsSQL, orderID, s.OrganizationID, roundID)
	batch.Queue(itemModifiersSQL, orderID, s.OrganizationID, roundID)
}

// readItems lee, en orden, los tres resultados que encola queueItems.
func readItems(results pgx.BatchResults) ([]domain.OrderItem, error) {
	items := []domain.OrderItem{}
	indexByID := map[string]int{}
	rows, err := results.Query()
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var it domain.OrderItem
		if err := rows.Scan(&it.ID, &it.ProductID, &it.Name, &it.Qty, &it.UnitPrice, &it.Note, &it.ItemType); err != nil {
			rows.Close()
			return nil, err
		}
		indexByID[it.ID] = len(items)
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if rows, err = results.Query(); err != nil {
		return nil, err
	}
	for rows.Next() {
		var itemID string
		var sel domain.OrderItemSelection
		if err := rows.Scan(&itemID, &sel.GroupID, &sel.GroupName, &sel.ProductID, &sel.Name, &sel.Surcharge); err != nil {
			rows.Close()
			return nil, err
		}
		if idx, ok := indexByID[itemID]; ok {
			items[idx].Selections = append(items[idx].Selections, sel)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if rows, err = results.Query(); err != nil {
		return nil, err
	}
	for rows.Next() {
		var itemID string
		var modifier domain.OrderItemModifier
		if err := rows.Scan(&itemID, &modifier.GroupID, &modifier.GroupName, &modifier.OptionID, &modifier.Name, &modifier.Surcharge); err != nil {
			rows.Close()
			return nil, err
		}
		if idx, ok := indexByID[itemID]; ok {
			items[idx].Modifiers = append(items[idx].Modifiers, modifier)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func readServiceItems(results pgx.BatchResults) ([]domain.OrderServiceItem, error) {
	rows, err := results.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.OrderServiceItem{}
	for rows.Next() {
		var item domain.OrderServiceItem
		if err = rows.Scan(&item.ID, &item.OrderItemID, &item.ProductID, &item.Name, &item.Qty, &item.Destination, &item.Status); err != nil {
			return nil, err
		}
		for _, destination := range menucontracts.ServiceDestinations {
			if destination["value"] == item.Destination {
				item.DestinationLabel = destination["shortLabel"]
			}
		}
		item.StatusLabel = serviceStatusLabels[item.Status]
		items = append(items, item)
	}
	return items, rows.Err()
}

// closeBatch cierra los resultados y conserva el primer error.
func closeBatch(results pgx.BatchResults, err error) error {
	if closeErr := results.Close(); err == nil {
		err = closeErr
	}
	return err
}

// Items lee las líneas con sus combos y modificadores en una ida y vuelta.
func (r *Orders) Items(ctx context.Context, s scope, orderID, roundID string) ([]domain.OrderItem, error) {
	batch := &pgx.Batch{}
	queueItems(batch, s, orderID, roundID)
	results := r.db(ctx).SendBatch(ctx, batch)
	items, err := readItems(results)
	if err = closeBatch(results, err); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Orders) ServiceItems(ctx context.Context, s scope, orderID string) ([]domain.OrderServiceItem, error) {
	batch := &pgx.Batch{}
	batch.Queue(serviceItemsSQL, s.OrganizationID, orderID)
	results := r.db(ctx).SendBatch(ctx, batch)
	items, err := readServiceItems(results)
	if err = closeBatch(results, err); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Orders) NetPaid(ctx context.Context, s scope, orderID string) (float64, error) {
	var paid float64
	err := r.db(ctx).QueryRow(ctx, orderNetPaidSQL, orderID, s.OrganizationID, s.LocationID).Scan(&paid)
	return paid, err
}

// Detail lee el pedido, sus entregas, sus líneas y lo cobrado neto en una sola
// ida y vuelta (antes eran seis consultas secuenciales). Devuelve ErrNotFound si
// el pedido no pertenece a la empresa y local.
func (r *Orders) Detail(ctx context.Context, s scope, id string) (domain.Order, float64, error) {
	batch := &pgx.Batch{}
	batch.Queue(`SELECT `+orderColumns+` FROM orders WHERE id=$1 AND organization_id=$2 AND location_id=$3`, id, s.OrganizationID, s.LocationID)
	batch.Queue(serviceItemsSQL, s.OrganizationID, id)
	queueItems(batch, s, id, "")
	batch.Queue(orderNetPaidSQL, id, s.OrganizationID, s.LocationID)
	results := r.db(ctx).SendBatch(ctx, batch)
	var paid float64
	o, err := scanOrder(results.QueryRow())
	if err == nil {
		o.ServiceItems, err = readServiceItems(results)
	}
	if err == nil {
		o.Items, err = readItems(results)
	}
	if err == nil {
		err = results.QueryRow().Scan(&paid)
	}
	if err = closeBatch(results, err); err != nil {
		return domain.Order{}, 0, orderError(err)
	}
	return o, paid, nil
}
