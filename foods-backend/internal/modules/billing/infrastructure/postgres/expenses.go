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
	e.void_reason
`

func scanExpense(row pgx.Row) (domain.ExpenseView, error) {
	var item domain.ExpenseView
	err := row.Scan(
		&item.ID, &item.CategoryID, &item.CategoryName, &item.Description, &item.Amount,
		&item.PaymentMethod, &item.BusinessDate, &item.Reference, &item.Notes, &item.Status,
		&item.CreatedByName, &item.CreatedAt, &item.VoidedByName, &item.VoidedAt, &item.VoidReason,
	)
	return item, err
}
func (r *Expenses) Get(ctx context.Context, s requestscope.Scope, id string) (domain.ExpenseView, error) {
	v, err := scanExpense(r.db(ctx).QueryRow(ctx, `
		SELECT `+expenseColumns+`
		FROM expenses e
		JOIN expense_categories c ON c.id=e.category_id AND c.organization_id=e.organization_id
		JOIN users created ON created.id=e.created_by AND created.organization_id=e.organization_id
		LEFT JOIN users voided ON voided.id=e.voided_by AND voided.organization_id=e.organization_id
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
	rows, err := r.db(ctx).Query(ctx, `SELECT `+expenseColumns+` FROM expenses e JOIN expense_categories c ON c.id=e.category_id AND c.organization_id=e.organization_id JOIN users created ON created.id=e.created_by AND created.organization_id=e.organization_id LEFT JOIN users voided ON voided.id=e.voided_by AND voided.organization_id=e.organization_id WHERE `+where+` ORDER BY e.business_date DESC,e.created_at DESC,e.id DESC LIMIT $8 OFFSET $9`, s.OrganizationID, s.LocationID, f.Query, f.CategoryID, f.Status, f.From, f.To, f.Size, (f.Page-1)*f.Size)
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
func (r *Expenses) Status(ctx context.Context, s requestscope.Scope, id string) (string, error) {
	var status string
	err := r.db(ctx).QueryRow(ctx, `SELECT status FROM expenses WHERE id=$1 AND organization_id=$2 AND location_id=$3`, id, s.OrganizationID, s.LocationID).Scan(&status)
	return status, billingError(err)
}
func (r *Expenses) Void(ctx context.Context, s requestscope.Scope, id, reason string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE expenses SET status='void',voided_by=$4,voided_at=now(),void_reason=$5,updated_at=now() WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND status='active'`, id, s.OrganizationID, s.LocationID, s.UserID, reason)
	return tag.RowsAffected(), err
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
