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
	ProductID    string  `json:"productId"`
	Name         string  `json:"name"`
	CategoryName *string `json:"categoryName"`
	ImageURL     *string `json:"imageUrl"`
	StockMode    string  `json:"stockMode"`
	Status       string  `json:"status"`
	Source       string  `json:"source"`
	ManualStatus string  `json:"manualStatus"`
	DailyQuota   *int    `json:"dailyQuota"`
	SoldQuantity int     `json:"soldQuantity"`
	Remaining    *int    `json:"remaining"`
	Note         string  `json:"note"`
	BusinessDate string  `json:"businessDate"`
}

type availabilityInput struct {
	Status     string `json:"status"`
	DailyQuota *int   `json:"dailyQuota"`
	Note       string `json:"note"`
}

func (a *API) businessDate(r *http.Request, s scope) (time.Time, error) {
	var day time.Time
	err := a.db.QueryRow(r.Context(), `SELECT (now() AT TIME ZONE timezone)::date FROM locations WHERE id=$1 AND organization_id=$2 AND active`, s.LocationID, s.OrganizationID).Scan(&day)
	return day, err
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
	var total int
	err = a.db.QueryRow(r.Context(), `SELECT count(*) FROM products p WHERE p.organization_id=$1 AND p.active AND p.name ILIKE $2 AND ($3='' OR p.category_id::text=$3)`, s.OrganizationID, search, categoryID).Scan(&total)
	if err != nil {
		fail(w, 503, "availability_unavailable", "No pudimos cargar la disponibilidad del menú.")
		return
	}
	rows, err := a.db.Query(r.Context(), `
		SELECT p.id,p.name,c.name,p.image_url,p.stock_mode,COALESCE(pa.daily_quota,p.default_daily_quota),
		       COALESCE(pa.sold_quantity,0),COALESCE(pa.manual_status,'available'),COALESCE(pa.note,''),
		       (p.available_from IS NULL OR now() >= p.available_from)
		       AND (p.available_until IS NULL OR now() <= p.available_until)
		       AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
		       AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time),
		       EXISTS (SELECT 1 FROM menu_combos mc WHERE mc.product_id=p.id),
		       NOT EXISTS (
		         SELECT 1
		         FROM menu_combo_groups mcg
		         WHERE mcg.combo_product_id=p.id AND mcg.required
		           AND (
		             SELECT count(*)
		             FROM menu_combo_options mco
		             JOIN products op ON op.id=mco.option_product_id AND op.organization_id=p.organization_id AND op.active
		             LEFT JOIN product_availability opa ON opa.organization_id=op.organization_id AND opa.location_id=$2
		               AND opa.product_id=op.id AND opa.business_date=$3
		             WHERE mco.group_id=mcg.id
		               AND COALESCE(opa.manual_status,'available') <> 'sold_out'
		               AND (op.available_from IS NULL OR now() >= op.available_from)
		               AND (op.available_until IS NULL OR now() <= op.available_until)
		               AND (op.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(op.available_days))
		               AND (op.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= op.available_until_time)
		               AND (op.stock_mode <> 'manual' OR COALESCE(opa.daily_quota,op.default_daily_quota) IS NULL
		                    OR COALESCE(opa.sold_quantity,0) < COALESCE(opa.daily_quota,op.default_daily_quota))
		           ) < mcg.min_selections
		       )
		FROM products p
		JOIN locations l ON l.id=$2 AND l.organization_id=p.organization_id AND l.active
		LEFT JOIN menu_categories c ON c.id=p.category_id AND c.organization_id=p.organization_id
		LEFT JOIN product_availability pa ON pa.organization_id=p.organization_id AND pa.location_id=$2 AND pa.product_id=p.id AND pa.business_date=$3
		WHERE p.organization_id=$1 AND p.active AND p.name ILIKE $4
		  AND ($5='' OR p.category_id::text=$5)
		ORDER BY c.sort_order NULLS LAST,c.name,p.name
		LIMIT $6 OFFSET $7`, s.OrganizationID, s.LocationID, day.Format("2006-01-02"), search, categoryID, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "availability_unavailable", "No pudimos cargar la disponibilidad del menú.")
		return
	}
	defer rows.Close()
	items := []availabilityItem{}
	for rows.Next() {
		var item availabilityItem
		var defaultQuota *int
		var manualStatus string
		var scheduleAvailable bool
		var isCombo, comboAvailable bool
		if err := rows.Scan(&item.ProductID, &item.Name, &item.CategoryName, &item.ImageURL, &item.StockMode, &defaultQuota, &item.SoldQuantity, &manualStatus, &item.Note, &scheduleAvailable, &isCombo, &comboAvailable); err != nil {
			fail(w, 503, "availability_unavailable", "No pudimos cargar la disponibilidad del menú.")
			return
		}
		item.BusinessDate = day.Format("2006-01-02")
		item.DailyQuota = defaultQuota
		if manualStatus == "low" {
			manualStatus = "available"
		}
		item.ManualStatus = manualStatus
		item.Status, item.Source = manualStatus, "manual_override"
		if manualStatus != "sold_out" && !scheduleAvailable {
			item.Status, item.Source = "unavailable", "schedule"
		} else if manualStatus != "sold_out" && isCombo && !comboAvailable {
			item.Status, item.Source = "sold_out", "combo_components"
		} else if manualStatus == "available" {
			item.Source = "always"
			if item.StockMode == "manual" && defaultQuota != nil {
				remaining := *defaultQuota - item.SoldQuantity
				if remaining < 0 {
					remaining = 0
				}
				item.Remaining = &remaining
				item.Source = "daily_quota"
				if remaining == 0 {
					item.Status = "sold_out"
				} else if remaining <= 3 {
					item.Status = "low"
				}
			}
		}
		items = append(items, item)
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
	if in.DailyQuota != nil && *in.DailyQuota < 0 {
		fail(w, 400, "invalid_availability", "El cupo no puede ser negativo.")
		return
	}
	productID := r.PathValue("productId")
	var stockMode string
	err = a.db.QueryRow(r.Context(), `SELECT stock_mode FROM products WHERE id=$1 AND organization_id=$2 AND active`, productID, s.OrganizationID).Scan(&stockMode)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "product_not_found", "El producto no existe o está inactivo.")
		return
	}
	if err != nil {
		fail(w, 503, "availability_unavailable", "No pudimos validar el producto.")
		return
	}
	if stockMode != "manual" {
		in.DailyQuota = nil
	} else if in.Status != "sold_out" && (in.DailyQuota == nil || *in.DailyQuota < 1) {
		fail(w, 400, "invalid_availability", "Ingresa un cupo mayor que cero para marcar el producto como disponible.")
		return
	}
	_, err = a.db.Exec(r.Context(), `
		INSERT INTO product_availability(organization_id,location_id,product_id,business_date,daily_quota,manual_status,note,updated_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (organization_id,location_id,product_id,business_date)
		DO UPDATE SET daily_quota=EXCLUDED.daily_quota,manual_status=EXCLUDED.manual_status,note=EXCLUDED.note,updated_by=EXCLUDED.updated_by,updated_at=now()`,
		s.OrganizationID, s.LocationID, productID, day.Format("2006-01-02"), in.DailyQuota, in.Status, strings.TrimSpace(in.Note), s.UserID)
	if err != nil {
		fail(w, 503, "availability_unavailable", "No pudimos actualizar la disponibilidad.")
		return
	}
	a.audit(r, "product.availability_updated", "product", productID)
	w.WriteHeader(http.StatusNoContent)
}
