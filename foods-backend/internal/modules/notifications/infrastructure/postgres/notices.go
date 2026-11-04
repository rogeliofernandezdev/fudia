package postgres

import (
	"context"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/notifications/application"
	"github.com/foods-platform/foods-backend/internal/modules/notifications/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
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

// pending excluye de la campana los avisos cuyo pedido ya se atendió:
//   - comanda lista: el pedido ya va en camino o se entregó;
//   - cuenta pedida: el pedido ya se completó (pagado);
//   - pedido nuevo (QR de mesa o mostrador, delivery, recojo): la cocina ya marcó
//     lista esa comanda (hay un aviso «listo» posterior del mismo pedido) o el pedido
//     se completó. No se usa el estado: los pedidos por QR llegan ya confirmados y
//     cada ronda nueva avisa sobre el mismo pedido;
//   - cuenta anterior sin cerrar: se cobró (completada) o el personal agregó productos;
//   - cualquier pedido cancelado.
//
// Los llamados al mozo y los avisos sin pedido siguen visibles sus 24 horas. Marcar
// como leído no usa este filtro, para que un aviso atendido mientras se abría no falle.
const pending = visible + ` AND NOT EXISTS (
	SELECT 1 FROM orders o
	WHERE o.id=n.order_id AND o.organization_id=n.organization_id
	  AND (o.status='cancelado' OR CASE
	    WHEN n.kind='order_ready' THEN o.status IN ('en_camino','entregado')
	    WHEN n.kind='bill_requested' THEN o.completed_at IS NOT NULL
	    WHEN n.kind='stale_account' THEN o.completed_at IS NOT NULL OR EXISTS (
	      SELECT 1 FROM order_kitchen_rounds k
	      WHERE k.organization_id=n.organization_id AND k.location_id=n.location_id AND k.order_id=n.order_id AND k.created_at>=n.created_at)
	    WHEN n.kind IN ('table_order','counter_order','delivery_order','pickup_order') THEN o.completed_at IS NOT NULL OR EXISTS (
	      SELECT 1 FROM staff_notices ready
	      WHERE ready.organization_id=n.organization_id AND ready.order_id=n.order_id
	        AND ready.kind='order_ready' AND ready.created_at>=n.created_at AND ready.id<>n.id)
	    ELSE false END))`

func (r *Notices) Insert(ctx context.Context, s scope, in domain.Input) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO staff_notices(organization_id,location_id,kind,audience,title,detail,order_id,table_id,dedupe_key)
		VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,'')::uuid,NULLIF($8,'')::uuid,NULLIF($9,''))
		ON CONFLICT (organization_id,dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING`,
		s.OrganizationID, s.LocationID, in.Kind, in.Audience, in.Title, in.Detail, in.OrderID, in.TableID, in.DedupeKey)
	return err
}

// Feed lee el contador de no leídos y los avisos en una sola ida y vuelta: la
// campana se refresca cada 15 s en cada sesión abierta.
func (r *Notices) Feed(ctx context.Context, s scope, audiences []string, limit int) (domain.Feed, error) {
	out := domain.Feed{Items: []domain.Notice{}}
	batch := &pgx.Batch{}
	batch.Queue(`
		SELECT count(*) FROM staff_notices n
		WHERE `+pending+` AND NOT EXISTS (SELECT 1 FROM staff_notice_reads nr WHERE nr.notice_id=n.id AND nr.user_id=$4)`,
		s.OrganizationID, s.LocationID, audiences, s.UserID)
	batch.Queue(`
		SELECT n.id::text,n.kind,n.title,n.detail,COALESCE(n.order_id::text,''),COALESCE(n.table_id::text,''),n.created_at,
		       EXISTS (SELECT 1 FROM staff_notice_reads nr WHERE nr.notice_id=n.id AND nr.user_id=$4)
		FROM staff_notices n
		WHERE `+pending+`
		ORDER BY n.created_at DESC,n.id DESC
		LIMIT $5`, s.OrganizationID, s.LocationID, audiences, s.UserID, limit)
	results := r.db(ctx).SendBatch(ctx, batch)
	err := readFeed(results, &out)
	if closeErr := results.Close(); err == nil {
		err = closeErr
	}
	return out, err
}

func readFeed(results pgx.BatchResults, out *domain.Feed) error {
	if err := results.QueryRow().Scan(&out.Unread); err != nil {
		return err
	}
	rows, err := results.Query()
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item domain.Notice
		var created time.Time
		if err := rows.Scan(&item.ID, &item.Kind, &item.Title, &item.Detail, &item.OrderID, &item.TableID, &created, &item.Read); err != nil {
			return err
		}
		item.CreatedAt = created.UTC().Format(time.RFC3339)
		out.Items = append(out.Items, item)
	}
	return rows.Err()
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
