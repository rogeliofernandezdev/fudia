package httpapi

import (
	"context"
	"errors"
	orderdomain "github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type orderServiceItem = orderdomain.OrderServiceItem

var serviceDestinations = []map[string]string{{"value": "kitchen", "label": "Cocina"}, {"value": "bar", "label": "Barra"}, {"value": "direct", "label": "Entrega directa"}}

func (a *API) listServiceDestinations(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"items": serviceDestinations})
}

// Snapshot routing on sale: changing a product must not reroute already sent food.
func insertOrderServiceItem(ctx context.Context, tx pgx.Tx, org, orderID, itemID, productID, name string, qty float64) error {
	_, err := tx.Exec(ctx, `INSERT INTO order_service_items(organization_id,order_id,order_item_id,product_id,name,qty,destination,status)
 SELECT $1,$2,$3,NULLIF($4,'')::uuid,$5,$6,COALESCE(p.service_destination,'kitchen'),
 CASE WHEN o.status='nuevo' THEN 'nuevo' WHEN COALESCE(p.service_destination,'kitchen')='direct' THEN 'listo' ELSE 'confirmado' END
 FROM orders o LEFT JOIN products p ON p.id=NULLIF($4,'')::uuid AND p.organization_id=$1
 WHERE o.id=$2 AND o.organization_id=$1`, org, orderID, itemID, productID, name, qty)
	return err
}

func loadOrderServiceItems(ctx context.Context, q orderRowsQuerier, org, orderID string) ([]orderServiceItem, error) {
	rows, err := q.Query(ctx, `SELECT id::text,order_item_id::text,COALESCE(product_id::text,''),name,qty::text,destination,status FROM order_service_items WHERE organization_id=$1 AND order_id=$2 ORDER BY created_at,id`, org, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []orderServiceItem{}
	for rows.Next() {
		var item orderServiceItem
		if err = rows.Scan(&item.ID, &item.OrderItemID, &item.ProductID, &item.Name, &item.Qty, &item.Destination, &item.Status); err != nil {
			return nil, err
		}
		for _, destination := range serviceDestinations {
			if destination["value"] == item.Destination {
				item.DestinationLabel = destination["label"]
			}
		}
		item.StatusLabel = map[string]string{"nuevo": "Sin enviar", "confirmado": "Pendiente", "preparando": "Preparando", "listo": "Listo", "entregado": "Entregado"}[item.Status]
		if item.Destination == "direct" && item.Status == "listo" {
			item.StatusLabel = "Pendiente de entrega"
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func syncOrderServiceStatus(ctx context.Context, tx pgx.Tx, s scope, id string) error {
	_, err := tx.Exec(ctx, `UPDATE orders o SET status=CASE
 WHEN EXISTS(SELECT 1 FROM order_service_items i WHERE i.order_id=o.id AND i.organization_id=o.organization_id AND i.status='preparando') THEN 'preparando'
 WHEN EXISTS(SELECT 1 FROM order_service_items i WHERE i.order_id=o.id AND i.organization_id=o.organization_id AND i.status='confirmado') THEN 'confirmado'
 WHEN EXISTS(SELECT 1 FROM order_service_items i WHERE i.order_id=o.id AND i.organization_id=o.organization_id AND i.status='nuevo') THEN 'nuevo'
 WHEN EXISTS(SELECT 1 FROM order_service_items i WHERE i.order_id=o.id AND i.organization_id=o.organization_id AND i.status='listo') THEN 'listo'
 ELSE 'entregado' END,updated_at=now()
 WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3 AND o.completed_at IS NULL AND o.status NOT IN ('cancelado','en_camino')
 AND EXISTS(SELECT 1 FROM order_service_items i WHERE i.order_id=o.id AND i.organization_id=o.organization_id)`, id, s.OrganizationID, s.LocationID)
	return err
}

func (a *API) serviceOrderResponse(w http.ResponseWriter, r *http.Request, id string) {
	a.getOrder(w, rWithOrderID(r, id))
}
func rWithOrderID(r *http.Request, id string) *http.Request { r.SetPathValue("id", id); return r }

// All account/append/delivery mutations lock the same row as payment. The
// tenant/local and waiter checks occur before any stock or service write.
func lockServiceOrder(w http.ResponseWriter, r *http.Request, tx pgx.Tx, s scope) (order, bool) {
	o, err := scanOrder(tx.QueryRow(r.Context(), `SELECT `+orderColumns+` FROM orders WHERE id=$1 AND organization_id=$2 AND location_id=$3 FOR UPDATE`, r.PathValue("id"), s.OrganizationID, s.LocationID))
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "order_not_found", "El pedido no existe en este local.")
		return o, false
	}
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos consultar la atención.")
		return o, false
	}
	if !requireOrderWaiter(w, s, o.Channel, o.WaiterID) {
		return o, false
	}
	if o.CompletedAt != "" || o.Status == "cancelado" {
		fail(w, 409, "order_closed", "Esta atención ya está finalizada.")
		return o, false
	}
	return o, true
}

func (a *API) closeOrderBill(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.CloseOrderBill(w, r)
}

func (a *API) deliverOrderServiceItem(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.DeliverOrderServiceItem(w, r)
}

func (a *API) appendOrderItems(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.AppendOrderItems(w, r)
}

// The caller holds the order lock shared by append, preparation and delivery.
// Do not trust an old aggregate or bill timestamp while any product is unserved.
func requireTableProductsDelivered(w http.ResponseWriter, r *http.Request, tx pgx.Tx, s scope, orderID, status string) bool {
	var pending bool
	if err := tx.QueryRow(r.Context(), `SELECT EXISTS(
		SELECT 1 FROM order_service_items WHERE order_id=$1 AND organization_id=$2 AND status<>'entregado'
	)`, orderID, s.OrganizationID).Scan(&pending); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos verificar la entrega de los productos.")
		return false
	}
	if status != "entregado" || pending {
		fail(w, 409, "products_not_delivered", "Entrega todos los productos antes de cerrar la cuenta o cobrar.")
		return false
	}
	return true
}
