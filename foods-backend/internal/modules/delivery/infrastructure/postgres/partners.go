// Package postgres persiste centrales de delivery, configuración por local,
// despachos, ofertas y la bandeja de mensajes hacia las centrales.
package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/delivery/application"
	"github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type scope = requestscope.Scope

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool} }

func (r *Repository) db(ctx context.Context) database.Executor {
	return database.ExecutorFor(ctx, r.pool)
}

var (
	_ application.PartnerRepository  = (*Repository)(nil)
	_ application.DispatchRepository = (*Repository)(nil)
)

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func optionalStamp(t *time.Time) string {
	if t == nil {
		return ""
	}
	return stamp(*t)
}

func duplicatePhone(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

const partnerColumns = `id::text,name,whatsapp_phone,active,priority,panel_token,created_at,updated_at`

func scanPartner(row pgx.Row) (domain.Partner, error) {
	var p domain.Partner
	var created, updated time.Time
	err := row.Scan(&p.ID, &p.Name, &p.WhatsAppPhone, &p.Active, &p.Priority, &p.PanelToken, &created, &updated)
	p.CreatedAt, p.UpdatedAt = stamp(created), stamp(updated)
	return p, err
}

func (r *Repository) ListPartners(ctx context.Context) ([]domain.Partner, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT p.id::text,p.name,p.whatsapp_phone,p.active,p.priority,p.panel_token,p.created_at,p.updated_at,
      ARRAY(SELECT c.area_code FROM delivery_partner_coverage c WHERE c.partner_id=p.id ORDER BY c.area_code),
      ARRAY(SELECT concat_ws(' · ',g.name,pr.name,dp.name)
        FROM delivery_partner_coverage c JOIN geographic_areas g ON g.code=c.area_code
        LEFT JOIN geographic_areas pr ON pr.code=g.parent_code
        LEFT JOIN geographic_areas dp ON dp.code=pr.parent_code
        WHERE c.partner_id=p.id ORDER BY c.area_code)
      FROM delivery_partners p ORDER BY p.priority,p.name,p.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Partner{}
	for rows.Next() {
		var p domain.Partner
		var created, updated time.Time
		err := rows.Scan(&p.ID, &p.Name, &p.WhatsAppPhone, &p.Active, &p.Priority, &p.PanelToken, &created, &updated, &p.CoverageCodes, &p.CoverageLabels)
		p.CreatedAt, p.UpdatedAt = stamp(created), stamp(updated)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) SetCoverage(ctx context.Context, id string, codes []string) error {
	var count int
	if err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM geographic_areas WHERE code=ANY($1::text[]) AND length(code)=6`, codes).Scan(&count); err != nil {
		return err
	}
	if count != len(codes) {
		return application.ErrInvalidCoverage
	}
	if _, err := r.db(ctx).Exec(ctx, `DELETE FROM delivery_partner_coverage WHERE partner_id=$1`, id); err != nil {
		return err
	}
	_, err := r.db(ctx).Exec(ctx, `INSERT INTO delivery_partner_coverage(partner_id,area_code) SELECT $1,unnest($2::text[])`, id, codes)
	return err
}

func (r *Repository) CreatePartner(ctx context.Context, in domain.PartnerInput) (domain.Partner, error) {
	active := in.Active == nil || *in.Active
	p, err := scanPartner(r.db(ctx).QueryRow(ctx, `
		INSERT INTO delivery_partners(name,whatsapp_phone,active,priority)
		VALUES ($1,$2,$3,(SELECT COALESCE(max(priority),0)+1 FROM delivery_partners))
		RETURNING `+partnerColumns, in.Name, in.WhatsAppPhone, active))
	if duplicatePhone(err) {
		return p, application.ErrDuplicatePhone
	}
	return p, err
}

func (r *Repository) UpdatePartner(ctx context.Context, id string, in domain.PartnerInput) (domain.Partner, error) {
	p, err := scanPartner(r.db(ctx).QueryRow(ctx, `
		UPDATE delivery_partners SET name=$2,whatsapp_phone=$3,active=COALESCE($4,active),updated_at=now()
		WHERE id=$1 RETURNING `+partnerColumns, id, in.Name, in.WhatsAppPhone, in.Active))
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return p, application.ErrNotFound
	case duplicatePhone(err):
		return p, application.ErrDuplicatePhone
	}
	return p, err
}

func (r *Repository) locationExists(ctx context.Context, s scope, locationID string) error {
	var exists bool
	if err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM locations WHERE id=$1 AND organization_id=$2)`, locationID, s.OrganizationID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return application.ErrNotFound
	}
	return nil
}

// ReorderPartners reasigna el orden global 1..n en el orden recibido. Bloquea
// las centrales para que un alta simultánea no quede fuera del orden.
func (r *Repository) ReorderPartners(ctx context.Context, ids []string) error {
	var registered []string
	rows, err := r.db(ctx).Query(ctx, `SELECT id::text FROM delivery_partners ORDER BY id FOR UPDATE`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		registered = append(registered, id)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	if len(registered) != len(ids) {
		return application.ErrOrderMismatch
	}
	known := map[string]bool{}
	for _, id := range registered {
		known[id] = true
	}
	for _, id := range ids {
		if !known[id] {
			return application.ErrOrderMismatch
		}
	}
	_, err = r.db(ctx).Exec(ctx, `
		UPDATE delivery_partners p SET priority=o.n,updated_at=now()
		FROM unnest($1::uuid[]) WITH ORDINALITY AS o(id,n)
		WHERE p.id=o.id AND p.priority<>o.n`, ids)
	return err
}

func (r *Repository) OrganizationDelivery(ctx context.Context, organizationID string) (domain.OrganizationDelivery, error) {
	out := domain.OrganizationDelivery{OrganizationID: organizationID}
	var updated *time.Time
	err := r.db(ctx).QueryRow(ctx, `
		SELECT COALESCE(s.enabled,false),s.updated_at
		FROM organizations o
		LEFT JOIN organization_delivery_settings s ON s.organization_id=o.id
		WHERE o.id=$1`, organizationID).Scan(&out.Enabled, &updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, application.ErrNotFound
	}
	out.UpdatedAt = optionalStamp(updated)
	return out, err
}

func (r *Repository) SetOrganizationDelivery(ctx context.Context, organizationID string, enabled bool) (domain.OrganizationDelivery, error) {
	out := domain.OrganizationDelivery{OrganizationID: organizationID}
	var updated time.Time
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO organization_delivery_settings(organization_id,enabled)
		SELECT o.id,$2 FROM organizations o WHERE o.id=$1
		ON CONFLICT (organization_id) DO UPDATE SET enabled=EXCLUDED.enabled,updated_at=now()
		RETURNING enabled,updated_at`, organizationID, enabled).Scan(&out.Enabled, &updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, application.ErrNotFound
	}
	out.UpdatedAt = stamp(updated)
	return out, err
}

func (r *Repository) DeliveryLink(ctx context.Context, s scope, locationID string) (domain.DeliveryLink, error) {
	out := domain.DeliveryLink{LocationID: locationID}
	if err := r.locationExists(ctx, s, locationID); err != nil {
		return out, err
	}
	var updated time.Time
	err := r.db(ctx).QueryRow(ctx, `
		SELECT qr_token,active,updated_at FROM location_delivery_links
		WHERE location_id=$1 AND organization_id=$2`, locationID, s.OrganizationID).Scan(&out.Token, &out.Active, &updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	out.UpdatedAt = stamp(updated)
	return out, err
}

func (r *Repository) RotateDeliveryLink(ctx context.Context, s scope, locationID string) (domain.DeliveryLink, error) {
	out := domain.DeliveryLink{LocationID: locationID}
	if err := r.locationExists(ctx, s, locationID); err != nil {
		return out, err
	}
	var updated time.Time
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO location_delivery_links(organization_id,location_id) VALUES ($1,$2)
		ON CONFLICT (location_id) DO UPDATE
		SET qr_token=replace(gen_random_uuid()::text,'-',''),active=true,updated_at=now()
		RETURNING qr_token,active,updated_at`, s.OrganizationID, locationID).Scan(&out.Token, &out.Active, &updated)
	out.UpdatedAt = stamp(updated)
	return out, err
}

// RotatePanelToken reemplaza el enlace privado del panel; el anterior deja de abrir.
func (r *Repository) RotatePanelToken(ctx context.Context, id string) (domain.PanelLink, error) {
	out := domain.PanelLink{PartnerID: id}
	err := r.db(ctx).QueryRow(ctx, `
		UPDATE delivery_partners SET panel_token=replace(gen_random_uuid()::text,'-',''),updated_at=now()
		WHERE id=$1 RETURNING panel_token`, id).Scan(&out.PanelToken)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, application.ErrNotFound
	}
	return out, err
}
