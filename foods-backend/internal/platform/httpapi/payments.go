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
	Amount           string `json:"amount"`
	RefundedAmount   string `json:"refundedAmount"`
	NetAmount        string `json:"netAmount"`
	Reference        string `json:"reference"`
	CreatedByName    string `json:"createdByName"`
	CreatedAt        string `json:"createdAt"`
}

type posOrderSummary struct {
	ID              string `json:"id"`
	Code            string `json:"code"`
	Channel         string `json:"channel"`
	Status          string `json:"status"`
	CustomerName    string `json:"customerName"`
	TableName       string `json:"tableName"`
	Total           string `json:"total"`
	PaidAmount      string `json:"paidAmount"`
	RemainingAmount string `json:"remainingAmount"`
	PaymentStatus   string `json:"paymentStatus"`
	CreatedAt       string `json:"createdAt"`
}

type posOrderDetail struct {
	Order           order         `json:"order"`
	PaidAmount      string        `json:"paidAmount"`
	RemainingAmount string        `json:"remainingAmount"`
	PaymentStatus   string        `json:"paymentStatus"`
	Payments        []paymentView `json:"payments"`
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
		filter = " AND (" + netPaidSQL + ") < o.total"
	case "pending":
		filter = " AND (" + netPaidSQL + ") <= 0.00001"
	case "partial":
		filter = " AND (" + netPaidSQL + ") > 0.00001 AND (" + netPaidSQL + ") < o.total"
	case "paid":
		filter = " AND (" + netPaidSQL + ") >= o.total"
	}

	where := `
		o.organization_id=$1 AND o.location_id=$2 AND o.status<>'cancelado'
		AND ($3='' OR o.code ILIKE '%'||$3||'%' OR o.customer_name ILIKE '%'||$3||'%'
		  OR COALESCE((SELECT name FROM tables t WHERE t.id=o.table_id),'') ILIKE '%'||$3||'%')
	` + filter

	var totalCount int
	if err := a.db.QueryRow(r.Context(), "SELECT count(*) FROM orders o WHERE "+where, s.OrganizationID, s.LocationID, q).Scan(&totalCount); err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos cargar los pedidos por cobrar.")
		return
	}

	rows, err := a.db.Query(r.Context(), `
		SELECT o.id::text,o.code,o.channel,o.status,o.customer_name,
		       COALESCE((SELECT name FROM tables t WHERE t.id=o.table_id),''),
		       o.total::text,
		       (`+netPaidSQL+`)::text,
		       GREATEST(o.total-(`+netPaidSQL+`),0)::text,
		       CASE
		         WHEN (`+netPaidSQL+`) <= 0.00001 THEN 'pending'
		         WHEN (`+netPaidSQL+`) >= o.total THEN 'paid'
		         ELSE 'partial'
		       END,
		       to_char(o.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
		FROM orders o
		WHERE `+where+`
		ORDER BY o.created_at DESC,o.id DESC
		LIMIT $4 OFFSET $5
	`, s.OrganizationID, s.LocationID, q, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "payments_unavailable", "No pudimos cargar los pedidos por cobrar.")
		return
	}
	defer rows.Close()

	items := []posOrderSummary{}
	for rows.Next() {
		var item posOrderSummary
		if err := rows.Scan(
			&item.ID,&item.Code,&item.Channel,&item.Status,&item.CustomerName,&item.TableName,
			&item.Total,&item.PaidAmount,&item.RemainingAmount,&item.PaymentStatus,&item.CreatedAt,
		); err != nil {
			fail(w, 503, "payments_unavailable", "No pudimos leer los pedidos por cobrar.")
			return
		}
		items = append(items,item)
	}
	writeJSON(w,200,map[string]any{"items":items,"total":totalCount,"page":page,"pageSize":size})
}

func (a *API) loadOrderPayments(r *http.Request, orderID string) ([]paymentView,error) {
	s := r.Context().Value(scopeKey{}).(scope)
	rows,err:=a.db.Query(r.Context(),`
		SELECT p.id::text,p.order_id::text,o.code,p.shift_id::text,cr.name,p.method,p.amount::text,
		       COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)::text,
		       (p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0))::text,
		       p.reference,u.full_name,to_char(p.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
		FROM payments p
		JOIN orders o ON o.id=p.order_id AND o.organization_id=p.organization_id
		JOIN cash_shifts cs ON cs.id=p.shift_id AND cs.organization_id=p.organization_id AND cs.location_id=p.location_id
		JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
		JOIN users u ON u.id=p.created_by AND u.organization_id=p.organization_id
		WHERE p.organization_id=$1 AND p.location_id=$2 AND p.order_id=$3
		ORDER BY p.created_at DESC,p.id DESC
	`,s.OrganizationID,s.LocationID,orderID)
	if err!=nil{return nil,err}
	defer rows.Close()
	items:=[]paymentView{}
	for rows.Next(){
		var item paymentView
		if err:=rows.Scan(&item.ID,&item.OrderID,&item.OrderCode,&item.ShiftID,&item.CashRegisterName,&item.Method,&item.Amount,&item.RefundedAmount,&item.NetAmount,&item.Reference,&item.CreatedByName,&item.CreatedAt);err!=nil{return nil,err}
		items=append(items,item)
	}
	return items,rows.Err()
}

func (a *API) getPOSOrder(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	o,err:=scanOrder(a.db.QueryRow(r.Context(),`
		SELECT `+orderColumns+`
		FROM orders
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
	`,r.PathValue("id"),s.OrganizationID,s.LocationID))
	if errors.Is(err,pgx.ErrNoRows){
		fail(w,404,"order_not_found","El pedido no existe en este local.")
		return
	}
	if err!=nil{
		fail(w,503,"payments_unavailable","No pudimos cargar el pedido.")
		return
	}
	items,err:=loadOrderItems(r.Context(),a.db,o.ID,s.OrganizationID)
	if err!=nil{
		fail(w,503,"payments_unavailable","No pudimos cargar el detalle del pedido.")
		return
	}
	o.Items=items
	payments,err:=a.loadOrderPayments(r,o.ID)
	if err!=nil{
		fail(w,503,"payments_unavailable","No pudimos cargar los pagos del pedido.")
		return
	}
	paid:=0.0
	for _,p:=range payments{
		value,_:=strconv.ParseFloat(p.NetAmount,64)
		paid+=value
	}
	total,_:=strconv.ParseFloat(o.Total,64)
	remaining:=math.Max(0,total-paid)
	writeJSON(w,200,posOrderDetail{
		Order:o,
		PaidAmount:strconv.FormatFloat(paid,'f',2,64),
		RemainingAmount:strconv.FormatFloat(remaining,'f',2,64),
		PaymentStatus:paymentStatus(total,paid),
		Payments:payments,
	})
}

func (a *API) getPaymentByID(r *http.Request,id string)(paymentView,error){
	s:=r.Context().Value(scopeKey{}).(scope)
	var item paymentView
	err:=a.db.QueryRow(r.Context(),`
		SELECT p.id::text,p.order_id::text,o.code,p.shift_id::text,cr.name,p.method,p.amount::text,
		       COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)::text,
		       (p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0))::text,
		       p.reference,u.full_name,to_char(p.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
		FROM payments p
		JOIN orders o ON o.id=p.order_id AND o.organization_id=p.organization_id
		JOIN cash_shifts cs ON cs.id=p.shift_id AND cs.organization_id=p.organization_id AND cs.location_id=p.location_id
		JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
		JOIN users u ON u.id=p.created_by AND u.organization_id=p.organization_id
		WHERE p.id=$1 AND p.organization_id=$2 AND p.location_id=$3
	`,id,s.OrganizationID,s.LocationID).Scan(
		&item.ID,&item.OrderID,&item.OrderCode,&item.ShiftID,&item.CashRegisterName,&item.Method,&item.Amount,
		&item.RefundedAmount,&item.NetAmount,&item.Reference,&item.CreatedByName,&item.CreatedAt,
	)
	return item,err
}

func (a *API) createPayment(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	var in struct{
		OrderID string `json:"orderId"`
		Method string `json:"method"`
		Amount float64 `json:"amount"`
		Reference string `json:"reference"`
	}
	if json.NewDecoder(r.Body).Decode(&in)!=nil||strings.TrimSpace(in.OrderID)==""||in.Amount<=0{
		fail(w,400,"invalid_payment","Revisa los datos del cobro.")
		return
	}
	if in.Method!="cash"&&in.Method!="card"&&in.Method!="transfer"&&in.Method!="other"{
		fail(w,400,"invalid_payment_method","Selecciona un método de pago válido.")
		return
	}
	in.OrderID=strings.TrimSpace(in.OrderID)
	in.Reference=strings.TrimSpace(in.Reference)
	if len(in.Reference)>120{
		fail(w,400,"invalid_payment","La referencia no puede superar 120 caracteres.")
		return
	}

	tx,err:=a.db.Begin(r.Context())
	if err!=nil{fail(w,503,"payments_unavailable","No pudimos iniciar el cobro.");return}
	defer tx.Rollback(r.Context())

	shiftID,err:=currentCashShiftID(r.Context(),tx,s)
	if errors.Is(err,pgx.ErrNoRows){
		fail(w,409,"cash_shift_required","Debes estar asignado a un turno de caja abierto para cobrar.")
		return
	}
	if err!=nil{fail(w,503,"payments_unavailable","No pudimos validar tu turno de caja.");return}

	var orderCode,status string
	var total,paid float64
	err=tx.QueryRow(r.Context(),`
		SELECT o.code,o.status,o.total::float8,(`+netPaidSQL+`)::float8
		FROM orders o
		WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3
		FOR UPDATE
	`,in.OrderID,s.OrganizationID,s.LocationID).Scan(&orderCode,&status,&total,&paid)
	if errors.Is(err,pgx.ErrNoRows){fail(w,404,"order_not_found","El pedido no existe en este local.");return}
	if err!=nil{fail(w,503,"payments_unavailable","No pudimos validar el pedido.");return}
	if status=="cancelado"{fail(w,409,"order_not_payable","Un pedido cancelado no admite cobros.");return}
	remaining:=math.Max(0,total-paid)
	if in.Amount>remaining+0.00001{
		fail(w,409,"payment_exceeds_remaining","El cobro supera el saldo pendiente del pedido.")
		return
	}

	var id string
	err=tx.QueryRow(r.Context(),`
		INSERT INTO payments(organization_id,location_id,order_id,shift_id,method,amount,reference,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id::text
	`,s.OrganizationID,s.LocationID,in.OrderID,shiftID,in.Method,in.Amount,in.Reference,s.UserID).Scan(&id)
	if err!=nil{fail(w,503,"payments_unavailable","No pudimos registrar el cobro.");return}

	if in.Method=="cash"{
		if _,err:=tx.Exec(r.Context(),`
			INSERT INTO cash_movements(
			  organization_id,location_id,shift_id,movement_type,source_type,source_id,
			  amount,reason,note,created_by
			)
			VALUES($1,$2,$3,'income','cash_sale',$4::uuid,$5,$6,$7,$8)
		`,s.OrganizationID,s.LocationID,shiftID,id,in.Amount,"Cobro "+orderCode,in.Reference,s.UserID);err!=nil{
			fail(w,503,"payments_unavailable","No pudimos reflejar el cobro en caja.")
			return
		}
	}
	if err:=tx.Commit(r.Context());err!=nil{fail(w,503,"payments_unavailable","No pudimos confirmar el cobro.");return}

	item,err:=a.getPaymentByID(r,id)
	if err!=nil{fail(w,503,"payments_unavailable","El cobro se registró, pero no pudimos cargar su detalle.");return}
	writeJSON(w,201,item)
}

func (a *API) refundPayment(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	var in struct{
		Amount float64 `json:"amount"`
		Reason string `json:"reason"`
		Note string `json:"note"`
	}
	if json.NewDecoder(r.Body).Decode(&in)!=nil||in.Amount<=0{
		fail(w,400,"invalid_refund","Ingresa un monto válido para la devolución.")
		return
	}
	in.Reason=strings.TrimSpace(in.Reason)
	in.Note=strings.TrimSpace(in.Note)
	if in.Reason==""||len(in.Reason)>120||len(in.Note)>240{
		fail(w,400,"invalid_refund","Indica el motivo de la devolución.")
		return
	}

	tx,err:=a.db.Begin(r.Context())
	if err!=nil{fail(w,503,"payments_unavailable","No pudimos iniciar la devolución.");return}
	defer tx.Rollback(r.Context())

	shiftID,err:=currentCashShiftID(r.Context(),tx,s)
	if errors.Is(err,pgx.ErrNoRows){
		fail(w,409,"cash_shift_required","Debes estar asignado a un turno de caja abierto para devolver un pago.")
		return
	}
	if err!=nil{fail(w,503,"payments_unavailable","No pudimos validar tu turno de caja.");return}

	var method,orderCode string
	var amount,refunded float64
	err=tx.QueryRow(r.Context(),`
		SELECT p.method,o.code,p.amount::float8,
		       COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)::float8
		FROM payments p
		JOIN orders o ON o.id=p.order_id AND o.organization_id=p.organization_id
		WHERE p.id=$1 AND p.organization_id=$2 AND p.location_id=$3
		FOR UPDATE
	`,r.PathValue("id"),s.OrganizationID,s.LocationID).Scan(&method,&orderCode,&amount,&refunded)
	if errors.Is(err,pgx.ErrNoRows){fail(w,404,"payment_not_found","El pago no existe en este local.");return}
	if err!=nil{fail(w,503,"payments_unavailable","No pudimos validar el pago.");return}
	if in.Amount>amount-refunded+0.00001{
		fail(w,409,"refund_exceeds_payment","La devolución supera el saldo disponible del pago.")
		return
	}
	if method=="cash"{
		expected,err:=cashShiftExpected(r.Context(),tx,s,shiftID)
		if err!=nil{fail(w,503,"payments_unavailable","No pudimos validar el efectivo disponible.");return}
		if in.Amount>expected+0.00001{
			fail(w,409,"refund_exceeds_expected","La devolución supera el efectivo esperado del turno.")
			return
		}
	}

	var refundID string
	err=tx.QueryRow(r.Context(),`
		INSERT INTO payment_refunds(organization_id,location_id,payment_id,shift_id,amount,reason,note,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id::text
	`,s.OrganizationID,s.LocationID,r.PathValue("id"),shiftID,in.Amount,in.Reason,in.Note,s.UserID).Scan(&refundID)
	if err!=nil{fail(w,503,"payments_unavailable","No pudimos registrar la devolución.");return}

	if method=="cash"{
		if _,err:=tx.Exec(r.Context(),`
			INSERT INTO cash_movements(
			  organization_id,location_id,shift_id,movement_type,source_type,source_id,
			  amount,reason,note,created_by
			)
			VALUES($1,$2,$3,'expense','cash_refund',$4::uuid,$5,$6,$7,$8)
		`,s.OrganizationID,s.LocationID,shiftID,refundID,in.Amount,"Devolución "+orderCode,in.Note,s.UserID);err!=nil{
			fail(w,503,"payments_unavailable","No pudimos reflejar la devolución en caja.")
			return
		}
	}
	if err:=tx.Commit(r.Context());err!=nil{fail(w,503,"payments_unavailable","No pudimos confirmar la devolución.");return}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) createPaymentBatch(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	var in struct{
		OrderID string `json:"orderId"`
		Payments []struct{
			Method string `json:"method"`
			Amount float64 `json:"amount"`
			Reference string `json:"reference"`
		} `json:"payments"`
	}
	if json.NewDecoder(r.Body).Decode(&in)!=nil||strings.TrimSpace(in.OrderID)==""||len(in.Payments)==0{
		fail(w,400,"invalid_payment","Revisa los datos del cobro.")
		return
	}
	in.OrderID=strings.TrimSpace(in.OrderID)
	totalBatch:=0.0
	for i:=range in.Payments{
		p:=&in.Payments[i]
		p.Method=strings.TrimSpace(p.Method)
		p.Reference=strings.TrimSpace(p.Reference)
		if p.Amount<=0||(p.Method!="cash"&&p.Method!="card"&&p.Method!="transfer"&&p.Method!="other"){
			fail(w,400,"invalid_payment","Todos los medios de pago deben tener método y monto válidos.")
			return
		}
		if len(p.Reference)>120{
			fail(w,400,"invalid_payment","La referencia no puede superar 120 caracteres.")
			return
		}
		totalBatch+=p.Amount
	}
	tx,err:=a.db.Begin(r.Context())
	if err!=nil{fail(w,503,"payments_unavailable","No pudimos iniciar el cobro.");return}
	defer tx.Rollback(r.Context())

	shiftID,err:=currentCashShiftID(r.Context(),tx,s)
	if errors.Is(err,pgx.ErrNoRows){fail(w,409,"cash_shift_required","Debes estar asignado a un turno de caja abierto para cobrar.");return}
	if err!=nil{fail(w,503,"payments_unavailable","No pudimos validar tu turno de caja.");return}

	var orderCode,status string
	var orderTotal,paid float64
	err=tx.QueryRow(r.Context(),`
		SELECT o.code,o.status,o.total::float8,(`+netPaidSQL+`)::float8
		FROM orders o
		WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3
		FOR UPDATE
	`,in.OrderID,s.OrganizationID,s.LocationID).Scan(&orderCode,&status,&orderTotal,&paid)
	if errors.Is(err,pgx.ErrNoRows){fail(w,404,"order_not_found","El pedido no existe en este local.");return}
	if err!=nil{fail(w,503,"payments_unavailable","No pudimos validar el pedido.");return}
	if status=="cancelado"{fail(w,409,"order_not_payable","Un pedido cancelado no admite cobros.");return}
	remaining:=math.Max(0,orderTotal-paid)
	if totalBatch>remaining+0.00001{
		fail(w,409,"payment_exceeds_remaining","El cobro supera el saldo pendiente del pedido.")
		return
	}

	ids:=make([]string,0,len(in.Payments))
	for _,p:=range in.Payments{
		var id string
		err=tx.QueryRow(r.Context(),`
			INSERT INTO payments(organization_id,location_id,order_id,shift_id,method,amount,reference,created_by)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8)
			RETURNING id::text
		`,s.OrganizationID,s.LocationID,in.OrderID,shiftID,p.Method,p.Amount,p.Reference,s.UserID).Scan(&id)
		if err!=nil{fail(w,503,"payments_unavailable","No pudimos registrar uno de los medios de pago.");return}
		ids=append(ids,id)
		if p.Method=="cash"{
			if _,err=tx.Exec(r.Context(),`
				INSERT INTO cash_movements(
				  organization_id,location_id,shift_id,movement_type,source_type,source_id,
				  amount,reason,note,created_by
				)
				VALUES($1,$2,$3,'income','cash_sale',$4::uuid,$5,$6,$7,$8)
			`,s.OrganizationID,s.LocationID,shiftID,id,p.Amount,"Cobro "+orderCode,p.Reference,s.UserID);err!=nil{
				fail(w,503,"payments_unavailable","No pudimos reflejar el cobro en caja.")
				return
			}
		}
	}
	if err=tx.Commit(r.Context());err!=nil{fail(w,503,"payments_unavailable","No pudimos confirmar el cobro.");return}
	writeJSON(w,201,map[string]any{
		"paymentIds":ids,
		"paidNow":strconv.FormatFloat(totalBatch,'f',2,64),
		"remainingAmount":strconv.FormatFloat(math.Max(0,remaining-totalBatch),'f',2,64),
		"paymentStatus":paymentStatus(orderTotal,paid+totalBatch),
	})
}

func (a *API) completePaidOrder(w http.ResponseWriter,r *http.Request){
	s:=r.Context().Value(scopeKey{}).(scope)
	tx,err:=a.db.Begin(r.Context())
	if err!=nil{fail(w,503,"payments_unavailable","No pudimos finalizar el pedido.");return}
	defer tx.Rollback(r.Context())
	var total,paid float64
	var status string
	err=tx.QueryRow(r.Context(),`
		SELECT o.total::float8,(`+netPaidSQL+`)::float8,o.status
		FROM orders o
		WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3
		FOR UPDATE
	`,r.PathValue("id"),s.OrganizationID,s.LocationID).Scan(&total,&paid,&status)
	if errors.Is(err,pgx.ErrNoRows){fail(w,404,"order_not_found","El pedido no existe en este local.");return}
	if err!=nil{fail(w,503,"payments_unavailable","No pudimos validar el pedido.");return}
	if status=="cancelado"{fail(w,409,"order_not_completable","Un pedido cancelado no puede finalizarse.");return}
	if paid+0.00001<total{fail(w,409,"payment_incomplete","El pedido todavía tiene saldo pendiente.");return}
	if _,err=tx.Exec(r.Context(),`
		UPDATE orders
		SET status='entregado',updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
	`,r.PathValue("id"),s.OrganizationID,s.LocationID);err!=nil{
		fail(w,503,"payments_unavailable","No pudimos liberar la mesa.")
		return
	}
	if err=tx.Commit(r.Context());err!=nil{fail(w,503,"payments_unavailable","No pudimos finalizar el pedido.");return}
	w.WriteHeader(http.StatusNoContent)
}
