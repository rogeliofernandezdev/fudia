package httpapi

import (
	menudomain "github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"net/http"
	"time"
)

type availabilityItem = menudomain.AvailabilityItem
type availabilityInput = menudomain.AvailabilityInput
type availabilityHistoryItem = menudomain.AvailabilityHistoryItem

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
	a.handlers.MenuCatalog.ListProductAvailability(w, r)
}
func (a *API) updateProductAvailability(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.UpdateProductAvailability(w, r)
}
func (a *API) listProductAvailabilityHistory(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.ListProductAvailabilityHistory(w, r)
}
