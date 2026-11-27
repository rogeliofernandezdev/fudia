package postgres

import (
	"context"

	"github.com/foods-platform/foods-backend/internal/modules/billing/application"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
)

var _ application.DepositRepository = (*Payments)(nil)

// heldDepositSQL selects a charged guarantee still without an order, net of
// refunds. Callers add the WHERE condition and keep the column order.
const heldDepositSQL = `
	SELECT p.id::text,r.id::text,r.status,
	       (p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id AND pr.location_id=p.location_id),0))::float8,
	       pm.affects_cash,r.customer_name,r.customer_phone
	FROM payments p
	JOIN reservations r ON r.id=p.reservation_id AND r.organization_id=p.organization_id AND r.location_id=p.location_id
	JOIN payment_methods pm ON pm.organization_id=p.organization_id AND pm.code=p.method
	WHERE p.organization_id=$2 AND p.location_id=$3 AND p.order_id IS NULL
	  AND p.amount>COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id AND pr.location_id=p.location_id),0)+0.00001`

func scanHeldDeposit(row interface{ Scan(...any) error }) (domain.HeldDeposit, error) {
	var d domain.HeldDeposit
	err := row.Scan(&d.PaymentID, &d.ReservationID, &d.ReservationStatus, &d.Amount, &d.AffectsCash, &d.CustomerName, &d.CustomerPhone)
	return d, billingError(err)
}

func (r *Payments) LockDepositReservation(ctx context.Context, s scope, reservationID string) (domain.DepositReservation, error) {
	var out domain.DepositReservation
	err := r.db(ctx).QueryRow(ctx, `
		SELECT r.status,r.deposit_amount::float8,r.customer_name,
		       EXISTS(SELECT 1 FROM payments p WHERE p.reservation_id=r.id AND p.organization_id=r.organization_id)
		FROM reservations r
		WHERE r.id=$1 AND r.organization_id=$2 AND r.location_id=$3
		FOR UPDATE`, reservationID, s.OrganizationID, s.LocationID).Scan(&out.Status, &out.Amount, &out.CustomerName, &out.Charged)
	return out, billingError(err)
}

func (r *Payments) CreateDepositPayment(ctx context.Context, s scope, reservationID, shiftID string, in domain.DepositInput, amount float64) (string, error) {
	var id string
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO payments(organization_id,location_id,reservation_id,shift_id,method,amount,reference,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id::text`, s.OrganizationID, s.LocationID, reservationID, shiftID, in.Method, amount, in.Reference, s.UserID).Scan(&id)
	return id, billingError(err)
}

// DepositMovement refleja en el efectivo del turno el cobro (income) o la
// devolución (expense) de una garantía.
func (r *Payments) DepositMovement(ctx context.Context, s scope, shiftID, sourceID, movement string, amount float64, reason, note string) error {
	source := "reservation_deposit"
	if movement == "expense" {
		source = "reservation_deposit_refund"
	}
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO cash_movements(organization_id,location_id,shift_id,movement_type,source_type,source_id,amount,reason,note,created_by)
		VALUES($1,$2,$3,$4,$5,$6::uuid,$7,$8,$9,$10)`,
		s.OrganizationID, s.LocationID, shiftID, movement, source, sourceID, amount, reason, note, s.UserID)
	return billingError(err)
}

func (r *Payments) DepositMethodName(ctx context.Context, s scope, method string) (string, error) {
	var name string
	err := r.db(ctx).QueryRow(ctx, `SELECT name FROM payment_methods WHERE organization_id=$1 AND code=$2`, s.OrganizationID, method).Scan(&name)
	return name, billingError(err)
}

func (r *Payments) LockHeldDeposit(ctx context.Context, s scope, reservationID string) (domain.HeldDeposit, error) {
	return scanHeldDeposit(r.db(ctx).QueryRow(ctx, heldDepositSQL+` AND p.reservation_id=$1 FOR UPDATE OF p`, reservationID, s.OrganizationID, s.LocationID))
}

func (r *Payments) LockHeldDepositPayment(ctx context.Context, s scope, paymentID string) (domain.HeldDeposit, error) {
	return scanHeldDeposit(r.db(ctx).QueryRow(ctx, heldDepositSQL+` AND p.id=$1 FOR UPDATE OF p`, paymentID, s.OrganizationID, s.LocationID))
}

func (r *Payments) AttachDeposit(ctx context.Context, s scope, paymentID, orderID, outcome string) error {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE payments SET order_id=$4,deposit_outcome=$5
		WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND order_id IS NULL`,
		paymentID, s.OrganizationID, s.LocationID, orderID, outcome)
	if err == nil && tag.RowsAffected() == 0 {
		return application.ErrNotFound
	}
	return billingError(err)
}

// SeatDepositReservation sienta la reserva cuya garantía se descontó: el
// cliente está en el local y el barrido ya no debe vencerla.
func (r *Payments) SeatDepositReservation(ctx context.Context, s scope, reservationID string) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE reservations SET status='seated',updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND status IN ('pending','confirmed')`,
		reservationID, s.OrganizationID, s.LocationID)
	return billingError(err)
}

// CreateRetainedDepositOrder registra la garantía retenida como una venta de
// mostrador ya entregada y cerrada, con una sola línea sin producto: así entra
// en Ventas y puede llevar su boleta como cualquier pedido.
func (r *Payments) CreateRetainedDepositOrder(ctx context.Context, s scope, d domain.HeldDeposit) (string, error) {
	var id string
	err := r.db(ctx).QueryRow(ctx, `
		WITH created AS (
		  INSERT INTO orders(organization_id,location_id,channel,status,customer_name,customer_phone,notes,subtotal,total,created_by,completed_at,bill_closed_at)
		  VALUES($1,$2,'mostrador','entregado',$3,$4,'',$5,$5,NULLIF($6,'')::uuid,now(),now())
		  RETURNING id,organization_id
		)
		INSERT INTO order_items(organization_id,order_id,name,qty,unit_price,item_type)
		SELECT organization_id,id,$7,1,$5,'reservation_deposit' FROM created
		RETURNING order_id::text`,
		s.OrganizationID, s.LocationID, d.CustomerName, d.CustomerPhone, d.Amount, s.UserID, domain.RetainedDepositItem).Scan(&id)
	return id, billingError(err)
}

// DepositOptions lista las garantías cobradas que esperan a su cliente hoy en
// el local, para descontarlas de la cuenta del pedido.
func (r *Payments) DepositOptions(ctx context.Context, s scope, orderID string) ([]domain.DepositOption, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT p.id::text,r.customer_name,
		       (p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id AND pr.location_id=p.location_id),0))::text,
		       pm.name,to_char(r.starts_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),COALESCE(t.name,''),
		       COALESCE(r.table_id=o.table_id,false)
		FROM payments p
		JOIN reservations r ON r.id=p.reservation_id AND r.organization_id=p.organization_id AND r.location_id=p.location_id
		JOIN payment_methods pm ON pm.organization_id=p.organization_id AND pm.code=p.method
		JOIN locations l ON l.id=p.location_id AND l.organization_id=p.organization_id
		LEFT JOIN tables t ON t.id=r.table_id AND t.organization_id=r.organization_id
		LEFT JOIN orders o ON o.id=$1 AND o.organization_id=p.organization_id AND o.location_id=p.location_id
		WHERE p.organization_id=$2 AND p.location_id=$3 AND p.order_id IS NULL
		  AND r.status IN ('pending','confirmed','seated')
		  AND (r.starts_at AT TIME ZONE l.timezone)::date=(now() AT TIME ZONE l.timezone)::date
		  AND p.amount>COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id AND pr.location_id=p.location_id),0)+0.00001
		ORDER BY 7 DESC,r.starts_at,r.customer_name`, orderID, s.OrganizationID, s.LocationID)
	if err != nil {
		return nil, billingError(err)
	}
	defer rows.Close()
	out := []domain.DepositOption{}
	for rows.Next() {
		var d domain.DepositOption
		if err := rows.Scan(&d.PaymentID, &d.CustomerName, &d.Amount, &d.MethodName, &d.StartsAt, &d.TableName, &d.Suggested); err != nil {
			return nil, billingError(err)
		}
		out = append(out, d)
	}
	return out, billingError(rows.Err())
}

// ForfeitedDeposits devuelve garantías cobradas cuya reserva terminó sin el
// cliente («No asistió» o cancelada sin devolverla), en todas las empresas.
func (r *Payments) ForfeitedDeposits(ctx context.Context, limit int) ([]domain.ForfeitedDeposit, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT p.organization_id::text,p.location_id::text,p.reservation_id::text
		FROM payments p
		JOIN reservations r ON r.id=p.reservation_id AND r.organization_id=p.organization_id
		WHERE p.reservation_id IS NOT NULL AND p.order_id IS NULL AND r.status IN ('no_show','cancelled')
		  AND p.amount>COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id AND pr.location_id=p.location_id),0)+0.00001
		ORDER BY p.created_at
		LIMIT $1`, limit)
	if err != nil {
		return nil, billingError(err)
	}
	defer rows.Close()
	out := []domain.ForfeitedDeposit{}
	for rows.Next() {
		var d domain.ForfeitedDeposit
		if err := rows.Scan(&d.OrganizationID, &d.LocationID, &d.ReservationID); err != nil {
			return nil, billingError(err)
		}
		out = append(out, d)
	}
	return out, billingError(rows.Err())
}
