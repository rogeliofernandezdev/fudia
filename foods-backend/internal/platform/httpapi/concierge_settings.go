package httpapi

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

var conciergeWhatsAppPhonePattern = regexp.MustCompile(`^\\+[1-9][0-9]{7,14}$`)

type conciergeSettingsView struct {
	Active           bool   `json:"active"`
	WhatsAppPhone    string `json:"whatsappPhone"`
	OrganizationName string `json:"organizationName"`
	LocationName     string `json:"locationName"`
}

type conciergeSettingsInput struct {
	Active        bool   `json:"active"`
	WhatsAppPhone string `json:"whatsappPhone"`
}

func (a *API) getConciergeSettings(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var out conciergeSettingsView
	err := a.db.QueryRow(r.Context(), `
		SELECT o.trade_name,l.name,
		       COALESCE(cs.whatsapp_phone,''),
		       COALESCE(cs.active,false)
		FROM organizations o
		JOIN locations l ON l.organization_id=o.id AND l.id=$2 AND l.active
		LEFT JOIN concierge_settings cs
		  ON cs.organization_id=o.id AND cs.location_id=l.id
		WHERE o.id=$1 AND o.active
	`, s.OrganizationID, s.LocationID).Scan(
		&out.OrganizationName,
		&out.LocationName,
		&out.WhatsAppPhone,
		&out.Active,
	)
	if err != nil {
		fail(w, 503, "concierge_settings_unavailable", "No pudimos cargar la configuración de Fudia Concierge.")
		return
	}
	writeJSON(w, 200, out)
}

func (a *API) updateConciergeSettings(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in conciergeSettingsInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_concierge_settings", "Revisa la configuración enviada.")
		return
	}
	in.WhatsAppPhone = strings.TrimSpace(in.WhatsAppPhone)
	if in.WhatsAppPhone != "" && !conciergeWhatsAppPhonePattern.MatchString(in.WhatsAppPhone) {
		fail(w, 400, "invalid_whatsapp_phone", "Usa un número internacional con prefijo + y código de país.")
		return
	}
	if in.Active && in.WhatsAppPhone == "" {
		fail(w, 400, "whatsapp_phone_required", "Configura el número de WhatsApp antes de activar Fudia Concierge.")
		return
	}

	tag, err := a.db.Exec(r.Context(), `
		INSERT INTO concierge_settings(
		  organization_id,location_id,whatsapp_phone,active
		)
		SELECT $1,$2,$3,$4
		WHERE EXISTS(
		  SELECT 1 FROM locations
		  WHERE id=$2 AND organization_id=$1 AND active
		)
		ON CONFLICT(organization_id,location_id)
		DO UPDATE SET
		  whatsapp_phone=EXCLUDED.whatsapp_phone,
		  active=EXCLUDED.active,
		  updated_at=now()
	`, s.OrganizationID, s.LocationID, in.WhatsAppPhone, in.Active)
	if err != nil || tag.RowsAffected() == 0 {
		fail(w, 503, "concierge_settings_unavailable", "No pudimos guardar la configuración de Fudia Concierge.")
		return
	}
	a.audit(r, "concierge.settings.updated", "location", s.LocationID)
	a.getConciergeSettings(w, r)
}
