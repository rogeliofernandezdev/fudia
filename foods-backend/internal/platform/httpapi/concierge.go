package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (a *API) conciergeServiceAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := strings.TrimSpace(os.Getenv("FUDIA_CONCIERGE_API_KEY"))
		provided := strings.TrimSpace(r.Header.Get("X-Fudia-Concierge-Key"))
		if expected == "" {
			fail(w, 503, "concierge_not_configured", "Fudia Concierge no está configurado.")
			return
		}
		if len(expected) != len(provided) || subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
			fail(w, 401, "concierge_unauthorized", "Credencial de Concierge inválida.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

type conciergeQRContext struct {
	Scope            scope
	TableID          string
	TableName        string
	OrganizationName string
	LocationName     string
	CurrencySymbol   string
}

type conciergeMenuItem struct {
	ProductID     string  `json:"productId"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Price         string  `json:"price"`
	CategoryName  *string `json:"categoryName"`
	ImageURL      *string `json:"imageUrl"`
	Status        string  `json:"status"`
	IsCombo       bool    `json:"isCombo"`
	HasModifiers  bool    `json:"hasModifiers"`
}

type conciergeOrderInput struct {
	CustomerName   string           `json:"customerName"`
	CustomerPhone  string           `json:"customerPhone"`
	ConversationID string           `json:"conversationId"`
	RequestID      string           `json:"requestId"`
	Notes          string           `json:"notes"`
	Items          []orderItemInput `json:"items"`
}

type conciergeBillRequestInput struct {
	ConversationID string `json:"conversationId"`
	CustomerPhone  string `json:"customerPhone"`
}

type conciergeBill struct {
	OrderID         string      `json:"orderId"`
	Code            string      `json:"code"`
	TableName       string      `json:"tableName"`
	Status          string      `json:"status"`
	CurrencySymbol  string      `json:"currencySymbol"`
	Items           []orderItem `json:"items"`
	Total           string      `json:"total"`
	PaidAmount      string      `json:"paidAmount"`
	RemainingAmount string      `json:"remainingAmount"`
	PaymentStatus   string      `json:"paymentStatus"`
}

func (a *API) resolveConciergeQR(ctx context.Context, token string) (conciergeQRContext, error) {
	var out conciergeQRContext
	err := a.db.QueryRow(ctx, `
		SELECT t.organization_id::text,t.location_id::text,t.id::text,t.name,
		       o.trade_name,l.name,p.currency_symbol
		FROM tables t
		JOIN organizations o ON o.id=t.organization_id AND o.active
		JOIN locations l ON l.id=t.location_id AND l.organization_id=t.organization_id AND l.active
		JOIN organization_modules om
		  ON om.organization_id=t.organization_id
		 AND om.module_key='whatsapp_bot'
		 AND om.active
		JOIN organization_fiscal_profiles p
		  ON p.id=l.fiscal_profile_id AND p.organization_id=l.organization_id AND p.active
		WHERE t.qr_token=$1 AND t.active AND t.qr_enabled
		LIMIT 1
	`, strings.TrimSpace(token)).Scan(
		&out.Scope.OrganizationID,
		&out.Scope.LocationID,
		&out.TableID,
		&out.TableName,
		&out.OrganizationName,
		&out.LocationName,
		&out.CurrencySymbol,
	)
	if err != nil {
		return conciergeQRContext{}, err
	}
	out.Scope.Name = "Fudia Concierge"
	return out, nil
}

func (a *API) listConciergeMenu(w http.ResponseWriter, r *http.Request) {
	ctx, err := a.resolveConciergeQR(r.Context(), r.PathValue("token"))
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "table_not_found", "La mesa no existe o el QR no está activo.")
		return
	}
	if err != nil {
		fail(w, 503, "concierge_unavailable", "No pudimos resolver el QR.")
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	productID := strings.TrimSpace(r.URL.Query().Get("productId"))
	search := "%" + q + "%"
	day, err := a.businessDate(r, ctx.Scope)
	if err != nil {
		fail(w, 503, "concierge_unavailable", "No pudimos determinar la fecha operativa.")
		return
	}

	rows, err := a.db.Query(r.Context(), `
		SELECT p.id::text,p.name,p.description,p.price::text,c.name,p.image_url,p.quantity_control,
		       pa.portion_quantity,COALESCE(pa.sold_quantity,0),COALESCE(pa.manual_status,'available'),
		       (p.available_from IS NULL OR now() >= p.available_from)
		       AND (p.available_until IS NULL OR now() <= p.available_until)
		       AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
		       AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time),
		       EXISTS(SELECT 1 FROM menu_combos mc WHERE mc.product_id=p.id AND mc.organization_id=p.organization_id),
		       EXISTS(SELECT 1 FROM product_modifier_groups pmg WHERE pmg.product_id=p.id AND pmg.organization_id=p.organization_id),
		       COALESCE(sb.quantity,0)::float8,COALESCE(ii.minimum_stock,0)::float8
		FROM products p
		JOIN locations l ON l.id=$2 AND l.organization_id=p.organization_id AND l.active
		LEFT JOIN menu_categories c ON c.id=p.category_id AND c.organization_id=p.organization_id AND c.active
		LEFT JOIN product_availability pa
		  ON pa.organization_id=p.organization_id AND pa.location_id=$2
		 AND pa.product_id=p.id AND pa.business_date=$3
		LEFT JOIN inventory_items ii
		  ON ii.organization_id=p.organization_id AND ii.product_id=p.id AND ii.active
		LEFT JOIN stock_balances sb
		  ON sb.organization_id=p.organization_id AND sb.location_id=$2 AND sb.inventory_item_id=ii.id
		WHERE p.organization_id=$1 AND p.active
		  AND ($4='' OR p.name ILIKE $5 OR p.description ILIKE $5 OR COALESCE(c.name,'') ILIKE $5)
		  AND ($6='' OR p.id::text=$6)
		ORDER BY c.sort_order NULLS LAST,c.name NULLS LAST,p.name
		LIMIT 30
	`, ctx.Scope.OrganizationID, ctx.Scope.LocationID, day.Format("2006-01-02"), q, search, productID)
	if err != nil {
		fail(w, 503, "concierge_unavailable", "No pudimos cargar la carta.")
		return
	}
	defer rows.Close()

	items := []conciergeMenuItem{}
	for rows.Next() {
		var item conciergeMenuItem
		var quantityControl, manualStatus string
		var portionQuantity *int
		var soldQuantity int
		var scheduleAvailable, isCombo bool
		var inventoryQuantity, minimumStock float64
		if err := rows.Scan(
			&item.ProductID,&item.Name,&item.Description,&item.Price,&item.CategoryName,&item.ImageURL,
			&quantityControl,&portionQuantity,&soldQuantity,&manualStatus,&scheduleAvailable,&isCombo,
			&item.HasModifiers,&inventoryQuantity,&minimumStock,
		); err != nil {
			fail(w, 503, "concierge_unavailable", "No pudimos leer la carta.")
			return
		}

		item.Status = "available"
		item.IsCombo = isCombo
		if manualStatus == "sold_out" {
			item.Status = "sold_out"
		} else if !scheduleAvailable {
			item.Status = "unavailable"
		} else {
			switch quantityControl {
			case "portions":
				if portionQuantity == nil || soldQuantity >= *portionQuantity {
					item.Status = "sold_out"
				} else if *portionQuantity-soldQuantity <= 3 {
					item.Status = "low"
				}
			case "inventory":
				if inventoryQuantity <= 0 {
					item.Status = "sold_out"
				} else if (minimumStock > 0 && inventoryQuantity <= minimumStock) || inventoryQuantity <= 3 {
					item.Status = "low"
				}
			}
		}
		if isCombo && item.Status == "available" {
			ok, comboErr := a.comboAvailableForLocation(r, ctx.Scope, item.ProductID, day)
			if comboErr != nil {
				fail(w, 503, "concierge_unavailable", "No pudimos validar un menú.")
				return
			}
			if !ok {
				item.Status = "sold_out"
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		fail(w, 503, "concierge_unavailable", "No pudimos completar la carta.")
		return
	}
	writeJSON(w, 200, map[string]any{"items":items,"currencySymbol":ctx.CurrencySymbol})
}



func (a *API) getConciergeProductModifiers(w http.ResponseWriter, r *http.Request) {
	qr, err := a.resolveConciergeQR(r.Context(), r.PathValue("token"))
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "table_not_found", "La mesa no existe o el QR no está activo.")
		return
	}
	if err != nil {
		fail(w, 503, "concierge_unavailable", "No pudimos resolver el QR.")
		return
	}

	productID := strings.TrimSpace(r.PathValue("id"))
	var exists, isCombo bool
	if err = a.db.QueryRow(r.Context(), `
		SELECT EXISTS(
		         SELECT 1 FROM products
		         WHERE id=$1 AND organization_id=$2 AND active
		       ),
		       EXISTS(
		         SELECT 1 FROM menu_combos
		         WHERE product_id=$1 AND organization_id=$2
		       )
	`, productID, qr.Scope.OrganizationID).Scan(&exists, &isCombo); err != nil {
		fail(w, 503, "modifiers_unavailable", "No pudimos validar el producto.")
		return
	}
	if !exists {
		fail(w, 404, "product_not_found", "El producto no existe o está inactivo.")
		return
	}
	if isCombo {
		fail(w, 409, "modifiers_not_supported_for_combo", "Este producto administra sus opciones como menú o combo.")
		return
	}

	out, err := loadProductModifiers(
		r.Context(), a.db, qr.Scope.OrganizationID, productID,
	)
	if err != nil {
		fail(w, 503, "modifiers_unavailable", "No pudimos cargar los modificadores.")
		return
	}
	writeJSON(w, 200, out)
}

type conciergeComboOption struct {
	ProductID string `json:"productId"`
	Name string `json:"name"`
	Surcharge string `json:"surcharge"`
	Available bool `json:"available"`
}

type conciergeComboGroup struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Required bool `json:"required"`
	MinSelections int `json:"minSelections"`
	MaxSelections int `json:"maxSelections"`
	Options []conciergeComboOption `json:"options"`
}

type conciergeComboDetail struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Description string `json:"description"`
	Price string `json:"price"`
	ImageURL *string `json:"imageUrl"`
	Groups []conciergeComboGroup `json:"groups"`
}

func (a *API) getConciergeCombo(w http.ResponseWriter, r *http.Request) {
	qr, err := a.resolveConciergeQR(r.Context(), r.PathValue("token"))
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "table_not_found", "La mesa no existe o el QR no está activo.")
		return
	}
	if err != nil {
		fail(w, 503, "concierge_unavailable", "No pudimos resolver el QR.")
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	day, err := a.businessDate(r, qr.Scope)
	if err != nil {
		fail(w, 503, "combo_unavailable", "No pudimos determinar la fecha operativa.")
		return
	}

	var out conciergeComboDetail
	out.ID = id
	var available bool
	err = a.db.QueryRow(r.Context(), `
		SELECT p.name,p.description,p.price::text,p.image_url,
		       p.active
		       AND (p.available_from IS NULL OR now() >= p.available_from)
		       AND (p.available_until IS NULL OR now() <= p.available_until)
		       AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
		       AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time)
		       AND COALESCE(pa.manual_status,'available') <> 'sold_out'
		FROM products p
		JOIN menu_combos mc ON mc.product_id=p.id AND mc.organization_id=p.organization_id
		JOIN locations l ON l.id=$3 AND l.organization_id=p.organization_id AND l.active
		LEFT JOIN product_availability pa
		  ON pa.organization_id=p.organization_id AND pa.location_id=l.id
		 AND pa.product_id=p.id AND pa.business_date=$4
		WHERE p.id::text=$1 AND p.organization_id=$2
	`, id, qr.Scope.OrganizationID, qr.Scope.LocationID, day.Format("2006-01-02")).
		Scan(&out.Name,&out.Description,&out.Price,&out.ImageURL,&available)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "combo_not_found", "El menú o combo no existe.")
		return
	}
	if err != nil {
		fail(w, 503, "combo_unavailable", "No pudimos cargar el menú.")
		return
	}
	if !available {
		fail(w, 409, "combo_unavailable", "Este menú o combo no está disponible en este momento.")
		return
	}
	viable, err := a.comboAvailableForLocation(r, qr.Scope, id, day)
	if err != nil {
		fail(w, 503, "combo_unavailable", "No pudimos validar las alternativas del menú.")
		return
	}
	if !viable {
		fail(w, 409, "combo_unavailable", "Este menú no tiene suficientes alternativas disponibles.")
		return
	}

	groupRows, err := a.db.Query(r.Context(), `
		SELECT id::text,name,required,min_selections,max_selections
		FROM menu_combo_groups
		WHERE combo_product_id::text=$1 AND organization_id=$2
		ORDER BY sort_order,id
	`, id, qr.Scope.OrganizationID)
	if err != nil {
		fail(w, 503, "combo_unavailable", "No pudimos cargar los grupos del menú.")
		return
	}
	defer groupRows.Close()

	out.Groups = []conciergeComboGroup{}
	for groupRows.Next() {
		var group conciergeComboGroup
		if err := groupRows.Scan(
			&group.ID,&group.Name,&group.Required,&group.MinSelections,&group.MaxSelections,
		); err != nil {
			fail(w, 503, "combo_unavailable", "No pudimos leer los grupos del menú.")
			return
		}

		rows, queryErr := a.db.Query(r.Context(), `
			SELECT o.option_product_id::text,p.name,o.surcharge::text,
			       p.active
			       AND (p.available_from IS NULL OR now() >= p.available_from)
			       AND (p.available_until IS NULL OR now() <= p.available_until)
			       AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
			       AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time)
			       AND COALESCE(pa.manual_status,'available') <> 'sold_out'
			       AND (
			         p.quantity_control='none'
			         OR (p.quantity_control='portions' AND pa.portion_quantity IS NOT NULL
			             AND COALESCE(pa.sold_quantity,0) < pa.portion_quantity)
			         OR (p.quantity_control='inventory' AND COALESCE((
			           SELECT sb.quantity
			           FROM inventory_items ii
			           JOIN stock_balances sb
			             ON sb.organization_id=ii.organization_id
			            AND sb.inventory_item_id=ii.id AND sb.location_id=l.id
			           WHERE ii.organization_id=p.organization_id AND ii.product_id=p.id AND ii.active
			         ),0) > 0)
			       )
			       AND (
			         o.default_quota IS NULL OR
			         COALESCE((
			           SELECT sum(oi.qty)
			           FROM order_item_combo_selections sel
			           JOIN order_items oi ON oi.id=sel.order_item_id AND oi.organization_id=sel.organization_id
			           JOIN orders ord ON ord.id=oi.order_id AND ord.organization_id=oi.organization_id
			           WHERE sel.organization_id=o.organization_id
			             AND oi.product_id=$5
			             AND lower(sel.group_name)=lower($6)
			             AND sel.option_product_id=o.option_product_id
			             AND ord.location_id=l.id
			             AND ord.status<>'cancelado'
			             AND (ord.created_at AT TIME ZONE l.timezone)::date=$4::date
			         ),0) < o.default_quota
			       )
			FROM menu_combo_options o
			JOIN products p ON p.id=o.option_product_id AND p.organization_id=o.organization_id
			JOIN locations l ON l.id=$3 AND l.organization_id=o.organization_id AND l.active
			LEFT JOIN product_availability pa
			  ON pa.organization_id=p.organization_id AND pa.location_id=l.id
			 AND pa.product_id=p.id AND pa.business_date=$4
			WHERE o.group_id::text=$1 AND o.organization_id=$2
			ORDER BY o.sort_order,o.id
		`, group.ID, qr.Scope.OrganizationID, qr.Scope.LocationID, day.Format("2006-01-02"), id, group.Name)
		if queryErr != nil {
			fail(w, 503, "combo_unavailable", "No pudimos cargar las opciones del menú.")
			return
		}
		group.Options = []conciergeComboOption{}
		for rows.Next() {
			var option conciergeComboOption
			if scanErr := rows.Scan(&option.ProductID,&option.Name,&option.Surcharge,&option.Available); scanErr != nil {
				rows.Close()
				fail(w, 503, "combo_unavailable", "No pudimos leer las opciones del menú.")
				return
			}
			group.Options = append(group.Options, option)
		}
		if rowsErr := rows.Err(); rowsErr != nil {
			rows.Close()
			fail(w, 503, "combo_unavailable", "No pudimos completar las opciones del menú.")
			return
		}
		rows.Close()
		out.Groups = append(out.Groups, group)
	}
	if err := groupRows.Err(); err != nil {
		fail(w, 503, "combo_unavailable", "No pudimos completar los grupos del menú.")
		return
	}
	writeJSON(w, 200, out)
}

func (a *API) createConciergeOrder(w http.ResponseWriter, r *http.Request) {
	qr, err := a.resolveConciergeQR(r.Context(), r.PathValue("token"))
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "table_not_found", "La mesa no existe o el QR no está activo.")
		return
	}
	if err != nil {
		fail(w, 503, "concierge_unavailable", "No pudimos resolver el QR.")
		return
	}

	var in conciergeOrderInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_order", "Revisa los datos enviados.")
		return
	}
	in.CustomerName = strings.TrimSpace(in.CustomerName)
	in.CustomerPhone = strings.TrimSpace(in.CustomerPhone)
	in.ConversationID = strings.TrimSpace(in.ConversationID)
	in.RequestID = strings.TrimSpace(in.RequestID)
	if in.RequestID == "" {
		in.RequestID = in.ConversationID
	}
	in.Notes = strings.TrimSpace(in.Notes)
	if len(in.Items) == 0 || len(in.Items) > 50 || len(in.CustomerPhone) > 32 ||
		in.ConversationID == "" || len(in.ConversationID) > 160 || in.RequestID == "" || len(in.RequestID) > 160 || len(in.Notes) > 240 {
		fail(w, 400, "invalid_order", "Revisa los productos y datos del pedido.")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos iniciar el pedido.")
		return
	}
	defer tx.Rollback(r.Context())

	var lockedTableID string
	err = tx.QueryRow(r.Context(), `
		SELECT id::text
		FROM tables
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		  AND qr_token=$4 AND active AND qr_enabled
		FOR UPDATE
	`, qr.TableID, qr.Scope.OrganizationID, qr.Scope.LocationID, r.PathValue("token")).Scan(&lockedTableID)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "table_not_found", "La mesa no existe o el QR ya no está activo.")
		return
	}
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos validar la mesa.")
		return
	}

	var existingOrderID string
	err = tx.QueryRow(r.Context(), `
		SELECT order_id::text
		FROM concierge_order_requests
		WHERE organization_id=$1 AND location_id=$2 AND table_id=$3 AND request_id=$4
	`, qr.Scope.OrganizationID, qr.Scope.LocationID, qr.TableID, in.RequestID).Scan(&existingOrderID)
	if err == nil {
		var existing order
		existing, err = scanOrder(tx.QueryRow(r.Context(), `
			SELECT `+orderColumns+`
			FROM orders
			WHERE id=$1 AND organization_id=$2 AND location_id=$3
		`, existingOrderID, qr.Scope.OrganizationID, qr.Scope.LocationID))
		if err != nil {
			fail(w, 503, "order_unavailable", "No pudimos recuperar el pedido confirmado.")
			return
		}
		existing.Items, err = loadOrderItems(r.Context(), tx, existing.ID, qr.Scope.OrganizationID)
		if err != nil {
			fail(w, 503, "order_unavailable", "No pudimos recuperar los productos del pedido.")
			return
		}
		for _, item := range existing.Items {
			if qty, parseErr := strconv.ParseFloat(item.Qty, 64); parseErr == nil {
				existing.ItemCount += int(qty)
			}
		}
		if err = applyOrderPaymentSummary(r.Context(), tx, &existing, qr.Scope.OrganizationID, qr.Scope.LocationID); err != nil {
			fail(w, 503, "order_unavailable", "No pudimos recuperar el estado del pedido.")
			return
		}
		writeJSON(w, 200, existing)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		fail(w, 503, "order_unavailable", "No pudimos validar la idempotencia del pedido.")
		return
	}

	var openOrderID,openStatus,openChannel string
	var openCreatedBy *string
	var openSubtotal,openDeliveryFee float64
	var conciergeOwned bool
	err = tx.QueryRow(r.Context(), `
		SELECT o.id::text,o.status,o.channel,o.subtotal::float8,o.delivery_fee::float8,
		       o.created_by::text,
		       EXISTS(
		         SELECT 1 FROM audit_log a
		         WHERE a.organization_id=o.organization_id
		           AND a.entity_id=o.id
		           AND a.entity_type='order'
		           AND a.action='concierge.order.created'
		       )
		FROM orders o
		WHERE o.organization_id=$1 AND o.location_id=$2 AND o.table_id=$3
		  AND o.status NOT IN ('entregado','cancelado')
		ORDER BY o.created_at DESC,o.id
		LIMIT 1
		FOR UPDATE
	`, qr.Scope.OrganizationID, qr.Scope.LocationID, qr.TableID).Scan(
		&openOrderID,&openStatus,&openChannel,&openSubtotal,&openDeliveryFee,&openCreatedBy,&conciergeOwned,
	)
	if err == nil {
		if openChannel=="whatsapp" {
			if openCreatedBy!=nil || !conciergeOwned {
				fail(w, 409, "table_occupied", "La mesa tiene una comanda abierta de WhatsApp que no pertenece a Fudia Concierge.")
				return
			}
		} else if openChannel!="salon" {
			fail(w, 409, "table_occupied", "La mesa tiene una comanda abierta gestionada por otro canal.")
			return
		}
		if openStatus!="confirmado" && openStatus!="preparando" && openStatus!="listo" {
			fail(w, 409, "concierge_order_not_editable", "La comanda ya no admite nuevas rondas desde Concierge.")
			return
		}
		paid,paidErr:=loadOrderNetPaid(r.Context(),tx,openOrderID,qr.Scope.OrganizationID,qr.Scope.LocationID)
		if paidErr!=nil {
			fail(w, 503, "order_unavailable", "No pudimos validar los pagos de la comanda.")
			return
		}
		if paid>0.00001 {
			fail(w, 409, "paid_order_not_editable", "La comanda ya tiene pagos registrados y no puede ampliarse desde Concierge.")
			return
		}

		if err=ensureKitchenRoundForOrder(r.Context(),tx,qr.Scope,openOrderID,openStatus);err!=nil {
			fail(w,503,"order_unavailable","No pudimos preparar la trazabilidad de las rondas de cocina.")
			return
		}
		roundID,_,roundErr:=createKitchenRound(r.Context(),tx,qr.Scope,openOrderID,"confirmado","concierge")
		if roundErr!=nil {
			fail(w,503,"order_unavailable","No pudimos abrir la nueva ronda de cocina.")
			return
		}
		prepared,addedSubtotal,preparationErr:=a.prepareOrderItems(r,tx,qr.Scope,"",in.Items)
		if preparationErr!=nil {
			fail(w,preparationErr.Status,preparationErr.Code,preparationErr.Message)
			return
		}
		if err=insertPreparedOrderItemsForRound(r.Context(),tx,qr.Scope.OrganizationID,openOrderID,roundID,prepared);err!=nil {
			fail(w,503,"order_unavailable","No pudimos agregar los productos a la comanda.")
			return
		}
		if quantityErr:=a.applyOrderQuantityDelta(
			r.Context(),tx,qr.Scope,openOrderID,preparedQuantityUsage(prepared),"sale",
		);quantityErr!=nil {
			fail(w,quantityErr.Status,quantityErr.Code,quantityErr.Message)
			return
		}
		recipeUsage,recipeErr:=desiredRecipeInventoryUsage(r.Context(),tx,qr.Scope,prepared)
		if recipeErr!=nil {
			fail(w,503,"recipe_inventory_unavailable","No pudimos validar el consumo de recetas.")
			return
		}
		if quantityErr:=applyRecipeUsageDelta(r.Context(),tx,qr.Scope,openOrderID,recipeUsage);quantityErr!=nil {
			fail(w,quantityErr.Status,quantityErr.Code,quantityErr.Message)
			return
		}
		nextSubtotal:=openSubtotal+addedSubtotal
		nextTotal:=nextSubtotal+openDeliveryFee
		if _,err=tx.Exec(r.Context(),`
			UPDATE orders
			SET subtotal=$4,total=$5,updated_at=now()
			WHERE id=$1 AND organization_id=$2 AND location_id=$3
		`,openOrderID,qr.Scope.OrganizationID,qr.Scope.LocationID,nextSubtotal,nextTotal);err!=nil {
			fail(w,503,"order_unavailable","No pudimos actualizar el total de la comanda.")
			return
		}
		if err=syncOrderKitchenStatus(r.Context(),tx,qr.Scope,openOrderID);err!=nil {
			fail(w,503,"order_unavailable","No pudimos actualizar el estado general de la comanda.")
			return
		}
		if _,err=tx.Exec(r.Context(),`
			INSERT INTO concierge_order_requests(
			  organization_id,location_id,table_id,conversation_id,request_id,order_id
			)
			VALUES($1,$2,$3,$4,$5,$6)
		`,qr.Scope.OrganizationID,qr.Scope.LocationID,qr.TableID,in.ConversationID,in.RequestID,openOrderID);err!=nil {
			fail(w,503,"order_unavailable","No pudimos registrar la idempotencia del pedido.")
			return
		}
		if _,err=tx.Exec(r.Context(),`
			INSERT INTO audit_log(
			  organization_id,location_id,user_id,action,entity_type,entity_id,metadata
			)
			VALUES($1,$2,NULL,'concierge.order.items_added','order',$3,
			  jsonb_build_object(
			    'source','fudia_concierge',
			    'conversationId',$4::text,
			    'requestId',$5::text,
			    'tableId',$6::text,
			    'existingChannel',$7::text
			  ))
		`,qr.Scope.OrganizationID,qr.Scope.LocationID,openOrderID,in.ConversationID,in.RequestID,qr.TableID,openChannel);err!=nil {
			fail(w,503,"order_unavailable","No pudimos registrar la trazabilidad del pedido.")
			return
		}
		if err=tx.Commit(r.Context());err!=nil {
			fail(w,503,"order_unavailable","No pudimos confirmar los productos adicionales.")
			return
		}

		var out order
		out,err=scanOrder(a.db.QueryRow(r.Context(),`
			SELECT `+orderColumns+`
			FROM orders
			WHERE id=$1 AND organization_id=$2 AND location_id=$3
		`,openOrderID,qr.Scope.OrganizationID,qr.Scope.LocationID))
		if err!=nil {
			fail(w,503,"order_unavailable","No pudimos recuperar la comanda actualizada.")
			return
		}
		out.Items,err=loadOrderItems(r.Context(),a.db,out.ID,qr.Scope.OrganizationID)
		if err!=nil {
			fail(w,503,"order_unavailable","No pudimos recuperar los productos actualizados.")
			return
		}
		for _,item:=range out.Items {
			if qty,parseErr:=strconv.ParseFloat(item.Qty,64);parseErr==nil {
				out.ItemCount+=int(qty)
			}
		}
		_ = applyOrderPaymentSummary(r.Context(),a.db,&out,qr.Scope.OrganizationID,qr.Scope.LocationID)
		writeJSON(w,200,out)
		return
	}
	if !errors.Is(err,pgx.ErrNoRows) {
		fail(w,503,"order_unavailable","No pudimos validar las comandas abiertas de la mesa.")
		return
	}

	prepared, subtotal, preparationErr := a.prepareOrderItems(r, tx, qr.Scope, "", in.Items)
	if preparationErr != nil {
		fail(w, preparationErr.Status, preparationErr.Code, preparationErr.Message)
		return
	}

	var out order
	out, err = scanOrder(tx.QueryRow(r.Context(), `
		INSERT INTO orders(
		  organization_id,location_id,channel,customer_name,customer_phone,
		  reference,table_id,notes,subtotal,delivery_fee,total,status,created_by
		)
		VALUES($1,$2,'whatsapp',$3,$4,'',$5,$6,$7,0,$7,'confirmado',NULL)
		RETURNING `+orderColumns,
		qr.Scope.OrganizationID,qr.Scope.LocationID,in.CustomerName,in.CustomerPhone,
		qr.TableID,in.Notes,subtotal,
	))
	if err != nil {
		fail(w, 503, "order_unavailable", "No pudimos guardar el pedido.")
		return
	}
	roundID, _, roundErr := createKitchenRound(r.Context(), tx, qr.Scope, out.ID, "confirmado", "concierge")
	if roundErr != nil {
		fail(w, 503, "order_unavailable", "No pudimos abrir la primera ronda de cocina.")
		return
	}
	if err = insertPreparedOrderItemsForRound(r.Context(), tx, qr.Scope.OrganizationID, out.ID, roundID, prepared); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos guardar los productos.")
		return
	}
	if quantityErr := a.applyOrderQuantityDelta(
		r.Context(), tx, qr.Scope, out.ID, preparedQuantityUsage(prepared), "sale",
	); quantityErr != nil {
		fail(w, quantityErr.Status, quantityErr.Code, quantityErr.Message)
		return
	}
	recipeUsage, recipeErr := desiredRecipeInventoryUsage(r.Context(), tx, qr.Scope, prepared)
	if recipeErr != nil {
		fail(w, 503, "recipe_inventory_unavailable", "No pudimos validar el consumo de recetas.")
		return
	}
	if quantityErr := applyRecipeUsageDelta(r.Context(), tx, qr.Scope, out.ID, recipeUsage); quantityErr != nil {
		fail(w, quantityErr.Status, quantityErr.Code, quantityErr.Message)
		return
	}
	if _, err = tx.Exec(r.Context(), `
		INSERT INTO concierge_order_requests(
		  organization_id,location_id,table_id,conversation_id,request_id,order_id
		)
		VALUES($1,$2,$3,$4,$5,$6)
	`, qr.Scope.OrganizationID, qr.Scope.LocationID, qr.TableID, in.ConversationID, in.RequestID, out.ID); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos registrar la idempotencia del pedido.")
		return
	}
	if _, err = tx.Exec(r.Context(), `
		INSERT INTO audit_log(
		  organization_id,location_id,user_id,action,entity_type,entity_id,metadata
		)
		VALUES($1,$2,NULL,'concierge.order.created','order',$3,
		  jsonb_build_object('source','fudia_concierge','conversationId',$4::text,'requestId',$5::text,'tableId',$6::text))
	`, qr.Scope.OrganizationID, qr.Scope.LocationID, out.ID, in.ConversationID, in.RequestID, qr.TableID); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos registrar la trazabilidad del pedido.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos confirmar el pedido.")
		return
	}

	out.Items, _ = loadOrderItems(r.Context(), a.db, out.ID, qr.Scope.OrganizationID)
	for _, item := range out.Items {
		if qty, parseErr := strconv.ParseFloat(item.Qty, 64); parseErr == nil {
			out.ItemCount += int(qty)
		}
	}
	_ = applyOrderPaymentSummary(r.Context(), a.db, &out, qr.Scope.OrganizationID, qr.Scope.LocationID)
	writeJSON(w, 201, out)
}


func (a *API) requestConciergeBill(w http.ResponseWriter, r *http.Request) {
	qr, err := a.resolveConciergeQR(r.Context(), r.PathValue("token"))
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "table_not_found", "La mesa no existe o el QR no está activo.")
		return
	}
	if err != nil {
		fail(w, 503, "concierge_unavailable", "No pudimos resolver el QR.")
		return
	}

	var in conciergeBillRequestInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_bill_request", "No pudimos procesar la solicitud de cuenta.")
		return
	}
	in.ConversationID = strings.TrimSpace(in.ConversationID)
	in.CustomerPhone = strings.TrimSpace(in.CustomerPhone)
	if in.ConversationID == "" || len(in.ConversationID) > 160 || len(in.CustomerPhone) > 32 {
		fail(w, 400, "invalid_bill_request", "Revisa los datos de la solicitud de cuenta.")
		return
	}

	out, err := scanOrder(a.db.QueryRow(r.Context(), `
		SELECT `+orderColumns+`
		FROM orders
		WHERE organization_id=$1 AND location_id=$2 AND table_id=$3
		  AND status NOT IN ('entregado','cancelado')
		ORDER BY created_at DESC,id
		LIMIT 1
	`, qr.Scope.OrganizationID, qr.Scope.LocationID, qr.TableID))
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "open_order_not_found", "La mesa no tiene una cuenta abierta.")
		return
	}
	if err != nil {
		fail(w, 503, "bill_unavailable", "No pudimos cargar la cuenta de la mesa.")
		return
	}
	out.Items, err = loadOrderItems(r.Context(), a.db, out.ID, qr.Scope.OrganizationID)
	if err != nil {
		fail(w, 503, "bill_unavailable", "No pudimos cargar el detalle de la cuenta.")
		return
	}
	if err = applyOrderPaymentSummary(r.Context(), a.db, &out, qr.Scope.OrganizationID, qr.Scope.LocationID); err != nil {
		fail(w, 503, "bill_unavailable", "No pudimos calcular el saldo pendiente.")
		return
	}

	if _, err = a.db.Exec(r.Context(), `
		INSERT INTO audit_log(
		  organization_id,location_id,user_id,action,entity_type,entity_id,metadata
		)
		VALUES($1,$2,NULL,'concierge.bill.requested','order',$3,
		  jsonb_build_object(
		    'source','fudia_concierge',
		    'conversationId',$4::text,
		    'tableId',$5::text,
		    'customerPhonePresent',($6::text<>'')
		  ))
	`, qr.Scope.OrganizationID, qr.Scope.LocationID, out.ID, in.ConversationID, qr.TableID, in.CustomerPhone); err != nil {
		fail(w, 503, "bill_unavailable", "No pudimos registrar la solicitud de cuenta.")
		return
	}

	writeJSON(w, 200, conciergeBill{
		OrderID: out.ID,
		Code: out.Code,
		TableName: qr.TableName,
		Status: out.Status,
		CurrencySymbol: qr.CurrencySymbol,
		Items: out.Items,
		Total: out.Total,
		PaidAmount: out.PaidAmount,
		RemainingAmount: out.RemainingAmount,
		PaymentStatus: out.PaymentStatus,
	})
}
