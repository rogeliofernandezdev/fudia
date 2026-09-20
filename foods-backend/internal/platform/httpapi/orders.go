package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

type orderItemSelection struct {
	GroupID   string `json:"groupId"`
	GroupName string `json:"groupName"`
	ProductID string `json:"productId"`
	Name      string `json:"name"`
	Surcharge string `json:"surcharge"`
}
type orderItem struct {
	ID         string               `json:"id"`
	ProductID  string               `json:"productId"`
	Name       string               `json:"name"`
	Qty        string               `json:"qty"`
	UnitPrice  string               `json:"unitPrice"`
	Note       string               `json:"note"`
	ItemType   string               `json:"itemType"`
	Selections []orderItemSelection `json:"selections,omitempty"`
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
type orderItemSelectionInput struct {
	GroupID   string `json:"groupId"`
	ProductID string `json:"productId"`
}
type orderItemInput struct {
	ID         string                    `json:"id"`
	ProductID  string                    `json:"productId"`
	Name       string                    `json:"name"`
	Qty        float64                   `json:"qty"`
	UnitPrice  float64                   `json:"unitPrice"`
	Note       string                    `json:"note"`
	Selections []orderItemSelectionInput `json:"selections"`
	Reprice    bool                      `json:"reprice"`
}
type preparedOrderSelection struct {
	GroupID   string
	GroupName string
	ProductID string
	Name      string
	Surcharge float64
}
type comboOptionUsageKey struct {
	ComboProductID string
	GroupName      string
	ProductID      string
}
type preparedOrderItem struct {
	ProductID  *string
	Name       string
	Qty        float64
	UnitPrice  float64
	Note       string
	ItemType   string
	Selections []preparedOrderSelection
}
type orderPreparationError struct {
	Status  int
	Code    string
	Message string
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

type orderRowsQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadOrderItems(ctx context.Context, q orderRowsQuerier, orderID, organizationID string) ([]orderItem, error) {
	rows, err := q.Query(ctx, `SELECT id,COALESCE(product_id::text,''),name,qty::text,unit_price::text,note,item_type FROM order_items WHERE order_id=$1 AND organization_id=$2 ORDER BY created_at,id`, orderID, organizationID)
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
		ORDER BY s.created_at,s.id`, orderID, organizationID)
	if err != nil {
		return nil, err
	}
	defer selectionRows.Close()
	for selectionRows.Next() {
		var itemID string
		var sel orderItemSelection
		if err := selectionRows.Scan(&itemID, &sel.GroupID, &sel.GroupName, &sel.ProductID, &sel.Name, &sel.Surcharge); err != nil {
			return nil, err
		}
		if idx, ok := indexByID[itemID]; ok {
			items[idx].Selections = append(items[idx].Selections, sel)
		}
	}
	if err := selectionRows.Err(); err != nil {
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
		var productID *string
		if productIDValue != "" {
			productID = &productIDValue
		}

		isCombo := false
		if productID != nil {
			if err := tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM menu_combos WHERE product_id=$1 AND organization_id=$2)`, *productID, s.OrganizationID).Scan(&isCombo); err != nil {
				return nil, 0, &orderPreparationError{Status: 503, Code: "order_unavailable", Message: "No pudimos validar el producto del pedido."}
			}
		}

		if !isCombo {
			if len(in.Selections) > 0 {
				return nil, 0, &orderPreparationError{Status: 400, Code: "invalid_combo_selection", Message: "Las opciones enviadas no pertenecen a un menú o combo."}
			}
			if strings.TrimSpace(in.Name) == "" || in.UnitPrice < 0 {
				return nil, 0, &orderPreparationError{Status: 400, Code: "invalid_order", Message: "Cada línea necesita producto y precio válido."}
			}
			item := preparedOrderItem{
				ProductID: productID,
				Name: strings.TrimSpace(in.Name),
				Qty: in.Qty,
				UnitPrice: in.UnitPrice,
				Note: strings.TrimSpace(in.Note),
				ItemType: "product",
			}
			prepared = append(prepared, item)
			subtotal += item.Qty * item.UnitPrice
			continue
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
						ProductID: productID,
						Name: existingName,
						Qty: in.Qty,
						UnitPrice: existingUnitPrice,
						Note: strings.TrimSpace(in.Note),
						ItemType: "combo",
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
			       AND (p.stock_mode <> 'manual' OR COALESCE(pa.daily_quota,p.default_daily_quota) IS NULL
			            OR COALESCE(pa.sold_quantity,0) < COALESCE(pa.daily_quota,p.default_daily_quota))
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
			Name string
			Surcharge float64
			Available bool
		}
		type comboGroup struct {
			ID string
			Name string
			Required bool
			Min int
			Max int
			Options map[string]comboOption
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
			         AND (op.stock_mode <> 'manual' OR COALESCE(opa.daily_quota,op.default_daily_quota) IS NULL
			              OR COALESCE(opa.sold_quantity,0) < COALESCE(opa.daily_quota,op.default_daily_quota))
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
					GroupID: group.ID,
					GroupName: group.Name,
					ProductID: option.ProductID,
					Name: option.Name,
					Surcharge: option.Surcharge,
				})
				surchargeTotal += option.Surcharge
			}
		}

		item := preparedOrderItem{
			ProductID: productID,
			Name: comboName,
			Qty: in.Qty,
			UnitPrice: basePrice + surchargeTotal,
			Note: strings.TrimSpace(in.Note),
			ItemType: "combo",
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
				Status: 409,
				Code: "combo_option_quota_exceeded",
				Message: optionName + " ya alcanzó el cupo reservado para este menú.",
			}
		}
	}
	return prepared, subtotal, nil
}

func insertPreparedOrderItems(ctx context.Context, tx pgx.Tx, organizationID, orderID string, items []preparedOrderItem) error {
	for _, it := range items {
		var itemID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO order_items(organization_id,order_id,product_id,name,qty,unit_price,note,item_type)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
			organizationID, orderID, it.ProductID, it.Name, it.Qty, it.UnitPrice, it.Note, it.ItemType).Scan(&itemID); err != nil {
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
	}
	return nil
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
	o.Items, err = loadOrderItems(r.Context(), a.db, o.ID, s.OrganizationID)
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos cargar los productos del pedido.")
		return
	}
	for _, it := range o.Items {
		if q, parseErr := strconv.ParseFloat(it.Qty, 64); parseErr == nil {
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

	prepared, subtotal, preparationErr := a.prepareOrderItems(r, tx, s, "", in.Items)
	if preparationErr != nil {
		fail(w, preparationErr.Status, preparationErr.Code, preparationErr.Message)
		return
	}
	total := subtotal + in.DeliveryFee

	var o order
	o, err = scanOrder(tx.QueryRow(r.Context(), `
		INSERT INTO orders(organization_id,location_id,channel,customer_id,customer_name,customer_phone,address,reference,table_id,notes,subtotal,delivery_fee,total,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING `+orderColumns,
		s.OrganizationID, s.LocationID, in.Channel, customerID, in.CustomerName,
		strings.TrimSpace(in.CustomerPhone), strings.TrimSpace(in.Address), strings.TrimSpace(in.Reference),
		tableID, strings.TrimSpace(in.Notes), subtotal, in.DeliveryFee, total, s.UserID))
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos guardar el pedido.")
		return
	}
	if err = insertPreparedOrderItems(r.Context(), tx, s.OrganizationID, o.ID, prepared); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos guardar los productos del pedido.")
		return
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

	prepared, subtotal, preparationErr := a.prepareOrderItems(r, tx, s, r.PathValue("id"), in.Items)
	if preparationErr != nil {
		fail(w, preparationErr.Status, preparationErr.Code, preparationErr.Message)
		return
	}
	total := subtotal + in.DeliveryFee

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
	if err = insertPreparedOrderItems(r.Context(), tx, s.OrganizationID, r.PathValue("id"), prepared); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos actualizar los productos del pedido.")
		return
	}

	o, err := scanOrder(tx.QueryRow(r.Context(), `SELECT `+orderColumns+` FROM orders WHERE id=$1 AND organization_id=$2`, r.PathValue("id"), s.OrganizationID))
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos cargar el pedido actualizado.")
		return
	}
	o.Items, err = loadOrderItems(r.Context(), tx, o.ID, s.OrganizationID)
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos cargar los productos actualizados.")
		return
	}
	for _, it := range o.Items {
		if qty, parseErr := strconv.ParseFloat(it.Qty, 64); parseErr == nil {
			o.ItemCount += int(qty)
		}
	}
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
