package httpapi

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

type paymentView struct {
	ID               string `json:"id"`
	OrderID          string `json:"orderId"`
	OrderCode        string `json:"orderCode"`
	ShiftID          string `json:"shiftId"`
	CashRegisterName string `json:"cashRegisterName"`
	Method           string `json:"method"`
	MethodName       string `json:"methodName"`
	Amount           string `json:"amount"`
	RefundedAmount   string `json:"refundedAmount"`
	NetAmount        string `json:"netAmount"`
	Reference        string `json:"reference"`
	CreatedByName    string `json:"createdByName"`
	CreatedAt        string `json:"createdAt"`
}

type posOrderSummary struct {
	BillClosed      bool     `json:"billClosed"`
	ID              string   `json:"id"`
	Code            string   `json:"code"`
	Channel         string   `json:"channel"`
	Status          string   `json:"status"`
	CustomerName    string   `json:"customerName"`
	TableName       string   `json:"tableName"`
	Total           string   `json:"total"`
	PaidAmount      string   `json:"paidAmount"`
	RemainingAmount string   `json:"remainingAmount"`
	PaymentStatus   string   `json:"paymentStatus"`
	PaymentMethods  []string `json:"paymentMethods"`
	CreatedAt       string   `json:"createdAt"`
}

type posOrderDetail struct {
	ReceiptContext  paymentReceiptContext `json:"receiptContext"`
	Order           order                 `json:"order"`
	PaidAmount      string                `json:"paidAmount"`
	RemainingAmount string                `json:"remainingAmount"`
	PaymentStatus   string                `json:"paymentStatus"`
	Payments        []paymentView         `json:"payments"`
}

// Current merchant identity for an operational payment ticket, not a fiscal document.
type paymentReceiptContext struct {
	OrganizationName string `json:"organizationName"`
	LegalName        string `json:"legalName"`
	TaxID            string `json:"taxId"`
	LocationName     string `json:"locationName"`
	Address          string `json:"address"`
	Phone            string `json:"phone"`
	Country          string `json:"country"`
	Timezone         string `json:"timezone"`
	Currency         string `json:"currency"`
	CurrencySymbol   string `json:"currencySymbol"`
	CurrencyPosition string `json:"currencyPosition"`
	CurrencyDecimals int    `json:"currencyDecimals"`
}

const netPaidSQL = `
	COALESCE((
	  SELECT sum(
	    p.amount
	    - COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)
	  )
	  FROM payments p
	  WHERE p.order_id=o.id AND p.organization_id=o.organization_id AND p.location_id=o.location_id
	),0)
`

func paymentStatus(total, paid float64) string {
	if paid <= 0.00001 {
		return "pending"
	}
	if paid+0.00001 >= total {
		return "paid"
	}
	return "partial"
}

func (a *API) listPOSOrders(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	from := strings.TrimSpace(r.URL.Query().Get("from"))
	to := strings.TrimSpace(r.URL.Query().Get("to"))
	if !validDateFilters(from, to) || strings.HasPrefix(from, "0000-") || strings.HasPrefix(to, "0000-") {
		fail(w, 400, "invalid_order_date_filter", "Ingresa fechas válidas con el formato AAAA-MM-DD.")
		return
	}
	if from != "" && to != "" && from > to {
		fail(w, 400, "invalid_order_date_filter", "La fecha final no puede ser anterior a la inicial.")
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("paymentStatus"))
	if status == "" {
		status = "unpaid"
	}
	if status != "unpaid" && status != "pending" && status != "partial" && status != "paid" && status != "all" {
		fail(w, 400, "invalid_payment_status", "El estado de cobro no es válido.")
		return
	}

	filter := ""
	switch status {
	case "unpaid":
		filter = " AND payment_summary.net_paid < o.total"
	case "pending":
		filter = " AND payment_summary.net_paid <= 0.00001"
	case "partial":
		filter = " AND payment_summary.net_paid > 0.00001 AND payment_summary.net_paid < o.total"
	case "paid":
		filter = " AND payment_summary.net_paid >= o.total"
	}

	baseFrom := `
		FROM orders o
		JOIN locations loc ON loc.id=o.location_id AND loc.organization_id=o.organization_id
		LEFT JOIN tables t
		  ON t.id=o.table_id
		 AND t.organization_id=o.organization_id
		 AND t.location_id=o.location_id
		LEFT JOIN LATERAL (
		  SELECT COALESCE(sum(p.amount-COALESCE(refunds.refunded,0)),0) AS net_paid,
		         COALESCE(array_agg(DISTINCT pm.name ORDER BY pm.name)
		           FILTER (WHERE p.amount-COALESCE(refunds.refunded,0)>0 AND pm.name IS NOT NULL),
		           '{}'::text[]) AS payment_methods
		  FROM payments p
		  LEFT JOIN payment_methods pm
		    ON pm.organization_id=p.organization_id AND pm.code=p.method
		  LEFT JOIN LATERAL (
		    SELECT COALESCE(sum(pr.amount),0) AS refunded
		    FROM payment_refunds pr
		    WHERE pr.payment_id=p.id
		      AND pr.organization_id=p.organization_id
		      AND pr.location_id=p.location_id
		  ) refunds ON true
		  WHERE p.order_id=o.id
		    AND p.organization_id=o.organization_id
		    AND p.location_id=o.location_id
		) payment_summary ON true
	`
	where := `
		o.organization_id=$1 AND o.location_id=$2 AND o.status<>'cancelado'
		AND ($3='' OR o.code ILIKE '%'||$3||'%' OR o.customer_name ILIKE '%'||$3||'%'
		  OR COALESCE(t.name,'') ILIKE '%'||$3||'%')
		AND ($4='' OR o.created_at >= (NULLIF($4,'')::date::timestamp AT TIME ZONE loc.timezone))
		AND ($5='' OR o.created_at < ((NULLIF($5,'')::date + interval '1 day') AT TIME ZONE loc.timezone))
	` + filter

	var totalCount int
	if err := a.db.QueryRow(r.Context(), "SELECT count(*) "+baseFrom+" WHERE "+where, s.OrganizationID, s.LocationID, q, from, to).Scan(&totalCount); err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos cargar los pedidos por cobrar.")
		return
	}

	rows, err := a.db.Query(r.Context(), `
		SELECT o.id::text,o.code,o.channel,o.status,o.customer_name,
		       COALESCE(t.name,''),
		       o.total::text,
		       payment_summary.net_paid::text,
		       GREATEST(o.total-payment_summary.net_paid,0)::text,
		       CASE
		         WHEN payment_summary.net_paid <= 0.00001 THEN 'pending'
		         WHEN payment_summary.net_paid >= o.total THEN 'paid'
		         ELSE 'partial'
		       END,
		       to_char(o.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),
		       payment_summary.payment_methods,o.bill_closed_at IS NOT NULL
	`+baseFrom+`
		WHERE `+where+`
		ORDER BY o.created_at DESC,o.id DESC
		LIMIT $6 OFFSET $7
	`, s.OrganizationID, s.LocationID, q, from, to, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos cargar los pedidos por cobrar.")
		return
	}
	defer rows.Close()

	items := []posOrderSummary{}
	for rows.Next() {
		var item posOrderSummary
		if err := rows.Scan(
			&item.ID, &item.Code, &item.Channel, &item.Status, &item.CustomerName, &item.TableName,
			&item.Total, &item.PaidAmount, &item.RemainingAmount, &item.PaymentStatus, &item.CreatedAt,
			&item.PaymentMethods, &item.BillClosed,
		); err != nil {
			fail(w, 503, "payments_unavailable", "No pudimos leer los pedidos por cobrar.")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos cargar los pedidos por cobrar.")
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": totalCount, "page": page, "pageSize": size})
}

func (a *API) loadOrderPayments(r *http.Request, orderID string) ([]paymentView, error) {
	s := r.Context().Value(scopeKey{}).(scope)
	rows, err := a.db.Query(r.Context(), `
		SELECT p.id::text,p.order_id::text,o.code,p.shift_id::text,cr.name,p.method,pm.name,p.amount::text,
		       COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)::text,
		       (p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0))::text,
		       p.reference,u.full_name,to_char(p.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
		FROM payments p
		JOIN orders o ON o.id=p.order_id AND o.organization_id=p.organization_id
		JOIN cash_shifts cs ON cs.id=p.shift_id AND cs.organization_id=p.organization_id AND cs.location_id=p.location_id
		JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
		JOIN users u ON u.id=p.created_by AND u.organization_id=p.organization_id
		JOIN payment_methods pm ON pm.organization_id=p.organization_id AND pm.code=p.method
		WHERE p.organization_id=$1 AND p.location_id=$2 AND p.order_id=$3
		ORDER BY p.created_at DESC,p.id DESC
	`, s.OrganizationID, s.LocationID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []paymentView{}
	for rows.Next() {
		var item paymentView
		if err := rows.Scan(&item.ID, &item.OrderID, &item.OrderCode, &item.ShiftID, &item.CashRegisterName, &item.Method, &item.MethodName, &item.Amount, &item.RefundedAmount, &item.NetAmount, &item.Reference, &item.CreatedByName, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (a *API) getPOSOrder(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	o, err := scanOrder(a.db.QueryRow(r.Context(), `
		SELECT `+orderColumns+`
		FROM orders
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
	`, r.PathValue("id"), s.OrganizationID, s.LocationID))
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "order_not_found", "El pedido no existe en este local.")
		return
	}
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos cargar el pedido.")
		return
	}
	items, err := loadOrderItems(r.Context(), a.db, o.ID, s.OrganizationID)
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos cargar el detalle del pedido.")
		return
	}
	o.Items = items
	payments, err := a.loadOrderPayments(r, o.ID)
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos cargar los pagos del pedido.")
		return
	}
	paid := 0.0
	for _, p := range payments {
		value, _ := strconv.ParseFloat(p.NetAmount, 64)
		paid += value
	}
	total, _ := strconv.ParseFloat(o.Total, 64)
	remaining := math.Max(0, total-paid)
	var identity paymentReceiptContext
	err = a.db.QueryRow(r.Context(), `SELECT COALESCE(NULLIF(org.trade_name,''),org.legal_name),org.legal_name,org.tax_id,
 loc.name,loc.address,loc.phone,fp.country_code,loc.timezone,fp.currency,fp.currency_symbol,fp.currency_position,fp.currency_decimals
 FROM locations loc JOIN organizations org ON org.id=loc.organization_id
 JOIN organization_fiscal_profiles fp ON fp.id=loc.fiscal_profile_id AND fp.organization_id=loc.organization_id
 WHERE loc.id=$1 AND loc.organization_id=$2`, s.LocationID, s.OrganizationID).Scan(
		&identity.OrganizationName, &identity.LegalName, &identity.TaxID, &identity.LocationName, &identity.Address, &identity.Phone,
		&identity.Country, &identity.Timezone, &identity.Currency, &identity.CurrencySymbol, &identity.CurrencyPosition, &identity.CurrencyDecimals)
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos cargar los datos del restaurante.")
		return
	}
	writeJSON(w, 200, posOrderDetail{
		ReceiptContext:  identity,
		Order:           o,
		PaidAmount:      strconv.FormatFloat(paid, 'f', 2, 64),
		RemainingAmount: strconv.FormatFloat(remaining, 'f', 2, 64),
		PaymentStatus:   paymentStatus(total, paid),
		Payments:        payments,
	})
}

func (a *API) getPaymentByID(r *http.Request, id string) (paymentView, error) {
	s := r.Context().Value(scopeKey{}).(scope)
	var item paymentView
	err := a.db.QueryRow(r.Context(), `
		SELECT p.id::text,p.order_id::text,o.code,p.shift_id::text,cr.name,p.method,pm.name,p.amount::text,
		       COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)::text,
		       (p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0))::text,
		       p.reference,u.full_name,to_char(p.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
		FROM payments p
		JOIN orders o ON o.id=p.order_id AND o.organization_id=p.organization_id
		JOIN cash_shifts cs ON cs.id=p.shift_id AND cs.organization_id=p.organization_id AND cs.location_id=p.location_id
		JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
		JOIN users u ON u.id=p.created_by AND u.organization_id=p.organization_id
		JOIN payment_methods pm ON pm.organization_id=p.organization_id AND pm.code=p.method
		WHERE p.id=$1 AND p.organization_id=$2 AND p.location_id=$3
	`, id, s.OrganizationID, s.LocationID).Scan(
		&item.ID, &item.OrderID, &item.OrderCode, &item.ShiftID, &item.CashRegisterName, &item.Method, &item.MethodName, &item.Amount,
		&item.RefundedAmount, &item.NetAmount, &item.Reference, &item.CreatedByName, &item.CreatedAt,
	)
	return item, err
}

func (a *API) createPayment(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		OrderID   string  `json:"orderId"`
		Method    string  `json:"method"`
		Amount    float64 `json:"amount"`
		Reference string  `json:"reference"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.OrderID) == "" || in.Amount <= 0 {
		fail(w, 400, "invalid_payment", "Revisa los datos del cobro.")
		return
	}
	in.OrderID = strings.TrimSpace(in.OrderID)
	in.Reference = strings.TrimSpace(in.Reference)
	if len(in.Reference) > 120 {
		fail(w, 400, "invalid_payment", "La referencia no puede superar 120 caracteres.")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos iniciar el cobro.")
		return
	}
	defer tx.Rollback(r.Context())

	methodDef, err := getPaymentMethod(r.Context(), tx, s.OrganizationID, in.Method, "sales")
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 400, "invalid_payment_method", "Selecciona un medio de pago activo.")
		return
	}
	if err != nil {
		fail(w, 503, "payment_methods_unavailable", "No pudimos validar el medio de pago.")
		return
	}

	shiftID, err := currentCashShiftID(r.Context(), tx, s)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 409, "cash_shift_required", "Debes estar asignado a un turno de caja abierto para cobrar.")
		return
	}
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos validar tu turno de caja.")
		return
	}

	var orderCode, status string
	var completed, salon, billClosed bool
	var total, paid float64
	err = tx.QueryRow(r.Context(), `
		SELECT o.code,o.status,o.total::float8,o.completed_at IS NOT NULL,o.channel='salon',o.bill_closed_at IS NOT NULL
		FROM orders o
		WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3
		FOR UPDATE
	`, in.OrderID, s.OrganizationID, s.LocationID).Scan(&orderCode, &status, &total, &completed, &salon, &billClosed)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "order_not_found", "El pedido no existe en este local.")
		return
	}
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos validar el pedido.")
		return
	}
	if err = tx.QueryRow(r.Context(), `
		SELECT COALESCE(sum(
		  p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)
		),0)::float8
		FROM payments p
		WHERE p.order_id=$1 AND p.organization_id=$2 AND p.location_id=$3
	`, in.OrderID, s.OrganizationID, s.LocationID).Scan(&paid); err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos validar el saldo pendiente.")
		return
	}
	if completed || status == "cancelado" || (salon && !billClosed) || (!salon && status != "listo" && status != "en_camino" && status != "entregado") {
		fail(w, 409, "order_not_ready_for_payment", "Cierra la cuenta de la mesa antes de cobrar. El pedido debe seguir abierto.")
		return
	}
	remaining := math.Max(0, total-paid)
	if salon && !requireTableProductsDelivered(w, r, tx, s, in.OrderID, status) {
		return
	}
	if in.Amount > remaining+0.00001 {
		fail(w, 409, "payment_exceeds_remaining", "El cobro supera el saldo pendiente del pedido.")
		return
	}

	var id string
	err = tx.QueryRow(r.Context(), `
		INSERT INTO payments(organization_id,location_id,order_id,shift_id,method,amount,reference,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id::text
	`, s.OrganizationID, s.LocationID, in.OrderID, shiftID, in.Method, in.Amount, in.Reference, s.UserID).Scan(&id)
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos registrar el cobro.")
		return
	}

	if methodDef.AffectsCash {
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO cash_movements(
			  organization_id,location_id,shift_id,movement_type,source_type,source_id,
			  amount,reason,note,created_by
			)
			VALUES($1,$2,$3,'income','cash_sale',$4::uuid,$5,$6,$7,$8)
		`, s.OrganizationID, s.LocationID, shiftID, id, in.Amount, "Cobro "+orderCode, in.Reference, s.UserID); err != nil {
			fail(w, 503, "payments_unavailable", "No pudimos reflejar el cobro en caja.")
			return
		}
	}
	closed, err := completeDeliveredSalonOrder(r.Context(), tx, s, in.OrderID)
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos finalizar la cuenta del pedido.")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos confirmar el cobro.")
		return
	}
	if closed {
		a.audit(r, "order.completed", "order", in.OrderID)
	}

	item, err := a.getPaymentByID(r, id)
	if err != nil {
		fail(w, 503, "payments_unavailable", "El cobro se registró, pero no pudimos cargar su detalle.")
		return
	}
	writeJSON(w, 201, item)
}

func (a *API) refundPayment(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		Amount float64 `json:"amount"`
		Reason string  `json:"reason"`
		Note   string  `json:"note"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.Amount <= 0 {
		fail(w, 400, "invalid_refund", "Ingresa un monto válido para la devolución.")
		return
	}
	in.Reason = strings.TrimSpace(in.Reason)
	in.Note = strings.TrimSpace(in.Note)
	if in.Reason == "" || len(in.Reason) > 120 || len(in.Note) > 240 {
		fail(w, 400, "invalid_refund", "Indica el motivo de la devolución.")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos iniciar la devolución.")
		return
	}
	defer tx.Rollback(r.Context())

	shiftID, err := currentCashShiftID(r.Context(), tx, s)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 409, "cash_shift_required", "Debes estar asignado a un turno de caja abierto para devolver un pago.")
		return
	}
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos validar tu turno de caja.")
		return
	}

	var affectsCash bool
	var orderCode, orderStatus string
	var completed, salon bool
	var amount, refunded float64
	err = tx.QueryRow(r.Context(), `
		SELECT pm.affects_cash,o.code,o.status,p.amount::float8,o.completed_at IS NOT NULL,o.channel='salon',
		       COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)::float8
		FROM payments p
		JOIN payment_methods pm ON pm.organization_id=p.organization_id AND pm.code=p.method
		JOIN orders o ON o.id=p.order_id AND o.organization_id=p.organization_id AND o.location_id=p.location_id
		WHERE p.id=$1 AND p.organization_id=$2 AND p.location_id=$3
		FOR UPDATE
	`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&affectsCash, &orderCode, &orderStatus, &amount, &completed, &salon, &refunded)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "payment_not_found", "El pago no existe en este local.")
		return
	}
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos validar el pago.")
		return
	}
	if completed || (orderStatus == "entregado" && !salon) || orderStatus == "cancelado" {
		fail(w, 409, "closed_order_refund_requires_void", "El pedido ya está cerrado. Usa un flujo de anulación o devolución posterior al cierre para no reabrir su saldo operativo.")
		return
	}
	if in.Amount > amount-refunded+0.00001 {
		fail(w, 409, "refund_exceeds_payment", "La devolución supera el saldo disponible del pago.")
		return
	}
	if affectsCash {
		expected, err := cashShiftExpected(r.Context(), tx, s, shiftID)
		if err != nil {
			fail(w, 503, "payments_unavailable", "No pudimos validar el efectivo disponible.")
			return
		}
		if in.Amount > expected+0.00001 {
			fail(w, 409, "refund_exceeds_expected", "La devolución supera el efectivo esperado del turno.")
			return
		}
	}

	var refundID string
	err = tx.QueryRow(r.Context(), `
		INSERT INTO payment_refunds(organization_id,location_id,payment_id,shift_id,amount,reason,note,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id::text
	`, s.OrganizationID, s.LocationID, r.PathValue("id"), shiftID, in.Amount, in.Reason, in.Note, s.UserID).Scan(&refundID)
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos registrar la devolución.")
		return
	}

	if affectsCash {
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO cash_movements(
			  organization_id,location_id,shift_id,movement_type,source_type,source_id,
			  amount,reason,note,created_by
			)
			VALUES($1,$2,$3,'expense','cash_refund',$4::uuid,$5,$6,$7,$8)
		`, s.OrganizationID, s.LocationID, shiftID, refundID, in.Amount, "Devolución "+orderCode, in.Note, s.UserID); err != nil {
			fail(w, 503, "payments_unavailable", "No pudimos reflejar la devolución en caja.")
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos confirmar la devolución.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) createPaymentBatch(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		OrderID  string `json:"orderId"`
		Payments []struct {
			Method    string  `json:"method"`
			Amount    float64 `json:"amount"`
			Reference string  `json:"reference"`
		} `json:"payments"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.OrderID) == "" || len(in.Payments) == 0 {
		fail(w, 400, "invalid_payment", "Revisa los datos del cobro.")
		return
	}
	in.OrderID = strings.TrimSpace(in.OrderID)
	totalBatch := 0.0
	for i := range in.Payments {
		p := &in.Payments[i]
		p.Method = strings.TrimSpace(p.Method)
		p.Reference = strings.TrimSpace(p.Reference)
		if p.Amount <= 0 || p.Method == "" {
			fail(w, 400, "invalid_payment", "Todos los medios de pago deben tener método y monto válidos.")
			return
		}
		if len(p.Reference) > 120 {
			fail(w, 400, "invalid_payment", "La referencia no puede superar 120 caracteres.")
			return
		}
		totalBatch += p.Amount
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos iniciar el cobro.")
		return
	}
	defer tx.Rollback(r.Context())

	methodDefs := map[string]paymentMethodView{}
	for _, p := range in.Payments {
		if _, ok := methodDefs[p.Method]; ok {
			continue
		}
		methodDef, lookupErr := getPaymentMethod(r.Context(), tx, s.OrganizationID, p.Method, "sales")
		if errors.Is(lookupErr, pgx.ErrNoRows) {
			fail(w, 400, "invalid_payment_method", "Uno de los medios de pago no está activo para ventas.")
			return
		}
		if lookupErr != nil {
			fail(w, 503, "payment_methods_unavailable", "No pudimos validar los medios de pago.")
			return
		}
		methodDefs[p.Method] = methodDef
	}

	shiftID, err := currentCashShiftID(r.Context(), tx, s)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 409, "cash_shift_required", "Debes estar asignado a un turno de caja abierto para cobrar.")
		return
	}
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos validar tu turno de caja.")
		return
	}

	var orderCode, status string
	var completed, salon, billClosed bool
	var orderTotal, paid float64
	err = tx.QueryRow(r.Context(), `
		SELECT o.code,o.status,o.total::float8,o.completed_at IS NOT NULL,o.channel='salon',o.bill_closed_at IS NOT NULL
		FROM orders o
		WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3
		FOR UPDATE
	`, in.OrderID, s.OrganizationID, s.LocationID).Scan(&orderCode, &status, &orderTotal, &completed, &salon, &billClosed)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "order_not_found", "El pedido no existe en este local.")
		return
	}
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos validar el pedido.")
		return
	}
	if err = tx.QueryRow(r.Context(), `
		SELECT COALESCE(sum(
		  p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)
		),0)::float8
		FROM payments p
		WHERE p.order_id=$1 AND p.organization_id=$2 AND p.location_id=$3
	`, in.OrderID, s.OrganizationID, s.LocationID).Scan(&paid); err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos validar el saldo pendiente.")
		return
	}
	if completed || status == "cancelado" || (salon && !billClosed) || (!salon && status != "listo" && status != "en_camino" && status != "entregado") {
		fail(w, 409, "order_not_ready_for_payment", "Cierra la cuenta de la mesa antes de cobrar. El pedido debe seguir abierto.")
		return
	}
	remaining := math.Max(0, orderTotal-paid)
	if salon && !requireTableProductsDelivered(w, r, tx, s, in.OrderID, status) {
		return
	}
	if totalBatch > remaining+0.00001 {
		fail(w, 409, "payment_exceeds_remaining", "El cobro supera el saldo pendiente del pedido.")
		return
	}

	ids := make([]string, 0, len(in.Payments))
	for _, p := range in.Payments {
		var id string
		err = tx.QueryRow(r.Context(), `
			INSERT INTO payments(organization_id,location_id,order_id,shift_id,method,amount,reference,created_by)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8)
			RETURNING id::text
		`, s.OrganizationID, s.LocationID, in.OrderID, shiftID, p.Method, p.Amount, p.Reference, s.UserID).Scan(&id)
		if err != nil {
			fail(w, 503, "payments_unavailable", "No pudimos registrar uno de los medios de pago.")
			return
		}
		ids = append(ids, id)
		if methodDefs[p.Method].AffectsCash {
			if _, err = tx.Exec(r.Context(), `
				INSERT INTO cash_movements(
				  organization_id,location_id,shift_id,movement_type,source_type,source_id,
				  amount,reason,note,created_by
				)
				VALUES($1,$2,$3,'income','cash_sale',$4::uuid,$5,$6,$7,$8)
			`, s.OrganizationID, s.LocationID, shiftID, id, p.Amount, "Cobro "+orderCode, p.Reference, s.UserID); err != nil {
				fail(w, 503, "payments_unavailable", "No pudimos reflejar el cobro en caja.")
				return
			}
		}
	}
	closed, err := completeDeliveredSalonOrder(r.Context(), tx, s, in.OrderID)
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos finalizar la cuenta del pedido.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos confirmar el cobro.")
		return
	}
	if closed {
		a.audit(r, "order.completed", "order", in.OrderID)
	}
	writeJSON(w, 201, map[string]any{
		"paymentIds":      ids,
		"paidNow":         strconv.FormatFloat(totalBatch, 'f', 2, 64),
		"remainingAmount": strconv.FormatFloat(math.Max(0, remaining-totalBatch), 'f', 2, 64),
		"paymentStatus":   paymentStatus(orderTotal, paid+totalBatch),
	})
}

func (a *API) completePaidOrder(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos finalizar el pedido.")
		return
	}
	defer tx.Rollback(r.Context())
	var total, paid float64
	var status string
	var salon bool
	err = tx.QueryRow(r.Context(), `
		SELECT o.total::float8,o.status,o.channel='salon'
		FROM orders o
		WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3
		FOR UPDATE
	`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&total, &status, &salon)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "order_not_found", "El pedido no existe en este local.")
		return
	}
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos validar el pedido.")
		return
	}
	if err = tx.QueryRow(r.Context(), `
		SELECT COALESCE(sum(
		  p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)
		),0)::float8
		FROM payments p
		WHERE p.order_id=$1 AND p.organization_id=$2 AND p.location_id=$3
	`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&paid); err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos validar el saldo del pedido.")
		return
	}
	if status == "cancelado" {
		fail(w, 409, "order_not_completable", "Un pedido cancelado no puede finalizarse.")
		return
	}
	if paid+0.00001 < total {
		fail(w, 409, "payment_incomplete", "El pedido todavía tiene saldo pendiente.")
		return
	}
	if salon {
		var closed bool
		closed, err = completeDeliveredSalonOrder(r.Context(), tx, s, r.PathValue("id"))
		if err != nil {
			fail(w, 503, "payments_unavailable", "No pudimos finalizar la cuenta.")
			return
		}
		if !closed {
			fail(w, 409, "delivery_required", "Cierra la cuenta y completa las entregas antes de finalizar.")
			return
		}
		if err = tx.Commit(r.Context()); err != nil {
			fail(w, 503, "payments_unavailable", "No pudimos finalizar.")
			return
		}
		a.audit(r, "order.completed", "order", r.PathValue("id"))
		w.WriteHeader(204)
		return
	}
	if status != "listo" && status != "en_camino" {
		fail(w, 409, "order_not_ready", "El pago está completo, pero la comanda todavía no está lista para entregarse.")
		return
	}
	var pending bool
	if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM order_service_items WHERE order_id=$1 AND organization_id=$2 AND status NOT IN ('listo','entregado'))`, r.PathValue("id"), s.OrganizationID).Scan(&pending); err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos validar la entrega.")
		return
	}
	if pending {
		fail(w, 409, "items_not_ready", "Todavía hay productos en preparación.")
		return
	}
	if _, err = tx.Exec(r.Context(), `UPDATE order_service_items SET status='entregado',delivered_at=now(),delivered_by=(SELECT id FROM users WHERE id=$3 AND organization_id=$2),updated_at=now() WHERE order_id=$1 AND organization_id=$2 AND status='listo'`, r.PathValue("id"), s.OrganizationID, s.UserID); err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos registrar la entrega.")
		return
	}
	if _, err = tx.Exec(r.Context(), `
		UPDATE orders
		SET status='entregado',updated_at=now(),completed_at=COALESCE(completed_at,now())
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
	`, r.PathValue("id"), s.OrganizationID, s.LocationID); err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos liberar la mesa.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos finalizar el pedido.")
		return
	}
	a.audit(r, "order.completed", "order", r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}
