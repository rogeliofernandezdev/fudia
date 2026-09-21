package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type cashMovementView struct {
	ID            string  `json:"id"`
	MovementType  string  `json:"movementType"`
	SourceType    string  `json:"sourceType"`
	SourceID      *string `json:"sourceId"`
	Amount        string  `json:"amount"`
	Reason        string  `json:"reason"`
	Note          string  `json:"note"`
	CreatedByName string  `json:"createdByName"`
	CreatedAt     string  `json:"createdAt"`
}

type cashShiftView struct {
	ID                    string             `json:"id"`
	Code                  string             `json:"code"`
	CashRegisterID        string             `json:"cashRegisterId"`
	CashRegisterName      string             `json:"cashRegisterName"`
	Status                string             `json:"status"`
	OpeningAmount         string             `json:"openingAmount"`
	IncomeAmount          string             `json:"incomeAmount"`
	ExpenseAmount         string             `json:"expenseAmount"`
	ExpectedAmount        string             `json:"expectedAmount"`
	ClosingExpectedAmount *string            `json:"closingExpectedAmount"`
	ClosingCountedAmount  *string            `json:"closingCountedAmount"`
	VarianceAmount        *string            `json:"varianceAmount"`
	OpeningNote           string             `json:"openingNote"`
	ClosingNote           string             `json:"closingNote"`
	OpenedByName          string             `json:"openedByName"`
	ClosedByName          string             `json:"closedByName"`
	BusinessDate          string             `json:"businessDate"`
	OpenedAt              string             `json:"openedAt"`
	ClosedAt              *string            `json:"closedAt"`
	MovementCount         int                `json:"movementCount"`
	Movements             []cashMovementView `json:"movements,omitempty"`
}

type cashRegisterView struct {
	ID        string         `json:"id"`
	Code      string         `json:"code"`
	Name      string         `json:"name"`
	Active    bool           `json:"active"`
	OpenShift *cashShiftView `json:"openShift"`
}

const cashShiftColumns = `
	cs.id::text,
	cs.code,
	cr.id::text,
	cr.name,
	cs.status,
	cs.opening_amount::text,
	COALESCE((SELECT sum(cm.amount) FROM cash_movements cm WHERE cm.shift_id=cs.id AND cm.organization_id=cs.organization_id AND cm.movement_type='income'),0)::text,
	COALESCE((SELECT sum(cm.amount) FROM cash_movements cm WHERE cm.shift_id=cs.id AND cm.organization_id=cs.organization_id AND cm.movement_type='expense'),0)::text,
	(
	  cs.opening_amount
	  + COALESCE((SELECT sum(cm.amount) FROM cash_movements cm WHERE cm.shift_id=cs.id AND cm.organization_id=cs.organization_id AND cm.movement_type='income'),0)
	  - COALESCE((SELECT sum(cm.amount) FROM cash_movements cm WHERE cm.shift_id=cs.id AND cm.organization_id=cs.organization_id AND cm.movement_type='expense'),0)
	)::text,
	CASE WHEN cs.closing_expected_amount IS NULL THEN NULL ELSE cs.closing_expected_amount::text END,
	CASE WHEN cs.closing_counted_amount IS NULL THEN NULL ELSE cs.closing_counted_amount::text END,
	CASE WHEN cs.variance_amount IS NULL THEN NULL ELSE cs.variance_amount::text END,
	cs.opening_note,
	cs.closing_note,
	opened.full_name,
	COALESCE(closed.full_name,''),
	cs.business_date::text,
	to_char(cs.opened_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),
	CASE WHEN cs.closed_at IS NULL THEN NULL ELSE to_char(cs.closed_at,'YYYY-MM-DD"T"HH24:MI:SSOF') END,
	(SELECT count(*) FROM cash_movements cm WHERE cm.shift_id=cs.id AND cm.organization_id=cs.organization_id)
`

func scanCashShift(row pgx.Row) (cashShiftView, error) {
	var shift cashShiftView
	err := row.Scan(
		&shift.ID,
		&shift.Code,
		&shift.CashRegisterID,
		&shift.CashRegisterName,
		&shift.Status,
		&shift.OpeningAmount,
		&shift.IncomeAmount,
		&shift.ExpenseAmount,
		&shift.ExpectedAmount,
		&shift.ClosingExpectedAmount,
		&shift.ClosingCountedAmount,
		&shift.VarianceAmount,
		&shift.OpeningNote,
		&shift.ClosingNote,
		&shift.OpenedByName,
		&shift.ClosedByName,
		&shift.BusinessDate,
		&shift.OpenedAt,
		&shift.ClosedAt,
		&shift.MovementCount,
	)
	return shift, err
}

func (a *API) loadCashMovements(r *http.Request, shiftID string) ([]cashMovementView, error) {
	s := r.Context().Value(scopeKey{}).(scope)
	rows, err := a.db.Query(r.Context(), `
		SELECT cm.id::text,cm.movement_type,cm.source_type,
		       CASE WHEN cm.source_id IS NULL THEN NULL ELSE cm.source_id::text END,
		       cm.amount::text,cm.reason,cm.note,u.full_name,
		       to_char(cm.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
		FROM cash_movements cm
		JOIN users u ON u.id=cm.created_by AND u.organization_id=cm.organization_id
		WHERE cm.shift_id=$1 AND cm.organization_id=$2 AND cm.location_id=$3
		ORDER BY cm.created_at DESC,cm.id DESC
	`, shiftID, s.OrganizationID, s.LocationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []cashMovementView{}
	for rows.Next() {
		var item cashMovementView
		if err := rows.Scan(&item.ID, &item.MovementType, &item.SourceType, &item.SourceID, &item.Amount, &item.Reason, &item.Note, &item.CreatedByName, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (a *API) getCashShiftByID(r *http.Request, id string) (cashShiftView, error) {
	s := r.Context().Value(scopeKey{}).(scope)
	shift, err := scanCashShift(a.db.QueryRow(r.Context(), `
		SELECT `+cashShiftColumns+`
		FROM cash_shifts cs
		JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
		JOIN users opened ON opened.id=cs.opened_by AND opened.organization_id=cs.organization_id
		LEFT JOIN users closed ON closed.id=cs.closed_by AND closed.organization_id=cs.organization_id
		WHERE cs.id=$1 AND cs.organization_id=$2 AND cs.location_id=$3
	`, id, s.OrganizationID, s.LocationID))
	if err != nil {
		return cashShiftView{}, err
	}
	movements, err := a.loadCashMovements(r, shift.ID)
	if err != nil {
		return cashShiftView{}, err
	}
	shift.Movements = movements
	return shift, nil
}

func (a *API) getOpenCashShiftByRegister(r *http.Request, registerID string) (*cashShiftView, error) {
	s := r.Context().Value(scopeKey{}).(scope)
	shift, err := scanCashShift(a.db.QueryRow(r.Context(), `
		SELECT `+cashShiftColumns+`
		FROM cash_shifts cs
		JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
		JOIN users opened ON opened.id=cs.opened_by AND opened.organization_id=cs.organization_id
		LEFT JOIN users closed ON closed.id=cs.closed_by AND closed.organization_id=cs.organization_id
		WHERE cs.organization_id=$1 AND cs.location_id=$2 AND cs.cash_register_id=$3 AND cs.status='open'
		LIMIT 1
	`, s.OrganizationID, s.LocationID, registerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &shift, nil
}

func (a *API) listCashRegisters(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	rows, err := a.db.Query(r.Context(), `
		SELECT id::text,code,name,active
		FROM cash_registers
		WHERE organization_id=$1 AND location_id=$2
		  AND ($3='' OR name ILIKE '%'||$3||'%' OR code ILIKE '%'||$3||'%')
		ORDER BY active DESC,name,created_at
	`, s.OrganizationID, s.LocationID, q)
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos cargar las cajas del local.")
		return
	}
	items := []cashRegisterView{}
	for rows.Next() {
		var item cashRegisterView
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Active); err != nil {
			rows.Close()
			fail(w, 503, "cash_unavailable", "No pudimos leer las cajas del local.")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		fail(w, 503, "cash_unavailable", "No pudimos completar la consulta de cajas.")
		return
	}
	rows.Close()

	for index := range items {
		openShift, err := a.getOpenCashShiftByRegister(r, items[index].ID)
		if err != nil {
			fail(w, 503, "cash_unavailable", "No pudimos cargar el turno abierto de la caja.")
			return
		}
		items[index].OpenShift = openShift
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (a *API) createCashRegister(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		Name string `json:"name"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_cash_register", "Revisa los datos de la caja.")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if len(in.Name) < 1 || len(in.Name) > 80 {
		fail(w, 400, "invalid_cash_register", "El nombre de la caja debe tener entre 1 y 80 caracteres.")
		return
	}

	var item cashRegisterView
	err := a.db.QueryRow(r.Context(), `
		INSERT INTO cash_registers(organization_id,location_id,name,created_by)
		VALUES($1,$2,$3,$4)
		RETURNING id::text,code,name,active
	`, s.OrganizationID, s.LocationID, in.Name, s.UserID).Scan(&item.ID, &item.Code, &item.Name, &item.Active)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			fail(w, 409, "cash_register_exists", "Ya existe una caja con ese nombre en este local.")
			return
		}
		fail(w, 503, "cash_unavailable", "No pudimos registrar la caja.")
		return
	}
	writeJSON(w, 201, item)
}

func (a *API) updateCashRegister(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		Name string `json:"name"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_cash_register", "Revisa los datos de la caja.")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if len(in.Name) < 1 || len(in.Name) > 80 {
		fail(w, 400, "invalid_cash_register", "El nombre de la caja debe tener entre 1 y 80 caracteres.")
		return
	}

	var item cashRegisterView
	err := a.db.QueryRow(r.Context(), `
		UPDATE cash_registers
		SET name=$4,updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		RETURNING id::text,code,name,active
	`, r.PathValue("id"), s.OrganizationID, s.LocationID, in.Name).Scan(&item.ID, &item.Code, &item.Name, &item.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "cash_register_not_found", "La caja no existe en este local.")
		return
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			fail(w, 409, "cash_register_exists", "Ya existe una caja con ese nombre en este local.")
			return
		}
		fail(w, 503, "cash_unavailable", "No pudimos actualizar la caja.")
		return
	}
	openShift, err := a.getOpenCashShiftByRegister(r, item.ID)
	if err != nil {
		fail(w, 503, "cash_unavailable", "La caja se actualizó, pero no pudimos cargar su turno.")
		return
	}
	item.OpenShift = openShift
	writeJSON(w, 200, item)
}

func (a *API) updateCashRegisterStatus(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		Active bool `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_cash_register", "Revisa el estado de la caja.")
		return
	}
	if !in.Active {
		var open bool
		if err := a.db.QueryRow(r.Context(), `
			SELECT EXISTS(
			  SELECT 1 FROM cash_shifts
			  WHERE cash_register_id=$1 AND organization_id=$2 AND location_id=$3 AND status='open'
			)
		`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&open); err != nil {
			fail(w, 503, "cash_unavailable", "No pudimos validar la caja.")
			return
		}
		if open {
			fail(w, 409, "cash_register_has_open_shift", "Cierra el turno activo antes de desactivar la caja.")
			return
		}
	}
	tag, err := a.db.Exec(r.Context(), `
		UPDATE cash_registers SET active=$4,updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
	`, r.PathValue("id"), s.OrganizationID, s.LocationID, in.Active)
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos actualizar el estado de la caja.")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 404, "cash_register_not_found", "La caja no existe en este local.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) getCurrentCashShift(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	shift, err := scanCashShift(a.db.QueryRow(r.Context(), `
		SELECT `+cashShiftColumns+`
		FROM cash_shifts cs
		JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
		JOIN users opened ON opened.id=cs.opened_by AND opened.organization_id=cs.organization_id
		LEFT JOIN users closed ON closed.id=cs.closed_by AND closed.organization_id=cs.organization_id
		WHERE cs.organization_id=$1 AND cs.location_id=$2 AND cs.opened_by=$3 AND cs.status='open'
		ORDER BY cs.opened_at DESC
		LIMIT 1
	`, s.OrganizationID, s.LocationID, s.UserID))
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, 200, map[string]any{"shift": nil})
		return
	}
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos cargar tu turno de caja.")
		return
	}
	movements, err := a.loadCashMovements(r, shift.ID)
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos cargar los movimientos del turno.")
		return
	}
	shift.Movements = movements
	writeJSON(w, 200, map[string]any{"shift": shift})
}

func (a *API) listCashShifts(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && status != "open" && status != "closed" {
		fail(w, 400, "invalid_cash_shift_status", "El estado del turno no es válido.")
		return
	}

	where := `cs.organization_id=$1 AND cs.location_id=$2
		AND ($3='' OR cs.code ILIKE '%'||$3||'%' OR cr.name ILIKE '%'||$3||'%' OR opened.full_name ILIKE '%'||$3||'%' OR COALESCE(closed.full_name,'') ILIKE '%'||$3||'%')
		AND ($4='' OR cs.status=$4)`

	var total int
	if err := a.db.QueryRow(r.Context(), `
		SELECT count(*)
		FROM cash_shifts cs
		JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
		JOIN users opened ON opened.id=cs.opened_by AND opened.organization_id=cs.organization_id
		LEFT JOIN users closed ON closed.id=cs.closed_by AND closed.organization_id=cs.organization_id
		WHERE `+where, s.OrganizationID, s.LocationID, q, status).Scan(&total); err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos cargar los turnos de caja.")
		return
	}

	rows, err := a.db.Query(r.Context(), `
		SELECT `+cashShiftColumns+`
		FROM cash_shifts cs
		JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
		JOIN users opened ON opened.id=cs.opened_by AND opened.organization_id=cs.organization_id
		LEFT JOIN users closed ON closed.id=cs.closed_by AND closed.organization_id=cs.organization_id
		WHERE `+where+`
		ORDER BY cs.opened_at DESC
		LIMIT $5 OFFSET $6
	`, s.OrganizationID, s.LocationID, q, status, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos cargar los turnos de caja.")
		return
	}
	defer rows.Close()

	items := []cashShiftView{}
	for rows.Next() {
		shift, err := scanCashShift(rows)
		if err != nil {
			fail(w, 503, "cash_unavailable", "No pudimos leer los turnos de caja.")
			return
		}
		items = append(items, shift)
	}
	if err := rows.Err(); err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos completar la consulta de turnos.")
		return
	}

	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size})
}

func (a *API) getCashShift(w http.ResponseWriter, r *http.Request) {
	shift, err := a.getCashShiftByID(r, r.PathValue("id"))
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "cash_shift_not_found", "El turno de caja no existe en este local.")
		return
	}
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos cargar el turno de caja.")
		return
	}
	writeJSON(w, 200, shift)
}

func (a *API) openCashShift(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		CashRegisterID string  `json:"cashRegisterId"`
		OpeningAmount  float64 `json:"openingAmount"`
		Note           string  `json:"note"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.CashRegisterID) == "" || in.OpeningAmount < 0 {
		fail(w, 400, "invalid_cash_shift", "Selecciona una caja e ingresa un fondo inicial válido.")
		return
	}
	in.CashRegisterID = strings.TrimSpace(in.CashRegisterID)
	in.Note = strings.TrimSpace(in.Note)
	if len(in.Note) > 240 {
		fail(w, 400, "invalid_cash_shift", "La observación no puede superar 240 caracteres.")
		return
	}

	var registerName string
	if err := a.db.QueryRow(r.Context(), `
		SELECT name FROM cash_registers
		WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND active
	`, in.CashRegisterID, s.OrganizationID, s.LocationID).Scan(&registerName); errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "cash_register_not_found", "La caja seleccionada no existe o está inactiva.")
		return
	} else if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos validar la caja.")
		return
	}

	var id string
	err := a.db.QueryRow(r.Context(), `
		INSERT INTO cash_shifts(organization_id,location_id,cash_register_id,business_date,opening_amount,opening_note,opened_by)
		VALUES(
		  $1,$2,$3,
		  (now() AT TIME ZONE (SELECT timezone FROM locations WHERE id=$2 AND organization_id=$1))::date,
		  $4,$5,$6
		)
		RETURNING id
	`, s.OrganizationID, s.LocationID, in.CashRegisterID, in.OpeningAmount, in.Note, s.UserID).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			fail(w, 409, "cash_shift_already_open", "Esa caja ya tiene un turno abierto o ya tienes otro turno abierto en este local.")
			return
		}
		fail(w, 503, "cash_unavailable", "No pudimos iniciar el turno de caja.")
		return
	}

	shift, err := a.getCashShiftByID(r, id)
	if err != nil {
		fail(w, 503, "cash_unavailable", "El turno se inició, pero no pudimos cargar su detalle.")
		return
	}
	writeJSON(w, 201, shift)
}

func (a *API) createCashMovement(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		MovementType string  `json:"movementType"`
		Amount       float64 `json:"amount"`
		Reason       string  `json:"reason"`
		Note         string  `json:"note"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || (in.MovementType != "income" && in.MovementType != "expense") || in.Amount <= 0 {
		fail(w, 400, "invalid_cash_movement", "Revisa el tipo y monto del movimiento.")
		return
	}
	in.Reason = strings.TrimSpace(in.Reason)
	in.Note = strings.TrimSpace(in.Note)
	if in.Reason == "" || len(in.Reason) > 120 || len(in.Note) > 240 {
		fail(w, 400, "invalid_cash_movement", "Indica un motivo válido y una observación de hasta 240 caracteres.")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos iniciar el movimiento de caja.")
		return
	}
	defer tx.Rollback(r.Context())

	var status string
	err = tx.QueryRow(r.Context(), `
		SELECT status
		FROM cash_shifts
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		FOR UPDATE
	`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "cash_shift_not_found", "El turno de caja no existe en este local.")
		return
	}
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos validar el turno de caja.")
		return
	}
	if status != "open" {
		fail(w, 409, "cash_shift_closed", "El turno ya fue cerrado y no admite nuevos movimientos.")
		return
	}

	var item cashMovementView
	err = tx.QueryRow(r.Context(), `
		INSERT INTO cash_movements(organization_id,location_id,shift_id,movement_type,source_type,amount,reason,note,created_by)
		VALUES($1,$2,$3,$4,'manual',$5,$6,$7,$8)
		RETURNING id::text,movement_type,source_type,
		          CASE WHEN source_id IS NULL THEN NULL ELSE source_id::text END,
		          amount::text,reason,note,
		          (SELECT full_name FROM users WHERE id=$8 AND organization_id=$1),
		          to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
	`, s.OrganizationID, s.LocationID, r.PathValue("id"), in.MovementType, in.Amount, in.Reason, in.Note, s.UserID).
		Scan(&item.ID, &item.MovementType, &item.SourceType, &item.SourceID, &item.Amount, &item.Reason, &item.Note, &item.CreatedByName, &item.CreatedAt)
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos registrar el movimiento de caja.")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos confirmar el movimiento de caja.")
		return
	}
	writeJSON(w, 201, item)
}

func (a *API) closeCashShift(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		CountedAmount float64 `json:"countedAmount"`
		Note          string  `json:"note"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.CountedAmount < 0 {
		fail(w, 400, "invalid_cash_close", "Ingresa un efectivo contado válido.")
		return
	}
	in.Note = strings.TrimSpace(in.Note)
	if len(in.Note) > 240 {
		fail(w, 400, "invalid_cash_close", "La observación no puede superar 240 caracteres.")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos iniciar el cierre de caja.")
		return
	}
	defer tx.Rollback(r.Context())

	var status string
	var opening, income, expense float64
	err = tx.QueryRow(r.Context(), `
		SELECT cs.status,cs.opening_amount::float8,
		       COALESCE((SELECT sum(cm.amount) FROM cash_movements cm WHERE cm.shift_id=cs.id AND cm.organization_id=cs.organization_id AND cm.movement_type='income'),0)::float8,
		       COALESCE((SELECT sum(cm.amount) FROM cash_movements cm WHERE cm.shift_id=cs.id AND cm.organization_id=cs.organization_id AND cm.movement_type='expense'),0)::float8
		FROM cash_shifts cs
		WHERE cs.id=$1 AND cs.organization_id=$2 AND cs.location_id=$3
		FOR UPDATE
	`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&status, &opening, &income, &expense)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "cash_shift_not_found", "El turno de caja no existe en este local.")
		return
	}
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos validar el turno de caja.")
		return
	}
	if status != "open" {
		fail(w, 409, "cash_shift_closed", "El turno ya fue cerrado.")
		return
	}

	expected := opening + income - expense
	variance := in.CountedAmount - expected
	if _, err := tx.Exec(r.Context(), `
		UPDATE cash_shifts
		SET status='closed',
		    closing_expected_amount=$4,
		    closing_counted_amount=$5,
		    variance_amount=$6,
		    closing_note=$7,
		    closed_by=$8,
		    closed_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
	`, r.PathValue("id"), s.OrganizationID, s.LocationID, expected, in.CountedAmount, variance, in.Note, s.UserID); err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos cerrar el turno de caja.")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos confirmar el cierre de caja.")
		return
	}

	shift, err := a.getCashShiftByID(r, r.PathValue("id"))
	if err != nil {
		fail(w, 503, "cash_unavailable", "El turno se cerró, pero no pudimos cargar su detalle.")
		return
	}
	writeJSON(w, 200, shift)
}
