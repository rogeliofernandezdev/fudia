package httpapi

import (
	"crypto/sha256"
	"encoding/json"
	"net/http"
)

func sha256Sum(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

type orgSummary struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Active        bool   `json:"active"`
	LocationCount int    `json:"locationCount"`
}

type locationSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Code   string `json:"code"`
	Active bool   `json:"active"`
}

type switchContextInput struct {
	OrganizationID string `json:"organizationId"`
	LocationID     string `json:"locationId"`
}

// listOrganizations lista todas las empresas (solo platform admin)
func (a *API) listOrganizations(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(r.Context(), `
		SELECT o.id, o.trade_name, o.active, count(l.id)
		FROM organizations o
		LEFT JOIN locations l ON l.organization_id=o.id AND l.active
		WHERE o.active
		GROUP BY o.id, o.trade_name, o.active
		ORDER BY o.trade_name`)
	if err != nil {
		fail(w, 503, "orgs_unavailable", "No pudimos cargar las empresas.")
		return
	}
	defer rows.Close()
	items := []orgSummary{}
	for rows.Next() {
		var o orgSummary
		if err = rows.Scan(&o.ID, &o.Name, &o.Active, &o.LocationCount); err != nil {
			fail(w, 503, "orgs_unavailable", "No pudimos cargar las empresas.")
			return
		}
		items = append(items, o)
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

// listOrgLocations lista los locales de una empresa específica (solo platform admin)
func (a *API) listOrgLocations(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		fail(w, 400, "invalid_request", "ID de empresa requerido.")
		return
	}
	rows, err := a.db.Query(r.Context(), `
		SELECT id, name, code, active
		FROM locations
		WHERE organization_id=$1 AND active
		ORDER BY name`, orgID)
	if err != nil {
		fail(w, 503, "locations_unavailable", "No pudimos cargar los locales.")
		return
	}
	defer rows.Close()
	items := []locationSummary{}
	for rows.Next() {
		var l locationSummary
		if err = rows.Scan(&l.ID, &l.Name, &l.Code, &l.Active); err != nil {
			fail(w, 503, "locations_unavailable", "No pudimos cargar los locales.")
			return
		}
		items = append(items, l)
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

// switchContext cambia la empresa/local activo de la sesión.
// platformAdmin puede ir a cualquier empresa/local; los demás solo a locales
// de su empresa donde tengan un rol asignado (o cualquiera con rol a nivel empresa).
func (a *API) switchContext(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in switchContextInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	if in.OrganizationID == "" || in.LocationID == "" {
		fail(w, 400, "invalid_request", "Empresa y local son obligatorios.")
		return
	}
	// Validar que la empresa y local existen y están activos
	var orgName, locName, locCountry, locTimezone string
	err := a.db.QueryRow(r.Context(), `
		SELECT o.trade_name, l.name, p.country_code, l.timezone
		FROM organizations o
		JOIN locations l ON l.organization_id=o.id AND l.active
		JOIN organization_fiscal_profiles p ON p.id=l.fiscal_profile_id AND p.organization_id=l.organization_id AND p.active
		WHERE o.id=$1 AND l.id=$2 AND o.active`, in.OrganizationID, in.LocationID).Scan(&orgName, &locName, &locCountry, &locTimezone)
	if err != nil {
		fail(w, 404, "context_not_found", "La empresa o local no existe o está inactivo.")
		return
	}
	var platformAdmin bool
	if err = a.db.QueryRow(r.Context(), `SELECT platform_admin FROM users WHERE id=$1`, s.UserID).Scan(&platformAdmin); err != nil {
		fail(w, 503, "session_error", "No pudimos cambiar el contexto.")
		return
	}
	if !platformAdmin {
		if in.OrganizationID != s.OrganizationID {
			fail(w, 403, "forbidden", "No puedes operar en otra empresa.")
			return
		}
		var allowed bool
		if err = a.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id=ur.role_id WHERE ur.user_id=$1 AND (ur.location_id IS NULL OR ur.location_id=$2) AND ro.active)`, s.UserID, in.LocationID).Scan(&allowed); err != nil {
			fail(w, 503, "session_error", "No pudimos cambiar el contexto.")
			return
		}
		if !allowed {
			fail(w, 403, "forbidden", "No tienes un rol asignado en ese local.")
			return
		}
	}
	// Actualizar la sesión actual con el nuevo contexto
	cookie, err := r.Cookie("foods_session")
	if err != nil || cookie.Value == "" {
		fail(w, 401, "session_required", "No hay sesión activa.")
		return
	}
	sum := sha256Sum(cookie.Value)
	_, err = a.db.Exec(r.Context(), `
		UPDATE sessions SET organization_id=$1, location_id=$2 WHERE token_hash=$3 AND revoked_at IS NULL`,
		in.OrganizationID, in.LocationID, sum[:])
	if err != nil {
		fail(w, 503, "session_error", "No pudimos cambiar el contexto.")
		return
	}
	a.audit(r, "context.switched", "organization", in.OrganizationID)
	writeJSON(w, 200, map[string]any{
		"organization": map[string]string{"id": in.OrganizationID, "name": orgName},
		"location":     map[string]string{"id": in.LocationID, "name": locName, "country": locCountry, "timezone": locTimezone},
	})
}

// listAvailableLocations lista los locales donde el usuario puede operar.
// platformAdmin y usuarios con rol a nivel empresa ven todos los locales activos;
// los demás solo los locales donde tienen un rol asignado.
func (a *API) listAvailableLocations(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	rows, err := a.db.Query(r.Context(), `
		SELECT l.id, l.name, l.code, l.active
		FROM locations l
		WHERE l.organization_id=$1 AND l.active
		  AND (
		    EXISTS(SELECT 1 FROM users u WHERE u.id=$2 AND u.platform_admin)
		    OR EXISTS(SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id=ur.role_id
		              WHERE ur.user_id=$2 AND ur.location_id IS NULL AND ro.active)
		    OR EXISTS(SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id=ur.role_id
		              WHERE ur.user_id=$2 AND ur.location_id=l.id AND ro.active)
		  )
		ORDER BY l.name`, s.OrganizationID, s.UserID)
	if err != nil {
		fail(w, 503, "locations_unavailable", "No pudimos cargar los locales.")
		return
	}
	defer rows.Close()
	items := []locationSummary{}
	for rows.Next() {
		var l locationSummary
		if err = rows.Scan(&l.ID, &l.Name, &l.Code, &l.Active); err != nil {
			fail(w, 503, "locations_unavailable", "No pudimos cargar los locales.")
			return
		}
		items = append(items, l)
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
