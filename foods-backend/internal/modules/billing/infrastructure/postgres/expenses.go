package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/billing/application"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Expenses struct{ pool *pgxpool.Pool }

func NewExpenses(pool *pgxpool.Pool) *Expenses { return &Expenses{pool} }
func (r *Expenses) db(ctx context.Context) database.Executor {
	return database.ExecutorFor(ctx, r.pool)
}

var _ application.ExpenseRepository = (*Expenses)(nil)

const expenseColumns = `
	e.id::text,e.category_id::text,c.name,e.description,e.amount::text,e.payment_method,
	e.business_date::text,e.reference,e.notes,e.status,created.full_name,
	to_char(e.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),
	COALESCE(voided.full_name,''),
	CASE WHEN e.voided_at IS NULL THEN NULL ELSE to_char(e.voided_at,'YYYY-MM-DD"T"HH24:MI:SSOF') END,
	e.void_reason,
	COALESCE(reg.name,''),
	COALESCE(cs.status='open',false)
`

// expenseJoins agrega la caja del turno de origen de un gasto en efectivo.
const expenseJoins = ` LEFT JOIN cash_shifts cs ON cs.id=e.cash_shift_id AND cs.organization_id=e.organization_id AND cs.location_id=e.location_id
	LEFT JOIN cash_registers reg ON reg.id=cs.cash_register_id AND reg.organization_id=cs.organization_id `

func scanExpense(row pgx.Row) (domain.ExpenseView, error) {
	var item domain.ExpenseView
	err := row.Scan(
		&item.ID, &item.CategoryID, &item.CategoryName, &item.Description, &item.Amount,
		&item.PaymentMethod, &item.BusinessDate, &item.Reference, &item.Notes, &item.Status,
		&item.CreatedByName, &item.CreatedAt, &item.VoidedByName, &item.VoidedAt, &item.VoidReason,
		&item.CashRegisterName, &item.CashShiftOpen,
	)
	return item, err
}
func (r *Expenses) Get(ctx context.Context, s requestscope.Scope, id string) (domain.ExpenseView, error) {
	v, err := scanExpense(r.db(ctx).QueryRow(ctx, `
		SELECT `+expenseColumns+`
		FROM expenses e
		JOIN expense_categories c ON c.id=e.category_id AND c.organization_id=e.organization_id
		JOIN users created ON created.id=e.created_by AND created.organization_id=e.organization_id
		LEFT JOIN users voided ON voided.id=e.voided_by AND voided.organization_id=e.organization_id`+expenseJoins+`
		WHERE e.id=$1 AND e.organization_id=$2 AND e.location_id=$3
	`, id, s.OrganizationID, s.LocationID))
	return v, billingError(err)
}
func (r *Expenses) List(ctx context.Context, s requestscope.Scope, f application.ExpenseFilter) (application.ExpensePage, error) {
	out := application.ExpensePage{Items: []domain.ExpenseView{}, Page: f.Page, PageSize: f.Size}
	where := `
		e.organization_id=$1 AND e.location_id=$2
		AND ($3='' OR e.description ILIKE '%'||$3||'%' OR e.reference ILIKE '%'||$3||'%' OR c.name ILIKE '%'||$3||'%' OR created.full_name ILIKE '%'||$3||'%')
		AND ($4='' OR e.category_id::text=$4)
		AND ($5='' OR e.status=$5)
		AND ($6='' OR e.business_date>=NULLIF($6,'')::date)
		AND ($7='' OR e.business_date<=NULLIF($7,'')::date)
	`
	if err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM expenses e JOIN expense_categories c ON c.id=e.category_id AND c.organization_id=e.organization_id JOIN users created ON created.id=e.created_by AND created.organization_id=e.organization_id WHERE `+where, s.OrganizationID, s.LocationID, f.Query, f.CategoryID, f.Status, f.From, f.To).Scan(&out.Total); err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `SELECT `+expenseColumns+` FROM expenses e JOIN expense_categories c ON c.id=e.category_id AND c.organization_id=e.organization_id JOIN users created ON created.id=e.created_by AND created.organization_id=e.organization_id LEFT JOIN users voided ON voided.id=e.voided_by AND voided.organization_id=e.organization_id`+expenseJoins+`WHERE `+where+` ORDER BY e.business_date DESC,e.created_at DESC,e.id DESC LIMIT $8 OFFSET $9`, s.OrganizationID, s.LocationID, f.Query, f.CategoryID, f.Status, f.From, f.To, f.Size, (f.Page-1)*f.Size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		v, e := scanExpense(rows)
		if e != nil {
			return out, e
		}
		out.Items = append(out.Items, v)
	}
	return out, rows.Err()
}
func (r *Expenses) CategoryActive(ctx context.Context, org, id string) (bool, error) {
	var active bool
	err := r.db(ctx).QueryRow(ctx, `SELECT active FROM expense_categories WHERE id::text=$1 AND organization_id=$2`, id, org).Scan(&active)
	return active, billingError(err)
}
func (r *Expenses) Create(ctx context.Context, s requestscope.Scope, in domain.ExpenseInput) (string, error) {
	var id string
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO expenses(organization_id,location_id,category_id,business_date,description,amount,payment_method,reference,notes,created_by)
		VALUES($1,$2,$3,$4,$5,$6::numeric,$7,$8,$9,$10)
		RETURNING id::text
	`, s.OrganizationID, s.LocationID, in.CategoryID, in.BusinessDate, in.Description, in.Amount, in.PaymentMethod, in.Reference, in.Notes, s.UserID).Scan(&id)
	return id, err
}
// Status devuelve el estado del gasto y, si salió de un turno de caja, el estado
// de ese turno ("" cuando no pasó por la caja).
func (r *Expenses) Status(ctx context.Context, s requestscope.Scope, id string) (string, string, error) {
	var status, shiftStatus string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT e.status,COALESCE(cs.status,'')
		FROM expenses e
		LEFT JOIN cash_shifts cs ON cs.id=e.cash_shift_id AND cs.organization_id=e.organization_id AND cs.location_id=e.location_id
		WHERE e.id=$1 AND e.organization_id=$2 AND e.location_id=$3`, id, s.OrganizationID, s.LocationID).Scan(&status, &shiftStatus)
	return status, shiftStatus, billingError(err)
}

// Void anula el gasto en una sola sentencia. Si salió del efectivo de un turno,
// solo procede con ese turno abierto (bloqueado frente al cierre) y devuelve el
// importe con un ingreso 'expense_void' en el mismo turno.
func (r *Expenses) Void(ctx context.Context, s requestscope.Scope, id, reason string) (int64, error) {
	var count int64
	err := r.db(ctx).QueryRow(ctx, `
		WITH shift AS (
		  SELECT cs.id FROM expenses e
		  JOIN cash_shifts cs ON cs.id=e.cash_shift_id AND cs.organization_id=e.organization_id AND cs.location_id=e.location_id
		  WHERE e.id=$1 AND e.organization_id=$2 AND e.location_id=$3 AND cs.status='open'
		  FOR UPDATE OF cs
		), voided AS (
		  UPDATE expenses SET status='void',voided_by=$4,voided_at=now(),void_reason=$5,updated_at=now()
		  WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND status='active'
		    AND (cash_shift_id IS NULL OR EXISTS (SELECT 1 FROM shift))
		  RETURNING id,cash_shift_id,amount,description
		), refund AS (
		  INSERT INTO cash_movements(organization_id,location_id,shift_id,movement_type,source_type,source_id,amount,reason,note,created_by)
		  SELECT $2,$3,v.cash_shift_id,'income','expense_void',v.id,v.amount,left('Anulación de gasto: '||v.description,120),$5,$4
		  FROM voided v WHERE v.cash_shift_id IS NOT NULL
		)
		SELECT count(*) FROM voided`, id, s.OrganizationID, s.LocationID, s.UserID, reason).Scan(&count)
	return count, err
}
func (r *Expenses) Categories(ctx context.Context, org string, f application.MethodFilter) (application.ExpenseCategoryPage, error) {
	out := application.ExpenseCategoryPage{Items: []domain.ExpenseCategoryView{}, Page: f.Page, PageSize: f.Size}
	where := `organization_id=$1 AND ($2='' OR name ILIKE '%'||$2||'%') AND ($3='' OR ($3='active' AND active) OR ($3='inactive' AND NOT active))`
	if err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM expense_categories WHERE `+where, org, f.Query, f.Status).Scan(&out.Total); err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `SELECT id::text,name,active FROM expense_categories WHERE `+where+` ORDER BY active DESC,name LIMIT $4 OFFSET $5`, org, f.Query, f.Status, f.Size, (f.Page-1)*f.Size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var v domain.ExpenseCategoryView
		if err = rows.Scan(&v.ID, &v.Name, &v.Active); err != nil {
			return out, err
		}
		out.Items = append(out.Items, v)
	}
	return out, rows.Err()
}
func (r *Expenses) SaveCategory(ctx context.Context, org, id, name string, updating bool) (domain.ExpenseCategoryView, error) {
	var v domain.ExpenseCategoryView
	var row pgx.Row
	if updating {
		row = r.db(ctx).QueryRow(ctx, `UPDATE expense_categories SET name=$3,updated_at=now() WHERE id=$1 AND organization_id=$2 RETURNING id::text,name,active`, id, org, name)
	} else {
		row = r.db(ctx).QueryRow(ctx, `INSERT INTO expense_categories(organization_id,name) VALUES($1,$2) RETURNING id::text,name,active`, org, name)
	}
	err := row.Scan(&v.ID, &v.Name, &v.Active)
	return v, billingError(err)
}
func (r *Expenses) SetCategoryActive(ctx context.Context, org, id string, active bool) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE expense_categories SET active=$3,updated_at=now() WHERE id=$1 AND organization_id=$2`, id, org, active)
	return tag.RowsAffected(), err
}
