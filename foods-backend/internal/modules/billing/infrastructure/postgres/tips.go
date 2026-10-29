package postgres

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/billing/application"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"github.com/jackc/pgx/v5"
	"math"
)

var _ application.TipRepository = (*Payments)(nil)

func (r *Payments) TipPolicy(ctx context.Context, s scope, global bool) (domain.TipPolicy, error) {
	var p domain.TipPolicy
	err := r.db(ctx).QueryRow(ctx, `SELECT COALESCE(ls.mode,os.mode,CASE WHEN fp.country_code='CL' THEN 'suggest' ELSE 'disabled' END),
 COALESCE(ls.percentage,os.percentage,10)::float8,COALESCE(ls.show_suggestion,os.show_suggestion,fp.country_code='CL'),COALESCE(ls.allow_fixed,os.allow_fixed,true),
 COALESCE(ls.channels,os.channels,ARRAY['salon']::text[]),ls.id IS NULL AND NOT $3,fp.country_code,fp.currency,fp.currency_decimals
 FROM locations l JOIN organization_fiscal_profiles fp ON fp.id=l.fiscal_profile_id AND fp.organization_id=l.organization_id
 LEFT JOIN tip_settings os ON os.organization_id=l.organization_id AND os.location_id IS NULL
 LEFT JOIN tip_settings ls ON ls.organization_id=l.organization_id AND ls.location_id=l.id AND NOT $3
 WHERE l.id=$2 AND l.organization_id=$1`, s.OrganizationID, s.LocationID, global).Scan(&p.Mode, &p.Percentage, &p.ShowSuggestion, &p.AllowFixed, &p.Channels, &p.Inherited, &p.Country, &p.Currency, &p.Decimals)
	return p, billingError(err)
}
func (r *Payments) SaveTipSettings(ctx context.Context, s scope, global bool, in domain.TipSettingsInput) error {
	var location any = s.LocationID
	if global {
		location = nil
	}
	if in.Inherit {
		_, err := r.db(ctx).Exec(ctx, `DELETE FROM tip_settings WHERE organization_id=$1 AND location_id=$2`, s.OrganizationID, s.LocationID)
		return billingError(err)
	}
	_, err := r.db(ctx).Exec(ctx, `INSERT INTO tip_settings(organization_id,location_id,mode,percentage,show_suggestion,allow_fixed,channels,updated_by)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8)
 ON CONFLICT (organization_id,(COALESCE(location_id,'00000000-0000-0000-0000-000000000000'::uuid))) DO UPDATE
 SET mode=excluded.mode,percentage=excluded.percentage,show_suggestion=excluded.show_suggestion,allow_fixed=excluded.allow_fixed,channels=excluded.channels,updated_by=excluded.updated_by,updated_at=now()`, s.OrganizationID, location, in.Mode, in.Percentage, in.ShowSuggestion, in.AllowFixed, in.Channels, s.UserID)
	return billingError(err)
}

const tipViewColumns = `t.id::text,t.payment_id::text,o.code,t.amount::text,pm.name,t.affects_cash,t.direct_to_staff,u.full_name,to_char(t.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')`
const tipViewJoin = ` FROM payment_tips t JOIN orders o ON o.id=t.order_id AND o.organization_id=t.organization_id AND o.location_id=t.location_id JOIN payment_methods pm ON pm.organization_id=t.organization_id AND pm.code=t.method JOIN users u ON u.id=t.created_by AND u.organization_id=t.organization_id `

func scanTip(row pgx.Row) (domain.TipView, error) {
	var t domain.TipView
	err := row.Scan(&t.ID, &t.PaymentID, &t.OrderCode, &t.Amount, &t.MethodName, &t.AffectsCash, &t.DirectToStaff, &t.CreatedByName, &t.CreatedAt)
	return t, billingError(err)
}
func (r *Payments) TipForOrder(ctx context.Context, s scope, orderID string) (*domain.TipView, error) {
	t, err := scanTip(r.db(ctx).QueryRow(ctx, `SELECT `+tipViewColumns+tipViewJoin+` WHERE t.organization_id=$1 AND t.location_id=$2 AND t.order_id=$3`, s.OrganizationID, s.LocationID, orderID))
	if errors.Is(err, application.ErrNotFound) {
		return nil, nil
	}
	return &t, err
}
func (r *Payments) TipOrderContext(ctx context.Context, s scope, id string) (string, float64, error) {
	var channel string
	var base float64
	err := r.db(ctx).QueryRow(ctx, `SELECT CASE WHEN channel='mostrador' THEN 'recojo' ELSE channel END,GREATEST(0,total-delivery_fee)::float8 FROM orders WHERE id=$1 AND organization_id=$2 AND location_id=$3`, id, s.OrganizationID, s.LocationID).Scan(&channel, &base)
	return channel, base, billingError(err)
}
func (r *Payments) InsertTip(ctx context.Context, s scope, orderID, paymentID, shiftID string, base float64, in domain.TipInput, amount float64, affectsCash bool) error {
	var id string
	err := r.db(ctx).QueryRow(ctx, `INSERT INTO payment_tips(organization_id,location_id,payment_id,order_id,shift_id,kind,value,base_amount,amount,method,affects_cash,direct_to_staff,created_by)
 SELECT $1,$2,p.id,p.order_id,p.shift_id,$6,$7,$8,$9,p.method,$10,$11,$12 FROM payments p
 WHERE p.id=$3 AND p.organization_id=$1 AND p.location_id=$2 AND p.order_id=$4 AND p.shift_id=$5 RETURNING id::text`, s.OrganizationID, s.LocationID, paymentID, orderID, shiftID, in.Kind, in.Value, base, amount, affectsCash, in.DirectToStaff, s.UserID).Scan(&id)
	if err != nil {
		return billingError(err)
	}
	return nil
}
func (r *Payments) TipRecipients(ctx context.Context, s scope) ([]domain.TipRecipient, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT u.id::text,u.full_name FROM users u WHERE u.organization_id=$1 AND u.active AND NOT u.platform_admin
 AND EXISTS(SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id=ur.role_id AND ro.organization_id=u.organization_id AND ro.active WHERE ur.user_id=u.id AND (ur.location_id IS NULL OR ur.location_id=$2))
 AND NOT EXISTS(SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id=ur.role_id AND ro.organization_id=u.organization_id WHERE ur.user_id=u.id AND ro.active AND ro.system_key='administrator') ORDER BY u.full_name,u.id`, s.OrganizationID, s.LocationID)
	if err != nil {
		return nil, billingError(err)
	}
	defer rows.Close()
	out := []domain.TipRecipient{}
	for rows.Next() {
		var v domain.TipRecipient
		if err = rows.Scan(&v.ID, &v.Name); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Each receipt is locked in a deterministic order; a competing payout rechecks
// allocations in a separate statement after acquiring those locks (READ COMMITTED).
func (r *Payments) LockTipBalances(ctx context.Context, s scope, all bool) ([]domain.TipBalance, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT t.id::text FROM payment_tips t JOIN cash_shifts cs ON cs.id=t.shift_id AND cs.organization_id=t.organization_id AND cs.location_id=t.location_id
 WHERE t.organization_id=$1 AND t.location_id=$2 AND NOT t.direct_to_staff AND ($3 OR cs.opened_by=$4)
 AND t.amount>COALESCE((SELECT sum(a.amount) FROM tip_payout_allocations a WHERE a.tip_id=t.id AND a.organization_id=$1 AND a.location_id=$2),0)
 ORDER BY t.id FOR UPDATE OF t`, s.OrganizationID, s.LocationID, all, s.UserID)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	out := []domain.TipBalance{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err = r.db(ctx).Query(ctx, `SELECT t.id::text,(t.amount-COALESCE((SELECT sum(a.amount) FROM tip_payout_allocations a WHERE a.tip_id=t.id AND a.organization_id=$2 AND a.location_id=$3),0))::float8 FROM payment_tips t WHERE t.id=ANY($1::uuid[]) AND t.organization_id=$2 AND t.location_id=$3 ORDER BY t.id`, ids, s.OrganizationID, s.LocationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b domain.TipBalance
		if err = rows.Scan(&b.ID, &b.Available); err != nil {
			return nil, err
		}
		if b.Available > 0 {
			out = append(out, b)
		}
	}
	return out, rows.Err()
}
func (r *Payments) TipPending(ctx context.Context, s scope, all bool) (string, error) {
	var pending string
	err := r.db(ctx).QueryRow(ctx, `SELECT COALESCE(sum(t.amount-COALESCE((SELECT sum(a.amount) FROM tip_payout_allocations a WHERE a.tip_id=t.id AND a.organization_id=$1 AND a.location_id=$2),0)),0)::text
 FROM payment_tips t JOIN cash_shifts cs ON cs.id=t.shift_id AND cs.organization_id=t.organization_id AND cs.location_id=t.location_id
 WHERE t.organization_id=$1 AND t.location_id=$2 AND NOT t.direct_to_staff AND ($3 OR cs.opened_by=$4)`, s.OrganizationID, s.LocationID, all, s.UserID).Scan(&pending)
	return pending, billingError(err)
}

const payoutColumns = `p.id::text,p.recipient_id::text,recipient.full_name,p.amount::text,p.method,p.note,u.full_name,to_char(p.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),p.shift_id::text,p.created_by::text`
const payoutJoin = ` FROM tip_payouts p JOIN users recipient ON recipient.id=p.recipient_id AND recipient.organization_id=p.organization_id JOIN users u ON u.id=p.created_by AND u.organization_id=p.organization_id `

func scanPayout(row pgx.Row) (domain.TipPayoutView, error) {
	var v domain.TipPayoutView
	err := row.Scan(&v.ID, &v.RecipientID, &v.RecipientName, &v.Amount, &v.Method, &v.Note, &v.CreatedByName, &v.CreatedAt, &v.ShiftID, &v.CreatedByID)
	return v, billingError(err)
}
func (r *Payments) FindTipPayout(ctx context.Context, s scope, key string) (domain.TipPayoutView, error) {
	return scanPayout(r.db(ctx).QueryRow(ctx, `SELECT `+payoutColumns+payoutJoin+` WHERE p.organization_id=$1 AND p.location_id=$2 AND p.request_key=$3`, s.OrganizationID, s.LocationID, key))
}
func (r *Payments) InsertTipPayout(ctx context.Context, s scope, shiftID string, in domain.TipPayoutInput, balances []domain.TipBalance) (domain.TipPayoutView, error) {
	var out domain.TipPayoutView
	var id string
	err := r.db(ctx).QueryRow(ctx, `INSERT INTO tip_payouts(organization_id,location_id,shift_id,recipient_id,amount,method,note,request_key,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id::text`, s.OrganizationID, s.LocationID, shiftID, in.RecipientID, in.Amount, in.Method, in.Note, in.RequestKey, s.UserID).Scan(&id)
	if err != nil {
		return out, billingError(err)
	}
	remaining := in.Amount
	for _, b := range balances {
		part := math.Min(remaining, b.Available)
		if part <= .00001 {
			continue
		}
		_, err = r.db(ctx).Exec(ctx, `INSERT INTO tip_payout_allocations(organization_id,location_id,payout_id,tip_id,amount) VALUES($1,$2,$3,$4,$5)`, s.OrganizationID, s.LocationID, id, b.ID, part)
		if err != nil {
			return out, err
		}
		remaining -= part
	}
	if remaining > .00001 {
		return out, application.ErrConflict
	}
	return r.FindTipPayout(ctx, s, in.RequestKey)
}
func (r *Payments) TipReport(ctx context.Context, s scope, shiftID string) (domain.TipReport, error) {
	out := domain.TipReport{Receipts: []domain.TipView{}, Payouts: []domain.TipPayoutView{}}
	err := r.db(ctx).QueryRow(ctx, `SELECT
 COALESCE(sum(t.amount) FILTER(WHERE t.affects_cash AND NOT t.direct_to_staff),0)::text,
 COALESCE(sum(t.amount) FILTER(WHERE NOT t.affects_cash AND NOT t.direct_to_staff),0)::text,
 COALESCE(sum(t.amount) FILTER(WHERE t.direct_to_staff),0)::text,
 COALESCE(sum(t.amount-COALESCE((SELECT sum(a.amount) FROM tip_payout_allocations a WHERE a.tip_id=t.id AND a.organization_id=$1 AND a.location_id=$2),0)) FILTER(WHERE NOT t.direct_to_staff),0)::text
 FROM payment_tips t WHERE t.organization_id=$1 AND t.location_id=$2 AND t.shift_id=$3`, s.OrganizationID, s.LocationID, shiftID).Scan(&out.CashCollected, &out.OtherCollected, &out.Direct, &out.Pending)
	if err != nil {
		return out, err
	}
	err = r.db(ctx).QueryRow(ctx, `SELECT COALESCE(sum(amount) FILTER(WHERE method='cash'),0)::text,COALESCE(sum(amount) FILTER(WHERE method='transfer'),0)::text FROM tip_payouts WHERE organization_id=$1 AND location_id=$2 AND shift_id=$3`, s.OrganizationID, s.LocationID, shiftID).Scan(&out.CashPaidOut, &out.OtherPaidOut)
	if err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `SELECT `+tipViewColumns+tipViewJoin+` WHERE t.organization_id=$1 AND t.location_id=$2 AND t.shift_id=$3 ORDER BY t.created_at,t.id`, s.OrganizationID, s.LocationID, shiftID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		v, e := scanTip(rows)
		if e != nil {
			rows.Close()
			return out, e
		}
		out.Receipts = append(out.Receipts, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = r.db(ctx).Query(ctx, `SELECT `+payoutColumns+payoutJoin+` WHERE p.organization_id=$1 AND p.location_id=$2 AND p.shift_id=$3 ORDER BY p.created_at,p.id`, s.OrganizationID, s.LocationID, shiftID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		v, e := scanPayout(rows)
		if e != nil {
			rows.Close()
			return out, e
		}
		out.Payouts = append(out.Payouts, v)
	}
	err = rows.Err()
	rows.Close()
	return out, err
}
