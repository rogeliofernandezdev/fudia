package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

type table struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Seats     int    `json:"seats"`
	Zone      string `json:"zone"`
	Active    bool   `json:"active"`
	QrToken   string `json:"qrToken"`
	QrEnabled bool   `json:"qrEnabled"`
}

type tableInput struct {
	Name      string `json:"name"`
	Seats     *int   `json:"seats"`
	Zone      string `json:"zone"`
	Active    *bool  `json:"active"`
	QrEnabled *bool  `json:"qrEnabled"`
}

type tableBatchInput struct {
	Items []tableInput `json:"items"`
}

func (a *API) listTables(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	q := "%" + strings.TrimSpace(r.URL.Query().Get("q")) + "%"
	status := r.URL.Query().Get("status")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if size < 1 {
		size = 10
	}
	rows, err := a.db.Query(r.Context(), `SELECT id,name,seats,zone,active,qr_token,qr_enabled FROM tables WHERE organization_id=$1 AND location_id=$2 AND name ILIKE $3 AND ($4='' OR active=($4='active')) ORDER BY name LIMIT $5 OFFSET $6`, s.OrganizationID, s.LocationID, q, status, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "tables_unavailable", "No pudimos cargar las mesas.")
		return
	}
	defer rows.Close()
	items := []table{}
	for rows.Next() {
		var t table
		if err = rows.Scan(&t.ID, &t.Name, &t.Seats, &t.Zone, &t.Active, &t.QrToken, &t.QrEnabled); err != nil {
			fail(w, 503, "tables_unavailable", "No pudimos cargar las mesas.")
			return
		}
		items = append(items, t)
	}
	var total int
	_ = a.db.QueryRow(r.Context(), `SELECT count(*) FROM tables WHERE organization_id=$1 AND location_id=$2 AND name ILIKE $3 AND ($4='' OR active=($4='active'))`, s.OrganizationID, s.LocationID, q, status).Scan(&total)
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size})
}

func (a *API) createTable(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in tableInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		fail(w, 400, "invalid_request", "El nombre de la mesa es obligatorio.")
		return
	}
	seats := 2
	if in.Seats != nil && *in.Seats > 0 {
		seats = *in.Seats
	}
	qrEnabled := true
	if in.QrEnabled != nil {
		qrEnabled = *in.QrEnabled
	}
	var t table
	err := a.db.QueryRow(r.Context(), `INSERT INTO tables(organization_id,location_id,name,seats,zone,active,qr_token,qr_enabled) VALUES($1,$2,$3,$4,$5,true,encode(gen_random_bytes(16),'hex'),$6) RETURNING id,name,seats,zone,active,qr_token,qr_enabled`, s.OrganizationID, s.LocationID, strings.TrimSpace(in.Name), seats, strings.TrimSpace(in.Zone), qrEnabled).Scan(&t.ID, &t.Name, &t.Seats, &t.Zone, &t.Active, &t.QrToken, &t.QrEnabled)
	if err != nil {
		fail(w, 409, "table_conflict", "Ya existe una mesa con ese nombre.")
		return
	}
	a.audit(r, "table.created", "table", t.ID)
	writeJSON(w, 201, t)
}

func (a *API) createTablesBatch(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in tableBatchInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	if len(in.Items) == 0 {
		fail(w, 400, "invalid_request", "Agrega al menos una mesa.")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "tables_unavailable", "No pudimos registrar las mesas.")
		return
	}
	defer tx.Rollback(r.Context())
	created := []table{}
	for _, item := range in.Items {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		seats := 2
		if item.Seats != nil && *item.Seats > 0 {
			seats = *item.Seats
		}
		qrEnabled := true
		if item.QrEnabled != nil {
			qrEnabled = *item.QrEnabled
		}
		var t table
		err := tx.QueryRow(r.Context(), `INSERT INTO tables(organization_id,location_id,name,seats,zone,active,qr_token,qr_enabled) VALUES($1,$2,$3,$4,$5,true,encode(gen_random_bytes(16),'hex'),$6) RETURNING id,name,seats,zone,active,qr_token,qr_enabled`, s.OrganizationID, s.LocationID, strings.TrimSpace(item.Name), seats, strings.TrimSpace(item.Zone), qrEnabled).Scan(&t.ID, &t.Name, &t.Seats, &t.Zone, &t.Active, &t.QrToken, &t.QrEnabled)
		if err != nil {
			fail(w, 409, "table_conflict", "Ya existe una mesa llamada \""+strings.TrimSpace(item.Name)+"\".")
			return
		}
		created = append(created, t)
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "tables_unavailable", "No pudimos registrar las mesas.")
		return
	}
	a.audit(r, "table.batch_created", "table", "")
	writeJSON(w, 201, map[string]any{"items": created})
}

func (a *API) updateTable(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in tableInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		fail(w, 400, "invalid_request", "El nombre de la mesa es obligatorio.")
		return
	}
	seats := 2
	if in.Seats != nil && *in.Seats > 0 {
		seats = *in.Seats
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	qrEnabled := true
	if in.QrEnabled != nil {
		qrEnabled = *in.QrEnabled
	}
	var t table
	err := a.db.QueryRow(r.Context(), `UPDATE tables SET name=$4,seats=$5,zone=$6,active=$7,qr_enabled=$8,updated_at=now() WHERE id=$1 AND organization_id=$2 AND location_id=$3 RETURNING id,name,seats,zone,active,qr_token,qr_enabled`, r.PathValue("id"), s.OrganizationID, s.LocationID, strings.TrimSpace(in.Name), seats, strings.TrimSpace(in.Zone), active, qrEnabled).Scan(&t.ID, &t.Name, &t.Seats, &t.Zone, &t.Active, &t.QrToken, &t.QrEnabled)
	if err == pgx.ErrNoRows {
		fail(w, 404, "table_not_found", "La mesa no existe.")
		return
	} else if err != nil {
		fail(w, 409, "table_conflict", "Ya existe una mesa con ese nombre.")
		return
	}
	a.audit(r, "table.updated", "table", t.ID)
	writeJSON(w, 200, t)
}

func (a *API) regenerateTableQR(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var t table
	err := a.db.QueryRow(r.Context(), `UPDATE tables SET qr_token=encode(gen_random_bytes(16),'hex'),updated_at=now() WHERE id=$1 AND organization_id=$2 AND location_id=$3 RETURNING id,name,seats,zone,active,qr_token,qr_enabled`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&t.ID, &t.Name, &t.Seats, &t.Zone, &t.Active, &t.QrToken, &t.QrEnabled)
	if err == pgx.ErrNoRows {
		fail(w, 404, "table_not_found", "La mesa no existe.")
		return
	} else if err != nil {
		fail(w, 503, "table_unavailable", "No pudimos regenerar el QR.")
		return
	}
	a.audit(r, "table.qr_regenerated", "table", t.ID)
	writeJSON(w, 200, t)
}

func (a *API) deactivateTable(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	ct, err := a.db.Exec(r.Context(), `UPDATE tables SET active=false,updated_at=now() WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND active=true`, r.PathValue("id"), s.OrganizationID, s.LocationID)
	if err != nil {
		fail(w, 503, "table_unavailable", "No pudimos desactivar la mesa.")
		return
	}
	if ct.RowsAffected() == 0 {
		fail(w, 404, "table_not_found", "La mesa no existe o ya está inactiva.")
		return
	}
	a.audit(r, "table.deactivated", "table", r.PathValue("id"))
	w.WriteHeader(204)
}

// getTableByQR es un endpoint público (sin sesión) que devuelve la información
// básica de la mesa a partir del token QR. Se usa para vincular el proceso
// de atención del cliente (carta digital, llamado de mozo, etc.).
func (a *API) getTableByQR(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		fail(w, 400, "invalid_request", "Token QR requerido.")
		return
	}
	var t struct {
		Name    string `json:"name"`
		Seats   int    `json:"seats"`
		Zone    string `json:"zone"`
		OrgName string `json:"organizationName"`
		LocName string `json:"locationName"`
	}
	err := a.db.QueryRow(r.Context(), `
		SELECT t.name, t.seats, t.zone, o.trade_name, l.name
		FROM tables t
		JOIN organizations o ON o.id = t.organization_id
		JOIN locations l ON l.id=t.location_id AND l.organization_id=t.organization_id AND l.active
		WHERE t.qr_token = $1 AND t.active = true AND t.qr_enabled = true
		LIMIT 1`, token).Scan(&t.Name, &t.Seats, &t.Zone, &t.OrgName, &t.LocName)
	if err != nil {
		fail(w, 404, "table_not_found", "La mesa no existe o el QR no está activo.")
		return
	}
	writeJSON(w, 200, t)
}
