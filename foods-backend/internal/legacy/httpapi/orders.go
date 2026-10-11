package httpapi

import (
	"context"
	"errors"
	orderdomain "github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

type orderItemSelection = orderdomain.OrderItemSelection
type orderItem = orderdomain.OrderItem
type order = orderdomain.Order
type orderItemSelectionInput = orderdomain.OrderItemSelectionInput
type orderItemInput = orderdomain.OrderItemInput
type preparedOrderSelection = orderdomain.PreparedOrderSelection
type comboOptionUsageKey = orderdomain.ComboOptionUsageKey
type preparedOrderItem = orderdomain.PreparedOrderItem
type orderPreparationError struct {
	Status  int
	Code    string
	Message string
}
type orderInput = orderdomain.OrderInput
type orderUpdateInput = orderdomain.OrderUpdateInput

const orderColumns = `id,code,channel,status,COALESCE(customer_id::text,''),customer_name,customer_phone,address,reference,COALESCE(table_id::text,''),COALESCE((SELECT name FROM tables t WHERE t.id=orders.table_id),''),notes,subtotal::text,delivery_fee::text,total::text,to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),COALESCE(to_char(completed_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),''),
COALESCE(waiter_id::text,''),COALESCE((SELECT u.full_name FROM users u WHERE u.id=orders.waiter_id AND u.organization_id=orders.organization_id),''),
ARRAY(SELECT DISTINCT u.full_name FROM payments p JOIN users u ON u.id=p.created_by AND u.organization_id=p.organization_id WHERE p.order_id=orders.id AND p.organization_id=orders.organization_id AND p.location_id=orders.location_id ORDER BY u.full_name),COALESCE(to_char(bill_closed_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),'')`

var orderChannels = orderdomain.Channels
var orderStatuses = []map[string]string{{"value": "nuevo", "label": "Nuevo"}, {"value": "confirmado", "label": "Confirmado"}, {"value": "preparando", "label": "Preparando"}, {"value": "listo", "label": "Listo"}, {"value": "en_camino", "label": "En camino"}, {"value": "entregado", "label": "Entregado"}, {"value": "cancelado", "label": "Cancelado"}}
var orderTransitions = map[string][]string{
	"nuevo":      {"confirmado", "cancelado"},
	"confirmado": {"cancelado"},
	"preparando": {},
	"listo":      {"en_camino", "entregado"},
	"en_camino":  {"entregado"},
	"entregado":  {},
	"cancelado":  {},
}

func scanOrder(row pgx.Row) (order, error) {
	var o order
	err := row.Scan(&o.ID, &o.Code, &o.Channel, &o.Status, &o.CustomerID, &o.CustomerName, &o.CustomerPhone, &o.Address, &o.Reference, &o.TableID, &o.TableName, &o.Notes, &o.Subtotal, &o.DeliveryFee, &o.Total, &o.CreatedAt, &o.UpdatedAt, &o.CompletedAt, &o.WaiterID, &o.WaiterName, &o.CollectedByNames, &o.BillClosedAt)
	o.AccountState = "open"
	if o.BillClosedAt != "" {
		o.AccountState = "awaiting_payment"
	}
	return o, err
}

// Ownership supplements permissions; it does not grant orders.manage.
// Unassigned legacy/remote services retain their existing permission policy.
func canManageOrderService(userID, channel, waiterID string) bool {
	return channel != "salon" || waiterID == "" || (userID != "" && userID == waiterID)
}

func requireOrderWaiter(w http.ResponseWriter, s scope, channel, waiterID string) bool {
	if canManageOrderService(s.UserID, channel, waiterID) {
		return true
	}
	fail(w, http.StatusForbidden, "order_assigned_to_another_waiter", "Esta mesa está siendo atendida por otro mozo. Solo puedes consultar su detalle.")
	return false
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

type orderRowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadOrderNetPaid(ctx context.Context, q orderRowQuerier, orderID, organizationID, locationID string) (float64, error) {
	var paid float64
	err := q.QueryRow(ctx, `
		SELECT COALESCE(sum(
		  p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)
		),0)::float8
		FROM payments p
		WHERE p.order_id=$1 AND p.organization_id=$2 AND p.location_id=$3
	`, orderID, organizationID, locationID).Scan(&paid)
	return paid, err
}

func applyOrderPaymentSummary(ctx context.Context, q orderRowQuerier, o *order, organizationID, locationID string) error {
	paid, err := loadOrderNetPaid(ctx, q, o.ID, organizationID, locationID)
	if err != nil {
		return err
	}
	total, err := strconv.ParseFloat(o.Total, 64)
	if err != nil {
		return err
	}
	remaining := math.Max(0, total-paid)
	o.PaidAmount = strconv.FormatFloat(paid, 'f', 2, 64)
	o.RemainingAmount = strconv.FormatFloat(remaining, 'f', 2, 64)
	o.PaymentStatus = paymentStatus(total, paid)
	if o.BillClosedAt != "" && paid >= total {
		o.AccountState = "paid"
	}
	return nil
}

type orderRowsQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadOrderItems(ctx context.Context, q orderRowsQuerier, orderID, organizationID string) ([]orderItem, error) {
	return loadOrderItemsForRound(ctx, q, orderID, organizationID, "")
}

func loadOrderItemsForRound(ctx context.Context, q orderRowsQuerier, orderID, organizationID, roundID string) ([]orderItem, error) {
	rows, err := q.Query(ctx, `SELECT id,COALESCE(product_id::text,''),name,qty::text,unit_price::text,note,item_type FROM order_items WHERE order_id=$1 AND organization_id=$2 AND ($3='' OR kitchen_round_id::text=$3) ORDER BY created_at,id`, orderID, organizationID, roundID)
	if err != nil {
		return nil, err
	}
	items := []orderItem{}
	indexByID := map[string]int{}
	for rows.Next() {
		var it orderItem
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

	selectionRows, err := q.Query(ctx, `
		SELECT s.order_item_id::text,s.group_id::text,s.group_name,s.option_product_id::text,s.option_name,s.surcharge::text
		FROM order_item_combo_selections s
		JOIN order_items i ON i.id=s.order_item_id AND i.organization_id=s.organization_id
		WHERE i.order_id=$1 AND i.organization_id=$2
		  AND ($3='' OR i.kitchen_round_id::text=$3)
		ORDER BY s.created_at,s.id`, orderID, organizationID, roundID)
	if err != nil {
		return nil, err
	}
	for selectionRows.Next() {
		var itemID string
		var sel orderItemSelection
		if err := selectionRows.Scan(&itemID, &sel.GroupID, &sel.GroupName, &sel.ProductID, &sel.Name, &sel.Surcharge); err != nil {
			selectionRows.Close()
			return nil, err
		}
		if idx, ok := indexByID[itemID]; ok {
			items[idx].Selections = append(items[idx].Selections, sel)
		}
	}
	if err := selectionRows.Err(); err != nil {
		selectionRows.Close()
		return nil, err
	}
	selectionRows.Close()

	modifierRows, err := q.Query(ctx, `
		SELECT m.order_item_id::text,m.group_id::text,m.group_name,
		       m.option_id::text,m.option_name,m.surcharge::text
		FROM order_item_modifier_selections m
		JOIN order_items i ON i.id=m.order_item_id AND i.organization_id=m.organization_id
		WHERE i.order_id=$1 AND i.organization_id=$2
		  AND ($3='' OR i.kitchen_round_id::text=$3)
		ORDER BY m.created_at,m.id`, orderID, organizationID, roundID)
	if err != nil {
		return nil, err
	}
	defer modifierRows.Close()
	for modifierRows.Next() {
		var itemID string
		var modifier orderItemModifier
		if err := modifierRows.Scan(
			&itemID, &modifier.GroupID, &modifier.GroupName,
			&modifier.OptionID, &modifier.Name, &modifier.Surcharge,
		); err != nil {
			return nil, err
		}
		if idx, ok := indexByID[itemID]; ok {
			items[idx].Modifiers = append(items[idx].Modifiers, modifier)
		}
	}
	if err := modifierRows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (a *API) prepareOrderItems(r *http.Request, tx pgx.Tx, s scope, existingOrderID string, inputs []orderItemInput) ([]preparedOrderItem, float64, *orderPreparationError) {
	prepared := make([]preparedOrderItem, 0, len(inputs))
	subtotal := 0.0
	plannedOptionUsage := map[comboOptionUsageKey]float64{}

	for _, in := range inputs {
		if in.Qty <= 0 {
			return nil, 0, &orderPreparationError{Status: 400, Code: "invalid_order", Message: "Cada línea necesita una cantidad mayor a cero."}
		}
		productIDValue := strings.TrimSpace(in.ProductID)
		if productIDValue == "" {
			return nil, 0, &orderPreparationError{Status: 400, Code: "product_id_required", Message: "Cada línea del pedido debe referenciar un producto."}
		}
		productID := &productIDValue

		isCombo := false
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM menu_combos WHERE product_id=$1 AND organization_id=$2)`, *productID, s.OrganizationID).Scan(&isCombo); err != nil {
			return nil, 0, &orderPreparationError{Status: 503, Code: "order_unavailable", Message: "No pudimos validar el producto del pedido."}
		}

		if !isCombo {
			if len(in.Selections) > 0 {
				return nil, 0, &orderPreparationError{Status: 400, Code: "invalid_combo_selection", Message: "Las opciones enviadas no pertenecen a un menú o combo."}
			}
			if in.UnitPrice < 0 {
				return nil, 0, &orderPreparationError{Status: 400, Code: "invalid_order", Message: "Cada línea necesita un precio válido."}
			}
			var productName string
			var productPrice float64
			var available bool
			err := tx.QueryRow(r.Context(), `
				SELECT p.name,p.price::float8,
				       p.active
				       AND (p.available_from IS NULL OR now() >= p.available_from)
				       AND (p.available_until IS NULL OR now() <= p.available_until)
				       AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
				       AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time)
				       AND COALESCE(pa.manual_status,'available') <> 'sold_out'
				FROM products p
				JOIN locations l ON l.id=$3 AND l.organization_id=p.organization_id AND l.active
				LEFT JOIN product_availability pa ON pa.organization_id=p.organization_id AND pa.location_id=l.id
				  AND pa.product_id=p.id AND pa.business_date=(now() AT TIME ZONE l.timezone)::date
				WHERE p.id=$1 AND p.organization_id=$2`, *productID, s.OrganizationID, s.LocationID).Scan(&productName, &productPrice, &available)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, 0, &orderPreparationError{Status: 404, Code: "product_not_found", Message: "El producto ya no existe."}
			}
			if err != nil {
				return nil, 0, &orderPreparationError{Status: 503, Code: "order_unavailable", Message: "No pudimos validar el producto."}
			}
			if !available {
				return nil, 0, &orderPreparationError{Status: 409, Code: "product_unavailable", Message: productName + " no está disponible en este momento."}
			}
			modifiers, modifierSurcharge, modifierErr := prepareProductModifiers(
				r.Context(), tx, s.OrganizationID, *productID, in.Modifiers,
			)
			if modifierErr != nil {
				return nil, 0, modifierErr
			}
			item := preparedOrderItem{
				ProductID: productID,
				Name:      productName,
				Qty:       in.Qty,
				UnitPrice: productPrice + modifierSurcharge,
				Note:      strings.TrimSpace(in.Note),
				ItemType:  "product",
				Modifiers: modifiers,
			}
			prepared = append(prepared, item)
			subtotal += item.Qty * item.UnitPrice
			continue
		}

		if len(in.Modifiers) > 0 {
			return nil, 0, &orderPreparationError{
				Status: 400, Code: "invalid_modifier_selection",
				Message: "Los modificadores simples no pertenecen a un menú o combo.",
			}
		}

		if existingOrderID != "" && strings.TrimSpace(in.ID) != "" && !in.Reprice {
			var existingName string
			var existingUnitPrice float64
			var existingProductID string
			var existingType string
			err := tx.QueryRow(r.Context(), `
				SELECT name,unit_price::float8,COALESCE(product_id::text,''),item_type
				FROM order_items
				WHERE id=$1 AND order_id=$2 AND organization_id=$3`,
				strings.TrimSpace(in.ID), existingOrderID, s.OrganizationID).Scan(&existingName, &existingUnitPrice, &existingProductID, &existingType)
			if err == nil && existingType == "combo" && existingProductID == *productID {
				rows, rowsErr := tx.Query(r.Context(), `
					SELECT group_id::text,group_name,option_product_id::text,option_name,surcharge::float8
					FROM order_item_combo_selections
					WHERE order_item_id=$1 AND organization_id=$2
					ORDER BY created_at,id`, strings.TrimSpace(in.ID), s.OrganizationID)
				if rowsErr != nil {
					return nil, 0, &orderPreparationError{Status: 503, Code: "order_unavailable", Message: "No pudimos validar la configuración existente del menú."}
				}
				existingSelections := []preparedOrderSelection{}
				existingKeys := map[string]bool{}
				for rows.Next() {
					var sel preparedOrderSelection
					if scanErr := rows.Scan(&sel.GroupID, &sel.GroupName, &sel.ProductID, &sel.Name, &sel.Surcharge); scanErr != nil {
						rows.Close()
						return nil, 0, &orderPreparationError{Status: 503, Code: "order_unavailable", Message: "No pudimos validar la configuración existente del menú."}
					}
					existingSelections = append(existingSelections, sel)
					existingKeys[sel.GroupID+":"+sel.ProductID] = true
				}
				rows.Close()
				requestedKeys := map[string]bool{}
				for _, selection := range in.Selections {
					requestedKeys[strings.TrimSpace(selection.GroupID)+":"+strings.TrimSpace(selection.ProductID)] = true
				}
				sameSelections := len(in.Selections) == len(requestedKeys) && len(existingKeys) == len(requestedKeys)
				if sameSelections {
					for key := range existingKeys {
						if !requestedKeys[key] {
							sameSelections = false
							break
						}
					}
				}
				if sameSelections {
					item := preparedOrderItem{
						ProductID:  productID,
						Name:       existingName,
						Qty:        in.Qty,
						UnitPrice:  existingUnitPrice,
						Note:       strings.TrimSpace(in.Note),
						ItemType:   "combo",
						Selections: existingSelections,
					}
					prepared = append(prepared, item)
					subtotal += item.Qty * item.UnitPrice
					for _, sel := range existingSelections {
						plannedOptionUsage[comboOptionUsageKey{ComboProductID: *productID, GroupName: sel.GroupName, ProductID: sel.ProductID}] += item.Qty
					}
					continue
				}
			} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, 0, &orderPreparationError{Status: 503, Code: "order_unavailable", Message: "No pudimos validar el menú existente."}
			}
		}

		var comboName string
		var basePrice float64
		var comboAvailable bool
		err := tx.QueryRow(r.Context(), `
			SELECT p.name,p.price::float8,
			       p.active
			       AND (p.available_from IS NULL OR now() >= p.available_from)
			       AND (p.available_until IS NULL OR now() <= p.available_until)
			       AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
			       AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time)
			       AND COALESCE(pa.manual_status,'available') <> 'sold_out'
			FROM products p
			JOIN menu_combos mc ON mc.product_id=p.id AND mc.organization_id=p.organization_id
			JOIN locations l ON l.id=$3 AND l.organization_id=p.organization_id AND l.active
			LEFT JOIN product_availability pa ON pa.organization_id=p.organization_id AND pa.location_id=l.id
			  AND pa.product_id=p.id AND pa.business_date=(now() AT TIME ZONE l.timezone)::date
			WHERE p.id=$1 AND p.organization_id=$2`, *productID, s.OrganizationID, s.LocationID).Scan(&comboName, &basePrice, &comboAvailable)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, &orderPreparationError{Status: 404, Code: "combo_not_found", Message: "El menú o combo ya no existe."}
		}
		if err != nil {
			return nil, 0, &orderPreparationError{Status: 503, Code: "order_unavailable", Message: "No pudimos validar el menú o combo."}
		}
		if !comboAvailable {
			return nil, 0, &orderPreparationError{Status: 409, Code: "combo_unavailable", Message: comboName + " no está disponible en este momento."}
		}

		type comboOption struct {
			ProductID string
			Name      string
			Surcharge float64
			Available bool
		}
		type comboGroup struct {
			ID       string
			Name     string
			Required bool
			Min      int
			Max      int
			Options  map[string]comboOption
		}
		groups := map[string]*comboGroup{}
		groupOrder := []string{}
		rows, err := tx.Query(r.Context(), `
			SELECT g.id::text,g.name,g.required,g.min_selections,g.max_selections,
			       COALESCE(o.option_product_id::text,''),COALESCE(op.name,''),COALESCE(o.surcharge::float8,0),
			       CASE WHEN op.id IS NULL THEN false ELSE
			         op.active
			         AND (op.available_from IS NULL OR now() >= op.available_from)
			         AND (op.available_until IS NULL OR now() <= op.available_until)
			         AND (op.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(op.available_days))
			         AND (op.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= op.available_until_time)
			         AND COALESCE(opa.manual_status,'available') <> 'sold_out'
			         AND (
			           op.quantity_control='none'
			           OR (op.quantity_control='portions' AND opa.portion_quantity IS NOT NULL
			               AND COALESCE(opa.sold_quantity,0) < opa.portion_quantity)
			           OR (op.quantity_control='inventory' AND COALESCE((
			               SELECT sb.quantity
			               FROM inventory_items ii
			               JOIN stock_balances sb ON sb.organization_id=ii.organization_id
			                 AND sb.inventory_item_id=ii.id AND sb.location_id=l.id
			               WHERE ii.organization_id=op.organization_id AND ii.product_id=op.id AND ii.active
			             ),0) > 0)
			         )
			       END
			FROM menu_combo_groups g
			JOIN locations l ON l.id=$3 AND l.organization_id=$2 AND l.active
			LEFT JOIN menu_combo_options o ON o.group_id=g.id AND o.organization_id=g.organization_id
			LEFT JOIN products op ON op.id=o.option_product_id AND op.organization_id=g.organization_id
			LEFT JOIN product_availability opa ON opa.organization_id=g.organization_id AND opa.location_id=l.id
			  AND opa.product_id=op.id AND opa.business_date=(now() AT TIME ZONE l.timezone)::date
			WHERE g.combo_product_id=$1 AND g.organization_id=$2
			ORDER BY g.sort_order,g.id,o.sort_order,o.id`, *productID, s.OrganizationID, s.LocationID)
		if err != nil {
			return nil, 0, &orderPreparationError{Status: 503, Code: "order_unavailable", Message: "No pudimos validar las opciones del menú."}
		}
		for rows.Next() {
			var groupID, groupName, optionID, optionName string
			var required bool
			var minSel, maxSel int
			var surcharge float64
			var available bool
			if err := rows.Scan(&groupID, &groupName, &required, &minSel, &maxSel, &optionID, &optionName, &surcharge, &available); err != nil {
				rows.Close()
				return nil, 0, &orderPreparationError{Status: 503, Code: "order_unavailable", Message: "No pudimos validar las opciones del menú."}
			}
			group, ok := groups[groupID]
			if !ok {
				group = &comboGroup{ID: groupID, Name: groupName, Required: required, Min: minSel, Max: maxSel, Options: map[string]comboOption{}}
				groups[groupID] = group
				groupOrder = append(groupOrder, groupID)
			}
			if optionID != "" {
				group.Options[optionID] = comboOption{ProductID: optionID, Name: optionName, Surcharge: surcharge, Available: available}
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, 0, &orderPreparationError{Status: 503, Code: "order_unavailable", Message: "No pudimos validar las opciones del menú."}
		}

		selectedByGroup := map[string][]string{}
		seen := map[string]bool{}
		for _, selection := range in.Selections {
			groupID := strings.TrimSpace(selection.GroupID)
			optionID := strings.TrimSpace(selection.ProductID)
			if groupID == "" || optionID == "" {
				return nil, 0, &orderPreparationError{Status: 400, Code: "invalid_combo_selection", Message: "Revisa las opciones seleccionadas del menú."}
			}
			key := groupID + ":" + optionID
			if seen[key] {
				return nil, 0, &orderPreparationError{Status: 400, Code: "invalid_combo_selection", Message: "Una opción del menú está repetida."}
			}
			seen[key] = true
			selectedByGroup[groupID] = append(selectedByGroup[groupID], optionID)
		}

		selections := []preparedOrderSelection{}
		surchargeTotal := 0.0
		for groupID := range selectedByGroup {
			if _, ok := groups[groupID]; !ok {
				return nil, 0, &orderPreparationError{Status: 400, Code: "invalid_combo_selection", Message: "Una de las opciones no pertenece a este menú."}
			}
		}
		for _, groupID := range groupOrder {
			group := groups[groupID]
			selected := selectedByGroup[groupID]
			minimum := group.Min
			if group.Required && minimum < 1 {
				minimum = 1
			}
			if len(selected) < minimum || len(selected) > group.Max {
				return nil, 0, &orderPreparationError{Status: 400, Code: "invalid_combo_selection", Message: "Completa correctamente el grupo " + group.Name + "."}
			}
			for _, optionID := range selected {
				option, ok := group.Options[optionID]
				if !ok {
					return nil, 0, &orderPreparationError{Status: 400, Code: "invalid_combo_selection", Message: "Una de las opciones no pertenece a " + group.Name + "."}
				}
				if !option.Available {
					return nil, 0, &orderPreparationError{Status: 409, Code: "combo_option_unavailable", Message: option.Name + " ya no está disponible."}
				}
				selections = append(selections, preparedOrderSelection{
					GroupID:   group.ID,
					GroupName: group.Name,
					ProductID: option.ProductID,
					Name:      option.Name,
					Surcharge: option.Surcharge,
				})
				surchargeTotal += option.Surcharge
			}
		}

		item := preparedOrderItem{
			ProductID:  productID,
			Name:       comboName,
			Qty:        in.Qty,
			UnitPrice:  basePrice + surchargeTotal,
			Note:       strings.TrimSpace(in.Note),
			ItemType:   "combo",
			Selections: selections,
		}
		prepared = append(prepared, item)
		subtotal += item.Qty * item.UnitPrice
		for _, sel := range selections {
			plannedOptionUsage[comboOptionUsageKey{ComboProductID: *productID, GroupName: sel.GroupName, ProductID: sel.ProductID}] += item.Qty
		}
	}

	usageKeys := make([]comboOptionUsageKey, 0, len(plannedOptionUsage))
	for key := range plannedOptionUsage {
		usageKeys = append(usageKeys, key)
	}
	sort.Slice(usageKeys, func(i, j int) bool {
		if usageKeys[i].ComboProductID != usageKeys[j].ComboProductID {
			return usageKeys[i].ComboProductID < usageKeys[j].ComboProductID
		}
		if usageKeys[i].GroupName != usageKeys[j].GroupName {
			return usageKeys[i].GroupName < usageKeys[j].GroupName
		}
		return usageKeys[i].ProductID < usageKeys[j].ProductID
	})
	for _, key := range usageKeys {
		var existingUsage float64
		if existingOrderID != "" {
			err := tx.QueryRow(r.Context(), `
				SELECT COALESCE(sum(oi.qty),0)::float8
				FROM order_item_combo_selections ssel
				JOIN order_items oi ON oi.id=ssel.order_item_id AND oi.organization_id=ssel.organization_id
				WHERE ssel.organization_id=$1
				  AND oi.order_id=$2
				  AND oi.product_id=$3
				  AND lower(ssel.group_name)=lower($4)
				  AND ssel.option_product_id=$5`,
				s.OrganizationID, existingOrderID, key.ComboProductID, key.GroupName, key.ProductID).Scan(&existingUsage)
			if err != nil {
				return nil, 0, &orderPreparationError{Status: 503, Code: "order_unavailable", Message: "No pudimos validar el cupo existente del menú."}
			}
		}

		var quota *int
		var optionName string
		err := tx.QueryRow(r.Context(), `
			SELECT o.default_quota,p.name
			FROM menu_combo_options o
			JOIN menu_combo_groups g ON g.id=o.group_id AND g.organization_id=o.organization_id
			JOIN products p ON p.id=o.option_product_id AND p.organization_id=o.organization_id
			WHERE g.combo_product_id=$1
			  AND lower(g.name)=lower($2)
			  AND o.option_product_id=$3
			  AND o.organization_id=$4
			FOR UPDATE OF o`,
			key.ComboProductID, key.GroupName, key.ProductID, s.OrganizationID).Scan(&quota, &optionName)
		if errors.Is(err, pgx.ErrNoRows) {
			if existingOrderID != "" && plannedOptionUsage[key] <= existingUsage {
				continue
			}
			return nil, 0, &orderPreparationError{Status: 409, Code: "combo_option_unavailable", Message: "Una opción del menú ya no está disponible."}
		}
		if err != nil {
			return nil, 0, &orderPreparationError{Status: 503, Code: "order_unavailable", Message: "No pudimos validar el cupo del menú."}
		}
		if quota == nil {
			continue
		}

		var externalUsage float64
		err = tx.QueryRow(r.Context(), `
			SELECT COALESCE(sum(oi.qty),0)::float8
			FROM order_item_combo_selections ssel
			JOIN order_items oi ON oi.id=ssel.order_item_id AND oi.organization_id=ssel.organization_id
			JOIN orders ord ON ord.id=oi.order_id AND ord.organization_id=oi.organization_id
			JOIN locations l ON l.id=ord.location_id AND l.organization_id=ord.organization_id
			WHERE ssel.organization_id=$1
			  AND oi.product_id=$2
			  AND lower(ssel.group_name)=lower($3)
			  AND ssel.option_product_id=$4
			  AND ord.location_id=$5
			  AND ord.status<>'cancelado'
			  AND (ord.created_at AT TIME ZONE l.timezone)::date=(now() AT TIME ZONE l.timezone)::date
			  AND ($6='' OR ord.id::text<>$6)`,
			s.OrganizationID, key.ComboProductID, key.GroupName, key.ProductID, s.LocationID, existingOrderID).Scan(&externalUsage)
		if err != nil {
			return nil, 0, &orderPreparationError{Status: 503, Code: "order_unavailable", Message: "No pudimos validar el cupo del menú."}
		}

		allowedForOrder := float64(*quota) - externalUsage
		if allowedForOrder < 0 {
			allowedForOrder = 0
		}
		if existingUsage > allowedForOrder {
			allowedForOrder = existingUsage
		}
		if plannedOptionUsage[key] > allowedForOrder {
			return nil, 0, &orderPreparationError{
				Status:  409,
				Code:    "combo_option_quota_exceeded",
				Message: optionName + " ya alcanzó el cupo reservado para este menú.",
			}
		}
	}
	return prepared, subtotal, nil
}

func insertPreparedOrderItems(ctx context.Context, tx pgx.Tx, organizationID, orderID string, items []preparedOrderItem) error {
	return insertPreparedOrderItemsForRound(ctx, tx, organizationID, orderID, "", items)
}

func insertPreparedOrderItemsForRound(ctx context.Context, tx pgx.Tx, organizationID, orderID, roundID string, items []preparedOrderItem) error {
	for _, it := range items {
		var itemID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO order_items(organization_id,order_id,product_id,name,qty,unit_price,note,item_type,kitchen_round_id)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,'')::uuid) RETURNING id`,
			organizationID, orderID, it.ProductID, it.Name, it.Qty, it.UnitPrice, it.Note, it.ItemType, roundID).Scan(&itemID); err != nil {
			return err
		}
		for _, sel := range it.Selections {
			if _, err := tx.Exec(ctx, `
				INSERT INTO order_item_combo_selections(organization_id,order_item_id,group_id,group_name,option_product_id,option_name,surcharge)
				VALUES($1,$2,$3,$4,$5,$6,$7)`,
				organizationID, itemID, sel.GroupID, sel.GroupName, sel.ProductID, sel.Name, sel.Surcharge); err != nil {
				return err
			}
		}
		if len(it.Selections) == 0 {
			productID := ""
			if it.ProductID != nil {
				productID = *it.ProductID
			}
			if err := insertOrderServiceItem(ctx, tx, organizationID, orderID, itemID, productID, it.Name, it.Qty); err != nil {
				return err
			}
		} else {
			for _, sel := range it.Selections {
				if err := insertOrderServiceItem(ctx, tx, organizationID, orderID, itemID, sel.ProductID, sel.Name, it.Qty); err != nil {
					return err
				}
			}
		}
		for _, modifier := range it.Modifiers {
			if _, err := tx.Exec(ctx, `
				INSERT INTO order_item_modifier_selections(
				  organization_id,order_item_id,group_id,group_name,option_id,option_name,surcharge
				)
				VALUES($1,$2,$3,$4,$5,$6,$7)`,
				organizationID, itemID, modifier.GroupID, modifier.GroupName,
				modifier.OptionID, modifier.Name, modifier.Surcharge); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *API) listOrders(w http.ResponseWriter, r *http.Request) { a.handlers.Orders.ListOrders(w, r) }

// getOrdersFloor devuelve el plano de salón: cada mesa activa con su pedido abierto (si tiene)
func (a *API) getOrdersFloor(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.GetOrdersFloor(w, r)
}

func (a *API) getOrder(w http.ResponseWriter, r *http.Request) { a.handlers.Orders.GetOrder(w, r) }

func (a *API) createOrder(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.CreateOrder(w, r)
}

func (a *API) updateOrder(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.UpdateOrder(w, r)
}

func (a *API) updateOrderStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.UpdateOrderStatus(w, r)
}
