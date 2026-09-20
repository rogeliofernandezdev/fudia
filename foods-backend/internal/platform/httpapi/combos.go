package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

type comboOptionInput struct {
	ProductID string `json:"productId"`
	Surcharge string `json:"surcharge"`
	Quota     *int   `json:"quota"`
}
type comboGroupInput struct {
	Name          string             `json:"name"`
	Required      bool               `json:"required"`
	MinSelections int                `json:"minSelections"`
	MaxSelections int                `json:"maxSelections"`
	Options       []comboOptionInput `json:"options"`
}
type comboInput struct {
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Price          string            `json:"price"`
	CategoryID     *string           `json:"categoryId"`
	AvailableFrom  *string           `json:"availableFrom"`
	AvailableUntil *string           `json:"availableUntil"`
	AvailableDays  []int             `json:"availableDays"`
	Groups         []comboGroupInput `json:"groups"`
}

func normalizedSurcharge(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "0"
	}
	return value
}

func (a *API) updateComboStatus(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		Active *bool `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.Active == nil {
		fail(w, 400, "invalid_combo_status", "El estado del menú no es válido.")
		return
	}
	tag, err := a.db.Exec(r.Context(), `UPDATE products p SET active=$3,updated_at=now() FROM menu_combos c WHERE p.id=$2 AND p.organization_id=$1 AND c.product_id=p.id AND c.organization_id=p.organization_id`, s.OrganizationID, r.PathValue("id"), *in.Active)
	if err != nil {
		fail(w, 503, "combo_status_unavailable", "No pudimos actualizar el estado del menú.")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 404, "combo_not_found", "El menú o combo no existe.")
		return
	}
	action := "combo.deactivated"
	if *in.Active {
		action = "combo.activated"
	}
	a.audit(r, action, "product", r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}

func validateCombo(in comboInput) string {
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Price) == "" {
		return "Nombre y precio son obligatorios."
	}
	if len(in.Groups) == 0 {
		return "Agrega al menos un grupo al menú."
	}
	for _, group := range in.Groups {
		if strings.TrimSpace(group.Name) == "" || group.MaxSelections < 1 || group.MinSelections < 0 || group.MaxSelections < group.MinSelections {
			return "Revisa las reglas de selección de cada grupo."
		}
		if group.Required && len(group.Options) < group.MinSelections {
			return "Cada grupo obligatorio necesita alternativas suficientes."
		}
	}
	return ""
}

func (a *API) listCombos(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	search := "%" + strings.TrimSpace(r.URL.Query().Get("q")) + "%"
	status := r.URL.Query().Get("status")
	var total int
	err := a.db.QueryRow(r.Context(), `SELECT count(*) FROM menu_combos c JOIN products p ON p.id=c.product_id AND p.organization_id=c.organization_id WHERE c.organization_id=$1 AND p.name ILIKE $2 AND ($3='' OR ($3='active' AND p.active) OR ($3='inactive' AND NOT p.active))`, s.OrganizationID, search, status).Scan(&total)
	if err != nil {
		fail(w, 503, "combos_unavailable", "No pudimos cargar los menús y combos.")
		return
	}
	rows, err := a.db.Query(r.Context(), `SELECT p.id,p.name,p.description,p.price::text,p.active,count(DISTINCT g.id),count(o.id) FROM menu_combos c JOIN products p ON p.id=c.product_id AND p.organization_id=c.organization_id LEFT JOIN menu_combo_groups g ON g.combo_product_id=c.product_id AND g.organization_id=c.organization_id LEFT JOIN menu_combo_options o ON o.group_id=g.id AND o.organization_id=c.organization_id WHERE c.organization_id=$1 AND p.name ILIKE $2 AND ($3='' OR ($3='active' AND p.active) OR ($3='inactive' AND NOT p.active)) GROUP BY p.id ORDER BY p.updated_at DESC,p.name LIMIT $4 OFFSET $5`, s.OrganizationID, search, status, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "combos_unavailable", "No pudimos cargar los menús y combos.")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, name, description, price string
		var active bool
		var groups, options int
		if rows.Scan(&id, &name, &description, &price, &active, &groups, &options) != nil {
			fail(w, 503, "combos_unavailable", "No pudimos cargar los menús y combos.")
			return
		}
		items = append(items, map[string]any{"id": id, "name": name, "description": description, "price": price, "active": active, "groupCount": groups, "optionCount": options})
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size})
}

func (a *API) listOrderCombos(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	search := "%" + strings.TrimSpace(r.URL.Query().Get("q")) + "%"
	day, err := a.businessDate(r, s)
	if err != nil {
		fail(w, 503, "combos_unavailable", "No pudimos determinar la fecha operativa del local.")
		return
	}

	availabilityWhere := `
		p.active
		AND (p.available_from IS NULL OR now() >= p.available_from)
		AND (p.available_until IS NULL OR now() <= p.available_until)
		AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
		AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time)
		AND COALESCE(pa.manual_status,'available') <> 'sold_out'
		AND (p.stock_mode <> 'manual' OR COALESCE(pa.daily_quota,p.default_daily_quota) IS NULL
		     OR COALESCE(pa.sold_quantity,0) < COALESCE(pa.daily_quota,p.default_daily_quota))
		AND NOT EXISTS (
			SELECT 1
			FROM menu_combo_groups g
			WHERE g.combo_product_id=p.id AND g.organization_id=p.organization_id AND g.required
			  AND (
				SELECT count(*)
				FROM menu_combo_options o
				JOIN products op ON op.id=o.option_product_id AND op.organization_id=p.organization_id AND op.active
				LEFT JOIN product_availability opa ON opa.organization_id=op.organization_id AND opa.location_id=l.id
				  AND opa.product_id=op.id AND opa.business_date=$3
				WHERE o.group_id=g.id AND o.organization_id=p.organization_id
				  AND (op.available_from IS NULL OR now() >= op.available_from)
				  AND (op.available_until IS NULL OR now() <= op.available_until)
				  AND (op.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(op.available_days))
				  AND (op.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= op.available_until_time)
				  AND COALESCE(opa.manual_status,'available') <> 'sold_out'
				  AND (op.stock_mode <> 'manual' OR COALESCE(opa.daily_quota,op.default_daily_quota) IS NULL
				       OR COALESCE(opa.sold_quantity,0) < COALESCE(opa.daily_quota,op.default_daily_quota))
				  AND (
				    o.default_quota IS NULL OR
				    COALESCE((
				      SELECT sum(oi.qty)
				      FROM order_item_combo_selections ssel
				      JOIN order_items oi ON oi.id=ssel.order_item_id AND oi.organization_id=ssel.organization_id
				      JOIN orders ord ON ord.id=oi.order_id AND ord.organization_id=oi.organization_id
				      WHERE ssel.organization_id=p.organization_id
				        AND ssel.group_id=o.group_id
				        AND ssel.option_product_id=o.option_product_id
				        AND ord.location_id=l.id
				        AND ord.status<>'cancelado'
				        AND (ord.created_at AT TIME ZONE l.timezone)::date=$3::date
				    ),0) < o.default_quota
				  )
			  ) < GREATEST(g.min_selections,1)
		)`

	var total int
	countSQL := `
		SELECT count(*)
		FROM menu_combos c
		JOIN products p ON p.id=c.product_id AND p.organization_id=c.organization_id
		JOIN locations l ON l.id=$2 AND l.organization_id=p.organization_id AND l.active
		LEFT JOIN product_availability pa ON pa.organization_id=p.organization_id AND pa.location_id=l.id
		  AND pa.product_id=p.id AND pa.business_date=$3
		WHERE c.organization_id=$1 AND p.name ILIKE $4 AND ` + availabilityWhere
	if err = a.db.QueryRow(r.Context(), countSQL, s.OrganizationID, s.LocationID, day.Format("2006-01-02"), search).Scan(&total); err != nil {
		fail(w, 503, "combos_unavailable", "No pudimos cargar los menús y combos.")
		return
	}

	listSQL := `
		SELECT p.id,p.name,p.description,p.price::text,p.image_url,
		       (SELECT count(*) FROM menu_combo_groups g WHERE g.combo_product_id=p.id AND g.organization_id=p.organization_id),
		       (SELECT count(*) FROM menu_combo_options o JOIN menu_combo_groups g ON g.id=o.group_id
		        WHERE g.combo_product_id=p.id AND g.organization_id=p.organization_id)
		FROM menu_combos c
		JOIN products p ON p.id=c.product_id AND p.organization_id=c.organization_id
		JOIN locations l ON l.id=$2 AND l.organization_id=p.organization_id AND l.active
		LEFT JOIN product_availability pa ON pa.organization_id=p.organization_id AND pa.location_id=l.id
		  AND pa.product_id=p.id AND pa.business_date=$3
		WHERE c.organization_id=$1 AND p.name ILIKE $4 AND ` + availabilityWhere + `
		ORDER BY p.featured DESC,p.name
		LIMIT $5 OFFSET $6`
	rows, err := a.db.Query(r.Context(), listSQL, s.OrganizationID, s.LocationID, day.Format("2006-01-02"), search, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "combos_unavailable", "No pudimos cargar los menús y combos.")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, name, description, price string
		var imageURL *string
		var groups, options int
		if err := rows.Scan(&id, &name, &description, &price, &imageURL, &groups, &options); err != nil {
			fail(w, 503, "combos_unavailable", "No pudimos cargar los menús y combos.")
			return
		}
		items = append(items, map[string]any{
			"id": id, "name": name, "description": description, "price": price,
			"imageUrl": imageURL, "groupCount": groups, "optionCount": options,
		})
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size})
}

func (a *API) getOrderCombo(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	id := r.PathValue("id")
	day, err := a.businessDate(r, s)
	if err != nil {
		fail(w, 503, "combo_unavailable", "No pudimos determinar la fecha operativa del local.")
		return
	}

	var name, description, price string
	var imageURL *string
	var available bool
	err = a.db.QueryRow(r.Context(), `
		SELECT p.name,p.description,p.price::text,p.image_url,
		       p.active
		       AND (p.available_from IS NULL OR now() >= p.available_from)
		       AND (p.available_until IS NULL OR now() <= p.available_until)
		       AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
		       AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time)
		       AND COALESCE(pa.manual_status,'available') <> 'sold_out'
		       AND (p.stock_mode <> 'manual' OR COALESCE(pa.daily_quota,p.default_daily_quota) IS NULL
		            OR COALESCE(pa.sold_quantity,0) < COALESCE(pa.daily_quota,p.default_daily_quota))
		FROM products p
		JOIN menu_combos c ON c.product_id=p.id AND c.organization_id=p.organization_id
		JOIN locations l ON l.id=$3 AND l.organization_id=p.organization_id AND l.active
		LEFT JOIN product_availability pa ON pa.organization_id=p.organization_id AND pa.location_id=l.id
		  AND pa.product_id=p.id AND pa.business_date=$4
		WHERE p.id=$1 AND p.organization_id=$2`,
		id, s.OrganizationID, s.LocationID, day.Format("2006-01-02")).Scan(&name, &description, &price, &imageURL, &available)
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

	groupRows, err := a.db.Query(r.Context(), `
		SELECT g.id,g.name,g.required,g.min_selections,g.max_selections
		FROM menu_combo_groups g
		WHERE g.combo_product_id=$1 AND g.organization_id=$2
		ORDER BY g.sort_order,g.id`, id, s.OrganizationID)
	if err != nil {
		fail(w, 503, "combo_unavailable", "No pudimos cargar los grupos del menú.")
		return
	}
	defer groupRows.Close()

	groups := []map[string]any{}
	for groupRows.Next() {
		var gid, gname string
		var required bool
		var minSel, maxSel int
		if err := groupRows.Scan(&gid, &gname, &required, &minSel, &maxSel); err != nil {
			fail(w, 503, "combo_unavailable", "No pudimos cargar los grupos del menú.")
			return
		}

		optRows, err := a.db.Query(r.Context(), `
			SELECT o.option_product_id,p.name,o.surcharge::text,
			       p.active
			       AND (p.available_from IS NULL OR now() >= p.available_from)
			       AND (p.available_until IS NULL OR now() <= p.available_until)
			       AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
			       AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time)
			       AND COALESCE(pa.manual_status,'available') <> 'sold_out'
			       AND (p.stock_mode <> 'manual' OR COALESCE(pa.daily_quota,p.default_daily_quota) IS NULL
			            OR COALESCE(pa.sold_quantity,0) < COALESCE(pa.daily_quota,p.default_daily_quota))
			       AND (
			         o.default_quota IS NULL OR
			         COALESCE((
			           SELECT sum(oi.qty)
			           FROM order_item_combo_selections ssel
			           JOIN order_items oi ON oi.id=ssel.order_item_id AND oi.organization_id=ssel.organization_id
			           JOIN orders ord ON ord.id=oi.order_id AND ord.organization_id=oi.organization_id
			           WHERE ssel.organization_id=o.organization_id
			             AND ssel.group_id=o.group_id
			             AND ssel.option_product_id=o.option_product_id
			             AND ord.location_id=l.id
			             AND ord.status<>'cancelado'
			             AND (ord.created_at AT TIME ZONE l.timezone)::date=$4::date
			         ),0) < o.default_quota
			       )
			FROM menu_combo_options o
			JOIN products p ON p.id=o.option_product_id AND p.organization_id=o.organization_id
			JOIN locations l ON l.id=$3 AND l.organization_id=o.organization_id AND l.active
			LEFT JOIN product_availability pa ON pa.organization_id=p.organization_id AND pa.location_id=l.id
			  AND pa.product_id=p.id AND pa.business_date=$4
			WHERE o.group_id=$1 AND o.organization_id=$2
			ORDER BY o.sort_order,o.id`, gid, s.OrganizationID, s.LocationID, day.Format("2006-01-02"))
		if err != nil {
			fail(w, 503, "combo_unavailable", "No pudimos cargar las opciones del menú.")
			return
		}
		options := []map[string]any{}
		for optRows.Next() {
			var productID, optionName, surcharge string
			var optionAvailable bool
			if err := optRows.Scan(&productID, &optionName, &surcharge, &optionAvailable); err != nil {
				optRows.Close()
				fail(w, 503, "combo_unavailable", "No pudimos cargar las opciones del menú.")
				return
			}
			options = append(options, map[string]any{
				"productId": productID, "name": optionName, "surcharge": surcharge, "available": optionAvailable,
			})
		}
		optRows.Close()
		groups = append(groups, map[string]any{
			"id": gid, "name": gname, "required": required,
			"minSelections": minSel, "maxSelections": maxSel, "options": options,
		})
	}

	writeJSON(w, 200, map[string]any{
		"id": id, "name": name, "description": description, "price": price,
		"imageUrl": imageURL, "groups": groups,
	})
}

func (a *API) getCombo(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	id := r.PathValue("id")
	var name, description, price string
	var active bool
	var availableFrom, availableUntil *string
	var availableDays []int
	err := a.db.QueryRow(r.Context(), `SELECT p.name,p.description,p.price::text,p.active,p.available_from::text,p.available_until::text,p.available_days FROM products p JOIN menu_combos c ON c.product_id=p.id AND c.organization_id=p.organization_id WHERE p.id=$1 AND p.organization_id=$2`, id, s.OrganizationID).Scan(&name, &description, &price, &active, &availableFrom, &availableUntil, &availableDays)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "combo_not_found", "El menú o combo no existe.")
		return
	}
	if err != nil {
		fail(w, 503, "combo_unavailable", "No pudimos cargar el menú.")
		return
	}
	groupRows, err := a.db.Query(r.Context(), `SELECT g.id,g.name,g.required,g.min_selections,g.max_selections FROM menu_combo_groups g WHERE g.combo_product_id=$1 AND g.organization_id=$2 ORDER BY g.sort_order`, id, s.OrganizationID)
	if err != nil {
		fail(w, 503, "combo_unavailable", "No pudimos cargar los grupos.")
		return
	}
	defer groupRows.Close()
	groups := []map[string]any{}
	for groupRows.Next() {
		var gid, gname string
		var required bool
		var minSel, maxSel int
		if groupRows.Scan(&gid, &gname, &required, &minSel, &maxSel) != nil {
			continue
		}
		optRows, err := a.db.Query(r.Context(), `SELECT o.option_product_id,p.name,o.surcharge::text,o.default_quota FROM menu_combo_options o JOIN products p ON p.id=o.option_product_id WHERE o.group_id=$1 AND o.organization_id=$2 ORDER BY o.sort_order`, gid, s.OrganizationID)
		if err != nil {
			continue
		}
		options := []map[string]any{}
		for optRows.Next() {
			var pid, pname, surcharge string
			var quota *int
			if optRows.Scan(&pid, &pname, &surcharge, &quota) != nil {
				continue
			}
			options = append(options, map[string]any{"productId": pid, "name": pname, "surcharge": surcharge, "quota": quota})
		}
		optRows.Close()
		groups = append(groups, map[string]any{"id": gid, "name": gname, "required": required, "minSelections": minSel, "maxSelections": maxSel, "options": options})
	}
	writeJSON(w, 200, map[string]any{"id": id, "name": name, "description": description, "price": price, "active": active, "availableFrom": availableFrom, "availableUntil": availableUntil, "availableDays": availableDays, "groups": groups})
}

func (a *API) saveCombo(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in comboInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_combo", "La información del menú no es válida.")
		return
	}
	if message := validateCombo(in); message != "" {
		fail(w, 400, "invalid_combo", message)
		return
	}
	for _, group := range in.Groups {
		for _, option := range group.Options {
			if option.Quota != nil && *option.Quota > 0 {
				var stock *int
				err := a.db.QueryRow(r.Context(), `SELECT default_daily_quota FROM products WHERE id=$1 AND organization_id=$2`, option.ProductID, s.OrganizationID).Scan(&stock)
				if err != nil {
					fail(w, 400, "invalid_combo", "Uno de los productos seleccionados no existe.")
					return
				}
				if stock != nil && *option.Quota > *stock {
					fail(w, 400, "quota_exceeded", "La reserva de una opción supera el stock disponible del producto.")
					return
				}
			}
		}
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "combo_unavailable", "No pudimos iniciar el registro.")
		return
	}
	defer tx.Rollback(r.Context())
	id := r.PathValue("id")
	if id == "" {
		sku, skuErr := a.nextSKU(r, s.OrganizationID)
		if skuErr != nil {
			fail(w, 503, "combo_unavailable", "No pudimos generar el código.")
			return
		}
		err = tx.QueryRow(r.Context(), `INSERT INTO products(organization_id,category_id,sku,name,description,price,active,stock_mode,available_from,available_until,available_days) VALUES($1,$2,$3,$4,$5,$6,true,'none',$7::timestamptz,$8::timestamptz,$9) RETURNING id`, s.OrganizationID, in.CategoryID, sku, strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), in.Price, in.AvailableFrom, in.AvailableUntil, in.AvailableDays).Scan(&id)
		if err == nil {
			_, err = tx.Exec(r.Context(), `INSERT INTO menu_combos(product_id,organization_id) VALUES($1,$2)`, id, s.OrganizationID)
		}
	} else {
		tag, updateErr := tx.Exec(r.Context(), `UPDATE products SET category_id=$3,name=$4,description=$5,price=$6,available_from=$7::timestamptz,available_until=$8::timestamptz,available_days=$9,updated_at=now() WHERE id=$2 AND organization_id=$1`, s.OrganizationID, id, in.CategoryID, strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), in.Price, in.AvailableFrom, in.AvailableUntil, in.AvailableDays)
		err = updateErr
		if err == nil && tag.RowsAffected() == 0 {
			err = pgx.ErrNoRows
		}
		if err == nil {
			_, err = tx.Exec(r.Context(), `DELETE FROM menu_combo_groups WHERE combo_product_id=$1 AND organization_id=$2`, id, s.OrganizationID)
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "combo_not_found", "El menú o combo no existe.")
		return
	}
	if err != nil {
		fail(w, 409, "combo_conflict", "No pudimos guardar el menú o combo.")
		return
	}
	for groupIndex, group := range in.Groups {
		var groupID string
		err = tx.QueryRow(r.Context(), `INSERT INTO menu_combo_groups(organization_id,combo_product_id,name,required,min_selections,max_selections,sort_order) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, s.OrganizationID, id, strings.TrimSpace(group.Name), group.Required, group.MinSelections, group.MaxSelections, groupIndex).Scan(&groupID)
		if err != nil {
			fail(w, 409, "combo_conflict", "No pudimos guardar los grupos.")
			return
		}
		for optionIndex, option := range group.Options {
			tag, optionErr := tx.Exec(r.Context(), `INSERT INTO menu_combo_options(organization_id,group_id,option_product_id,surcharge,sort_order,default_quota) SELECT $1,$2,p.id,$4,$5,$6 FROM products p WHERE p.id=$3 AND p.organization_id=$1 AND p.active AND p.id<>$7`, s.OrganizationID, groupID, option.ProductID, normalizedSurcharge(option.Surcharge), optionIndex, option.Quota, id)
			if optionErr != nil {
				fail(w, 409, "combo_option_conflict", "No pudimos guardar una de las alternativas.")
				return
			}
			if tag.RowsAffected() == 0 {
				fail(w, 409, "combo_option_invalid", "Una alternativa no existe, está inactiva o crea una referencia circular.")
				return
			}
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "combo_unavailable", "No pudimos confirmar el registro.")
		return
	}
	a.audit(r, "combo.saved", "product", id)
	writeJSON(w, 201, map[string]string{"id": id})
}
