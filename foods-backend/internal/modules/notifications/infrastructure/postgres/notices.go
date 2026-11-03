package postgres

import (
	"context"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/notifications/application"
	"github.com/foods-platform/foods-backend/internal/modules/notifications/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5/pgxpool"
)

type scope = requestscope.Scope

type Notices struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Notices { return &Notices{pool} }
func (r *Notices) db(ctx context.Context) database.Executor {
	return database.ExecutorFor(ctx, r.pool)
}

var _ application.Repository = (*Notices)(nil)

// visible filtra los avisos del local de las últimas 24 horas dirigidos a las audiencias del usuario.
const visible = `n.organization_id=$1 AND n.location_id=$2 AND n.audience=ANY($3::text[]) AND n.created_at>now()-interval '24 hours'`

func (r *Notices) Insert(ctx context.Context, s scope, in domain.Input) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO staff_notices(organization_id,location_id,kind,audience,title,detail,order_id,table_id,dedupe_key)
		VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,'')::uuid,NULLIF($8,'')::uuid,NULLIF($9,''))
		ON CONFLICT (organization_id,dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING`,
		s.OrganizationID, s.LocationID, in.Kind, in.Audience, in.Title, in.Detail, in.OrderID, in.TableID, in.DedupeKey)
	return err
}

func (r *Notices) Feed(ctx context.Context, s scope, audiences []string, limit int) (domain.Feed, error) {
	out := domain.Feed{Items: []domain.Notice{}}
	if err := r.db(ctx).QueryRow(ctx, `
		SELECT count(*) FROM staff_notices n
		WHERE `+visible+` AND NOT EXISTS (SELECT 1 FROM staff_notice_reads nr WHERE nr.notice_id=n.id AND nr.user_id=$4)`,
		s.OrganizationID, s.LocationID, audiences, s.UserID).Scan(&out.Unread); err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT n.id::text,n.kind,n.title,n.detail,COALESCE(n.order_id::text,''),COALESCE(n.table_id::text,''),n.created_at,
		       EXISTS (SELECT 1 FROM staff_notice_reads nr WHERE nr.notice_id=n.id AND nr.user_id=$4)
		FROM staff_notices n
		WHERE `+visible+`
		ORDER BY n.created_at DESC,n.id DESC
		LIMIT $5`, s.OrganizationID, s.LocationID, audiences, s.UserID, limit)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var item domain.Notice
		var created time.Time
		if err := rows.Scan(&item.ID, &item.Kind, &item.Title, &item.Detail, &item.OrderID, &item.TableID, &created, &item.Read); err != nil {
			return out, err
		}
		item.CreatedAt = created.UTC().Format(time.RFC3339)
		out.Items = append(out.Items, item)
	}
	return out, rows.Err()
}

func (r *Notices) MarkRead(ctx context.Context, s scope, id string, audiences []string) error {
	var exists bool
	if err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM staff_notices n WHERE n.id=$4 AND `+visible+`)`,
		s.OrganizationID, s.LocationID, audiences, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return application.ErrNotFound
	}
	_, err := r.db(ctx).Exec(ctx, `INSERT INTO staff_notice_reads(notice_id,user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, id, s.UserID)
	return err
}

func (r *Notices) MarkAllRead(ctx context.Context, s scope, audiences []string) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO staff_notice_reads(notice_id,user_id)
		SELECT n.id,$4 FROM staff_notices n WHERE `+visible+`
		ON CONFLICT DO NOTHING`, s.OrganizationID, s.LocationID, audiences, s.UserID)
	return err
}
