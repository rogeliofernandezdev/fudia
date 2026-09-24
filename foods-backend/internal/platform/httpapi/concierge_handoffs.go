package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

type conciergeHandoffView struct {
	ID             string  `json:"id"`
	TableID        string  `json:"tableId"`
	TableName      string  `json:"tableName"`
	ConversationID string  `json:"conversationId"`
	CustomerPhone  string  `json:"customerPhone"`
	Reason         string  `json:"reason"`
	Status         string  `json:"status"`
	RequestedAt    string  `json:"requestedAt"`
	ResolvedAt     *string `json:"resolvedAt"`
	ResolvedByName *string `json:"resolvedByName"`
}

const conciergeHandoffColumns = `
	h.id::text,h.table_id::text,t.name,h.conversation_id,h.customer_phone,h.reason,h.status,
	to_char(h.requested_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),
	CASE WHEN h.resolved_at IS NULL THEN NULL ELSE to_char(h.resolved_at,'YYYY-MM-DD"T"HH24:MI:SSOF') END,
	u.full_name
`

func scanConciergeHandoff(row pgx.Row) (conciergeHandoffView, error) {
	var out conciergeHandoffView
	err := row.Scan(
		&out.ID,&out.TableID,&out.TableName,&out.ConversationID,&out.CustomerPhone,
		&out.Reason,&out.Status,&out.RequestedAt,&out.ResolvedAt,&out.ResolvedByName,
	)
	return out, err
}

func (a *API) requestConciergeHandoff(w http.ResponseWriter, r *http.Request) {
	qr, err := a.resolveConciergeQR(r.Context(), r.PathValue("token"))
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "table_not_found", "La mesa no existe o el QR no está activo.")
		return
	}
	if err != nil {
		fail(w, 503, "concierge_unavailable", "No pudimos resolver el QR.")
		return
	}

	var in struct {
		ConversationID string `json:"conversationId"`
		CustomerPhone  string `json:"customerPhone"`
		Reason         string `json:"reason"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_handoff", "Revisa la solicitud de atención.")
		return
	}
	in.ConversationID = strings.TrimSpace(in.ConversationID)
	in.CustomerPhone = strings.TrimSpace(in.CustomerPhone)
	in.Reason = strings.TrimSpace(in.Reason)
	if in.ConversationID == "" || len(in.ConversationID) > 160 || len(in.CustomerPhone) > 32 || len([]rune(in.Reason)) > 240 {
		fail(w, 400, "invalid_handoff", "Revisa la solicitud de atención.")
		return
	}
	if in.Reason == "" {
		in.Reason = "El comensal solicitó atención del personal."
	}

	var id string
	err = a.db.QueryRow(r.Context(), `
		INSERT INTO concierge_handoffs(
		  organization_id,location_id,table_id,conversation_id,customer_phone,reason
		)
		VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(organization_id,location_id,conversation_id)
		  WHERE status='pending'
		DO UPDATE SET
		  customer_phone=EXCLUDED.customer_phone,
		  reason=EXCLUDED.reason
		RETURNING id::text
	`, qr.Scope.OrganizationID,qr.Scope.LocationID,qr.TableID,in.ConversationID,in.CustomerPhone,in.Reason).Scan(&id)
	if err != nil {
		fail(w, 503, "handoff_unavailable", "No pudimos avisar al personal.")
		return
	}

	out, err := scanConciergeHandoff(a.db.QueryRow(r.Context(), `
		SELECT `+conciergeHandoffColumns+`
		FROM concierge_handoffs h
		JOIN tables t
		  ON t.id=h.table_id AND t.organization_id=h.organization_id AND t.location_id=h.location_id
		LEFT JOIN users u ON u.id=h.resolved_by AND u.organization_id=h.organization_id
		WHERE h.id=$1 AND h.organization_id=$2 AND h.location_id=$3
	`, id, qr.Scope.OrganizationID, qr.Scope.LocationID))
	if err != nil {
		fail(w, 503, "handoff_unavailable", "No pudimos recuperar la solicitud de atención.")
		return
	}
	writeJSON(w, 200, out)
}

func (a *API) getConciergeHandoffStatus(w http.ResponseWriter, r *http.Request) {
	qr, err := a.resolveConciergeQR(r.Context(), r.PathValue("token"))
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "table_not_found", "La mesa no existe o el QR no está activo.")
		return
	}
	if err != nil {
		fail(w, 503, "concierge_unavailable", "No pudimos resolver el QR.")
		return
	}
	conversationID := strings.TrimSpace(r.PathValue("conversationId"))
	if conversationID == "" || len(conversationID) > 160 {
		fail(w, 400, "invalid_handoff", "La conversación no es válida.")
		return
	}

	out, err := scanConciergeHandoff(a.db.QueryRow(r.Context(), `
		SELECT `+conciergeHandoffColumns+`
		FROM concierge_handoffs h
		JOIN tables t
		  ON t.id=h.table_id AND t.organization_id=h.organization_id AND t.location_id=h.location_id
		LEFT JOIN users u ON u.id=h.resolved_by AND u.organization_id=h.organization_id
		WHERE h.organization_id=$1 AND h.location_id=$2 AND h.table_id=$3
		  AND h.conversation_id=$4
		ORDER BY h.requested_at DESC,h.id DESC
		LIMIT 1
	`, qr.Scope.OrganizationID,qr.Scope.LocationID,qr.TableID,conversationID))
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, 200, map[string]any{"status":"none"})
		return
	}
	if err != nil {
		fail(w, 503, "handoff_unavailable", "No pudimos consultar la solicitud de atención.")
		return
	}
	writeJSON(w, 200, out)
}

func (a *API) listOperationalConciergeHandoffs(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = "pending"
	}
	if status != "pending" && status != "resolved" {
		fail(w, 400, "invalid_status", "El estado de atención no es válido.")
		return
	}

	rows, err := a.db.Query(r.Context(), `
		SELECT `+conciergeHandoffColumns+`
		FROM concierge_handoffs h
		JOIN tables t
		  ON t.id=h.table_id AND t.organization_id=h.organization_id AND t.location_id=h.location_id
		LEFT JOIN users u ON u.id=h.resolved_by AND u.organization_id=h.organization_id
		WHERE h.organization_id=$1 AND h.location_id=$2 AND h.status=$3
		ORDER BY h.requested_at DESC,h.id DESC
		LIMIT 50
	`, s.OrganizationID,s.LocationID,status)
	if err != nil {
		fail(w, 503, "handoffs_unavailable", "No pudimos cargar las solicitudes de atención.")
		return
	}
	defer rows.Close()

	items := []conciergeHandoffView{}
	for rows.Next() {
		var out conciergeHandoffView
		if err := rows.Scan(
			&out.ID,&out.TableID,&out.TableName,&out.ConversationID,&out.CustomerPhone,
			&out.Reason,&out.Status,&out.RequestedAt,&out.ResolvedAt,&out.ResolvedByName,
		); err != nil {
			fail(w, 503, "handoffs_unavailable", "No pudimos leer las solicitudes de atención.")
			return
		}
		items = append(items, out)
	}
	if err := rows.Err(); err != nil {
		fail(w, 503, "handoffs_unavailable", "No pudimos completar las solicitudes de atención.")
		return
	}
	writeJSON(w, 200, map[string]any{"items":items,"total":len(items)})
}

func (a *API) resolveOperationalConciergeHandoff(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		fail(w, 400, "invalid_handoff", "La solicitud de atención no es válida.")
		return
	}

	var resolvedID string
	err := a.db.QueryRow(r.Context(), `
		UPDATE concierge_handoffs
		SET status='resolved',resolved_at=now(),resolved_by=$4
		WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND status='pending'
		RETURNING id::text
	`, id,s.OrganizationID,s.LocationID,s.UserID).Scan(&resolvedID)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "handoff_not_found", "La solicitud ya fue atendida o no existe.")
		return
	}
	if err != nil {
		fail(w, 503, "handoff_unavailable", "No pudimos cerrar la solicitud de atención.")
		return
	}
	a.audit(r, "concierge.handoff.resolved", "concierge_handoff", resolvedID)
	w.WriteHeader(http.StatusNoContent)
}
