package postgres

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/billing/application"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type Methods struct{ pool *pgxpool.Pool }

func NewMethods(pool *pgxpool.Pool) *Methods { return &Methods{pool} }

var _ application.MethodRepository = (*Methods)(nil)

func (r *Methods) db(ctx context.Context) database.Executor { return database.ExecutorFor(ctx, r.pool) }
func billingError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return application.ErrConflict
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	return err
}
func paymentMethodUsageColumn(usage string) (string, bool) {
	switch strings.TrimSpace(usage) {
	case "sales":
		return "sales_enabled", true
	case "expenses":
		return "expenses_enabled", true
	default:
		return "", false
	}
}
func scanPaymentMethod(row pgx.Row) (domain.PaymentMethodView, error) {
	var item domain.PaymentMethodView
	err := row.Scan(
		&item.Code, &item.Name, &item.Description, &item.Active,
		&item.SalesEnabled, &item.ExpensesEnabled, &item.AffectsCash, &item.SortOrder,
	)
	return item, err
}
func readMethods(rows pgx.Rows, err error) ([]domain.PaymentMethodView, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.PaymentMethodView{}
	for rows.Next() {
		v, e := scanPaymentMethod(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, v)
	}
	return items, rows.Err()
}
func (r *Methods) Operational(ctx context.Context, org, usage string) ([]domain.PaymentMethodView, error) {
	column, ok := paymentMethodUsageColumn(usage)
	if !ok {
		return []domain.PaymentMethodView{}, nil
	}
	return readMethods(r.db(ctx).Query(ctx, `
		SELECT code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		FROM payment_methods
		WHERE organization_id=$1 AND active AND `+column+`
		ORDER BY sort_order,name
	`, org))
}
func (r *Methods) Find(ctx context.Context, org, code, usage string) (domain.PaymentMethodView, error) {
	column, ok := paymentMethodUsageColumn(usage)
	if !ok {
		return domain.PaymentMethodView{}, application.ErrNotFound
	}
	v, err := scanPaymentMethod(r.db(ctx).QueryRow(ctx, `
		SELECT code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		FROM payment_methods
		WHERE organization_id=$1 AND code=$2 AND active AND `+column+`
	`, org, code))
	return v, billingError(err)
}
func (r *Methods) List(ctx context.Context, org string, f application.MethodFilter) (application.MethodPage, error) {
	out := application.MethodPage{Page: f.Page, PageSize: f.Size}
	where := `organization_id=$1
		AND ($2='' OR code ILIKE '%'||$2||'%' OR name ILIKE '%'||$2||'%' OR description ILIKE '%'||$2||'%')
		AND ($3='' OR ($3='active' AND active) OR ($3='inactive' AND NOT active))`
	if err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM payment_methods WHERE `+where, org, f.Query, f.Status).Scan(&out.Total); err != nil {
		return out, err
	}
	items, err := readMethods(r.db(ctx).Query(ctx, `
		SELECT code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		FROM payment_methods
		WHERE `+where+`
		ORDER BY active DESC,sort_order,name
		LIMIT $4 OFFSET $5
	`, org, f.Query, f.Status, f.Size, (f.Page-1)*f.Size))
	out.Items = items
	return out, err
}
func (r *Methods) Save(ctx context.Context, org, code string, in domain.PaymentMethodInput, updating bool) (domain.PaymentMethodView, error) {
	var row pgx.Row
	if updating {
		row = r.db(ctx).QueryRow(ctx, `
			UPDATE payment_methods
			SET name=$3,description=$4,sales_enabled=$5,expenses_enabled=$6,affects_cash=$7,sort_order=$8,updated_at=now()
			WHERE organization_id=$1 AND code=$2
			RETURNING code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		`, org, code, in.Name, in.Description, in.SalesEnabled, in.ExpensesEnabled, in.AffectsCash, in.SortOrder)
	} else {
		row = r.db(ctx).QueryRow(ctx, `
			INSERT INTO payment_methods(
				organization_id,code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
			) VALUES($1,$2,$3,$4,true,$5,$6,$7,$8)
			RETURNING code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		`, org, code, in.Name, in.Description, in.SalesEnabled, in.ExpensesEnabled, in.AffectsCash, in.SortOrder)
	}
	v, err := scanPaymentMethod(row)
	return v, billingError(err)
}
func (r *Methods) SetActive(ctx context.Context, org, code string, active bool) (bool, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE payment_methods SET active=$3,updated_at=now()
		WHERE organization_id=$1 AND code=$2
	`, org, code, active)
	return tag.RowsAffected() > 0, err
}
func (r *Methods) Continuity(ctx context.Context, org string) (bool, error) {
	var sales, expenses int
	err := r.db(ctx).QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE active AND sales_enabled),
		       count(*) FILTER (WHERE active AND expenses_enabled)
		FROM payment_methods
		WHERE organization_id=$1
	`, org).Scan(&sales, &expenses)
	return sales > 0 && expenses > 0, err
}
