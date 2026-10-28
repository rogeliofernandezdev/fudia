package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/billing/application"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"github.com/jackc/pgx/v5"
	"strings"
	"time"
)

var _ application.CashRepository = (*Payments)(nil)

// Shared by the selector and the write validation. Reserved administrators are
// excluded even when another local role would otherwise make them eligible.
const cashCollaboratorPredicate = `
 u.organization_id=$1 AND u.active AND NOT u.platform_admin AND u.id<>$3
 AND EXISTS (SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id=ur.role_id
   AND ro.organization_id=u.organization_id AND ro.active
   WHERE ur.user_id=u.id AND (ur.location_id IS NULL OR ur.location_id=$2))
 AND NOT EXISTS (SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id=ur.role_id
   AND ro.organization_id=u.organization_id AND ro.active
   WHERE ur.user_id=u.id AND ro.system_key='administrator')
`

func (r *Payments) CreateRegister(ctx context.Context, s scope, in domain.RegisterInput) (domain.CashRegisterView, error) {
	var item domain.CashRegisterView
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO cash_registers(organization_id,location_id,name,blind_close,created_by)
		VALUES($1,$2,$3,$4,$5)
		RETURNING id::text,code,name,active,blind_close
	`, s.OrganizationID, s.LocationID, in.Name, in.BlindClose, s.UserID).Scan(&item.ID, &item.Code, &item.Name, &item.Active, &item.BlindClose)
	return item, billingError(err)
}
func (r *Payments) UpdateRegister(ctx context.Context, s scope, id string, in domain.RegisterInput) (domain.CashRegisterView, error) {
	var item domain.CashRegisterView
	err := r.db(ctx).QueryRow(ctx, `
		UPDATE cash_registers
		SET name=$4,blind_close=$5,updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		RETURNING id::text,code,name,active,blind_close
	`, id, s.OrganizationID, s.LocationID, in.Name, in.BlindClose).Scan(&item.ID, &item.Code, &item.Name, &item.Active, &item.BlindClose)
	return item, billingError(err)
}
func (r *Payments) RegisterHasOpenShift(ctx context.Context, s scope, id string) (bool, error) {
	var open bool
	err := r.db(ctx).QueryRow(ctx, `
			SELECT EXISTS(
			  SELECT 1 FROM cash_shifts
			  WHERE cash_register_id=$1 AND organization_id=$2 AND location_id=$3 AND status='open'
			)
		`, id, s.OrganizationID, s.LocationID).Scan(&open)
	return open, billingError(err)
}
func (r *Payments) SetRegisterActive(ctx context.Context, s scope, id string, in domain.RegisterStatusInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE cash_registers SET active=$4,updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
	`, id, s.OrganizationID, s.LocationID, in.Active)
	return tag.RowsAffected(), billingError(err)
}
func (r *Payments) LockRegister(ctx context.Context, s scope, in domain.OpenShiftInput) (string, error) {
	var registerName string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT name FROM cash_registers
		WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND active
		FOR UPDATE
	`, in.CashRegisterID, s.OrganizationID, s.LocationID).Scan(&registerName)
	return registerName, billingError(err)
}
func (r *Payments) OpenShift(ctx context.Context, s scope, in domain.OpenShiftInput) (string, error) {
	var id string
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO cash_shifts(organization_id,location_id,cash_register_id,business_date,opening_amount,opening_note,opened_by)
		VALUES(
		  $1,$2,$3,
		  (now() AT TIME ZONE (SELECT timezone FROM locations WHERE id=$2 AND organization_id=$1))::date,
		  $4,$5,$6
		)
		RETURNING id
	`, s.OrganizationID, s.LocationID, in.CashRegisterID, in.OpeningAmount, in.Note, s.UserID).Scan(&id)
	return id, billingError(err)
}
func (r *Payments) AssignOpeningUser(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		INSERT INTO cash_shift_users(organization_id,location_id,shift_id,user_id,assigned_by)
		VALUES($1,$2,$3,$4,$4)
	`, s.OrganizationID, s.LocationID, id, s.UserID)
	return tag.RowsAffected(), billingError(err)
}
func (r *Payments) LockMovementShift(ctx context.Context, s scope, shiftID string) (string, error) {
	var status string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT status
		FROM cash_shifts
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		FOR UPDATE
	`, shiftID, s.OrganizationID, s.LocationID).Scan(&status)
	return status, billingError(err)
}
func (r *Payments) CreateMovement(ctx context.Context, s scope, shiftID string, in domain.CashMovementInput) (domain.CashMovementView, error) {
	var item domain.CashMovementView
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO cash_movements(organization_id,location_id,shift_id,movement_type,source_type,amount,reason,note,created_by)
		VALUES($1,$2,$3,$4,'manual',$5,$6,$7,$8)
		RETURNING id::text,movement_type,source_type,
		          CASE WHEN source_id IS NULL THEN NULL ELSE source_id::text END,
		          amount::text,reason,note,
		          (SELECT full_name FROM users WHERE id=$8 AND organization_id=$1),
		          to_char(created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
	`, s.OrganizationID, s.LocationID, shiftID, in.MovementType, in.Amount, in.Reason, in.Note, s.UserID).
		Scan(&item.ID, &item.MovementType, &item.SourceType, &item.SourceID, &item.Amount, &item.Reason, &item.Note, &item.CreatedByName, &item.CreatedAt)
	return item, billingError(err)
}
func (r *Payments) LockClosingShift(ctx context.Context, s scope, shiftID string) (string, bool, float64, float64, float64, error) {
	var status string
	var owned bool
	var opening float64
	var income float64
	var expense float64
	err := r.db(ctx).QueryRow(ctx, `
		SELECT cs.status,cs.opened_by=$4,cs.opening_amount::float8,
		       COALESCE((SELECT sum(cm.amount) FROM cash_movements cm WHERE cm.shift_id=cs.id AND cm.organization_id=cs.organization_id AND cm.movement_type='income'),0)::float8,
		       COALESCE((SELECT sum(cm.amount) FROM cash_movements cm WHERE cm.shift_id=cs.id AND cm.organization_id=cs.organization_id AND cm.movement_type='expense'),0)::float8
		FROM cash_shifts cs
		WHERE cs.id=$1 AND cs.organization_id=$2 AND cs.location_id=$3
		FOR UPDATE
	`, shiftID, s.OrganizationID, s.LocationID, s.UserID).Scan(&status, &owned, &opening, &income, &expense)
	return status, owned, opening, income, expense, billingError(err)
}
func (r *Payments) SaveCountLine(ctx context.Context, s scope, shiftID string, line domain.CountLineInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
				INSERT INTO cash_count_lines(organization_id,location_id,shift_id,denomination,quantity)
				VALUES($1,$2,$3,$4,$5)
			`, s.OrganizationID, s.LocationID, shiftID, line.Denomination, line.Quantity)
	return tag.RowsAffected(), billingError(err)
}
func (r *Payments) CloseShift(ctx context.Context, s scope, shiftID string, in domain.CloseShiftInput, expected float64, counted float64, variance float64) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE cash_shifts
		SET status='closed',
		    closing_expected_amount=$4,
		    closing_counted_amount=$5,
		    variance_amount=$6,
		    closing_note=$7,
		    closed_by=$8,
		    closed_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND opened_by=$8 AND status='open'
	`, shiftID, s.OrganizationID, s.LocationID, expected, counted, variance, in.Note, s.UserID)
	return tag.RowsAffected(), billingError(err)
}
func (r *Payments) UnassignClosingUsers(ctx context.Context, s scope, shiftID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE cash_shift_users
		SET unassigned_at=now()
		WHERE organization_id=$1 AND location_id=$2 AND shift_id=$3 AND unassigned_at IS NULL
	`, s.OrganizationID, s.LocationID, shiftID)
	return tag.RowsAffected(), billingError(err)
}
func (r *Payments) SaveReport(ctx context.Context, s scope, shiftID string, data []byte) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `INSERT INTO cash_shift_reports(organization_id,location_id,shift_id,report) VALUES($1,$2,$3,$4)`, s.OrganizationID, s.LocationID, shiftID, data)
	return tag.RowsAffected(), billingError(err)
}
func (r *Payments) AuditClosedShift(ctx context.Context, s scope, shiftID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,metadata) VALUES($1,$2,$3,'cash.shift_closed','cash_shift',$4,jsonb_build_object('reportVersion',1))`, s.OrganizationID, s.LocationID, s.UserID, shiftID)
	return tag.RowsAffected(), billingError(err)
}
func (r *Payments) LockAssignShift(ctx context.Context, s scope, id string) (bool, error) {
	var open bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT status='open'
		FROM cash_shifts
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		FOR UPDATE
	`, id, s.OrganizationID, s.LocationID).Scan(&open)
	return open, billingError(err)
}
func (r *Payments) AssignableUser(ctx context.Context, s scope, in domain.ShiftUserInput) (string, error) {
	var name string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT u.full_name
		FROM users u
		WHERE `+cashCollaboratorPredicate+` AND u.id=$4
		FOR SHARE OF u
	`, s.OrganizationID, s.LocationID, s.UserID, in.UserID).Scan(&name)
	return name, billingError(err)
}
func (r *Payments) AssignUser(ctx context.Context, s scope, id string, in domain.ShiftUserInput, name string) (domain.CashShiftUserView, error) {
	var item domain.CashShiftUserView
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO cash_shift_users(organization_id,location_id,shift_id,user_id,assigned_by)
		VALUES($1,$2,$3,$4,$5)
		RETURNING user_id::text,$6,to_char(assigned_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
	`, s.OrganizationID, s.LocationID, id, in.UserID, s.UserID, name).
		Scan(&item.UserID, &item.Name, &item.AssignedAt)
	return item, billingError(err)
}
func (r *Payments) UnassignUser(ctx context.Context, s scope, id string, userID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE cash_shift_users
		SET unassigned_at=now()
		WHERE organization_id=$1
		  AND location_id=$2
		  AND shift_id=$3
		  AND user_id=$4
		  AND unassigned_at IS NULL
		  AND EXISTS (SELECT 1 FROM cash_shifts cs WHERE cs.id=$3
		    AND cs.organization_id=$1 AND cs.location_id=$2 AND cs.opened_by<>$4)
	`, s.OrganizationID, s.LocationID, id, userID)
	return tag.RowsAffected(), billingError(err)
}
func (r *Payments) LockTargetShift(ctx context.Context, s scope, in domain.CashOperationInput) (bool, error) {
	var targetOpen bool
	err := r.db(ctx).QueryRow(ctx, `
			SELECT status='open'
			FROM cash_shifts
			WHERE id=$1 AND organization_id=$2 AND location_id=$3
			FOR UPDATE
		`, in.TargetShiftID, s.OrganizationID, s.LocationID).Scan(&targetOpen)
	return targetOpen, billingError(err)
}
func (r *Payments) CreateOperation(ctx context.Context, s scope, sourceShiftID string, in domain.CashOperationInput) (domain.CashOperationView, error) {
	var operation domain.CashOperationView
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO cash_operations(
		  organization_id,location_id,operation_type,source_shift_id,target_shift_id,
		  amount,reason,note,created_by
		)
		VALUES($1,$2,$3,$4,NULLIF($5,''::text)::uuid,$6,$7,$8,$9)
		RETURNING id::text,operation_type,amount::text,reason,note,to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
	`, s.OrganizationID, s.LocationID, in.OperationType, sourceShiftID, in.TargetShiftID, in.Amount, in.Reason, in.Note, s.UserID).
		Scan(&operation.ID, &operation.OperationType, &operation.Amount, &operation.Reason, &operation.Note, &operation.CreatedAt)
	return operation, billingError(err)
}
func (r *Payments) OperationOutflow(ctx context.Context, s scope, sourceShiftID string, sourceType string, operation domain.CashOperationView, in domain.CashOperationInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		INSERT INTO cash_movements(
		  organization_id,location_id,shift_id,movement_type,source_type,source_id,
		  amount,reason,note,created_by
		)
		VALUES($1,$2,$3,'expense',$4,$5::uuid,$6,$7,$8,$9)
	`, s.OrganizationID, s.LocationID, sourceShiftID, sourceType, operation.ID, in.Amount, in.Reason, in.Note, s.UserID)
	return tag.RowsAffected(), billingError(err)
}
func (r *Payments) OperationInflow(ctx context.Context, s scope, operation domain.CashOperationView, in domain.CashOperationInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
			INSERT INTO cash_movements(
			  organization_id,location_id,shift_id,movement_type,source_type,source_id,
			  amount,reason,note,created_by
			)
			VALUES($1,$2,$3,'income','transfer_in',$4::uuid,$5,$6,$7,$8)
		`, s.OrganizationID, s.LocationID, in.TargetShiftID, operation.ID, in.Amount, in.Reason, in.Note, s.UserID)
	return tag.RowsAffected(), billingError(err)
}

const cashShiftColumns = `
	cs.id::text,
	cs.code,
	cr.id::text,
	cr.name,
	cr.blind_close,
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
	cs.opened_by::text,
	opened.full_name,
	COALESCE(closed.full_name,''),
	ARRAY(
	  SELECT u.full_name
	  FROM cash_shift_users su
	  JOIN users u ON u.id=su.user_id AND u.organization_id=su.organization_id
	  WHERE su.shift_id=cs.id AND su.organization_id=cs.organization_id
	    AND su.location_id=cs.location_id AND su.unassigned_at IS NULL
	    AND su.user_id<>cs.opened_by
	    AND cs.status='open'
	  ORDER BY su.assigned_at,u.full_name,su.user_id
	),
	cs.business_date::text,
	to_char(cs.opened_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
	CASE WHEN cs.closed_at IS NULL THEN NULL ELSE to_char(cs.closed_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END,
	(SELECT count(*) FROM cash_movements cm WHERE cm.shift_id=cs.id AND cm.organization_id=cs.organization_id)
`

func scanCashShift(row pgx.Row) (domain.CashShiftView, error) {
	shift := domain.CashShiftView{DetailsVisible: true}
	err := row.Scan(
		&shift.ID,
		&shift.Code,
		&shift.CashRegisterID,
		&shift.CashRegisterName,
		&shift.BlindClose,
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
		&shift.OpenedByID,
		&shift.OpenedByName,
		&shift.ClosedByName,
		&shift.ActiveUserNames,
		&shift.BusinessDate,
		&shift.OpenedAt,
		&shift.ClosedAt,
		&shift.MovementCount,
	)
	return shift, err
}
func (r *Payments) Shift(ctx context.Context, s scope, id string) (domain.CashShiftView, error) {
	shift, err := scanCashShift(r.db(ctx).QueryRow(ctx, `
		SELECT `+cashShiftColumns+`
		FROM cash_shifts cs
		JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
		JOIN users opened ON opened.id=cs.opened_by AND opened.organization_id=cs.organization_id
		LEFT JOIN users closed ON closed.id=cs.closed_by AND closed.organization_id=cs.organization_id
		WHERE cs.id=$1 AND cs.organization_id=$2 AND cs.location_id=$3
	`, id, s.OrganizationID, s.LocationID))
	return shift, billingError(err)
}
func (r *Payments) OpenShiftForRegister(ctx context.Context, s scope, registerID string) (domain.CashShiftView, error) {
	shift, err := scanCashShift(r.db(ctx).QueryRow(ctx, `
		SELECT `+cashShiftColumns+`
		FROM cash_shifts cs
		JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
		JOIN users opened ON opened.id=cs.opened_by AND opened.organization_id=cs.organization_id
		LEFT JOIN users closed ON closed.id=cs.closed_by AND closed.organization_id=cs.organization_id
		WHERE cs.organization_id=$1 AND cs.location_id=$2 AND cs.cash_register_id=$3 AND cs.status='open'
		LIMIT 1
	`, s.OrganizationID, s.LocationID, registerID))
	return shift, billingError(err)
}
func (r *Payments) UserShift(ctx context.Context, s scope) (domain.CashShiftView, error) {
	shift, err := scanCashShift(r.db(ctx).QueryRow(ctx, `
		SELECT `+cashShiftColumns+`
		FROM cash_shifts cs
		JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
		JOIN users opened ON opened.id=cs.opened_by AND opened.organization_id=cs.organization_id
		LEFT JOIN users closed ON closed.id=cs.closed_by AND closed.organization_id=cs.organization_id
		JOIN cash_shift_users su ON su.shift_id=cs.id
		  AND su.organization_id=cs.organization_id
		  AND su.location_id=cs.location_id
		  AND su.user_id=$3
		  AND su.unassigned_at IS NULL
		WHERE cs.organization_id=$1 AND cs.location_id=$2 AND cs.status='open'
		ORDER BY su.assigned_at DESC
		LIMIT 1
	`, s.OrganizationID, s.LocationID, s.UserID))
	return shift, billingError(err)
}
func (r *Payments) Movements(ctx context.Context, s scope, shiftID string) ([]domain.CashMovementView, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT cm.id::text,cm.movement_type,cm.source_type,
		       CASE WHEN cm.source_id IS NULL THEN NULL ELSE cm.source_id::text END,
		       cm.amount::text,cm.reason,cm.note,u.full_name,
		       to_char(cm.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
		FROM cash_movements cm
		JOIN users u ON u.id=cm.created_by AND u.organization_id=cm.organization_id
		WHERE cm.shift_id=$1 AND cm.organization_id=$2 AND cm.location_id=$3
		ORDER BY cm.created_at DESC,cm.id DESC
	`, shiftID, s.OrganizationID, s.LocationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.CashMovementView{}
	for rows.Next() {
		var item domain.CashMovementView
		if err := rows.Scan(&item.ID, &item.MovementType, &item.SourceType, &item.SourceID, &item.Amount, &item.Reason, &item.Note, &item.CreatedByName, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r *Payments) Registers(ctx context.Context, s scope, query string) ([]domain.CashRegisterView, error) {
	var out []domain.CashRegisterView
	q := strings.TrimSpace(query)
	rows, err := r.db(ctx).Query(ctx, `
		SELECT id::text,code,name,active,blind_close
		FROM cash_registers
		WHERE organization_id=$1 AND location_id=$2
		  AND ($3='' OR name ILIKE '%'||$3||'%' OR code ILIKE '%'||$3||'%')
		ORDER BY active DESC,name,created_at
	`, s.OrganizationID, s.LocationID, q)
	if err != nil {
		return out, err
	}
	items := []domain.CashRegisterView{}
	for rows.Next() {
		var item domain.CashRegisterView
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Active, &item.BlindClose); err != nil {
			rows.Close()
			return out, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()

	return items, nil
}
func (r *Payments) Shifts(ctx context.Context, s scope, f application.CashFilter) (application.CashPage, error) {
	var out application.CashPage
	page, size, q, status := f.Page, f.Size, f.Query, f.Status
	where := `cs.organization_id=$1 AND cs.location_id=$2
		AND ($3='' OR cs.code ILIKE '%'||$3||'%' OR cr.name ILIKE '%'||$3||'%' OR opened.full_name ILIKE '%'||$3||'%' OR COALESCE(closed.full_name,'') ILIKE '%'||$3||'%'
		  OR EXISTS (
		    SELECT 1 FROM cash_shift_users su
		    JOIN users u ON u.id=su.user_id AND u.organization_id=su.organization_id
		    WHERE su.shift_id=cs.id AND su.organization_id=cs.organization_id
		      AND su.location_id=cs.location_id AND su.unassigned_at IS NULL
		      AND cs.status='open' AND u.full_name ILIKE '%'||$3||'%'
		  ))
		AND ($4='' OR cs.status=$4)
		AND ($5::uuid IS NULL OR cs.opened_by=$5)`
	var ownerID any
	if f.OwnerID != "" {
		ownerID = f.OwnerID
	}

	var total int
	if err := r.db(ctx).QueryRow(ctx, `
		SELECT count(*)
		FROM cash_shifts cs
		JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
		JOIN users opened ON opened.id=cs.opened_by AND opened.organization_id=cs.organization_id
		LEFT JOIN users closed ON closed.id=cs.closed_by AND closed.organization_id=cs.organization_id
		WHERE `+where, s.OrganizationID, s.LocationID, q, status, ownerID).Scan(&total); err != nil {
		return out, err
	}

	rows, err := r.db(ctx).Query(ctx, `
		SELECT `+cashShiftColumns+`
		FROM cash_shifts cs
		JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
		JOIN users opened ON opened.id=cs.opened_by AND opened.organization_id=cs.organization_id
		LEFT JOIN users closed ON closed.id=cs.closed_by AND closed.organization_id=cs.organization_id
		WHERE `+where+`
		ORDER BY cs.opened_at DESC
		LIMIT $6 OFFSET $7
	`, s.OrganizationID, s.LocationID, q, status, ownerID, size, (page-1)*size)
	if err != nil {
		return out, err
	}
	defer rows.Close()

	items := []domain.CashShiftView{}
	for rows.Next() {
		shift, err := scanCashShift(rows)
		if err != nil {
			return out, err
		}

		items = append(items, shift)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}

	out.Items = items
	out.Total = total
	out.Page = page
	out.PageSize = size
	return out, rows.Err()
}
func (r *Payments) UserOptions(ctx context.Context, s scope) ([]domain.CashUserOption, error) {
	var out []domain.CashUserOption
	rows, err := r.db(ctx).Query(ctx, `
		SELECT u.id::text,u.full_name,
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
		WHERE `+cashCollaboratorPredicate+`
		ORDER BY u.full_name
	`, s.OrganizationID, s.LocationID, s.UserID)
	if err != nil {
		return out, err
	}
	defer rows.Close()

	items := []domain.CashUserOption{}
	for rows.Next() {
		var item domain.CashUserOption
		if err := rows.Scan(&item.ID, &item.Name, &item.AssignedShiftID); err != nil {
			return out, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r *Payments) ShiftUsers(ctx context.Context, s scope, id string) ([]domain.CashShiftUserView, error) {
	var out []domain.CashShiftUserView
	rows, err := r.db(ctx).Query(ctx, `
		SELECT su.user_id::text,u.full_name,to_char(su.assigned_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
		FROM cash_shift_users su
		JOIN users u ON u.id=su.user_id AND u.organization_id=su.organization_id
		JOIN cash_shifts cs ON cs.id=su.shift_id AND cs.organization_id=su.organization_id AND cs.location_id=su.location_id
		WHERE su.organization_id=$1
		  AND su.location_id=$2
		  AND su.shift_id=$3
		  AND su.user_id<>cs.opened_by
		  AND su.unassigned_at IS NULL
		ORDER BY su.assigned_at,u.full_name
	`, s.OrganizationID, s.LocationID, id)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	items := []domain.CashShiftUserView{}
	for rows.Next() {
		var item domain.CashShiftUserView
		if err := rows.Scan(&item.UserID, &item.Name, &item.AssignedAt); err != nil {
			return out, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r *Payments) BuildReport(ctx context.Context, s scope, id string) (domain.CashShiftReport, error) {
	out := domain.CashShiftReport{Version: 1, GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Sales: []domain.CashReportSale{}, Payments: []domain.CashReportPayment{}, Refunds: []domain.CashReportPayment{}, Methods: []domain.CashReportMethod{}, Counts: []domain.CashReportCount{}}
	var err error
	out.Shift, err = scanCashShift(r.db(ctx).QueryRow(ctx, `SELECT `+cashShiftColumns+`
 FROM cash_shifts cs JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
 JOIN users opened ON opened.id=cs.opened_by AND opened.organization_id=cs.organization_id
 LEFT JOIN users closed ON closed.id=cs.closed_by AND closed.organization_id=cs.organization_id
 WHERE cs.id=$1 AND cs.organization_id=$2 AND cs.location_id=$3`, id, s.OrganizationID, s.LocationID))
	if err != nil {
		return out, billingError(err)
	}
	out.Shift.ExpectedVisible = true
	err = r.db(ctx).QueryRow(ctx, `SELECT COALESCE(NULLIF(o.trade_name,''),o.legal_name),l.name,l.timezone,fp.country_code,fp.currency,fp.currency_symbol,fp.currency_position,fp.currency_decimals
 FROM locations l JOIN organizations o ON o.id=l.organization_id
 JOIN organization_fiscal_profiles fp ON fp.id=l.fiscal_profile_id AND fp.organization_id=l.organization_id
 WHERE l.id=$1 AND l.organization_id=$2`, s.LocationID, s.OrganizationID).Scan(&out.OrganizationName, &out.LocationName, &out.Timezone, &out.Country, &out.Currency, &out.CurrencySymbol, &out.CurrencyPosition, &out.CurrencyDecimals)
	if err != nil {
		return out, billingError(err)
	}
	rows, err := r.db(ctx).Query(ctx, `SELECT cm.id::text,cm.movement_type,cm.source_type,cm.source_id::text,cm.amount::text,cm.reason,cm.note,u.full_name,
 to_char(cm.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
 FROM cash_movements cm JOIN users u ON u.id=cm.created_by AND u.organization_id=cm.organization_id
 WHERE cm.shift_id=$1 AND cm.organization_id=$2 AND cm.location_id=$3 ORDER BY cm.created_at,cm.id`, id, s.OrganizationID, s.LocationID)
	if err != nil {
		return out, billingError(err)
	}
	out.Shift.Movements = []domain.CashMovementView{}
	for rows.Next() {
		var v domain.CashMovementView
		err = rows.Scan(&v.ID, &v.MovementType, &v.SourceType, &v.SourceID, &v.Amount, &v.Reason, &v.Note, &v.CreatedByName, &v.CreatedAt)
		if err != nil {
			rows.Close()
			return out, billingError(err)
		}
		out.Shift.Movements = append(out.Shift.Movements, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, billingError(err)
	}
	// EXISTS includes each order line only once, regardless of the number of payments.
	rows, err = r.db(ctx).Query(ctx, `SELECT i.id::text,o.code,COALESCE(NULLIF(t.name,''),NULLIF(o.customer_name,''),'Venta sin mesa'),i.name,
 concat_ws(' · ',NULLIF(i.note,''),(SELECT string_agg(sel.group_name||': '||sel.option_name,' · ' ORDER BY sel.id) FROM order_item_combo_selections sel WHERE sel.order_item_id=i.id AND sel.organization_id=i.organization_id)),
 i.qty::text,i.unit_price::text,round(i.qty*i.unit_price,2)::text,
 to_char(i.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),COALESCE(u.full_name,'')
 FROM order_items i JOIN orders o ON o.id=i.order_id AND o.organization_id=i.organization_id
 LEFT JOIN tables t ON t.id=o.table_id AND t.organization_id=o.organization_id
 LEFT JOIN users u ON u.id=o.waiter_id AND u.organization_id=o.organization_id
 WHERE o.organization_id=$2 AND o.location_id=$3 AND EXISTS(SELECT 1 FROM payments p WHERE p.order_id=o.id AND p.organization_id=o.organization_id AND p.location_id=o.location_id AND p.shift_id=$1)
 ORDER BY o.created_at,o.id,i.created_at,i.id`, id, s.OrganizationID, s.LocationID)
	if err != nil {
		return out, billingError(err)
	}
	for rows.Next() {
		var v domain.CashReportSale
		err = rows.Scan(&v.ID, &v.OrderCode, &v.Customer, &v.Name, &v.Details, &v.Quantity, &v.UnitPrice, &v.Total, &v.CreatedAt, &v.WaiterName)
		if err != nil {
			rows.Close()
			return out, billingError(err)
		}
		out.Sales = append(out.Sales, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, billingError(err)
	}
	// Refunds belong to the shift where they were registered, not the original payment's shift.
	rows, err = r.db(ctx).Query(ctx, `SELECT p.id::text,o.code,pm.name,p.amount::text,p.reference,u.full_name,to_char(p.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),false
 FROM payments p JOIN orders o ON o.id=p.order_id AND o.organization_id=p.organization_id AND o.location_id=p.location_id
 JOIN payment_methods pm ON pm.organization_id=p.organization_id AND pm.code=p.method JOIN users u ON u.id=p.created_by AND u.organization_id=p.organization_id
 WHERE p.shift_id=$1 AND p.organization_id=$2 AND p.location_id=$3
 UNION ALL SELECT pr.id::text,o.code,pm.name,pr.amount::text,pr.reason,u.full_name,to_char(pr.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),true
 FROM payment_refunds pr JOIN payments p ON p.id=pr.payment_id AND p.organization_id=pr.organization_id AND p.location_id=pr.location_id
 JOIN orders o ON o.id=p.order_id AND o.organization_id=p.organization_id AND o.location_id=p.location_id
 JOIN payment_methods pm ON pm.organization_id=p.organization_id AND pm.code=p.method JOIN users u ON u.id=pr.created_by AND u.organization_id=pr.organization_id
 WHERE pr.shift_id=$1 AND pr.organization_id=$2 AND pr.location_id=$3 ORDER BY 7,1`, id, s.OrganizationID, s.LocationID)
	if err != nil {
		return out, billingError(err)
	}
	for rows.Next() {
		var v domain.CashReportPayment
		var refund bool
		err = rows.Scan(&v.ID, &v.OrderCode, &v.MethodName, &v.Amount, &v.Reference, &v.CreatedByName, &v.CreatedAt, &refund)
		if err != nil {
			rows.Close()
			return out, billingError(err)
		}
		if refund {
			out.Refunds = append(out.Refunds, v)
		} else {
			out.Payments = append(out.Payments, v)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, billingError(err)
	}
	err = r.db(ctx).QueryRow(ctx, `WITH totals AS(SELECT COALESCE((SELECT sum(amount) FROM payments WHERE shift_id=$1 AND organization_id=$2 AND location_id=$3),0) collected,COALESCE((SELECT sum(amount) FROM payment_refunds WHERE shift_id=$1 AND organization_id=$2 AND location_id=$3),0) refunded) SELECT collected::text,refunded::text,(collected-refunded)::text FROM totals`, id, s.OrganizationID, s.LocationID).Scan(&out.CollectedAmount, &out.RefundedAmount, &out.NetCollectedAmount)
	if err != nil {
		return out, billingError(err)
	}
	rows, err = r.db(ctx).Query(ctx, `SELECT pm.name,sum(p.amount)::text FROM payments p JOIN payment_methods pm ON pm.organization_id=p.organization_id AND pm.code=p.method WHERE p.shift_id=$1 AND p.organization_id=$2 AND p.location_id=$3 GROUP BY pm.code,pm.name ORDER BY pm.name,pm.code`, id, s.OrganizationID, s.LocationID)
	if err != nil {
		return out, billingError(err)
	}
	for rows.Next() {
		var v domain.CashReportMethod
		err = rows.Scan(&v.Name, &v.Amount)
		if err != nil {
			rows.Close()
			return out, billingError(err)
		}
		out.Methods = append(out.Methods, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, billingError(err)
	}
	rows, err = r.db(ctx).Query(ctx, `SELECT denomination::text,quantity,(denomination*quantity)::text FROM cash_count_lines WHERE shift_id=$1 AND organization_id=$2 AND location_id=$3 ORDER BY denomination DESC`, id, s.OrganizationID, s.LocationID)
	if err != nil {
		return out, billingError(err)
	}
	for rows.Next() {
		var v domain.CashReportCount
		err = rows.Scan(&v.Denomination, &v.Quantity, &v.Total)
		if err != nil {
			rows.Close()
			return out, billingError(err)
		}
		out.Counts = append(out.Counts, v)
	}
	err = rows.Err()
	rows.Close()
	return out, billingError(err)
}
func (r *Payments) StoredReport(ctx context.Context, s scope, id string) ([]byte, error) {
	var stored []byte
	err := r.db(ctx).QueryRow(ctx, `SELECT report FROM cash_shift_reports WHERE shift_id=$1 AND organization_id=$2 AND location_id=$3`, id, s.OrganizationID, s.LocationID).Scan(&stored)
	return stored, billingError(err)
}
func (r *Payments) IsPlatformAdmin(ctx context.Context, s scope) (bool, error) {
	var platformAdmin bool
	err := r.db(ctx).QueryRow(ctx, `SELECT platform_admin FROM users WHERE id=$1 AND active`, s.UserID).Scan(&platformAdmin)
	return platformAdmin, billingError(err)
}

func (r *Payments) CanReadAllShifts(ctx context.Context, s scope) (bool, error) {
	var allowed bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS (
	 SELECT 1 FROM users u WHERE u.id=$1 AND u.active AND (
	  u.platform_admin OR (u.organization_id=$2 AND EXISTS (
	   SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id=ur.role_id
	   WHERE ur.user_id=u.id AND ro.organization_id=$2
	    AND ro.active AND ro.system_key='administrator'
	  ))
	 )
	)`, s.UserID, s.OrganizationID).Scan(&allowed)
	return allowed, billingError(err)
}

func (r *Payments) OwnsShift(ctx context.Context, s scope, id string) (bool, error) {
	var owned bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS (
	 SELECT 1 FROM cash_shifts WHERE id=$1 AND organization_id=$2
	  AND location_id=$3 AND opened_by=$4
	)`, id, s.OrganizationID, s.LocationID, s.UserID).Scan(&owned)
	return owned, billingError(err)
}
func (r *Payments) CanSeeExpected(ctx context.Context, s scope) (bool, error) {
	var allowed bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT EXISTS(
		  SELECT 1
		  FROM user_roles ur
		  JOIN roles ro ON ro.id=ur.role_id
		  WHERE ur.user_id=$1
		    AND (ur.location_id IS NULL OR ur.location_id=$2 OR ro.system_key='administrator')
		    AND ro.active
		    AND ro.organization_id=$3
		    AND (
		      ro.permissions @> ARRAY['*']::text[]
		      OR ro.permissions @> ARRAY['cash.expected.read']::text[]
		    )
		)
	`, s.UserID, s.LocationID, s.OrganizationID).Scan(&allowed)
	return allowed, billingError(err)
}
