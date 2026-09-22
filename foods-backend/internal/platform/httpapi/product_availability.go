package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type availabilityItem struct {
	ProductID       string   `json:"productId"`
	Name            string   `json:"name"`
	Price           string   `json:"price"`
	CategoryID      *string  `json:"categoryId"`
	CategoryName    *string  `json:"categoryName"`
	ImageURL        *string  `json:"imageUrl"`
	QuantityControl string   `json:"quantityControl"`
	Status          string   `json:"status"`
	Source          string   `json:"source"`
	ManualStatus    string   `json:"manualStatus"`
	PortionQuantity *int     `json:"portionQuantity"`
	SoldQuantity    int      `json:"soldQuantity"`
	Remaining       *float64 `json:"remaining"`
	InventoryUnit   *string  `json:"inventoryUnit"`
	Note            string   `json:"note"`
	BusinessDate    string   `json:"businessDate"`
}

type availabilityInput struct {
	Status          string `json:"status"`
	PortionQuantity *int   `json:"portionQuantity"`
	Note            string `json:"note"`
}

func (a *API) businessDate(r *http.Request, s scope) (time.Time, error) {
	var day time.Time
	err := a.db.QueryRow(r.Context(), `SELECT (now() AT TIME ZONE timezone)::date FROM locations WHERE id=$1 AND organization_id=$2 AND active`, s.LocationID, s.OrganizationID).Scan(&day)
	return day, err
}

func (a *API) comboAvailableForLocation(r *http.Request, s scope, comboProductID string, day time.Time) (bool, error) {
	var available bool
	err := a.db.QueryRow(r.Context(), `
		SELECT NOT EXISTS (
		  SELECT 1
		  FROM menu_combo_groups g
		  WHERE g.combo_product_id=$1 AND g.organization_id=$2 AND g.required
		    AND (
		      SELECT count(*)
		      FROM menu_combo_options o
		      JOIN products p ON p.id=o.option_product_id AND p.organization_id=o.organization_id
		      LEFT JOIN product_availability pa
		        ON pa.organization_id=p.organization_id AND pa.location_id=$3
		       AND pa.product_id=p.id AND pa.business_date=$4
		      LEFT JOIN inventory_items ii
		        ON ii.organization_id=p.organization_id AND ii.product_id=p.id AND ii.active
		      LEFT JOIN stock_balances sb
		        ON sb.organization_id=p.organization_id AND sb.location_id=$3 AND sb.inventory_item_id=ii.id
		      JOIN locations l ON l.id=$3 AND l.organization_id=p.organization_id AND l.active
		      WHERE o.group_id=g.id
		        AND p.active
		        AND COALESCE(pa.manual_status,'available') <> 'sold_out'
		        AND (p.available_from IS NULL OR now() >= p.available_from)
		        AND (p.available_until IS NULL OR now() <= p.available_until)
		        AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
		        AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time)
		        AND (
		          p.quantity_control='none'
		          OR (p.quantity_control='portions' AND pa.portion_quantity IS NOT NULL
		              AND COALESCE(pa.sold_quantity,0) < pa.portion_quantity)
		          OR (p.quantity_control='inventory' AND COALESCE(sb.quantity,0) > 0)
		        )
		    ) < GREATEST(g.min_selections, CASE WHEN g.required THEN 1 ELSE 0 END)
		)`, comboProductID, s.OrganizationID, s.LocationID, day.Format("2006-01-02")).Scan(&available)
	return available, err
}

func (a *API) listProductAvailability(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	day, err := a.businessDate(r, s)
	if err != nil {
		fail(w, 503, "availability_unavailable", "No pudimos determinar la fecha operativa del local.")
		return
	}
	search := "%" + strings.TrimSpace(r.URL.Query().Get("q")) + "%"
	categoryID := r.URL.Query().Get("categoryId")
	excludeCombos := r.URL.Query().Get("excludeCombos") == "true"
	var total int
	if err = a.db.QueryRow(r.Context(), `
		SELECT count(*)
		FROM products p
		WHERE p.organization_id=$1 AND p.active AND p.name ILIKE $2
		  AND ($3='' OR p.category_id::text=$3)
		  AND (NOT $4 OR NOT EXISTS (SELECT 1 FROM menu_combos mc WHERE mc.product_id=p.id AND mc.organization_id=p.organization_id))`,
		s.OrganizationID, search, categoryID, excludeCombos).Scan(&total); err != nil {
		fail(w, 503, "availability_unavailable", "No pudimos cargar la disponibilidad de la carta.")
		return
	}

	rows, err := a.db.Query(r.Context(), `
		SELECT p.id,p.name,p.price::text,p.category_id,c.name,p.image_url,p.quantity_control,
		       pa.portion_quantity,COALESCE(pa.sold_quantity,0),COALESCE(pa.manual_status,'available'),COALESCE(pa.note,''),
		       (p.available_from IS NULL OR now() >= p.available_from)
		       AND (p.available_until IS NULL OR now() <= p.available_until)
		       AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
		       AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time),
		       EXISTS (SELECT 1 FROM menu_combos mc WHERE mc.product_id=p.id AND mc.organization_id=p.organization_id),
		       COALESCE(sb.quantity,0)::float8,ii.unit,COALESCE(ii.minimum_stock,0)::float8
		FROM products p
		JOIN locations l ON l.id=$2 AND l.organization_id=p.organization_id AND l.active
		LEFT JOIN menu_categories c ON c.id=p.category_id AND c.organization_id=p.organization_id
		LEFT JOIN product_availability pa
		  ON pa.organization_id=p.organization_id AND pa.location_id=$2
		 AND pa.product_id=p.id AND pa.business_date=$3
		LEFT JOIN inventory_items ii
		  ON ii.organization_id=p.organization_id AND ii.product_id=p.id AND ii.active
		LEFT JOIN stock_balances sb
		  ON sb.organization_id=p.organization_id AND sb.location_id=$2 AND sb.inventory_item_id=ii.id
		WHERE p.organization_id=$1 AND p.active AND p.name ILIKE $4
		  AND ($5='' OR p.category_id::text=$5)
		  AND (NOT $6 OR NOT EXISTS (SELECT 1 FROM menu_combos mc WHERE mc.product_id=p.id AND mc.organization_id=p.organization_id))
		ORDER BY c.sort_order NULLS LAST,c.name,p.name
		LIMIT $7 OFFSET $8`,
		s.OrganizationID, s.LocationID, day.Format("2006-01-02"), search, categoryID, excludeCombos, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "availability_unavailable", "No pudimos cargar la disponibilidad de la carta.")
		return
	}
	defer rows.Close()

	items := []availabilityItem{}
	for rows.Next() {
		var item availabilityItem
		var manualStatus string
		var scheduleAvailable, isCombo bool
		var inventoryQuantity, minimumStock float64
		if err := rows.Scan(
			&item.ProductID, &item.Name, &item.Price, &item.CategoryID, &item.CategoryName, &item.ImageURL, &item.QuantityControl,
			&item.PortionQuantity, &item.SoldQuantity, &manualStatus, &item.Note,
			&scheduleAvailable, &isCombo, &inventoryQuantity, &item.InventoryUnit, &minimumStock,
		); err != nil {
			fail(w, 503, "availability_unavailable", "No pudimos cargar la disponibilidad de la carta.")
			return
		}
		item.BusinessDate = day.Format("2006-01-02")
		if manualStatus == "low" {
			manualStatus = "available"
		}
		item.ManualStatus = manualStatus
		item.Status, item.Source = "available", "always"

		if manualStatus == "sold_out" {
			item.Status, item.Source = "sold_out", "manual_override"
		} else if !scheduleAvailable {
			item.Status, item.Source = "unavailable", "schedule"
		} else {
			switch item.QuantityControl {
			case "portions":
				item.Source = "portions"
				remaining := 0.0
				if item.PortionQuantity != nil {
					remaining = float64(*item.PortionQuantity - item.SoldQuantity)
					if remaining < 0 {
						remaining = 0
					}
				}
				item.Remaining = &remaining
				if item.PortionQuantity == nil || remaining <= 0 {
					item.Status = "sold_out"
				} else if remaining <= 3 {
					item.Status = "low"
				}
			case "inventory":
				item.Source = "inventory"
				remaining := inventoryQuantity
				item.Remaining = &remaining
				if inventoryQuantity <= 0 {
					item.Status = "sold_out"
				} else if (minimumStock > 0 && inventoryQuantity <= minimumStock) || inventoryQuantity <= 3 {
					item.Status = "low"
				}
			}
		}

		if isCombo && item.Status == "available" {
			comboAvailable, comboErr := a.comboAvailableForLocation(r, s, item.ProductID, day)
			if comboErr != nil {
				fail(w, 503, "availability_unavailable", "No pudimos validar las opciones de un menú.")
				return
			}
			if !comboAvailable {
				item.Status, item.Source = "sold_out", "combo_components"
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		fail(w, 503, "availability_unavailable", "No pudimos cargar la disponibilidad de la carta.")
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "businessDate": day.Format("2006-01-02"), "total": total, "page": page, "pageSize": size})
}

func (a *API) updateProductAvailability(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	day, err := a.businessDate(r, s)
	if err != nil {
		fail(w, 503, "availability_unavailable", "No pudimos determinar la fecha operativa del local.")
		return
	}
	var in availabilityInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_availability", "La disponibilidad enviada no es válida.")
		return
	}
	if in.Status != "available" && in.Status != "sold_out" {
		fail(w, 400, "invalid_availability", "Selecciona disponible o agotado.")
		return
	}
	if in.PortionQuantity != nil && *in.PortionQuantity < 0 {
		fail(w, 400, "invalid_availability", "La cantidad de porciones no puede ser negativa.")
		return
	}
	in.Note = strings.TrimSpace(in.Note)
	if len([]rune(in.Note)) > 240 {
		fail(w, 400, "invalid_availability", "La nota no puede superar 240 caracteres.")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "availability_unavailable", "No pudimos iniciar la actualización de disponibilidad.")
		return
	}
	defer tx.Rollback(r.Context())

	productID := r.PathValue("productId")
	var quantityControl string
	err = tx.QueryRow(r.Context(), `
		SELECT quantity_control
		FROM products
		WHERE id=$1 AND organization_id=$2 AND active
		FOR UPDATE`,
		productID, s.OrganizationID).Scan(&quantityControl)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "product_not_found", "El producto no existe o está inactivo.")
		return
	}
	if err != nil {
		fail(w, 503, "availability_unavailable", "No pudimos validar el producto.")
		return
	}

	var currentPortionQuantity *int
	var soldQuantity int
	err = tx.QueryRow(r.Context(), `
		SELECT portion_quantity,sold_quantity
		FROM product_availability
		WHERE organization_id=$1 AND location_id=$2 AND product_id=$3 AND business_date=$4::date
		FOR UPDATE`,
		s.OrganizationID, s.LocationID, productID, day.Format("2006-01-02")).Scan(&currentPortionQuantity, &soldQuantity)
	hasDailyAvailability := true
	if errors.Is(err, pgx.ErrNoRows) {
		hasDailyAvailability = false
		soldQuantity = 0
		err = nil
	}
	if err != nil {
		fail(w, 503, "availability_unavailable", "No pudimos validar la disponibilidad actual.")
		return
	}

	if quantityControl != "portions" {
		in.PortionQuantity = nil
	} else {
		if in.PortionQuantity == nil && hasDailyAvailability {
			in.PortionQuantity = currentPortionQuantity
		}
		if in.PortionQuantity != nil && *in.PortionQuantity < soldQuantity {
			fail(w, 409, "portion_quantity_below_sold", "El cupo de hoy no puede ser menor que las porciones ya vendidas.")
			return
		}
		if in.Status != "sold_out" && (in.PortionQuantity == nil || *in.PortionQuantity < 1) {
			fail(w, 400, "invalid_availability", "Ingresa cuántas porciones están disponibles hoy.")
			return
		}
	}

	_, err = tx.Exec(r.Context(), `
		INSERT INTO product_availability(organization_id,location_id,product_id,business_date,portion_quantity,manual_status,note,updated_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (organization_id,location_id,product_id,business_date)
		DO UPDATE SET portion_quantity=EXCLUDED.portion_quantity,manual_status=EXCLUDED.manual_status,note=EXCLUDED.note,updated_by=EXCLUDED.updated_by,updated_at=now()`,
		s.OrganizationID, s.LocationID, productID, day.Format("2006-01-02"), in.PortionQuantity, in.Status, in.Note, s.UserID)
	if err != nil {
		fail(w, 503, "availability_unavailable", "No pudimos actualizar la disponibilidad.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "availability_unavailable", "No pudimos confirmar la actualización de disponibilidad.")
		return
	}
	a.audit(r, "product.availability_updated", "product", productID)
	w.WriteHeader(http.StatusNoContent)
}
