package postgres

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/customers/application"
	customercontracts "github.com/foods-platform/foods-backend/internal/modules/customers/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/customers/domain"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) *Repository { return &Repository{db} }

var _ application.Repository = (*Repository)(nil)

// consentColumns deriva el estado vigente del último evento de consentimiento;
// sin eventos, un consentimiento cargado antes de esta evidencia figura como granted.
const consentColumns = `COALESCE((SELECT e.decision FROM customer_marketing_consents e WHERE e.organization_id=customers.organization_id AND e.customer_id=customers.id ORDER BY e.created_at DESC,e.id DESC LIMIT 1),CASE WHEN marketing_consent THEN 'granted' ELSE 'pending' END),(SELECT to_char(e.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF') FROM customer_marketing_consents e WHERE e.organization_id=customers.organization_id AND e.customer_id=customers.id ORDER BY e.created_at DESC,e.id DESC LIMIT 1)`

const customerColumns = `id,code,customer_type,display_name,document_type,document_number,phone,email,preferred_channel,preferences,notes,marketing_consent,` + consentColumns + `,vip_override,visit_count,total_spent::text,CASE WHEN last_purchase_at IS NULL THEN NULL ELSE to_char(last_purchase_at,'YYYY-MM-DD"T"HH24:MI:SSOF') END,active,CASE WHEN vip_override THEN 'vip' WHEN visit_count>=5 THEN 'frequent' WHEN visit_count>=2 THEN 'recurrent' ELSE 'new' END`

func scanCustomer(row pgx.Row) (domain.Customer, error) {
	var c domain.Customer
	err := row.Scan(&c.ID, &c.Code, &c.CustomerType, &c.DisplayName, &c.DocumentType, &c.DocumentNumber, &c.Phone, &c.Email, &c.PreferredChannel, &c.Preferences, &c.Notes, &c.MarketingConsent, &c.MarketingConsentStatus, &c.MarketingConsentUpdatedAt, &c.VIPOverride, &c.VisitCount, &c.TotalSpent, &c.LastPurchaseAt, &c.Active, &c.Segment)
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
	rows, err := r.db.Query(ctx, `SELECT id,label,address,reference,district,city,country_code,is_default,latitude::float8,longitude::float8 FROM customer_addresses WHERE customer_id=$1 AND organization_id=$2 AND active ORDER BY is_default DESC,created_at`, c.ID, organizationID)
	if err != nil {
		return c, err
	}
	defer rows.Close()
	for rows.Next() {
		var ad domain.Address
		if err = rows.Scan(&ad.ID, &ad.Label, &ad.Address, &ad.Reference, &ad.District, &ad.City, &ad.CountryCode, &ad.Default, &ad.Latitude, &ad.Longitude); err != nil {
			return c, err
		}
		c.Addresses = append(c.Addresses, ad)
	}
	return c, rows.Err()
}
func (r *Repository) Save(ctx context.Context, s requestscope.Scope, id string, in domain.Input) (domain.Customer, error) {
	var c domain.Customer
	organizationID := s.OrganizationID
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return c, err
	}
	defer tx.Rollback(ctx)
	previousConsent := false
	if id != "" {
		// Bloquea al cliente para registrar el cambio de consentimiento una sola vez.
		err = tx.QueryRow(ctx, `SELECT marketing_consent FROM customers WHERE id=$1 AND organization_id=$2 FOR UPDATE`, id, organizationID).Scan(&previousConsent)
		if errors.Is(err, pgx.ErrNoRows) {
			return c, application.ErrNotFound
		}
		if err != nil {
			return c, err
		}
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
		_, err = tx.Exec(ctx, `INSERT INTO customer_addresses(organization_id,customer_id,label,address,reference,district,city,country_code,is_default,latitude,longitude) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, organizationID, c.ID, ad.Label, ad.Address, ad.Reference, ad.District, ad.City, ad.CountryCode, ad.Default, ad.Latitude, ad.Longitude)
		if err != nil {
			return c, application.ErrAddress
		}
	}
	if in.MarketingConsent != previousConsent {
		decision := customercontracts.ConsentWithdrawn
		if in.MarketingConsent {
			decision = customercontracts.ConsentGranted
		}
		if _, err = tx.Exec(ctx, `INSERT INTO customer_marketing_consents(organization_id,location_id,customer_id,decision,source,evidence,created_by) VALUES($1,$2,$3,$4,'admin','Actualizado en Clientes',NULLIF($5,'')::uuid)`, organizationID, s.LocationID, c.ID, decision, s.UserID); err != nil {
			return c, err
		}
		if c, err = scanCustomer(tx.QueryRow(ctx, `SELECT `+customerColumns+` FROM customers WHERE id=$1 AND organization_id=$2`, c.ID, organizationID)); err != nil {
			return c, err
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

// whatsAppCustomerSQL encuentra al cliente activo con el mismo número, también
// guardado sin código de país (mismo criterio del prerregistro de Concierge).
const whatsAppCustomerSQL = `
	SELECT c.id::text,c.display_name,c.marketing_consent
	FROM (
	  SELECT id,created_at,display_name,marketing_consent,regexp_replace(phone,'\D','','g') AS digits
	  FROM customers
	  WHERE organization_id=$1 AND active
	) c
	WHERE c.digits=$2 OR (length(c.digits)>=9 AND right($2,length(c.digits))=c.digits)
	ORDER BY c.created_at,c.id
	LIMIT 1`

const latestConsentSQL = `SELECT decision FROM customer_marketing_consents WHERE organization_id=$1 AND customer_id=$2 ORDER BY created_at DESC,id DESC LIMIT 1`

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func whatsAppProfile(ctx context.Context, q rowQuerier, organizationID, digits string) (string, customercontracts.WhatsAppProfile, error) {
	var id string
	var consent bool
	out := customercontracts.WhatsAppProfile{MarketingConsent: customercontracts.ConsentPending}
	err := q.QueryRow(ctx, whatsAppCustomerSQL, organizationID, digits).Scan(&id, &out.CustomerName, &consent)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", out, nil
	}
	if err != nil {
		return "", out, err
	}
	out.Known = true
	err = q.QueryRow(ctx, latestConsentSQL, organizationID, id).Scan(&out.MarketingConsent)
	if errors.Is(err, pgx.ErrNoRows) {
		out.MarketingConsent = customercontracts.ConsentPending
		if consent {
			out.MarketingConsent = customercontracts.ConsentGranted
		}
		err = nil
	}
	return id, out, err
}

// WhatsAppProfile lee nombre y consentimiento del cliente de ese número.
func (r *Repository) WhatsAppProfile(ctx context.Context, organizationID, digits string) (customercontracts.WhatsAppProfile, error) {
	_, out, err := whatsAppProfile(ctx, r.db, organizationID, digits)
	return out, err
}

// RecordWhatsAppConsent guarda la pregunta o respuesta con su evidencia y, si
// es una respuesta, actualiza el consentimiento vigente en la misma
// transacción. requestId repetido no duplica la evidencia y la pregunta se
// registra una sola vez por cliente y empresa.
func (r *Repository) RecordWhatsAppConsent(ctx context.Context, s requestscope.Scope, digits string, in customercontracts.ConsentEvidence) (customercontracts.WhatsAppProfile, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return customercontracts.WhatsAppProfile{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('customer_consent|'||$1::text||'|'||$2::text,0))`, s.OrganizationID, digits); err != nil {
		return customercontracts.WhatsAppProfile{}, err
	}
	id, profile, err := whatsAppProfile(ctx, tx, s.OrganizationID, digits)
	if err != nil {
		return profile, err
	}
	if id == "" {
		return profile, application.ErrNotFound
	}
	if in.Decision == customercontracts.ConsentAsked && profile.MarketingConsent != customercontracts.ConsentPending {
		return profile, tx.Commit(ctx)
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO customer_marketing_consents(organization_id,location_id,customer_id,decision,source,evidence,conversation_id,request_id)
		VALUES($1,$2,$3,$4,'whatsapp',$5,$6,$7)
		ON CONFLICT (organization_id,request_id) WHERE request_id IS NOT NULL DO NOTHING
	`, s.OrganizationID, s.LocationID, id, in.Decision, in.Evidence, in.ConversationID, in.RequestID)
	if err != nil {
		return profile, err
	}
	if tag.RowsAffected() > 0 && in.Decision != customercontracts.ConsentAsked {
		if _, err = tx.Exec(ctx, `UPDATE customers SET marketing_consent=$3,updated_at=now() WHERE id=$1 AND organization_id=$2`, id, s.OrganizationID, in.Decision == customercontracts.ConsentGranted); err != nil {
			return profile, err
		}
	}
	if _, profile, err = whatsAppProfile(ctx, tx, s.OrganizationID, digits); err != nil {
		return profile, err
	}
	return profile, tx.Commit(ctx)
}
