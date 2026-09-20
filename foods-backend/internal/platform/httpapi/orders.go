package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

type orderItem struct {
	ID        string `json:"id"`
	ProductID string `json:"productId"`
	Name      string `json:"name"`
	Qty       string `json:"qty"`
	UnitPrice string `json:"unitPrice"`
	Note      string `json:"note"`
}
type order struct {
	ID            string      `json:"id"`
	Code          string      `json:"code"`
	Channel       string      `json:"channel"`
	Status        string      `json:"status"`
	CustomerID    string      `json:"customerId"`
	CustomerName  string      `json:"customerName"`
	CustomerPhone string      `json:"customerPhone"`
	Address       string      `json:"address"`
	Reference     string      `json:"reference"`
	TableID       string      `json:"tableId"`
	TableName     string      `json:"tableName"`
	Notes         string      `json:"notes"`
	Subtotal      string      `json:"subtotal"`
	DeliveryFee   string      `json:"deliveryFee"`
	Total         string      `json:"total"`
	CreatedAt     string      `json:"createdAt"`
	UpdatedAt     string      `json:"updatedAt"`
	ItemCount     int         `json:"itemCount"`
	Items         []orderItem `json:"items,omitempty"`
}
type orderItemInput struct {
	ProductID string  `json:"productId"`
	Name      string  `json:"name"`
	Qty       float64 `json:"qty"`
	UnitPrice float64 `json:"unitPrice"`
	Note      string  `json:"note"`
}
type orderInput struct {
	Channel       string           `json:"channel"`
	CustomerID    string           `json:"customerId"`
	CustomerName  string           `json:"customerName"`
	CustomerPhone string           `json:"customerPhone"`
	Address       string           `json:"address"`
	Reference     string           `json:"reference"`
	TableID       string           `json:"tableId"`
	Notes         string           `json:"notes"`
	DeliveryFee   float64          `json:"deliveryFee"`
	Items         []orderItemInput `json:"items"`
}
type orderUpdateInput struct {
	CustomerName  string           `json:"customerName"`
	CustomerPhone string           `json:"customerPhone"`
	Address       string           `json:"address"`
	Reference     string           `json:"reference"`
	Notes         string           `json:"notes"`
	DeliveryFee   float64          `json:"deliveryFee"`
	Items         []orderItemInput `json:"items"`
}

const orderColumns = `id,code,channel,status,COALESCE(customer_id::text,''),customer_name,customer_phone,address,reference,COALESCE(table_id::text,''),COALESCE((SELECT name FROM tables t WHERE t.id=orders.table_id),''),notes,subtotal::text,delivery_fee::text,total::text,to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SSOF')`

var orderChannels = []map[string]string{{"value": "salon", "label": "Salón"}, {"value": "mostrador", "label": "Mostrador"}, {"value": "recojo", "label": "Recojo"}, {"value": "delivery", "label": "Delivery"}, {"value": "whatsapp", "label": "WhatsApp"}}
var orderStatuses = []map[string]string{{"value": "nuevo", "label": "Nuevo"}, {"value": "confirmado", "label": "Confirmado"}, {"value": "preparando", "label": "Preparando"}, {"value": "listo", "label": "Listo"}, {"value": "en_camino", "label": "En camino"}, {"value": "entregado", "label": "Entregado"}, {"value": "cancelado", "label": "Cancelado"}}
var orderTransitions = map[string][]string{
	"nuevo":      {"confirmado", "cancelado"},
	"confirmado": {"preparando", "cancelado"},
	"preparando": {"listo", "cancelado"},
	"listo":      {"en_camino", "entregado", "cancelado"},
	"en_camino":  {"entregado"},
	"entregado":  {},
	"cancelado":  {},
}

func scanOrder(row pgx.Row) (order, error) {
	var o order
	err := row.Scan(&o.ID, &o.Code, &o.Channel, &o.Status, &o.CustomerID, &o.CustomerName, &o.CustomerPhone, &o.Address, &o.Reference, &o.TableID, &o.TableName, &o.Notes, &o.Subtotal, &o.DeliveryFee, &o.Total, &o.CreatedAt, &o.UpdatedAt)
	return o, err
}
func validOrderChannel(ch string) bool {
	return ch == "salon" || ch == "mostrador" || ch == "recojo" || ch == "delivery" || ch == "whatsapp"
}
func validOrderStatus(st string) bool {
	for _, s := range orderStatuses {
		if s["value"] == st {
			return true
		}
	}
	return false
}
func editableOrderStatus(st string) bool {
	return st == "nuevo" || st == "confirmado"
}

func (a *API) listOrders(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	channel := r.URL.Query().Get("channel")
	status := r.URL.Query().Get("status")
	where := `organization_id=$1 AND location_id=$2 AND ($3='' OR code ILIKE '%'||$3||'%' OR customer_name ILIKE '%'||$3||'%' OR customer_phone ILIKE '%'||$3||'%') AND ($4='' OR channel=$4) AND ($5='' OR ($5='abiertos' AND status NOT IN ('entregado','cancelado')) OR status=$5)`
	var total int
	if err := a.db.QueryRow(r.Context(), `SELECT count(*) FROM orders WHERE `+where, s.OrganizationID, s.LocationID, q, channel, status).Scan(&total); err != nil {
		fail(w, 503, "orders_unavailable", "No pudimos cargar los pedidos.")
		return
	}
	rows, err := a.db.Query(r.Context(), `SELECT `+orderColumns+` FROM orders WHERE `+where+` ORDER BY CASE WHEN status IN ('entregado','cancelado') THEN 1 ELSE 0 END, created_at DESC LIMIT $6 OFFSET $7`, s.OrganizationID, s.LocationID, q, channel, status, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "orders_unavailable", "No pudimos cargar los pedidos.")
		return
	}
	defer rows.Close()
	items := []order{}
	for rows.Next() {
		o, scanErr := scanOrder(rows)
		if scanErr != nil {
			fail(w, 503, "orders_unavailable", "No pudimos cargar los pedidos.")
			return
		}
		items = append(items, o)
	}
	channelCounts := map[string]int{}
	countRows, err := a.db.Query(r.Context(), `SELECT channel, count(*) FROM orders WHERE organization_id=$1 AND location_id=$2 AND status NOT IN ('entregado','cancelado') GROUP BY channel`, s.OrganizationID, s.LocationID)
	if err == nil {
		defer countRows.Close()
		for countRows.Next() {
			var ch string
			var n int
			if countRows.Scan(&ch, &n) == nil {
				channelCounts[ch] = n
			}
		}
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size, "channelCounts": channelCounts, "channelOptions": orderChannels, "statusOptions": orderStatuses})
}

// getOrdersFloor devuelve el plano de salón: cada mesa activa con su pedido abierto (si tiene)
func (a *API) getOrdersFloor(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	type floorTable struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Zone  string `json:"zone"`
		Seats int    `json:"seats"`
		Order *order `json:"order"`
	}
	rows, err := a.db.Query(r.Context(), `
		SELECT t.id,t.name,t.zone,t.seats,
		  o.id,COALESCE(o.code,''),COALESCE(o.channel,''),COALESCE(o.status,''),COALESCE(o.customer_id::text,''),COALESCE(o.customer_name,''),COALESCE(o.customer_phone,''),COALESCE(o.address,''),COALESCE(o.reference,''),COALESCE(o.table_id::text,''),'',COALESCE(o.notes,''),COALESCE(o.subtotal::text,'0'),COALESCE(o.delivery_fee::text,'0'),COALESCE(o.total::text,'0'),COALESCE(to_char(o.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),''),COALESCE(to_char(o.updated_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),''),COALESCE((SELECT sum(i.qty)::int FROM order_items i WHERE i.order_id=o.id),0)
		FROM tables t
		LEFT JOIN orders o ON o.table_id=t.id AND o.organization_id=t.organization_id AND o.status NOT IN ('entregado','cancelado')
		WHERE t.organization_id=$1 AND t.active
		ORDER BY t.zone,t.name`, s.OrganizationID)
	if err != nil {
		fail(w, 503, "orders_unavailable", "No pudimos cargar el salón.")
		return
	}
	defer rows.Close()
	items := []floorTable{}
	for rows.Next() {
		var ft floorTable
		var o order
		var oid *string
		if err = rows.Scan(&ft.ID, &ft.Name, &ft.Zone, &ft.Seats, &oid, &o.Code, &o.Channel, &o.Status, &o.CustomerID, &o.CustomerName, &o.CustomerPhone, &o.Address, &o.Reference, &o.TableID, &o.TableName, &o.Notes, &o.Subtotal, &o.DeliveryFee, &o.Total, &o.CreatedAt, &o.UpdatedAt, &o.ItemCount); err != nil {
			fail(w, 503, "orders_unavailable", "No pudimos cargar el salón.")
			return
		}
		if oid != nil {
			o.ID = *oid
			ft.Order = &o
		}
		items = append(items, ft)
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (a *API) getOrder(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	o, err := scanOrder(a.db.QueryRow(r.Context(), `SELECT `+orderColumns+` FROM orders WHERE id=$1 AND organization_id=$2`, r.PathValue("id"), s.OrganizationID))
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "order_not_found", "El pedido no existe.")
		return
	}
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos cargar el pedido.")
		return
	}
	o.Items = []orderItem{}
	rows, err := a.db.Query(r.Context(), `SELECT id,COALESCE(product_id::text,''),name,qty::text,unit_price::text,note FROM order_items WHERE order_id=$1 AND organization_id=$2 ORDER BY created_at`, o.ID, s.OrganizationID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var it orderItem
			if rows.Scan(&it.ID, &it.ProductID, &it.Name, &it.Qty, &it.UnitPrice, &it.Note) == nil {
				o.Items = append(o.Items, it)
			}
		}
	}
	for _, it := range o.Items {
		if q, err := strconv.ParseFloat(it.Qty, 64); err == nil {
			o.ItemCount += int(q)
		}
	}
	writeJSON(w, 200, o)
}

func (a *API) createOrder(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in orderInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_order", "Revisa los datos enviados.")
		return
	}
	in.Channel = strings.TrimSpace(in.Channel)
	in.CustomerName = strings.TrimSpace(in.CustomerName)
	if !validOrderChannel(in.Channel) || len(in.Items) == 0 || in.DeliveryFee < 0 {
		fail(w, 400, "invalid_order", "El canal y al menos un producto son obligatorios.")
		return
	}
	if in.Channel == "delivery" && strings.TrimSpace(in.Address) == "" {
		fail(w, 400, "invalid_order", "El pedido de delivery necesita una dirección.")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos guardar el pedido.")
		return
	}
	defer tx.Rollback(r.Context())
	var customerID, tableID *string
	if in.CustomerID != "" {
		var ok bool
		if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM customers WHERE id=$1 AND organization_id=$2 AND active)`, in.CustomerID, s.OrganizationID).Scan(&ok); err != nil || !ok {
			fail(w, 400, "invalid_order", "El cliente no existe.")
			return
		}
		customerID = &in.CustomerID
	}
	if in.TableID != "" {
		var ok bool
		if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM tables WHERE id=$1 AND organization_id=$2 AND active)`, in.TableID, s.OrganizationID).Scan(&ok); err != nil || !ok {
			fail(w, 400, "invalid_order", "La mesa no existe.")
			return
		}
		var occupied bool
		if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM orders WHERE organization_id=$1 AND table_id=$2 AND status NOT IN ('entregado','cancelado'))`, s.OrganizationID, in.TableID).Scan(&occupied); err != nil {
			fail(w, 503, "order_unavailable", "No pudimos guardar el pedido.")
			return
		}
		if occupied {
			fail(w, 409, "table_occupied", "La mesa ya tiene un pedido abierto.")
			return
		}
		tableID = &in.TableID
	}
	subtotal := 0.0
	for _, it := range in.Items {
		if strings.TrimSpace(it.Name) == "" || it.Qty <= 0 || it.UnitPrice < 0 {
			fail(w, 400, "invalid_order", "Cada línea necesita producto, cantidad mayor a cero y precio válido.")
			return
		}
		subtotal += it.Qty * it.UnitPrice
	}
	total := subtotal + in.DeliveryFee
	var o order
	o, err = scanOrder(tx.QueryRow(r.Context(), `INSERT INTO orders(organization_id,location_id,channel,customer_id,customer_name,customer_phone,address,reference,table_id,notes,subtotal,delivery_fee,total,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) RETURNING `+orderColumns,
		s.OrganizationID, s.LocationID, in.Channel, customerID, in.CustomerName, strings.TrimSpace(in.CustomerPhone), strings.TrimSpace(in.Address), strings.TrimSpace(in.Reference), tableID, strings.TrimSpace(in.Notes), subtotal, in.DeliveryFee, total, s.UserID))
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos guardar el pedido.")
		return
	}
	for _, it := range in.Items {
		var productID *string
		if it.ProductID != "" {
			productID = &it.ProductID
		}
		if _, err = tx.Exec(r.Context(), `INSERT INTO order_items(organization_id,order_id,product_id,name,qty,unit_price,note) VALUES($1,$2,$3,$4,$5,$6,$7)`, s.OrganizationID, o.ID, productID, strings.TrimSpace(it.Name), it.Qty, it.UnitPrice, strings.TrimSpace(it.Note)); err != nil {
			fail(w, 503, "order_unavailable", "No pudimos guardar el pedido.")
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos guardar el pedido.")
		return
	}
	a.audit(r, "created", "order", o.ID)
	writeJSON(w, 201, o)
}

func (a *API) updateOrder(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in orderUpdateInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_order", "Revisa los datos enviados.")
		return
	}
	in.CustomerName = strings.TrimSpace(in.CustomerName)
	if len(in.Items) == 0 || in.DeliveryFee < 0 {
		fail(w, 400, "invalid_order", "La comanda necesita al menos un producto y montos válidos.")
		return
	}
	subtotal := 0.0
	for _, it := range in.Items {
		if strings.TrimSpace(it.Name) == "" || it.Qty <= 0 || it.UnitPrice < 0 {
			fail(w, 400, "invalid_order", "Cada línea necesita producto, cantidad mayor a cero y precio válido.")
			return
		}
		subtotal += it.Qty * it.UnitPrice
	}
	total := subtotal + in.DeliveryFee

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos actualizar el pedido.")
		return
	}
	defer tx.Rollback(r.Context())

	var currentStatus, channel string
	err = tx.QueryRow(r.Context(), `SELECT status,channel FROM orders WHERE id=$1 AND organization_id=$2 AND location_id=$3 FOR UPDATE`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&currentStatus, &channel)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "order_not_found", "El pedido no existe.")
		return
	}
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos actualizar el pedido.")
		return
	}
	if !editableOrderStatus(currentStatus) {
		fail(w, 409, "order_not_editable", "Solo se puede editar un pedido en estado Nuevo o Confirmado.")
		return
	}
	if channel == "delivery" && strings.TrimSpace(in.Address) == "" {
		fail(w, 400, "invalid_order", "El pedido de delivery necesita una dirección.")
		return
	}

	_, err = tx.Exec(r.Context(), `UPDATE orders
		SET customer_name=$4,customer_phone=$5,address=$6,reference=$7,notes=$8,subtotal=$9,delivery_fee=$10,total=$11,updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3`,
		r.PathValue("id"), s.OrganizationID, s.LocationID,
		in.CustomerName, strings.TrimSpace(in.CustomerPhone), strings.TrimSpace(in.Address), strings.TrimSpace(in.Reference), strings.TrimSpace(in.Notes),
		subtotal, in.DeliveryFee, total)
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos actualizar el pedido.")
		return
	}
	if _, err = tx.Exec(r.Context(), `DELETE FROM order_items WHERE order_id=$1 AND organization_id=$2`, r.PathValue("id"), s.OrganizationID); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos actualizar los productos del pedido.")
		return
	}
	for _, it := range in.Items {
		var productID *string
		if it.ProductID != "" {
			productID = &it.ProductID
		}
		if _, err = tx.Exec(r.Context(), `INSERT INTO order_items(organization_id,order_id,product_id,name,qty,unit_price,note) VALUES($1,$2,$3,$4,$5,$6,$7)`,
			s.OrganizationID, r.PathValue("id"), productID, strings.TrimSpace(it.Name), it.Qty, it.UnitPrice, strings.TrimSpace(it.Note)); err != nil {
			fail(w, 503, "order_unavailable", "No pudimos actualizar los productos del pedido.")
			return
		}
	}

	o, err := scanOrder(tx.QueryRow(r.Context(), `SELECT `+orderColumns+` FROM orders WHERE id=$1 AND organization_id=$2`, r.PathValue("id"), s.OrganizationID))
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos cargar el pedido actualizado.")
		return
	}
	o.Items = []orderItem{}
	rows, err := tx.Query(r.Context(), `SELECT id,COALESCE(product_id::text,''),name,qty::text,unit_price::text,note FROM order_items WHERE order_id=$1 AND organization_id=$2 ORDER BY created_at`, o.ID, s.OrganizationID)
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos cargar los productos actualizados.")
		return
	}
	for rows.Next() {
		var it orderItem
		if err = rows.Scan(&it.ID, &it.ProductID, &it.Name, &it.Qty, &it.UnitPrice, &it.Note); err != nil {
			rows.Close()
			fail(w, 503, "order_unavailable", "No pudimos cargar los productos actualizados.")
			return
		}
		o.Items = append(o.Items, it)
		if qty, parseErr := strconv.ParseFloat(it.Qty, 64); parseErr == nil {
			o.ItemCount += int(qty)
		}
	}
	rows.Close()
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos actualizar el pedido.")
		return
	}
	a.audit(r, "updated", "order", o.ID)
	writeJSON(w, 200, o)
}

func (a *API) updateOrderStatus(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		Status string `json:"status"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || !validOrderStatus(in.Status) {
		fail(w, 400, "invalid_status", "El estado no es válido.")
		return
	}
	var current string
	err := a.db.QueryRow(r.Context(), `SELECT status FROM orders WHERE id=$1 AND organization_id=$2`, r.PathValue("id"), s.OrganizationID).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "order_not_found", "El pedido no existe.")
		return
	}
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos actualizar el pedido.")
		return
	}
	allowed := false
	for _, next := range orderTransitions[current] {
		if next == in.Status {
			allowed = true
		}
	}
	if !allowed {
		fail(w, 409, "invalid_transition", "No se puede pasar de "+current+" a "+in.Status+".")
		return
	}
	tag, err := a.db.Exec(r.Context(), `UPDATE orders SET status=$3,updated_at=now() WHERE id=$1 AND organization_id=$2`, r.PathValue("id"), s.OrganizationID, in.Status)
	if err != nil || tag.RowsAffected() == 0 {
		fail(w, 404, "order_not_found", "El pedido no existe.")
		return
	}
	a.audit(r, "status_updated", "order", r.PathValue("id"))
	o, _ := scanOrder(a.db.QueryRow(r.Context(), `SELECT `+orderColumns+` FROM orders WHERE id=$1`, r.PathValue("id")))
	writeJSON(w, 200, o)
}
