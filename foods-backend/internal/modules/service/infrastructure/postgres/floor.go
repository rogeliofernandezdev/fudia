package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/service/application"
	"github.com/foods-platform/foods-backend/internal/modules/service/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Floor struct{ pool *pgxpool.Pool }
type scope = requestscope.Scope

func NewFloor(pool *pgxpool.Pool) *Floor { return &Floor{pool} }

var _ application.FloorRepository = (*Floor)(nil)

func (r *Floor) db(ctx context.Context) database.Executor { return database.ExecutorFor(ctx, r.pool) }
func serviceError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return application.ErrConflict
	}
	return err
}
func scanZone(row pgx.Row) (domain.Zone, error) {
	var v domain.Zone
	err := row.Scan(&v.ID, &v.Name, &v.SortOrder, &v.Active)
	return v, serviceError(err)
}
func scanTable(row pgx.Row) (domain.Table, error) {
	var v domain.Table
	err := row.Scan(&v.ID, &v.Name, &v.Seats, &v.Zone, &v.Active, &v.QrToken, &v.QrEnabled)
	return v, serviceError(err)
}
func (r *Floor) Zones(ctx context.Context, s scope, f application.FloorFilter) (application.ZonePage, error) {
	out := application.ZonePage{Items: []domain.Zone{}, Page: f.Page, PageSize: f.Size}
	rows, err := r.db(ctx).Query(ctx, `SELECT id,name,sort_order,active FROM zones WHERE organization_id=$1 AND location_id=$2 AND ($3='' OR ($3='active' AND active) OR ($3='inactive' AND NOT active)) ORDER BY sort_order,name LIMIT $4 OFFSET $5`, s.OrganizationID, s.LocationID, f.Status, f.Size, (f.Page-1)*f.Size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		v, e := scanZone(rows)
		if e != nil {
			return out, e
		}
		out.Items = append(out.Items, v)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	err = r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM zones WHERE organization_id=$1 AND location_id=$2 AND ($3='' OR ($3='active' AND active) OR ($3='inactive' AND NOT active))`, s.OrganizationID, s.LocationID, f.Status).Scan(&out.Total)
	return out, err
}
func (r *Floor) CreateZone(ctx context.Context, s scope, name string, sort int) (domain.Zone, error) {
	return scanZone(r.db(ctx).QueryRow(ctx, `INSERT INTO zones(organization_id,location_id,name,sort_order,active) VALUES($1,$2,$3,$4,true) RETURNING id,name,sort_order,active`, s.OrganizationID, s.LocationID, name, sort))
}
func (r *Floor) LockZone(ctx context.Context, s scope, id string) (domain.Zone, error) {
	var v domain.Zone
	err := r.db(ctx).QueryRow(ctx, `
		SELECT name,active
		FROM zones
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		FOR UPDATE
	`, id, s.OrganizationID, s.LocationID).Scan(&v.Name, &v.Active)
	return v, serviceError(err)
}
func (r *Floor) ZoneInUse(ctx context.Context, s scope, name string) (bool, error) {
	var used bool
	err := r.db(ctx).QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM tables
				WHERE organization_id=$1 AND location_id=$2 AND zone=$3 AND active
			)
		`, s.OrganizationID, s.LocationID, name).Scan(&used)
	return used, err
}
func (r *Floor) UpdateZone(ctx context.Context, s scope, id, name string, sort int, active bool) (domain.Zone, error) {
	return scanZone(r.db(ctx).QueryRow(ctx, `
		UPDATE zones
		SET name=$4,sort_order=$5,active=$6
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		RETURNING id,name,sort_order,active
	`, id, s.OrganizationID, s.LocationID, name, sort, active))
}
func (r *Floor) RenameTableZone(ctx context.Context, s scope, oldName, newName string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
			UPDATE tables
			SET zone=$4,updated_at=now()
			WHERE organization_id=$1 AND location_id=$2 AND zone=$3
		`, s.OrganizationID, s.LocationID, oldName, newName)
	return tag.RowsAffected(), err
}
func (r *Floor) DeactivateZone(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE zones SET active=false
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
	`, id, s.OrganizationID, s.LocationID)
	return tag.RowsAffected(), err
}
func (r *Floor) Tables(ctx context.Context, s scope, f application.FloorFilter) (application.TablePage, error) {
	out := application.TablePage{Items: []domain.Table{}, Page: f.Page, PageSize: f.Size}
	query := "%" + f.Query + "%"
	rows, err := r.db(ctx).Query(ctx, `SELECT id,name,seats,zone,active,qr_token,qr_enabled FROM tables WHERE organization_id=$1 AND location_id=$2 AND name ILIKE $3 AND ($4='' OR active=($4='active')) ORDER BY name LIMIT $5 OFFSET $6`, s.OrganizationID, s.LocationID, query, f.Status, f.Size, (f.Page-1)*f.Size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		v, e := scanTable(rows)
		if e != nil {
			return out, e
		}
		out.Items = append(out.Items, v)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	err = r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM tables WHERE organization_id=$1 AND location_id=$2 AND name ILIKE $3 AND ($4='' OR active=($4='active'))`, s.OrganizationID, s.LocationID, query, f.Status).Scan(&out.Total)
	return out, err
}
func (r *Floor) LockTable(ctx context.Context, s scope, id string) (bool, error) {
	var active bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT active
		FROM tables
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		FOR UPDATE
	`, id, s.OrganizationID, s.LocationID).Scan(&active)
	return active, serviceError(err)
}
func (r *Floor) ActiveZone(ctx context.Context, s scope, name string) error {
	if name == "" {
		return nil
	}
	var id string
	return serviceError(r.db(ctx).QueryRow(ctx, `
		SELECT id
		FROM zones
		WHERE organization_id=$1 AND location_id=$2 AND name=$3 AND active
		FOR SHARE
	`, s.OrganizationID, s.LocationID, name).Scan(&id))
}
func (r *Floor) TableOccupied(ctx context.Context, s scope, id string) (bool, error) {
	var occupied bool
	err := r.db(ctx).QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM orders
				WHERE organization_id=$1 AND location_id=$2 AND table_id=$3
				  AND status<>'cancelado' AND (status<>'entregado' OR (channel='salon' AND completed_at IS NULL))
			)
		`, s.OrganizationID, s.LocationID, id).Scan(&occupied)
	return occupied, err
}
func (r *Floor) CreateTable(ctx context.Context, s scope, name string, seats int, zone string, qr bool) (domain.Table, error) {
	return scanTable(r.db(ctx).QueryRow(ctx, `
		INSERT INTO tables(organization_id,location_id,name,seats,zone,active,qr_token,qr_enabled)
		VALUES($1,$2,$3,$4,$5,true,encode(gen_random_bytes(16),'hex'),$6)
		RETURNING id,name,seats,zone,active,qr_token,qr_enabled
	`, s.OrganizationID, s.LocationID, name, seats, zone, qr))
}
func (r *Floor) UpdateTable(ctx context.Context, s scope, id, name string, seats int, zone string, active, qr bool) (domain.Table, error) {
	return scanTable(r.db(ctx).QueryRow(ctx, `
		UPDATE tables
		SET name=$4,seats=$5,zone=$6,active=$7,qr_enabled=$8,updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		RETURNING id,name,seats,zone,active,qr_token,qr_enabled
	`, id, s.OrganizationID, s.LocationID, name, seats, zone, active, qr))
}
func (r *Floor) DeactivateTable(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE tables SET active=false,updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
	`, id, s.OrganizationID, s.LocationID)
	return tag.RowsAffected(), err
}
func (r *Floor) RegenerateQR(ctx context.Context, s scope, id string) (domain.Table, error) {
	return scanTable(r.db(ctx).QueryRow(ctx, `UPDATE tables SET qr_token=encode(gen_random_bytes(16),'hex'),updated_at=now() WHERE id=$1 AND organization_id=$2 AND location_id=$3 RETURNING id,name,seats,zone,active,qr_token,qr_enabled`, id, s.OrganizationID, s.LocationID))
}
func (r *Floor) TableQR(ctx context.Context, token string) (domain.TableQR, error) {
	var v domain.TableQR
	err := r.db(ctx).QueryRow(ctx, `
		SELECT t.name,t.seats,t.zone,o.trade_name,l.name,
		       COALESCE(om.active,false)
		FROM tables t
		JOIN organizations o ON o.id=t.organization_id AND o.active
		JOIN locations l
		  ON l.id=t.location_id
		 AND l.organization_id=t.organization_id
		 AND l.active
		LEFT JOIN organization_modules om
		  ON om.organization_id=t.organization_id
		 AND om.module_key='whatsapp_bot'
		WHERE t.qr_token=$1 AND t.active=true AND t.qr_enabled=true
		LIMIT 1
	`, token).Scan(&v.Name, &v.Seats, &v.Zone, &v.OrgName, &v.LocName, &v.ConciergeEnabled)
	return v, serviceError(err)
}

const tableBatchSQL = `
	WITH input AS MATERIALIZED (
		SELECT item->>'name' AS name, (item->>'seats')::integer AS seats,
		       item->>'zone' AS zone, (item->>'qrEnabled')::boolean AS qr_enabled,
		       position
		FROM jsonb_array_elements($3::jsonb) WITH ORDINALITY AS entries(item,position)
	), locked_zones AS MATERIALIZED (
		SELECT name FROM zones
		WHERE organization_id=$1 AND location_id=$2 AND active
		  AND name IN (SELECT zone FROM input WHERE zone<>'')
		ORDER BY name FOR SHARE
	), invalid_zones AS MATERIALIZED (
		SELECT zone FROM input
		WHERE zone<>'' AND NOT EXISTS (SELECT 1 FROM locked_zones WHERE name=input.zone)
	), created AS (
		INSERT INTO tables(organization_id,location_id,name,seats,zone,active,qr_token,qr_enabled)
		SELECT $1,$2,name,seats,zone,true,encode(gen_random_bytes(16),'hex'),qr_enabled
		FROM input WHERE NOT EXISTS (SELECT 1 FROM invalid_zones)
		ORDER BY name
		RETURNING id,name,seats,zone,active,qr_token,qr_enabled
	), audited AS (
		INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,metadata)
		SELECT $1::uuid,$2::uuid,$4::uuid,'table.batch_created','table',NULL,
		       jsonb_build_object('tableIds',jsonb_agg(id),'count',count(*))
		FROM created HAVING count(*)>0
		RETURNING id
	)
	SELECT EXISTS(SELECT 1 FROM invalid_zones),
	       jsonb_build_object('items',COALESCE((
		SELECT jsonb_agg(jsonb_build_object(
			'id',c.id,'name',c.name,'seats',c.seats,'zone',c.zone,'active',c.active,
			'qrToken',c.qr_token,'qrEnabled',c.qr_enabled
		) ORDER BY i.position)
		FROM created c JOIN input i ON i.name=c.name
	       ),'[]'::jsonb))
`

type tableRowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func persistTableBatch(ctx context.Context, q tableRowQuerier, s scope, payload []byte) (json.RawMessage, bool, error) {
	var result json.RawMessage
	var invalidZone bool
	err := q.QueryRow(ctx, tableBatchSQL, s.OrganizationID, s.LocationID, string(payload), s.UserID).Scan(&invalidZone, &result)
	return result, invalidZone, err
}
func (r *Floor) Batch(ctx context.Context, s scope, items []domain.TableBatchItem) (domain.TableBatchResult, bool, error) {
	var out domain.TableBatchResult
	payload, err := json.Marshal(items)
	if err != nil {
		return out, false, err
	}
	raw, invalid, err := persistTableBatch(ctx, r.db(ctx), s, payload)
	if err != nil || invalid {
		return out, invalid, serviceError(err)
	}
	err = json.Unmarshal(raw, &out)
	return out, false, err
}
