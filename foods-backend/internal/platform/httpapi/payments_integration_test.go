package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPaymentAndRefundUpdateCashShiftAutomatically(t *testing.T) {
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool)
	api:=New(pool)
	ctx:=context.Background()

	var registerID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO cash_registers(organization_id,location_id,name,created_by)
		VALUES($1,$2,'Caja pagos',$3)
		RETURNING id
	`,s.OrganizationID,s.LocationID,s.UserID).Scan(&registerID);err!=nil{t.Fatal(err)}

	openBody:=[]byte(fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":100}`,registerID))
	openReq:=httptest.NewRequest("POST","/v1/admin/cash-shifts",bytes.NewReader(openBody))
	openReq=openReq.WithContext(context.WithValue(openReq.Context(),scopeKey{},s))
	openRec:=httptest.NewRecorder()
	api.openCashShift(openRec,openReq)
	if openRec.Code!=201{t.Fatalf("open shift: %d %s",openRec.Code,openRec.Body.String())}
	var shift cashShiftView
	if err:=json.Unmarshal(openRec.Body.Bytes(),&shift);err!=nil{t.Fatal(err)}

	var orderID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO orders(organization_id,location_id,code,channel,status,total,created_by)
		VALUES($1,$2,$3,'mostrador','listo',60,$4)
		RETURNING id
	`,s.OrganizationID,s.LocationID,fmt.Sprintf("PED-PAY-%d",time.Now().UnixNano()),s.UserID).Scan(&orderID);err!=nil{t.Fatal(err)}

	cashBody:=[]byte(fmt.Sprintf(`{"orderId":%q,"method":"cash","amount":40}`,orderID))
	cashReq:=httptest.NewRequest("POST","/v1/admin/payments",bytes.NewReader(cashBody))
	cashReq=cashReq.WithContext(context.WithValue(cashReq.Context(),scopeKey{},s))
	cashRec:=httptest.NewRecorder()
	api.createPayment(cashRec,cashReq)
	if cashRec.Code!=201{t.Fatalf("cash payment: %d %s",cashRec.Code,cashRec.Body.String())}
	var cashPayment paymentView
	if err:=json.Unmarshal(cashRec.Body.Bytes(),&cashPayment);err!=nil{t.Fatal(err)}
	if cashPayment.Method!="cash"||cashPayment.Amount!="40.00"{t.Fatalf("unexpected cash payment: %#v",cashPayment)}

	cardBody:=[]byte(fmt.Sprintf(`{"orderId":%q,"method":"card","amount":20,"reference":"VISA"}`,orderID))
	cardReq:=httptest.NewRequest("POST","/v1/admin/payments",bytes.NewReader(cardBody))
	cardReq=cardReq.WithContext(context.WithValue(cardReq.Context(),scopeKey{},s))
	cardRec:=httptest.NewRecorder()
	api.createPayment(cardRec,cardReq)
	if cardRec.Code!=201{t.Fatalf("card payment: %d %s",cardRec.Code,cardRec.Body.String())}

	var cashSales int
	if err:=pool.QueryRow(ctx,`
		SELECT count(*) FROM cash_movements
		WHERE organization_id=$1 AND location_id=$2 AND shift_id=$3 AND source_type='cash_sale' AND amount=40
	`,s.OrganizationID,s.LocationID,shift.ID).Scan(&cashSales);err!=nil{t.Fatal(err)}
	if cashSales!=1{t.Fatalf("expected one cash sale movement, got %d",cashSales)}

	detailReq:=httptest.NewRequest("GET","/v1/admin/pos/orders/"+orderID,nil)
	detailReq.SetPathValue("id",orderID)
	detailReq=detailReq.WithContext(context.WithValue(detailReq.Context(),scopeKey{},s))
	detailRec:=httptest.NewRecorder()
	api.getPOSOrder(detailRec,detailReq)
	if detailRec.Code!=200{t.Fatalf("pos detail: %d %s",detailRec.Code,detailRec.Body.String())}
	var detail posOrderDetail
	if err:=json.Unmarshal(detailRec.Body.Bytes(),&detail);err!=nil{t.Fatal(err)}
	if detail.PaymentStatus!="paid"||detail.PaidAmount!="60.00"||detail.RemainingAmount!="0.00"{
		t.Fatalf("unexpected payment summary: %#v",detail)
	}

	overBody:=[]byte(fmt.Sprintf(`{"orderId":%q,"method":"cash","amount":1}`,orderID))
	overReq:=httptest.NewRequest("POST","/v1/admin/payments",bytes.NewReader(overBody))
	overReq=overReq.WithContext(context.WithValue(overReq.Context(),scopeKey{},s))
	overRec:=httptest.NewRecorder()
	api.createPayment(overRec,overReq)
	if overRec.Code!=409{t.Fatalf("expected overpayment 409, got %d %s",overRec.Code,overRec.Body.String())}

	refundBody:=[]byte(`{"amount":10,"reason":"Corrección de cobro"}`)
	refundReq:=httptest.NewRequest("POST","/v1/admin/payments/"+cashPayment.ID+"/refund",bytes.NewReader(refundBody))
	refundReq.SetPathValue("id",cashPayment.ID)
	refundReq=refundReq.WithContext(context.WithValue(refundReq.Context(),scopeKey{},s))
	refundRec:=httptest.NewRecorder()
	api.refundPayment(refundRec,refundReq)
	if refundRec.Code!=204{t.Fatalf("cash refund: %d %s",refundRec.Code,refundRec.Body.String())}

	var cashRefunds int
	if err:=pool.QueryRow(ctx,`
		SELECT count(*) FROM cash_movements
		WHERE organization_id=$1 AND location_id=$2 AND shift_id=$3 AND source_type='cash_refund' AND amount=10
	`,s.OrganizationID,s.LocationID,shift.ID).Scan(&cashRefunds);err!=nil{t.Fatal(err)}
	if cashRefunds!=1{t.Fatalf("expected one cash refund movement, got %d",cashRefunds)}

	detailRec=httptest.NewRecorder()
	detailReq=httptest.NewRequest("GET","/v1/admin/pos/orders/"+orderID,nil)
	detailReq.SetPathValue("id",orderID)
	detailReq=detailReq.WithContext(context.WithValue(detailReq.Context(),scopeKey{},s))
	api.getPOSOrder(detailRec,detailReq)
	if err:=json.Unmarshal(detailRec.Body.Bytes(),&detail);err!=nil{t.Fatal(err)}
	if detail.PaymentStatus!="partial"||detail.PaidAmount!="50.00"||detail.RemainingAmount!="10.00"{
		t.Fatalf("refund must reopen remaining balance: %#v",detail)
	}

	currentReq:=httptest.NewRequest("GET","/v1/admin/cash-shifts/current",nil)
	currentReq=currentReq.WithContext(context.WithValue(currentReq.Context(),scopeKey{},s))
	currentRec:=httptest.NewRecorder()
	api.getCurrentCashShift(currentRec,currentReq)
	var current struct{Shift *cashShiftView `json:"shift"`}
	if err:=json.Unmarshal(currentRec.Body.Bytes(),&current);err!=nil{t.Fatal(err)}
	if current.Shift==nil||current.Shift.ExpectedAmount!="130.00"{
		t.Fatalf("expected cash 100 + sale 40 - refund 10 = 130, got %#v",current.Shift)
	}
}


func TestSplitPaymentBatchAndCompletePaidOrder(t *testing.T) {
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool)
	api:=New(pool)
	ctx:=context.Background()

	var registerID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO cash_registers(organization_id,location_id,name,created_by)
		VALUES($1,$2,'Caja split',$3)
		RETURNING id
	`,s.OrganizationID,s.LocationID,s.UserID).Scan(&registerID);err!=nil{t.Fatal(err)}

	openReq:=httptest.NewRequest("POST","/v1/operations/cash-shifts",bytes.NewReader([]byte(fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":100}`,registerID))))
	openReq=openReq.WithContext(context.WithValue(openReq.Context(),scopeKey{},s))
	openRec:=httptest.NewRecorder()
	api.openCashShift(openRec,openReq)
	if openRec.Code!=201{t.Fatalf("open shift: %d %s",openRec.Code,openRec.Body.String())}

	var orderID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO orders(organization_id,location_id,code,channel,status,total,created_by)
		VALUES($1,$2,$3,'mostrador','listo',60,$4)
		RETURNING id
	`,s.OrganizationID,s.LocationID,fmt.Sprintf("PED-SPLIT-%d",time.Now().UnixNano()),s.UserID).Scan(&orderID);err!=nil{t.Fatal(err)}

	batchBody:=[]byte(fmt.Sprintf(`{
		"orderId":%q,
		"payments":[
			{"method":"cash","amount":20,"reference":"Recibido 20.00"},
			{"method":"card","amount":40,"reference":"VISA-01"}
		]
	}`,orderID))
	batchReq:=httptest.NewRequest("POST","/v1/operations/payments/batch",bytes.NewReader(batchBody))
	batchReq=batchReq.WithContext(context.WithValue(batchReq.Context(),scopeKey{},s))
	batchRec:=httptest.NewRecorder()
	api.createPaymentBatch(batchRec,batchReq)
	if batchRec.Code!=201{t.Fatalf("split payment: %d %s",batchRec.Code,batchRec.Body.String())}

	var result struct{
		PaymentIDs []string `json:"paymentIds"`
		RemainingAmount string `json:"remainingAmount"`
		PaymentStatus string `json:"paymentStatus"`
	}
	if err:=json.Unmarshal(batchRec.Body.Bytes(),&result);err!=nil{t.Fatal(err)}
	if len(result.PaymentIDs)!=2||result.RemainingAmount!="0.00"||result.PaymentStatus!="paid"{
		t.Fatalf("unexpected split result: %#v",result)
	}

	var payments,cashMovements int
	if err:=pool.QueryRow(ctx,`SELECT count(*) FROM payments WHERE organization_id=$1 AND location_id=$2 AND order_id=$3`,s.OrganizationID,s.LocationID,orderID).Scan(&payments);err!=nil{t.Fatal(err)}
	if err:=pool.QueryRow(ctx,`SELECT count(*) FROM cash_movements WHERE organization_id=$1 AND location_id=$2 AND source_type='cash_sale' AND amount=20`,s.OrganizationID,s.LocationID).Scan(&cashMovements);err!=nil{t.Fatal(err)}
	if payments!=2||cashMovements!=1{t.Fatalf("expected 2 payments and 1 cash movement, got payments=%d cash=%d",payments,cashMovements)}

	completeReq:=httptest.NewRequest("POST","/v1/operations/pos/orders/"+orderID+"/complete",nil)
	completeReq.SetPathValue("id",orderID)
	completeReq=completeReq.WithContext(context.WithValue(completeReq.Context(),scopeKey{},s))
	completeRec:=httptest.NewRecorder()
	api.completePaidOrder(completeRec,completeReq)
	if completeRec.Code!=204{t.Fatalf("complete order: %d %s",completeRec.Code,completeRec.Body.String())}

	var status string
	if err:=pool.QueryRow(ctx,`SELECT status FROM orders WHERE id=$1`,orderID).Scan(&status);err!=nil{t.Fatal(err)}
	if status!="entregado"{t.Fatalf("expected delivered order after completion, got %s",status)}
}
