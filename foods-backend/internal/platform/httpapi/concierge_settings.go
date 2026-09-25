package httpapi

import (
	"net/http"
	"os"
	"regexp"
	"strings"
)

var conciergeWhatsAppPhonePattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

func globalConciergeWhatsAppPhone() string {
	phone := strings.TrimSpace(os.Getenv("FUDIA_WHATSAPP_PHONE"))
	if !conciergeWhatsAppPhonePattern.MatchString(phone) {
		return ""
	}
	return phone
}

type conciergeSettingsView struct {
	Active            bool   `json:"active"`
	Available         bool   `json:"available"`
	WhatsAppPhone     string `json:"whatsappPhone"`
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
	out.WhatsAppPhone = globalConciergeWhatsAppPhone()
	out.Available = out.Active && out.WhatsAppPhone != ""
	writeJSON(w, 200, out)
}
