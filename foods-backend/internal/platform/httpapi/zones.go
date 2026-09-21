package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

type zone struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder"`
	Active    bool   `json:"active"`
}

type zoneInput struct {
	Name      string `json:"name"`
	SortOrder *int   `json:"sortOrder"`
	Active    *bool  `json:"active"`
}

func (a *API) listZones(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if size < 1 {
		size = 50
	}
	rows, err := a.db.Query(r.Context(), `SELECT id,name,sort_order,active FROM zones WHERE organization_id=$1 AND location_id=$2 ORDER BY sort_order,name LIMIT $3 OFFSET $4`, s.OrganizationID, s.LocationID, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "zones_unavailable", "No pudimos cargar las zonas.")
		return
	}
	defer rows.Close()
	items := []zone{}
	for rows.Next() {
		var z zone
		if err = rows.Scan(&z.ID, &z.Name, &z.SortOrder, &z.Active); err != nil {
			fail(w, 503, "zones_unavailable", "No pudimos cargar las zonas.")
			return
		}
		items = append(items, z)
	}
	var total int
	_ = a.db.QueryRow(r.Context(), `SELECT count(*) FROM zones WHERE organization_id=$1 AND location_id=$2`, s.OrganizationID, s.LocationID).Scan(&total)
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size})
}

func (a *API) createZone(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in zoneInput
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Name) == "" {
		fail(w, 400, "invalid_request", "El nombre de la zona es obligatorio.")
		return
	}
	sortOrder := 0
	if in.SortOrder != nil {
		sortOrder = *in.SortOrder
	}
	var z zone
	err := a.db.QueryRow(r.Context(), `INSERT INTO zones(organization_id,location_id,name,sort_order,active) VALUES($1,$2,$3,$4,true) RETURNING id,name,sort_order,active`, s.OrganizationID, s.LocationID, strings.TrimSpace(in.Name), sortOrder).Scan(&z.ID, &z.Name, &z.SortOrder, &z.Active)
	if err != nil {
		fail(w, 409, "zone_conflict", "Ya existe una zona con ese nombre.")
		return
	}
	a.audit(r, "zone.created", "zone", z.ID)
	writeJSON(w, 201, z)
}

func (a *API) updateZone(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in zoneInput
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Name) == "" {
		fail(w, 400, "invalid_request", "El nombre de la zona es obligatorio.")
		return
	}
	sortOrder := 0
	if in.SortOrder != nil {
		sortOrder = *in.SortOrder
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	var z zone
	err := a.db.QueryRow(r.Context(), `UPDATE zones SET name=$4,sort_order=$5,active=$6 WHERE id=$1 AND organization_id=$2 AND location_id=$3 RETURNING id,name,sort_order,active`, r.PathValue("id"), s.OrganizationID, s.LocationID, strings.TrimSpace(in.Name), sortOrder, active).Scan(&z.ID, &z.Name, &z.SortOrder, &z.Active)
	if err == pgx.ErrNoRows {
		fail(w, 404, "zone_not_found", "La zona no existe.")
		return
	} else if err != nil {
		fail(w, 409, "zone_conflict", "Ya existe una zona con ese nombre.")
		return
	}
	a.audit(r, "zone.updated", "zone", z.ID)
	writeJSON(w, 200, z)
}

func (a *API) deactivateZone(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	ct, err := a.db.Exec(r.Context(), `UPDATE zones SET active=false WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND active=true`, r.PathValue("id"), s.OrganizationID, s.LocationID)
	if err != nil {
		fail(w, 503, "zone_unavailable", "No pudimos desactivar la zona.")
		return
	}
	if ct.RowsAffected() == 0 {
		fail(w, 404, "zone_not_found", "La zona no existe o ya está inactiva.")
		return
	}
	a.audit(r, "zone.deactivated", "zone", r.PathValue("id"))
	w.WriteHeader(204)
}
