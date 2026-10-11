package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

type orderServiceItem struct {
	DestinationLabel string `json:"destinationLabel"`
	StatusLabel      string `json:"statusLabel"`
	ID               string `json:"id"`
	OrderItemID      string `json:"orderItemId"`
	ProductID        string `json:"productId"`
	Name             string `json:"name"`
	Qty              string `json:"qty"`
	Destination      string `json:"destination"`
	Status           string `json:"status"`
}

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
	s := r.Context().Value(scopeKey{}).(scope)
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos cerrar la cuenta.")
		return
	}
	defer tx.Rollback(r.Context())
	o, ok := lockServiceOrder(w, r, tx, s)
	if !ok {
		return
	}
	if o.Channel != "salon" {
		fail(w, 409, "table_account_required", "Esta acción corresponde a una cuenta de Salón.")
		return
	}
	if o.Status == "nuevo" {
		fail(w, 409, "order_not_submitted", "Registra el pedido antes de cerrar la cuenta.")
		return
	}
	if !requireTableProductsDelivered(w, r, tx, s, o.ID, o.Status) {
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE orders SET bill_closed_at=COALESCE(bill_closed_at,now()),bill_closed_by=COALESCE(bill_closed_by,CASE WHEN EXISTS(SELECT 1 FROM users WHERE id=$4 AND organization_id=$2) THEN $4::uuid END),updated_at=now() WHERE id=$1 AND organization_id=$2 AND location_id=$3`, o.ID, s.OrganizationID, s.LocationID, s.UserID)
	closed := false
	if err == nil {
		closed, err = completeDeliveredSalonOrder(r.Context(), tx, s, o.ID)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos cerrar la cuenta.")
		return
	}
	a.audit(r, "order.bill.closed", "order", o.ID)
	if closed {
		a.audit(r, "order.completed", "order", o.ID)
	}
	a.serviceOrderResponse(w, r, o.ID)
}

func (a *API) deliverOrderServiceItem(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos confirmar la entrega.")
		return
	}
	defer tx.Rollback(r.Context())
	o, ok := lockServiceOrder(w, r, tx, s)
	if !ok {
		return
	}
	if o.Channel != "salon" {
		fail(w, 409, "table_account_required", "La entrega por producto corresponde a la atención de Salón.")
		return
	}
	tag, err := tx.Exec(r.Context(), `UPDATE order_service_items SET status='entregado',delivered_at=now(),delivered_by=CASE WHEN EXISTS(SELECT 1 FROM users WHERE id=$4 AND organization_id=$2) THEN $4::uuid END,updated_at=now() WHERE order_id=$1 AND organization_id=$2 AND id=$3 AND status='listo'`, o.ID, s.OrganizationID, r.PathValue("itemId"), s.UserID)
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos confirmar la entrega.")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 409, "item_not_ready", "El producto no existe en esta cuenta o todavía no está listo para entregar.")
		return
	}
	closed := false
	if err = syncOrderServiceStatus(r.Context(), tx, s, o.ID); err == nil {
		closed, err = completeDeliveredSalonOrder(r.Context(), tx, s, o.ID)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos confirmar la entrega.")
		return
	}
	a.audit(r, "order.item.delivered", "order", o.ID)
	if closed {
		a.audit(r, "order.completed", "order", o.ID)
	}
	a.serviceOrderResponse(w, r, o.ID)
}

func (a *API) appendOrderItems(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		Items      []orderItemInput `json:"items"`
		RequestKey string           `json:"requestKey"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || len(in.Items) == 0 || len(in.Items) > 200 || len(strings.TrimSpace(in.RequestKey)) < 1 || len(in.RequestKey) > 160 {
		fail(w, 400, "invalid_order", "Agrega productos y un identificador de envío válido.")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos agregar productos.")
		return
	}
	defer tx.Rollback(r.Context())
	o, ok := lockServiceOrder(w, r, tx, s)
	if !ok {
		return
	}
	if o.Channel != "salon" || o.Status == "nuevo" {
		fail(w, 409, "order_not_appendable", "Registra primero el pedido de mesa.")
		return
	}
	var exists bool
	if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM order_kitchen_rounds WHERE organization_id=$1 AND order_id=$2 AND request_key=$3)`, s.OrganizationID, o.ID, in.RequestKey).Scan(&exists); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos validar el envío.")
		return
	}
	if exists {
		if err = tx.Commit(r.Context()); err != nil {
			fail(w, 503, "order_unavailable", "No pudimos consultar el envío.")
			return
		}
		a.serviceOrderResponse(w, r, o.ID)
		return
	}
	if o.BillClosedAt != "" {
		fail(w, 409, "order_bill_closed", "La cuenta está por cobrar. No se pueden agregar productos después de cerrarla.")
		return
	}
	var paid bool
	if err = tx.QueryRow(r.Context(), `SELECT `+netPaidSQL+` >= o.total AND (o.bill_closed_at IS NOT NULL OR `+netPaidSQL+` > 0) FROM orders o WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3`, o.ID, s.OrganizationID, s.LocationID).Scan(&paid); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos validar el saldo.")
		return
	}
	if paid {
		fail(w, 409, "paid_order_not_editable", "La cuenta ya está pagada. El nuevo consumo necesita una nueva atención.")
		return
	}
	if err = ensureKitchenRoundForOrder(r.Context(), tx, s, o.ID, o.Status); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos conservar las comandas anteriores.")
		return
	}
	roundID, _, err := createKitchenRound(r.Context(), tx, s, o.ID, "confirmado", "order")
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos registrar la nueva comanda.")
		return
	}
	if _, err = tx.Exec(r.Context(), `UPDATE order_kitchen_rounds SET request_key=$2 WHERE id=$1`, roundID, in.RequestKey); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos identificar el envío.")
		return
	}
	items, added, prepErr := a.prepareOrderItems(r, tx, s, "", in.Items)
	if prepErr != nil {
		fail(w, prepErr.Status, prepErr.Code, prepErr.Message)
		return
	}
	// New lines must enter Pending even when the previous order was Delivered.
	if _, err = tx.Exec(r.Context(), `UPDATE orders SET status='confirmado' WHERE id=$1 AND organization_id=$2`, o.ID, s.OrganizationID); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos abrir la comanda.")
		return
	}
	if err = insertPreparedOrderItemsForRound(r.Context(), tx, s.OrganizationID, o.ID, roundID, items); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos agregar los productos.")
		return
	}
	if quantityErr := a.applyOrderQuantityDelta(r.Context(), tx, s, o.ID, preparedQuantityUsage(items), "sale"); quantityErr != nil {
		fail(w, quantityErr.Status, quantityErr.Code, quantityErr.Message)
		return
	}
	recipeUsage, recipeErr := desiredRecipeInventoryUsage(r.Context(), tx, s, items)
	if recipeErr != nil {
		fail(w, 503, "recipe_inventory_unavailable", "No pudimos validar los insumos.")
		return
	}
	if quantityErr := applyRecipeUsageDelta(r.Context(), tx, s, o.ID, recipeUsage); quantityErr != nil {
		fail(w, quantityErr.Status, quantityErr.Code, quantityErr.Message)
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE orders SET subtotal=subtotal+$4,total=total+$4,updated_at=now() WHERE id=$1 AND organization_id=$2 AND location_id=$3`, o.ID, s.OrganizationID, s.LocationID, added)
	if err == nil {
		err = syncOrderServiceStatus(r.Context(), tx, s, o.ID)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos confirmar los productos adicionales.")
		return
	}
	a.audit(r, "order.items.added", "order", o.ID)
	a.serviceOrderResponse(w, r, o.ID)
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
