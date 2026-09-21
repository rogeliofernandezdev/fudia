package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type cashRowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type cashShiftUserView struct {
	UserID     string  `json:"userId"`
	Name       string  `json:"name"`
	AssignedAt string  `json:"assignedAt"`
}

type cashUserOption struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	AssignedShiftID *string `json:"assignedShiftId"`
}

type cashOperationView struct {
	ID            string `json:"id"`
	OperationType string `json:"operationType"`
	Amount        string `json:"amount"`
	Reason        string `json:"reason"`
	Note          string `json:"note"`
	CreatedAt     string `json:"createdAt"`
}

func currentCashShiftID(ctx context.Context, q cashRowQuerier, s scope) (string, error) {
	var id string
	err := q.QueryRow(ctx, `
		SELECT cs.id::text
		FROM cash_shift_users su
		JOIN cash_shifts cs ON cs.id=su.shift_id
		  AND cs.organization_id=su.organization_id
		  AND cs.location_id=su.location_id
		WHERE su.organization_id=$1
		  AND su.location_id=$2
		  AND su.user_id=$3
		  AND su.unassigned_at IS NULL
		  AND cs.status='open'
		ORDER BY su.assigned_at DESC
		LIMIT 1
	`, s.OrganizationID, s.LocationID, s.UserID).Scan(&id)
	return id, err
}

func requireCashShiftAssignment(ctx context.Context, q cashRowQuerier, s scope, shiftID string) error {
	var assigned bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS(
		  SELECT 1
		  FROM cash_shift_users su
		  JOIN cash_shifts cs ON cs.id=su.shift_id
		    AND cs.organization_id=su.organization_id
		    AND cs.location_id=su.location_id
		  WHERE su.organization_id=$1
		    AND su.location_id=$2
		    AND su.shift_id=$3
		    AND su.user_id=$4
		    AND su.unassigned_at IS NULL
		    AND cs.status='open'
		)
	`, s.OrganizationID, s.LocationID, shiftID, s.UserID).Scan(&assigned)
	if err != nil {
		return err
	}
	if !assigned {
		return pgx.ErrNoRows
	}
	return nil
}

func cashShiftExpected(ctx context.Context, q cashRowQuerier, s scope, shiftID string) (float64, error) {
	var expected float64
	err := q.QueryRow(ctx, `
		SELECT (
		  cs.opening_amount
		  + COALESCE((SELECT sum(cm.amount) FROM cash_movements cm WHERE cm.shift_id=cs.id AND cm.organization_id=cs.organization_id AND cm.movement_type='income'),0)
		  - COALESCE((SELECT sum(cm.amount) FROM cash_movements cm WHERE cm.shift_id=cs.id AND cm.organization_id=cs.organization_id AND cm.movement_type='expense'),0)
		)::float8
		FROM cash_shifts cs
		WHERE cs.id=$1 AND cs.organization_id=$2 AND cs.location_id=$3 AND cs.status='open'
	`, shiftID, s.OrganizationID, s.LocationID).Scan(&expected)
	return expected, err
}

func (a *API) listCashUserOptions(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	rows, err := a.db.Query(r.Context(), `
		SELECT DISTINCT u.id::text,u.full_name,
		       (
		         SELECT su.shift_id::text
		         FROM cash_shift_users su
		         JOIN cash_shifts cs ON cs.id=su.shift_id
		           AND cs.organization_id=su.organization_id
		           AND cs.location_id=su.location_id
		         WHERE su.organization_id=$1
		           AND su.location_id=$2
		           AND su.user_id=u.id
		           AND su.unassigned_at IS NULL
		           AND cs.status='open'
		         LIMIT 1
		       )
		FROM users u
		JOIN user_roles ur ON ur.user_id=u.id
		WHERE u.organization_id=$1
		  AND u.active
		  AND (ur.location_id IS NULL OR ur.location_id=$2)
		ORDER BY u.full_name
	`, s.OrganizationID, s.LocationID)
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos cargar los usuarios disponibles.")
		return
	}
	defer rows.Close()

	items := []cashUserOption{}
	for rows.Next() {
		var item cashUserOption
		if err := rows.Scan(&item.ID, &item.Name, &item.AssignedShiftID); err != nil {
			fail(w, 503, "cash_unavailable", "No pudimos leer los usuarios disponibles.")
			return
		}
		items = append(items, item)
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (a *API) listCashShiftUsers(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	rows, err := a.db.Query(r.Context(), `
		SELECT su.user_id::text,u.full_name,to_char(su.assigned_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
		FROM cash_shift_users su
		JOIN users u ON u.id=su.user_id AND u.organization_id=su.organization_id
		WHERE su.organization_id=$1
		  AND su.location_id=$2
		  AND su.shift_id=$3
		  AND su.unassigned_at IS NULL
		ORDER BY su.assigned_at,u.full_name
	`, s.OrganizationID, s.LocationID, r.PathValue("id"))
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos cargar el equipo del turno.")
		return
	}
	defer rows.Close()
	items := []cashShiftUserView{}
	for rows.Next() {
		var item cashShiftUserView
		if err := rows.Scan(&item.UserID, &item.Name, &item.AssignedAt); err != nil {
			fail(w, 503, "cash_unavailable", "No pudimos leer el equipo del turno.")
			return
		}
		items = append(items, item)
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (a *API) assignCashShiftUser(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		UserID string `json:"userId"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.UserID) == "" {
		fail(w, 400, "invalid_cash_user", "Selecciona un usuario válido.")
		return
	}
	in.UserID = strings.TrimSpace(in.UserID)

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos asignar el usuario.")
		return
	}
	defer tx.Rollback(r.Context())

	var open bool
	if err := tx.QueryRow(r.Context(), `
		SELECT status='open'
		FROM cash_shifts
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		FOR UPDATE
	`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&open); errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "cash_shift_not_found", "El turno no existe en este local.")
		return
	} else if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos validar el turno.")
		return
	}
	if !open {
		fail(w, 409, "cash_shift_closed", "No puedes asignar usuarios a un turno cerrado.")
		return
	}

	var name string
	if err := tx.QueryRow(r.Context(), `
		SELECT DISTINCT u.full_name
		FROM users u
		JOIN user_roles ur ON ur.user_id=u.id
		WHERE u.id=$1 AND u.organization_id=$2 AND u.active
		  AND (ur.location_id IS NULL OR ur.location_id=$3)
	`, in.UserID, s.OrganizationID, s.LocationID).Scan(&name); errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "cash_user_not_found", "El usuario no está disponible en este local.")
		return
	} else if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos validar el usuario.")
		return
	}

	var item cashShiftUserView
	err = tx.QueryRow(r.Context(), `
		INSERT INTO cash_shift_users(organization_id,location_id,shift_id,user_id,assigned_by)
		VALUES($1,$2,$3,$4,$5)
		RETURNING user_id::text,$6,to_char(assigned_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
	`, s.OrganizationID, s.LocationID, r.PathValue("id"), in.UserID, s.UserID, name).
		Scan(&item.UserID, &item.Name, &item.AssignedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			fail(w, 409, "cash_user_already_assigned", "El usuario ya está asignado a un turno abierto.")
			return
		}
		fail(w, 503, "cash_unavailable", "No pudimos asignar el usuario.")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos confirmar la asignación.")
		return
	}
	writeJSON(w, 201, item)
}

func (a *API) unassignCashShiftUser(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	tag, err := a.db.Exec(r.Context(), `
		UPDATE cash_shift_users
		SET unassigned_at=now()
		WHERE organization_id=$1
		  AND location_id=$2
		  AND shift_id=$3
		  AND user_id=$4
		  AND unassigned_at IS NULL
	`, s.OrganizationID, s.LocationID, r.PathValue("id"), r.PathValue("userId"))
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos retirar al usuario del turno.")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 404, "cash_user_not_assigned", "El usuario ya no está asignado a este turno.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) createCashOperation(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		OperationType string  `json:"operationType"`
		TargetShiftID string  `json:"targetShiftId"`
		Amount        float64 `json:"amount"`
		Reason        string  `json:"reason"`
		Note          string  `json:"note"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.Amount <= 0 {
		fail(w, 400, "invalid_cash_operation", "Ingresa un monto válido.")
		return
	}
	if in.OperationType != "cash_pull" && in.OperationType != "deposit" && in.OperationType != "transfer" {
		fail(w, 400, "invalid_cash_operation", "Selecciona una operación de caja válida.")
		return
	}
	in.Reason = strings.TrimSpace(in.Reason)
	in.Note = strings.TrimSpace(in.Note)
	in.TargetShiftID = strings.TrimSpace(in.TargetShiftID)
	if in.Reason == "" || len(in.Reason) > 120 || len(in.Note) > 240 || (in.OperationType == "transfer" && in.TargetShiftID == "") {
		fail(w, 400, "invalid_cash_operation", "Completa el motivo y los datos de la operación.")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos iniciar la operación de caja.")
		return
	}
	defer tx.Rollback(r.Context())

	sourceShiftID := r.PathValue("id")
	if err := requireCashShiftAssignment(r.Context(), tx, s, sourceShiftID); errors.Is(err, pgx.ErrNoRows) {
		fail(w, 403, "cash_shift_not_assigned", "Debes estar asignado al turno para operar esta caja.")
		return
	} else if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos validar tu asignación.")
		return
	}

	expected, err := cashShiftExpected(r.Context(), tx, s, sourceShiftID)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 409, "cash_shift_closed", "El turno no está abierto.")
		return
	}
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos calcular el saldo disponible.")
		return
	}
	if in.Amount > expected+0.00001 {
		fail(w, 409, "cash_operation_exceeds_expected", "La operación supera el efectivo esperado del turno.")
		return
	}

	if in.OperationType == "transfer" {
		if in.TargetShiftID == sourceShiftID {
			fail(w, 400, "invalid_cash_transfer", "La caja de destino debe ser diferente.")
			return
		}
		var targetOpen bool
		if err := tx.QueryRow(r.Context(), `
			SELECT status='open'
			FROM cash_shifts
			WHERE id=$1 AND organization_id=$2 AND location_id=$3
			FOR UPDATE
		`, in.TargetShiftID, s.OrganizationID, s.LocationID).Scan(&targetOpen); errors.Is(err, pgx.ErrNoRows) || !targetOpen {
			fail(w, 409, "cash_transfer_target_unavailable", "La caja de destino no tiene un turno abierto.")
			return
		} else if err != nil {
			fail(w, 503, "cash_unavailable", "No pudimos validar la caja de destino.")
			return
		}
	}

	var operation cashOperationView
	err = tx.QueryRow(r.Context(), `
		INSERT INTO cash_operations(
		  organization_id,location_id,operation_type,source_shift_id,target_shift_id,
		  amount,reason,note,created_by
		)
		VALUES($1,$2,$3,$4,NULLIF($5,''::text)::uuid,$6,$7,$8,$9)
		RETURNING id::text,operation_type,amount::text,reason,note,to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
	`, s.OrganizationID, s.LocationID, in.OperationType, sourceShiftID, in.TargetShiftID, in.Amount, in.Reason, in.Note, s.UserID).
		Scan(&operation.ID, &operation.OperationType, &operation.Amount, &operation.Reason, &operation.Note, &operation.CreatedAt)
	if err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos registrar la operación.")
		return
	}

	sourceType := in.OperationType
	if in.OperationType == "transfer" {
		sourceType = "transfer_out"
	}
	if _, err := tx.Exec(r.Context(), `
		INSERT INTO cash_movements(
		  organization_id,location_id,shift_id,movement_type,source_type,source_id,
		  amount,reason,note,created_by
		)
		VALUES($1,$2,$3,'expense',$4,$5::uuid,$6,$7,$8,$9)
	`, s.OrganizationID, s.LocationID, sourceShiftID, sourceType, operation.ID, in.Amount, in.Reason, in.Note, s.UserID); err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos afectar la caja de origen.")
		return
	}

	if in.OperationType == "transfer" {
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO cash_movements(
			  organization_id,location_id,shift_id,movement_type,source_type,source_id,
			  amount,reason,note,created_by
			)
			VALUES($1,$2,$3,'income','transfer_in',$4::uuid,$5,$6,$7,$8)
		`, s.OrganizationID, s.LocationID, in.TargetShiftID, operation.ID, in.Amount, in.Reason, in.Note, s.UserID); err != nil {
			fail(w, 503, "cash_unavailable", "No pudimos afectar la caja de destino.")
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		fail(w, 503, "cash_unavailable", "No pudimos confirmar la operación.")
		return
	}
	writeJSON(w, 201, operation)
}

func (a *API) canSeeCashExpected(r *http.Request, s scope) bool {
	var platformAdmin bool
	if err := a.db.QueryRow(r.Context(), `SELECT platform_admin FROM users WHERE id=$1 AND active`, s.UserID).Scan(&platformAdmin); err == nil && platformAdmin {
		return true
	}
	var allowed bool
	if err := a.db.QueryRow(r.Context(), `
		SELECT EXISTS(
		  SELECT 1
		  FROM user_roles ur
		  JOIN roles ro ON ro.id=ur.role_id
		  WHERE ur.user_id=$1
		    AND (ur.location_id IS NULL OR ur.location_id=$2)
		    AND ro.active
		    AND (
		      ro.permissions @> ARRAY['*']::text[]
		      OR ro.permissions @> ARRAY['cash.expected.read']::text[]
		    )
		)
	`, s.UserID, s.LocationID).Scan(&allowed); err != nil {
		return false
	}
	return allowed
}

func (a *API) applyCashExpectedVisibility(r *http.Request, shift *cashShiftView) {
	if shift == nil {
		return
	}
	s := r.Context().Value(scopeKey{}).(scope)
	shift.ExpectedVisible = true
	if shift.Status == "open" && shift.BlindClose && !a.canSeeCashExpected(r, s) {
		shift.ExpectedAmount = ""
		shift.ExpectedVisible = false
	}
}
