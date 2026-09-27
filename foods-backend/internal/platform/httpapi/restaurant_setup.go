package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type restaurantSetupCounts struct {
	Categories     int `json:"categories"`
	Products       int `json:"products"`
	Tables         int `json:"tables"`
	CashRegisters  int `json:"cashRegisters"`
	Users          int `json:"users"`
	InventoryItems int `json:"inventoryItems"`
	Recipes        int `json:"recipes"`
	Suppliers      int `json:"suppliers"`
}

type restaurantSetupView struct {
	ServiceMode string                `json:"serviceMode"`
	CompletedAt *time.Time            `json:"completedAt"`
	CoreReady   bool                  `json:"coreReady"`
	Counts      restaurantSetupCounts `json:"counts"`
	Modules     map[string]bool       `json:"modules"`
}

func validServiceMode(value string) bool {
	return value == "counter" || value == "dine_in" || value == "mixed"
}

func restaurantSetupCoreReady(serviceMode string, counts restaurantSetupCounts) bool {
	requiresTables := serviceMode == "dine_in" || serviceMode == "mixed"
	return validServiceMode(serviceMode) &&
		counts.Categories > 0 && counts.Products > 0 && counts.CashRegisters > 0 &&
		(!requiresTables || counts.Tables > 0)
}

func (a *API) loadRestaurantSetup(r *http.Request) (restaurantSetupView, error) {
	s := r.Context().Value(scopeKey{}).(scope)
	var out restaurantSetupView
	var serviceMode *string
	err := a.db.QueryRow(r.Context(), `
		SELECT ros.service_mode,ros.completed_at,
		  (SELECT count(*) FROM menu_categories mc WHERE mc.organization_id=$1 AND mc.active),
		  (SELECT count(*) FROM products p WHERE p.organization_id=$1 AND p.active AND p.category_id IS NOT NULL),
		  (SELECT count(*) FROM tables t WHERE t.organization_id=$1 AND t.location_id=$2 AND t.active),
		  (SELECT count(*) FROM cash_registers cr WHERE cr.organization_id=$1 AND cr.location_id=$2 AND cr.active),
		  (SELECT count(*) FROM users u WHERE u.organization_id=$1 AND u.active AND NOT u.platform_admin),
		  (SELECT count(*) FROM inventory_items ii WHERE ii.organization_id=$1 AND ii.active),
		  (SELECT count(*) FROM product_recipes pr WHERE pr.organization_id=$1 AND pr.active),
		  (SELECT count(*) FROM suppliers sp WHERE sp.organization_id=$1 AND sp.active)
		FROM organization_operational_setup ros
		WHERE ros.organization_id=$1
	`, s.OrganizationID, s.LocationID).Scan(
		&serviceMode, &out.CompletedAt,
		&out.Counts.Categories, &out.Counts.Products, &out.Counts.Tables,
		&out.Counts.CashRegisters, &out.Counts.Users, &out.Counts.InventoryItems,
		&out.Counts.Recipes, &out.Counts.Suppliers,
	)
	if err != nil {
		return out, err
	}
	if serviceMode != nil {
		out.ServiceMode = *serviceMode
	}
	out.Modules = a.activeModules(s.OrganizationID)
	if out.Modules == nil {
		return out, errors.New("organization modules unavailable")
	}
	out.CoreReady = restaurantSetupCoreReady(out.ServiceMode, out.Counts)
	return out, nil
}

func (a *API) getRestaurantSetup(w http.ResponseWriter, r *http.Request) {
	out, err := a.loadRestaurantSetup(r)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "restaurant_setup_not_found", "La empresa todavía no tiene una puesta en marcha registrada.")
		return
	}
	if err != nil {
		fail(w, 503, "restaurant_setup_unavailable", "No pudimos cargar la puesta en marcha del restaurante.")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) updateRestaurantSetup(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		ServiceMode string `json:"serviceMode"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_restaurant_setup", "Selecciona cómo atenderá el restaurante.")
		return
	}
	in.ServiceMode = strings.TrimSpace(in.ServiceMode)
	if !validServiceMode(in.ServiceMode) {
		fail(w, 400, "invalid_service_mode", "Selecciona atención en mostrador, salón o ambas.")
		return
	}
	tag, err := a.db.Exec(r.Context(), `
		UPDATE organization_operational_setup
		SET service_mode=$2,completed_at=NULL,completed_by=NULL,updated_at=now()
		WHERE organization_id=$1
	`, s.OrganizationID, in.ServiceMode)
	if err != nil {
		fail(w, 503, "restaurant_setup_unavailable", "No pudimos guardar el tipo de atención.")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 404, "restaurant_setup_not_found", "La empresa todavía no tiene una puesta en marcha registrada.")
		return
	}
	a.audit(r, "restaurant_setup.service_mode_updated", "organization", s.OrganizationID)
	a.getRestaurantSetup(w, r)
}

func (a *API) completeRestaurantSetup(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	out, err := a.loadRestaurantSetup(r)
	if err != nil {
		fail(w, 503, "restaurant_setup_unavailable", "No pudimos validar la puesta en marcha del restaurante.")
		return
	}
	if !out.CoreReady {
		fail(w, 409, "restaurant_setup_incomplete", "Completa el tipo de atención, la carta y las mesas requeridas antes de finalizar.")
		return
	}
	var completedAt time.Time
	err = a.db.QueryRow(r.Context(), `
		UPDATE organization_operational_setup
		SET completed_at=COALESCE(completed_at,now()),completed_by=$2,updated_at=now()
		WHERE organization_id=$1
		RETURNING completed_at
	`, s.OrganizationID, s.UserID).Scan(&completedAt)
	if err != nil {
		fail(w, 503, "restaurant_setup_unavailable", "No pudimos finalizar la puesta en marcha.")
		return
	}
	a.audit(r, "restaurant_setup.completed", "organization", s.OrganizationID)
	out.CompletedAt = &completedAt
	writeJSON(w, http.StatusOK, out)
}
