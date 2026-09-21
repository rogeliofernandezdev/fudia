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
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	rows, err := a.db.Query(r.Context(), `SELECT id,name,sort_order,active FROM zones WHERE organization_id=$1 AND location_id=$2 AND ($3='' OR ($3='active' AND active) OR ($3='inactive' AND NOT active)) ORDER BY sort_order,name LIMIT $4 OFFSET $5`, s.OrganizationID, s.LocationID, status, size, (page-1)*size)
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
	_ = a.db.QueryRow(r.Context(), `SELECT count(*) FROM zones WHERE organization_id=$1 AND location_id=$2 AND ($3='' OR ($3='active' AND active) OR ($3='inactive' AND NOT active))`, s.OrganizationID, s.LocationID, status).Scan(&total)
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
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "zone_unavailable", "No pudimos actualizar la zona.")
		return
	}
	defer tx.Rollback(r.Context())

	var oldName string
	var oldActive bool
	err = tx.QueryRow(r.Context(), `
		SELECT name,active
		FROM zones
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		FOR UPDATE
	`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&oldName, &oldActive)
	if err == pgx.ErrNoRows {
		fail(w, 404, "zone_not_found", "La zona no existe.")
		return
	}
	if err != nil {
		fail(w, 503, "zone_unavailable", "No pudimos bloquear la zona.")
		return
	}
	if oldActive && !active {
		var used bool
		if err = tx.QueryRow(r.Context(), `
			SELECT EXISTS(
				SELECT 1 FROM tables
				WHERE organization_id=$1 AND location_id=$2 AND zone=$3 AND active
			)
		`, s.OrganizationID, s.LocationID, oldName).Scan(&used); err != nil {
			fail(w, 503, "zone_unavailable", "No pudimos validar las mesas de la zona.")
			return
		}
		if used {
			fail(w, 409, "zone_in_use", "Mueve o desactiva las mesas activas de esta zona antes de desactivarla.")
			return
		}
	}

	newName := strings.TrimSpace(in.Name)
	var z zone
	err = tx.QueryRow(r.Context(), `
		UPDATE zones
		SET name=$4,sort_order=$5,active=$6
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		RETURNING id,name,sort_order,active
	`, r.PathValue("id"), s.OrganizationID, s.LocationID, newName, sortOrder, active).
		Scan(&z.ID, &z.Name, &z.SortOrder, &z.Active)
	if err != nil {
		fail(w, 409, "zone_conflict", "Ya existe una zona con ese nombre.")
		return
	}
	if oldName != newName {
		if _, err = tx.Exec(r.Context(), `
			UPDATE tables
			SET zone=$4,updated_at=now()
			WHERE organization_id=$1 AND location_id=$2 AND zone=$3
		`, s.OrganizationID, s.LocationID, oldName, newName); err != nil {
			fail(w, 503, "zone_unavailable", "No pudimos actualizar las mesas de la zona.")
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "zone_unavailable", "No pudimos confirmar la actualización.")
		return
	}
	a.audit(r, "zone.updated", "zone", z.ID)
	writeJSON(w, 200, z)
}

func (a *API) deactivateZone(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "zone_unavailable", "No pudimos desactivar la zona.")
		return
	}
	defer tx.Rollback(r.Context())

	var name string
	var active bool
	err = tx.QueryRow(r.Context(), `
		SELECT name,active
		FROM zones
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		FOR UPDATE
	`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&name, &active)
	if err == pgx.ErrNoRows || !active {
		fail(w, 404, "zone_not_found", "La zona no existe o ya está inactiva.")
		return
	}
	if err != nil {
		fail(w, 503, "zone_unavailable", "No pudimos validar la zona.")
		return
	}
	var used bool
	if err = tx.QueryRow(r.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM tables
			WHERE organization_id=$1 AND location_id=$2 AND zone=$3 AND active
		)
	`, s.OrganizationID, s.LocationID, name).Scan(&used); err != nil {
		fail(w, 503, "zone_unavailable", "No pudimos validar las mesas de la zona.")
		return
	}
	if used {
		fail(w, 409, "zone_in_use", "Mueve o desactiva las mesas activas de esta zona antes de desactivarla.")
		return
	}
	if _, err = tx.Exec(r.Context(), `
		UPDATE zones SET active=false
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
	`, r.PathValue("id"), s.OrganizationID, s.LocationID); err != nil {
		fail(w, 503, "zone_unavailable", "No pudimos desactivar la zona.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "zone_unavailable", "No pudimos confirmar la desactivación.")
		return
	}
	a.audit(r, "zone.deactivated", "zone", r.PathValue("id"))
	w.WriteHeader(204)
}
