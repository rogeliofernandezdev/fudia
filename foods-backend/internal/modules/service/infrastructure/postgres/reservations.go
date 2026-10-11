package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/service/application"
	"github.com/foods-platform/foods-backend/internal/modules/service/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Reservations struct{ pool *pgxpool.Pool }

func NewReservations(pool *pgxpool.Pool) *Reservations { return &Reservations{pool} }
func (r *Reservations) db(ctx context.Context) database.Executor {
	return database.ExecutorFor(ctx, r.pool)
}

var _ application.ReservationRepository = (*Reservations)(nil)

func reservationColumns() string {
	return `r.id::text,r.customer_id::text,r.customer_name,r.customer_phone,
	to_char(r.starts_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),r.guests,r.duration_minutes,r.table_id::text,
	COALESCE((SELECT name FROM tables rt WHERE rt.id=r.table_id AND rt.organization_id=r.organization_id AND rt.location_id=r.location_id),''),
	r.status,r.notes,to_char(r.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF')`
}
func scanReservation(row pgx.Row) (domain.ReservationView, error) {
	var item domain.ReservationView
	err := row.Scan(&item.ID, &item.CustomerID, &item.CustomerName, &item.CustomerPhone, &item.StartsAt, &item.Guests, &item.DurationMinutes, &item.TableID, &item.TableName, &item.Status, &item.Notes, &item.CreatedAt)
	return item, err
}
func (r *Reservations) List(ctx context.Context, s scope, f application.FloorFilter) (application.ReservationPage, error) {
	out := application.ReservationPage{Items: []domain.ReservationView{}, Page: f.Page, PageSize: f.Size}
	where := `r.organization_id=$1 AND r.location_id=$2
		AND ($3='' OR r.customer_name ILIKE '%'||$3||'%' OR r.customer_phone ILIKE '%'||$3||'%' OR COALESCE(t.name,'') ILIKE '%'||$3||'%')
		AND ($4='' OR r.status=$4)`
	if err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM reservations r LEFT JOIN tables t ON t.id=r.table_id AND t.organization_id=r.organization_id AND t.location_id=r.location_id WHERE `+where, s.OrganizationID, s.LocationID, f.Query, f.Status).Scan(&out.Total); err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT `+reservationColumns()+`
		FROM reservations r
		LEFT JOIN tables t ON t.id=r.table_id AND t.organization_id=r.organization_id AND t.location_id=r.location_id
		WHERE `+where+`
		ORDER BY CASE WHEN r.starts_at>=now() THEN 0 ELSE 1 END,r.starts_at ASC,r.created_at DESC
		LIMIT $5 OFFSET $6`, s.OrganizationID, s.LocationID, f.Query, f.Status, f.Size, (f.Page-1)*f.Size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		v, e := scanReservation(rows)
		if e != nil {
			return out, e
		}
		out.Items = append(out.Items, v)
	}
	return out, rows.Err()
}
func (r *Reservations) CustomerAvailable(ctx context.Context, s scope, id string) (bool, error) {
	var ok bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM customers WHERE id=$1 AND organization_id=$2 AND active)`, id, s.OrganizationID).Scan(&ok)
	return ok, err
}
func (r *Reservations) TableSeats(ctx context.Context, s scope, id string) (int, error) {
	var seats int
	err := r.db(ctx).QueryRow(ctx, `SELECT seats FROM tables WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND active`, id, s.OrganizationID, s.LocationID).Scan(&seats)
	return seats, serviceError(err)
}
func (r *Reservations) Overlaps(ctx context.Context, s scope, in domain.ReservationInput, id string) (bool, error) {
	var ok bool
	err := r.db(ctx).QueryRow(ctx, `
			SELECT EXISTS(
			 SELECT 1 FROM reservations
			 WHERE organization_id=$1 AND location_id=$2 AND table_id=$3
			   AND status IN ('pending','confirmed','seated')
			   AND starts_at < $4::timestamptz + make_interval(mins=>$5)
			   AND starts_at + make_interval(mins=>duration_minutes) > $4::timestamptz
			   AND ($6='' OR id::text<>$6)
			)`, s.OrganizationID, s.LocationID, *in.TableID, in.StartsAt, in.DurationMinutes, id).Scan(&ok)
	return ok, err
}
func (r *Reservations) Create(ctx context.Context, s scope, in domain.ReservationInput, startsAt time.Time) (domain.ReservationView, error) {
	v, err := scanReservation(r.db(ctx).QueryRow(ctx, `
		INSERT INTO reservations AS r(organization_id,location_id,customer_id,customer_name,customer_phone,starts_at,guests,duration_minutes,table_id,notes,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING `+reservationColumns()+`
	`, s.OrganizationID, s.LocationID, in.CustomerID, in.CustomerName, in.CustomerPhone, startsAt, in.Guests, in.DurationMinutes, in.TableID, in.Notes, s.UserID))
	return v, serviceError(err)
}
func (r *Reservations) Update(ctx context.Context, s scope, id string, in domain.ReservationInput, startsAt time.Time) (domain.ReservationView, error) {
	v, err := scanReservation(r.db(ctx).QueryRow(ctx, `
		UPDATE reservations r SET customer_id=$4,customer_name=$5,customer_phone=$6,starts_at=$7,guests=$8,duration_minutes=$9,table_id=$10,notes=$11,updated_at=now()
		WHERE r.id=$1 AND r.organization_id=$2 AND r.location_id=$3
		RETURNING `+reservationColumns()+`
	`, id, s.OrganizationID, s.LocationID, in.CustomerID, in.CustomerName, in.CustomerPhone, startsAt, in.Guests, in.DurationMinutes, in.TableID, in.Notes))
	return v, serviceError(err)
}
func (r *Reservations) Status(ctx context.Context, s scope, id string, lock bool) (string, error) {
	query := `SELECT status FROM reservations WHERE id=$1 AND organization_id=$2 AND location_id=$3`
	if lock {
		query += " FOR UPDATE"
	}
	var status string
	err := r.db(ctx).QueryRow(ctx, query, id, s.OrganizationID, s.LocationID).Scan(&status)
	return status, serviceError(err)
}
func (r *Reservations) SetStatus(ctx context.Context, s scope, id, status string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE reservations SET status=$4,updated_at=now() WHERE id=$1 AND organization_id=$2 AND location_id=$3`, id, s.OrganizationID, s.LocationID, status)
	return tag.RowsAffected(), err
}
