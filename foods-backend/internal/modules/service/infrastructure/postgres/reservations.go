package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
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

// reservationStatusSQL proyecta «Asistió» (attended): una reserva en mesa
// (seated) cuyo tiempo reservado ya terminó. No se guarda: la base conserva
// seated para que Cobros y la garantía sigan igual.
const reservationStatusSQL = `CASE WHEN r.status='seated' AND r.starts_at+make_interval(mins=>r.duration_minutes)<=now() THEN 'attended' ELSE r.status END`

func reservationColumns() string {
	return `r.id::text,r.customer_id::text,r.customer_name,r.customer_phone,
	to_char(r.starts_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),r.guests,r.duration_minutes,r.tolerance_minutes,r.table_id::text,
	COALESCE((SELECT name FROM tables rt WHERE rt.id=r.table_id AND rt.organization_id=r.organization_id AND rt.location_id=r.location_id),''),
	r.deposit_amount::text,`+reservationStatusSQL+`,r.expired_at IS NOT NULL,r.notes,to_char(r.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),
	` + reservationDepositSQL
}

// reservationDepositSQL resume el cobro de la garantía (payments con
// reservation_id): estado, medio y, si quedó retenida, su boleta.
const reservationDepositSQL = `COALESCE((
	  SELECT json_build_object(
	    'status',CASE WHEN p.deposit_outcome IS NOT NULL THEN p.deposit_outcome
	                  WHEN COALESCE(rf.amount,0)>=p.amount-0.00001 THEN 'refunded' ELSE 'paid' END,
	    'method',pm.name,
	    'document',COALESCE((SELECT d.series||'-'||d.number FROM fiscal_documents d
	      WHERE p.deposit_outcome='retained' AND d.order_id=p.order_id AND d.organization_id=p.organization_id LIMIT 1),''))
	  FROM payments p
	  JOIN payment_methods pm ON pm.organization_id=p.organization_id AND pm.code=p.method
	  LEFT JOIN LATERAL (SELECT sum(pr.amount) AS amount FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id) rf ON true
	  WHERE p.reservation_id=r.id AND p.organization_id=r.organization_id
	),'{}'::json)::text`

func scanReservation(row pgx.Row) (domain.ReservationView, error) {
	var item domain.ReservationView
	var deposit string
	err := row.Scan(&item.ID, &item.CustomerID, &item.CustomerName, &item.CustomerPhone, &item.StartsAt, &item.Guests, &item.DurationMinutes, &item.ToleranceMinutes, &item.TableID, &item.TableName, &item.DepositAmount, &item.Status, &item.Expired, &item.Notes, &item.CreatedAt, &deposit)
	if err != nil {
		return item, err
	}
	var charged struct{ Status, Method, Document string }
	if err := json.Unmarshal([]byte(deposit), &charged); err != nil {
		return item, err
	}
	item.DepositStatus, item.DepositMethodName, item.DepositDocument = charged.Status, charged.Method, charged.Document
	if item.DepositStatus == "" {
		item.DepositStatus = "none"
		if amount, _ := strconv.ParseFloat(item.DepositAmount, 64); amount > 0 {
			item.DepositStatus = "pending"
		}
	}
	return item, nil
}
func (r *Reservations) List(ctx context.Context, s scope, f application.FloorFilter) (application.ReservationPage, error) {
	out := application.ReservationPage{Items: []domain.ReservationView{}, Page: f.Page, PageSize: f.Size}
	where := `r.organization_id=$1 AND r.location_id=$2
		AND ($3='' OR r.customer_name ILIKE '%'||$3||'%' OR r.customer_phone ILIKE '%'||$3||'%' OR COALESCE(t.name,'') ILIKE '%'||$3||'%')
		AND ($4='' OR `+reservationStatusSQL+`=$4)`
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
		INSERT INTO reservations AS r(organization_id,location_id,customer_id,customer_name,customer_phone,starts_at,guests,duration_minutes,table_id,notes,created_by,tolerance_minutes,deposit_amount)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::numeric)
		RETURNING `+reservationColumns()+`
	`, s.OrganizationID, s.LocationID, in.CustomerID, in.CustomerName, in.CustomerPhone, startsAt, in.Guests, in.DurationMinutes, in.TableID, in.Notes, s.UserID, in.ToleranceMinutes, in.DepositAmount))
	return v, serviceError(err)
}
func (r *Reservations) Update(ctx context.Context, s scope, id string, in domain.ReservationInput, startsAt time.Time) (domain.ReservationView, error) {
	v, err := scanReservation(r.db(ctx).QueryRow(ctx, `
		UPDATE reservations r SET customer_id=$4,customer_name=$5,customer_phone=$6,starts_at=$7,guests=$8,duration_minutes=$9,table_id=$10,notes=$11,
		  tolerance_minutes=COALESCE($12::integer,r.tolerance_minutes),deposit_amount=CASE WHEN EXISTS(SELECT 1 FROM payments p WHERE p.reservation_id=r.id AND p.organization_id=r.organization_id) THEN r.deposit_amount ELSE COALESCE($13::numeric,r.deposit_amount) END,updated_at=now()
		WHERE r.id=$1 AND r.organization_id=$2 AND r.location_id=$3
		RETURNING `+reservationColumns()+`
	`, id, s.OrganizationID, s.LocationID, in.CustomerID, in.CustomerName, in.CustomerPhone, startsAt, in.Guests, in.DurationMinutes, in.TableID, in.Notes, in.ToleranceMinutes, in.DepositAmount))
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
// Seat pasa la reserva a «En mesa» con la mesa elegida, o con la que ya tenía si
// tableID está vacío. Devuelve false si la reserva quedaría sin mesa.
func (r *Reservations) Seat(ctx context.Context, s scope, id, tableID string) (bool, error) {
	var seated bool
	err := r.db(ctx).QueryRow(ctx, `
		UPDATE reservations SET status='seated',table_id=COALESCE(NULLIF($4,'')::uuid,table_id),updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND COALESCE(NULLIF($4,'')::uuid,table_id) IS NOT NULL
		RETURNING true`, id, s.OrganizationID, s.LocationID, tableID).Scan(&seated)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return seated, err
}
func (r *Reservations) Defaults(ctx context.Context, s scope) (application.ReservationDefaults, error) {
	var out application.ReservationDefaults
	err := r.db(ctx).QueryRow(ctx, `SELECT reservation_tolerance_minutes,reservation_deposit_amount::text FROM locations WHERE id=$1 AND organization_id=$2`, s.LocationID, s.OrganizationID).Scan(&out.ToleranceMinutes, &out.DepositAmount)
	return out, serviceError(err)
}

// ExpireOverdue vence en una sola sentencia las reservas abiertas cuya hora más
// la tolerancia ya pasó. SKIP LOCKED evita que dos instancias del API, o una
// edición en curso, tomen la misma reserva; la condición de estado se repite en
// el UPDATE para no pisar una reserva que se sentó o canceló entretanto.
func (r *Reservations) ExpireOverdue(ctx context.Context, limit int) ([]domain.ExpiredReservation, error) {
	rows, err := r.db(ctx).Query(ctx, `
		WITH due AS (
		  SELECT id FROM reservations
		  WHERE status IN ('pending','confirmed') AND starts_at<=now()
		    AND starts_at+make_interval(mins=>tolerance_minutes)<=now()
		  ORDER BY starts_at
		  LIMIT $1
		  FOR UPDATE SKIP LOCKED
		)
		UPDATE reservations r SET status='no_show',expired_at=now(),updated_at=now()
		FROM due,locations l
		WHERE r.id=due.id AND r.status IN ('pending','confirmed')
		  AND l.id=r.location_id AND l.organization_id=r.organization_id
		RETURNING r.id::text,r.organization_id::text,r.location_id::text,r.customer_name,
		  COALESCE((SELECT name FROM tables t WHERE t.id=r.table_id AND t.organization_id=r.organization_id AND t.location_id=r.location_id),''),
		  to_char(r.starts_at AT TIME ZONE l.timezone,'HH24:MI'),r.guests,
		  COALESCE((SELECT CASE WHEN fp.currency_position='after' THEN p.amount::text||' '||fp.currency_symbol ELSE fp.currency_symbol||' '||p.amount::text END
		    FROM payments p JOIN organization_fiscal_profiles fp ON fp.id=l.fiscal_profile_id AND fp.organization_id=l.organization_id
		    WHERE p.reservation_id=r.id AND p.organization_id=r.organization_id AND p.order_id IS NULL
		      AND NOT EXISTS(SELECT 1 FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id)),'')`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ExpiredReservation{}
	for rows.Next() {
		var item domain.ExpiredReservation
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.LocationID, &item.CustomerName, &item.TableName, &item.LocalTime, &item.Guests, &item.Deposit); err != nil {
			return out, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
