package postgres

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/customers/application"
	"github.com/foods-platform/foods-backend/internal/modules/customers/domain"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) *Repository { return &Repository{db} }

var _ application.Repository = (*Repository)(nil)

const customerColumns = `id,code,customer_type,display_name,document_type,document_number,phone,email,preferred_channel,preferences,notes,marketing_consent,vip_override,visit_count,total_spent::text,CASE WHEN last_purchase_at IS NULL THEN NULL ELSE to_char(last_purchase_at,'YYYY-MM-DD"T"HH24:MI:SSOF') END,active,CASE WHEN vip_override THEN 'vip' WHEN visit_count>=5 THEN 'frequent' WHEN visit_count>=2 THEN 'recurrent' ELSE 'new' END`

func scanCustomer(row pgx.Row) (domain.Customer, error) {
	var c domain.Customer
	err := row.Scan(&c.ID, &c.Code, &c.CustomerType, &c.DisplayName, &c.DocumentType, &c.DocumentNumber, &c.Phone, &c.Email, &c.PreferredChannel, &c.Preferences, &c.Notes, &c.MarketingConsent, &c.VIPOverride, &c.VisitCount, &c.TotalSpent, &c.LastPurchaseAt, &c.Active, &c.Segment)
	return c, err
}

func (r *Repository) List(ctx context.Context, organizationID string, f application.Filter) (application.Page, error) {
	const where = `organization_id=$1 AND ($2='' OR display_name ILIKE '%'||$2||'%' OR code ILIKE '%'||$2||'%' OR phone ILIKE '%'||$2||'%' OR email ILIKE '%'||$2||'%' OR document_number ILIKE '%'||$2||'%') AND ($3='' OR ($3='active' AND active) OR ($3='inactive' AND NOT active)) AND ($4='' OR $4=CASE WHEN vip_override THEN 'vip' WHEN visit_count>=5 THEN 'frequent' WHEN visit_count>=2 THEN 'recurrent' ELSE 'new' END)`
	out := application.Page{Items: []domain.Customer{}}
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM customers WHERE `+where, organizationID, f.Query, f.Status, f.Segment).Scan(&out.Total); err != nil {
		return out, err
	}
	rows, err := r.db.Query(ctx, `SELECT `+customerColumns+` FROM customers WHERE `+where+` ORDER BY updated_at DESC,display_name LIMIT $5 OFFSET $6`, organizationID, f.Query, f.Status, f.Segment, f.PageSize, (f.Page-1)*f.PageSize)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return out, err
		}
		out.Items = append(out.Items, c)
	}
	return out, rows.Err()
}
func (r *Repository) Get(ctx context.Context, organizationID, id string) (domain.Customer, error) {
	c, err := scanCustomer(r.db.QueryRow(ctx, `SELECT `+customerColumns+` FROM customers WHERE id=$1 AND organization_id=$2`, id, organizationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return c, application.ErrNotFound
	}
	if err != nil {
		return c, err
	}
	c.Addresses = []domain.Address{}
	rows, err := r.db.Query(ctx, `SELECT id,label,address,reference,district,city,country_code,is_default FROM customer_addresses WHERE customer_id=$1 AND organization_id=$2 AND active ORDER BY is_default DESC,created_at`, c.ID, organizationID)
	if err != nil {
		return c, err
	}
	defer rows.Close()
	for rows.Next() {
		var ad domain.Address
		if err = rows.Scan(&ad.ID, &ad.Label, &ad.Address, &ad.Reference, &ad.District, &ad.City, &ad.CountryCode, &ad.Default); err != nil {
			return c, err
		}
		c.Addresses = append(c.Addresses, ad)
	}
	return c, rows.Err()
}
func (r *Repository) Save(ctx context.Context, organizationID, id string, in domain.Input) (domain.Customer, error) {
	var c domain.Customer
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return c, err
	}
	defer tx.Rollback(ctx)
	if id != "" {
		c, err = scanCustomer(tx.QueryRow(ctx, `UPDATE customers SET customer_type=$3,display_name=$4,document_type=$5,document_number=$6,phone=$7,email=$8,preferred_channel=$9,preferences=$10,notes=$11,marketing_consent=$12,vip_override=$13,updated_at=now() WHERE id=$1 AND organization_id=$2 RETURNING `+customerColumns, id, organizationID, in.CustomerType, in.DisplayName, in.DocumentType, in.DocumentNumber, in.Phone, in.Email, in.PreferredChannel, in.Preferences, in.Notes, in.MarketingConsent, in.VIPOverride))
	} else {
		c, err = scanCustomer(tx.QueryRow(ctx, `INSERT INTO customers(organization_id,customer_type,display_name,document_type,document_number,phone,email,preferred_channel,preferences,notes,marketing_consent,vip_override) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING `+customerColumns, organizationID, in.CustomerType, in.DisplayName, in.DocumentType, in.DocumentNumber, in.Phone, in.Email, in.PreferredChannel, in.Preferences, in.Notes, in.MarketingConsent, in.VIPOverride))
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return c, application.ErrNotFound
	}
	if err != nil {
		return c, application.ErrConflict
	}
	if id != "" {
		if _, err = tx.Exec(ctx, `DELETE FROM customer_addresses WHERE customer_id=$1 AND organization_id=$2`, c.ID, organizationID); err != nil {
			return c, err
		}
	}
	for _, ad := range in.Addresses {
		_, err = tx.Exec(ctx, `INSERT INTO customer_addresses(organization_id,customer_id,label,address,reference,district,city,country_code,is_default) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, organizationID, c.ID, ad.Label, ad.Address, ad.Reference, ad.District, ad.City, ad.CountryCode, ad.Default)
		if err != nil {
			return c, application.ErrAddress
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return c, err
	}
	return c, nil
}
func (r *Repository) SetActive(ctx context.Context, organizationID, id string, active bool) error {
	tag, err := r.db.Exec(ctx, `UPDATE customers SET active=$3,updated_at=now() WHERE id=$1 AND organization_id=$2`, id, organizationID, active)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return nil
}

// Audit retains the existing best-effort policy; transactional audit is separate follow-up work.
func (r *Repository) Audit(ctx context.Context, s requestscope.Scope, action, id string) {
	_, _ = r.db.Exec(ctx, `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id) VALUES($1,$2,$3,$4,'customer',$5)`, s.OrganizationID, s.LocationID, s.UserID, action, id)
}
