package httpapi

import "net/http"

type conciergeSettingsView struct {
	Active            bool   `json:"active"`
	OrganizationName  string `json:"organizationName"`
	LocationName      string `json:"locationName"`
	ManagedByPlatform bool   `json:"managedByPlatform"`
}

func (a *API) getConciergeSettings(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var out conciergeSettingsView
	err := a.db.QueryRow(r.Context(), `
		SELECT o.trade_name,l.name,COALESCE(om.active,false)
		FROM organizations o
		JOIN locations l
		  ON l.organization_id=o.id
		 AND l.id=$2
		 AND l.active
		LEFT JOIN organization_modules om
		  ON om.organization_id=o.id
		 AND om.module_key='whatsapp_bot'
		WHERE o.id=$1 AND o.active
	`, s.OrganizationID, s.LocationID).Scan(
		&out.OrganizationName,
		&out.LocationName,
		&out.Active,
	)
	if err != nil {
		fail(w, 503, "concierge_settings_unavailable", "No pudimos cargar el estado de Fudia Concierge.")
		return
	}
	out.ManagedByPlatform = true
	writeJSON(w, 200, out)
}
