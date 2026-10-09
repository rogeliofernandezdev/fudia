package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

type cashReportSale struct {
	ID         string `json:"id"`
	OrderCode  string `json:"orderCode"`
	Customer   string `json:"customer"`
	Name       string `json:"name"`
	Details    string `json:"details"`
	Quantity   string `json:"quantity"`
	UnitPrice  string `json:"unitPrice"`
	Total      string `json:"total"`
	CreatedAt  string `json:"createdAt"`
	WaiterName string `json:"waiterName"`
}
type cashReportPayment struct {
	ID            string `json:"id"`
	OrderCode     string `json:"orderCode"`
	MethodName    string `json:"methodName"`
	Amount        string `json:"amount"`
	Reference     string `json:"reference"`
	CreatedByName string `json:"createdByName"`
	CreatedAt     string `json:"createdAt"`
}
type cashReportCount struct {
	Denomination string `json:"denomination"`
	Quantity     int    `json:"quantity"`
	Total        string `json:"total"`
}
type cashReportMethod struct {
	Name   string `json:"name"`
	Amount string `json:"amount"`
}
type cashShiftReport struct {
	Version            int                 `json:"version"`
	Persisted          bool                `json:"persisted"`
	GeneratedAt        string              `json:"generatedAt"`
	OrganizationName   string              `json:"organizationName"`
	LocationName       string              `json:"locationName"`
	Timezone           string              `json:"timezone"`
	Country            string              `json:"country"`
	Currency           string              `json:"currency"`
	CurrencySymbol     string              `json:"currencySymbol"`
	CurrencyPosition   string              `json:"currencyPosition"`
	CurrencyDecimals   int                 `json:"currencyDecimals"`
	Shift              cashShiftView       `json:"shift"`
	Sales              []cashReportSale    `json:"sales"`
	Payments           []cashReportPayment `json:"payments"`
	Refunds            []cashReportPayment `json:"refunds"`
	Methods            []cashReportMethod  `json:"methods"`
	Counts             []cashReportCount   `json:"counts"`
	CollectedAmount    string              `json:"collectedAmount"`
	RefundedAmount     string              `json:"refundedAmount"`
	NetCollectedAmount string              `json:"netCollectedAmount"`
}

// A fixed number of scoped queries, using the caller's transaction/snapshot.
func loadCashShiftReport(ctx context.Context, tx pgx.Tx, s scope, id string) (cashShiftReport, error) {
	out := cashShiftReport{Version: 1, GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Sales: []cashReportSale{}, Payments: []cashReportPayment{}, Refunds: []cashReportPayment{}, Methods: []cashReportMethod{}, Counts: []cashReportCount{}}
	var err error
	out.Shift, err = scanCashShift(tx.QueryRow(ctx, `SELECT `+cashShiftColumns+`
 FROM cash_shifts cs JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
 JOIN users opened ON opened.id=cs.opened_by AND opened.organization_id=cs.organization_id
 LEFT JOIN users closed ON closed.id=cs.closed_by AND closed.organization_id=cs.organization_id
 WHERE cs.id=$1 AND cs.organization_id=$2 AND cs.location_id=$3`, id, s.OrganizationID, s.LocationID))
	if err != nil {
		return out, err
	}
	out.Shift.ExpectedVisible = true
	err = tx.QueryRow(ctx, `SELECT COALESCE(NULLIF(o.trade_name,''),o.legal_name),l.name,l.timezone,fp.country_code,fp.currency,fp.currency_symbol,fp.currency_position,fp.currency_decimals
 FROM locations l JOIN organizations o ON o.id=l.organization_id
 JOIN organization_fiscal_profiles fp ON fp.id=l.fiscal_profile_id AND fp.organization_id=l.organization_id
 WHERE l.id=$1 AND l.organization_id=$2`, s.LocationID, s.OrganizationID).Scan(&out.OrganizationName, &out.LocationName, &out.Timezone, &out.Country, &out.Currency, &out.CurrencySymbol, &out.CurrencyPosition, &out.CurrencyDecimals)
	if err != nil {
		return out, err
	}
	rows, err := tx.Query(ctx, `SELECT cm.id::text,cm.movement_type,cm.source_type,cm.source_id::text,cm.amount::text,cm.reason,cm.note,u.full_name,
 to_char(cm.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
 FROM cash_movements cm JOIN users u ON u.id=cm.created_by AND u.organization_id=cm.organization_id
 WHERE cm.shift_id=$1 AND cm.organization_id=$2 AND cm.location_id=$3 ORDER BY cm.created_at,cm.id`, id, s.OrganizationID, s.LocationID)
	if err != nil {
		return out, err
	}
	out.Shift.Movements = []cashMovementView{}
	for rows.Next() {
		var v cashMovementView
		err = rows.Scan(&v.ID, &v.MovementType, &v.SourceType, &v.SourceID, &v.Amount, &v.Reason, &v.Note, &v.CreatedByName, &v.CreatedAt)
		if err != nil {
			rows.Close()
			return out, err
		}
		out.Shift.Movements = append(out.Shift.Movements, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	// EXISTS includes each order line only once, regardless of the number of payments.
	rows, err = tx.Query(ctx, `SELECT i.id::text,o.code,COALESCE(NULLIF(t.name,''),NULLIF(o.customer_name,''),'Mostrador'),i.name,
 concat_ws(' · ',NULLIF(i.note,''),(SELECT string_agg(sel.group_name||': '||sel.option_name,' · ' ORDER BY sel.id) FROM order_item_combo_selections sel WHERE sel.order_item_id=i.id AND sel.organization_id=i.organization_id)),
 i.qty::text,i.unit_price::text,round(i.qty*i.unit_price,2)::text,
 to_char(i.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),COALESCE(u.full_name,'')
 FROM order_items i JOIN orders o ON o.id=i.order_id AND o.organization_id=i.organization_id
 LEFT JOIN tables t ON t.id=o.table_id AND t.organization_id=o.organization_id
 LEFT JOIN users u ON u.id=o.waiter_id AND u.organization_id=o.organization_id
 WHERE o.organization_id=$2 AND o.location_id=$3 AND EXISTS(SELECT 1 FROM payments p WHERE p.order_id=o.id AND p.organization_id=o.organization_id AND p.location_id=o.location_id AND p.shift_id=$1)
 ORDER BY o.created_at,o.id,i.created_at,i.id`, id, s.OrganizationID, s.LocationID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var v cashReportSale
		err = rows.Scan(&v.ID, &v.OrderCode, &v.Customer, &v.Name, &v.Details, &v.Quantity, &v.UnitPrice, &v.Total, &v.CreatedAt, &v.WaiterName)
		if err != nil {
			rows.Close()
			return out, err
		}
		out.Sales = append(out.Sales, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	// Refunds belong to the shift where they were registered, not the original payment's shift.
	rows, err = tx.Query(ctx, `SELECT p.id::text,o.code,pm.name,p.amount::text,p.reference,u.full_name,to_char(p.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),false
 FROM payments p JOIN orders o ON o.id=p.order_id AND o.organization_id=p.organization_id AND o.location_id=p.location_id
 JOIN payment_methods pm ON pm.organization_id=p.organization_id AND pm.code=p.method JOIN users u ON u.id=p.created_by AND u.organization_id=p.organization_id
 WHERE p.shift_id=$1 AND p.organization_id=$2 AND p.location_id=$3
 UNION ALL SELECT pr.id::text,o.code,pm.name,pr.amount::text,pr.reason,u.full_name,to_char(pr.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),true
 FROM payment_refunds pr JOIN payments p ON p.id=pr.payment_id AND p.organization_id=pr.organization_id AND p.location_id=pr.location_id
 JOIN orders o ON o.id=p.order_id AND o.organization_id=p.organization_id AND o.location_id=p.location_id
 JOIN payment_methods pm ON pm.organization_id=p.organization_id AND pm.code=p.method JOIN users u ON u.id=pr.created_by AND u.organization_id=pr.organization_id
 WHERE pr.shift_id=$1 AND pr.organization_id=$2 AND pr.location_id=$3 ORDER BY 7,1`, id, s.OrganizationID, s.LocationID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var v cashReportPayment
		var refund bool
		err = rows.Scan(&v.ID, &v.OrderCode, &v.MethodName, &v.Amount, &v.Reference, &v.CreatedByName, &v.CreatedAt, &refund)
		if err != nil {
			rows.Close()
			return out, err
		}
		if refund {
			out.Refunds = append(out.Refunds, v)
		} else {
			out.Payments = append(out.Payments, v)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	err = tx.QueryRow(ctx, `WITH totals AS(SELECT COALESCE((SELECT sum(amount) FROM payments WHERE shift_id=$1 AND organization_id=$2 AND location_id=$3),0) collected,COALESCE((SELECT sum(amount) FROM payment_refunds WHERE shift_id=$1 AND organization_id=$2 AND location_id=$3),0) refunded) SELECT collected::text,refunded::text,(collected-refunded)::text FROM totals`, id, s.OrganizationID, s.LocationID).Scan(&out.CollectedAmount, &out.RefundedAmount, &out.NetCollectedAmount)
	if err != nil {
		return out, err
	}
	rows, err = tx.Query(ctx, `SELECT pm.name,sum(p.amount)::text FROM payments p JOIN payment_methods pm ON pm.organization_id=p.organization_id AND pm.code=p.method WHERE p.shift_id=$1 AND p.organization_id=$2 AND p.location_id=$3 GROUP BY pm.code,pm.name ORDER BY pm.name,pm.code`, id, s.OrganizationID, s.LocationID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var v cashReportMethod
		err = rows.Scan(&v.Name, &v.Amount)
		if err != nil {
			rows.Close()
			return out, err
		}
		out.Methods = append(out.Methods, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = tx.Query(ctx, `SELECT denomination::text,quantity,(denomination*quantity)::text FROM cash_count_lines WHERE shift_id=$1 AND organization_id=$2 AND location_id=$3 ORDER BY denomination DESC`, id, s.OrganizationID, s.LocationID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var v cashReportCount
		err = rows.Scan(&v.Denomination, &v.Quantity, &v.Total)
		if err != nil {
			rows.Close()
			return out, err
		}
		out.Counts = append(out.Counts, v)
	}
	err = rows.Err()
	rows.Close()
	return out, err
}

func (a *API) getCashShiftReport(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	tx, err := a.db.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		fail(w, 503, "cash_report_unavailable", "No pudimos cargar el resumen de caja.")
		return
	}
	defer tx.Rollback(r.Context())
	var stored []byte
	err = tx.QueryRow(r.Context(), `SELECT report FROM cash_shift_reports WHERE shift_id=$1 AND organization_id=$2 AND location_id=$3`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&stored)
	var out cashShiftReport
	if err == nil {
		err = json.Unmarshal(stored, &out)
	} else if errors.Is(err, pgx.ErrNoRows) {
		out, err = loadCashShiftReport(r.Context(), tx, s, r.PathValue("id"))
	}
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "cash_shift_not_found", "El turno de caja no existe en este local.")
		return
	}
	if err != nil {
		fail(w, 503, "cash_report_unavailable", "No pudimos cargar el resumen de caja.")
		return
	}
	if out.Shift.Status == "open" && out.Shift.BlindClose && !a.canSeeCashExpected(r, s) {
		fail(w, 403, "cash_report_blind_close", "El detalle de este cierre ciego estará disponible después de cerrar el turno.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "cash_report_unavailable", "No pudimos confirmar el resumen de caja.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, out)
}
