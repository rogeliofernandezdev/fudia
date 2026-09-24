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
	ProductID    string  `json:"productId"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Price        string  `json:"price"`
	CategoryName *string `json:"categoryName"`
	ImageURL     *string `json:"imageUrl"`
	Status       string  `json:"status"`
	IsCombo      bool    `json:"isCombo"`
}

type conciergeOrderInput struct {
	CustomerName   string           `json:"customerName"`
	CustomerPhone  string           `json:"customerPhone"`
	ConversationID string           `json:"conversationId"`
	Notes          string           `json:"notes"`
	Items          []orderItemInput `json:"items"`
}

func (a *API) resolveConciergeQR(ctx context.Context, token string) (conciergeQRContext, error) {
	var out conciergeQRContext
	err := a.db.QueryRow(ctx, `
		SELECT t.organization_id::text,t.location_id::text,t.id::text,t.name,
		       o.trade_name,l.name,p.currency_symbol
		FROM tables t
		JOIN organizations o ON o.id=t.organization_id AND o.active
		JOIN locations l ON l.id=t.location_id AND l.organization_id=t.organization_id AND l.active
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
		  AND ($4='' OR p.name ILIKE $5 OR p.description ILIKE $5)
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
			&inventoryQuantity,&minimumStock,
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
	in.Notes = strings.TrimSpace(in.Notes)
	if len(in.Items) == 0 || len(in.Items) > 50 || len(in.CustomerPhone) > 32 ||
		in.ConversationID == "" || len(in.ConversationID) > 160 || len(in.Notes) > 240 {
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
		WHERE organization_id=$1 AND location_id=$2 AND table_id=$3 AND conversation_id=$4
	`, qr.Scope.OrganizationID, qr.Scope.LocationID, qr.TableID, in.ConversationID).Scan(&existingOrderID)
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

	var occupied bool
	if err = tx.QueryRow(r.Context(), `
		SELECT EXISTS(
		  SELECT 1 FROM orders
		  WHERE organization_id=$1 AND location_id=$2 AND table_id=$3
		    AND status NOT IN ('entregado','cancelado')
		)
	`, qr.Scope.OrganizationID, qr.Scope.LocationID, qr.TableID).Scan(&occupied); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos validar la mesa.")
		return
	}
	if occupied {
		fail(w, 409, "table_occupied", "La mesa ya tiene un pedido abierto.")
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
	if err = insertPreparedOrderItems(r.Context(), tx, qr.Scope.OrganizationID, out.ID, prepared); err != nil {
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
		  organization_id,location_id,table_id,conversation_id,order_id
		)
		VALUES($1,$2,$3,$4,$5)
	`, qr.Scope.OrganizationID, qr.Scope.LocationID, qr.TableID, in.ConversationID, out.ID); err != nil {
		fail(w, 503, "order_unavailable", "No pudimos registrar la idempotencia del pedido.")
		return
	}
	if _, err = tx.Exec(r.Context(), `
		INSERT INTO audit_log(
		  organization_id,location_id,user_id,action,entity_type,entity_id,metadata
		)
		VALUES($1,$2,NULL,'concierge.order.created','order',$3,
		  jsonb_build_object('source','fudia_concierge','conversationId',$4::text,'tableId',$5::text))
	`, qr.Scope.OrganizationID, qr.Scope.LocationID, out.ID, in.ConversationID, qr.TableID); err != nil {
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
