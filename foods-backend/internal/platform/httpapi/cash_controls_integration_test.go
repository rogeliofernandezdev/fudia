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

func TestCashShiftUsersTransferCountsAndBlindClose(t *testing.T) {
	pool:=integrationPool(t)
	s:=seedInventoryScope(t,pool)
	api:=New(pool)
	ctx:=context.Background()

	var roleID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO roles(organization_id,name,permissions,menu_access)
		VALUES($1,$2,ARRAY['cash.read','cash.manage']::text[],ARRAY['caja']::text[])
		RETURNING id
	`,s.OrganizationID,fmt.Sprintf("Caja test %d",time.Now().UnixNano())).Scan(&roleID);err!=nil{t.Fatal(err)}

	var user2ID,user3ID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO users(organization_id,email,full_name,password_hash)
		VALUES($1,$2,'Cajero Dos','test') RETURNING id
	`,s.OrganizationID,fmt.Sprintf("cash2-%d@example.test",time.Now().UnixNano())).Scan(&user2ID);err!=nil{t.Fatal(err)}
	if err:=pool.QueryRow(ctx,`
		INSERT INTO users(organization_id,email,full_name,password_hash)
		VALUES($1,$2,'Cajero Tres','test') RETURNING id
	`,s.OrganizationID,fmt.Sprintf("cash3-%d@example.test",time.Now().UnixNano())).Scan(&user3ID);err!=nil{t.Fatal(err)}
	if _,err:=pool.Exec(ctx,`
		INSERT INTO user_roles(user_id,role_id,location_id)
		VALUES($1,$3,$4),($2,$3,$4)
	`,user2ID,user3ID,roleID,s.LocationID);err!=nil{t.Fatal(err)}

	var register1ID,register2ID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO cash_registers(organization_id,location_id,name,created_by)
		VALUES($1,$2,'Caja uno',$3) RETURNING id
	`,s.OrganizationID,s.LocationID,s.UserID).Scan(&register1ID);err!=nil{t.Fatal(err)}
	if err:=pool.QueryRow(ctx,`
		INSERT INTO cash_registers(organization_id,location_id,name,created_by)
		VALUES($1,$2,'Caja dos',$3) RETURNING id
	`,s.OrganizationID,s.LocationID,s.UserID).Scan(&register2ID);err!=nil{t.Fatal(err)}

	open1Req:=httptest.NewRequest("POST","/v1/admin/cash-shifts",bytes.NewReader([]byte(fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":100}`,register1ID))))
	open1Req=open1Req.WithContext(context.WithValue(open1Req.Context(),scopeKey{},s))
	open1Rec:=httptest.NewRecorder()
	api.openCashShift(open1Rec,open1Req)
	if open1Rec.Code!=201{t.Fatalf("open first shift: %d %s",open1Rec.Code,open1Rec.Body.String())}
	var shift1 cashShiftView
	if err:=json.Unmarshal(open1Rec.Body.Bytes(),&shift1);err!=nil{t.Fatal(err)}

	s2:=scope{OrganizationID:s.OrganizationID,LocationID:s.LocationID,UserID:user2ID,Name:"Cajero Dos"}
	open2Req:=httptest.NewRequest("POST","/v1/admin/cash-shifts",bytes.NewReader([]byte(fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":50}`,register2ID))))
	open2Req=open2Req.WithContext(context.WithValue(open2Req.Context(),scopeKey{},s2))
	open2Rec:=httptest.NewRecorder()
	api.openCashShift(open2Rec,open2Req)
	if open2Rec.Code!=201{t.Fatalf("open second shift: %d %s",open2Rec.Code,open2Rec.Body.String())}
	var shift2 cashShiftView
	if err:=json.Unmarshal(open2Rec.Body.Bytes(),&shift2);err!=nil{t.Fatal(err)}

	assignBody:=[]byte(fmt.Sprintf(`{"userId":%q}`,user3ID))
	assignReq:=httptest.NewRequest("POST","/v1/admin/cash-shifts/"+shift1.ID+"/users",bytes.NewReader(assignBody))
	assignReq.SetPathValue("id",shift1.ID)
	assignReq=assignReq.WithContext(context.WithValue(assignReq.Context(),scopeKey{},s))
	assignRec:=httptest.NewRecorder()
	api.assignCashShiftUser(assignRec,assignReq)
	if assignRec.Code!=201{t.Fatalf("assign user: %d %s",assignRec.Code,assignRec.Body.String())}

	s3:=scope{OrganizationID:s.OrganizationID,LocationID:s.LocationID,UserID:user3ID,Name:"Cajero Tres"}
	currentReq:=httptest.NewRequest("GET","/v1/admin/cash-shifts/current",nil)
	currentReq=currentReq.WithContext(context.WithValue(currentReq.Context(),scopeKey{},s3))
	currentRec:=httptest.NewRecorder()
	api.getCurrentCashShift(currentRec,currentReq)
	var current struct{Shift *cashShiftView `json:"shift"`}
	if err:=json.Unmarshal(currentRec.Body.Bytes(),&current);err!=nil{t.Fatal(err)}
	if current.Shift==nil||current.Shift.ID!=shift1.ID{t.Fatalf("assigned user must resolve first shift, got %#v",current.Shift)}

	conflictBody:=[]byte(fmt.Sprintf(`{"userId":%q}`,user2ID))
	conflictReq:=httptest.NewRequest("POST","/v1/admin/cash-shifts/"+shift1.ID+"/users",bytes.NewReader(conflictBody))
	conflictReq.SetPathValue("id",shift1.ID)
	conflictReq=conflictReq.WithContext(context.WithValue(conflictReq.Context(),scopeKey{},s))
	conflictRec:=httptest.NewRecorder()
	api.assignCashShiftUser(conflictRec,conflictReq)
	if conflictRec.Code!=409{t.Fatalf("assigned user cannot join another open shift: %d %s",conflictRec.Code,conflictRec.Body.String())}

	transferBody:=[]byte(fmt.Sprintf(`{"operationType":"transfer","targetShiftId":%q,"amount":30,"reason":"Cambio entre cajas"}`,shift2.ID))
	transferReq:=httptest.NewRequest("POST","/v1/admin/cash-shifts/"+shift1.ID+"/operations",bytes.NewReader(transferBody))
	transferReq.SetPathValue("id",shift1.ID)
	transferReq=transferReq.WithContext(context.WithValue(transferReq.Context(),scopeKey{},s))
	transferRec:=httptest.NewRecorder()
	api.createCashOperation(transferRec,transferReq)
	if transferRec.Code!=201{t.Fatalf("transfer: %d %s",transferRec.Code,transferRec.Body.String())}

	var sourceExpected,targetExpected float64
	if err:=pool.QueryRow(ctx,`
		SELECT (
		  cs.opening_amount
		  + COALESCE((SELECT sum(amount) FROM cash_movements WHERE shift_id=cs.id AND movement_type='income'),0)
		  - COALESCE((SELECT sum(amount) FROM cash_movements WHERE shift_id=cs.id AND movement_type='expense'),0)
		)::float8
		FROM cash_shifts cs WHERE id=$1
	`,shift1.ID).Scan(&sourceExpected);err!=nil{t.Fatal(err)}
	if err:=pool.QueryRow(ctx,`
		SELECT (
		  cs.opening_amount
		  + COALESCE((SELECT sum(amount) FROM cash_movements WHERE shift_id=cs.id AND movement_type='income'),0)
		  - COALESCE((SELECT sum(amount) FROM cash_movements WHERE shift_id=cs.id AND movement_type='expense'),0)
		)::float8
		FROM cash_shifts cs WHERE id=$1
	`,shift2.ID).Scan(&targetExpected);err!=nil{t.Fatal(err)}
	if sourceExpected!=70||targetExpected!=80{t.Fatalf("transfer must move expected cash atomically: source=%v target=%v",sourceExpected,targetExpected)}

	closeBody:=[]byte(`{"counts":[{"denomination":50,"quantity":1},{"denomination":20,"quantity":1}],"note":"Conteo por denominaciones"}`)
	closeReq:=httptest.NewRequest("POST","/v1/admin/cash-shifts/"+shift1.ID+"/close",bytes.NewReader(closeBody))
	closeReq.SetPathValue("id",shift1.ID)
	closeReq=closeReq.WithContext(context.WithValue(closeReq.Context(),scopeKey{},s))
	closeRec:=httptest.NewRecorder()
	api.closeCashShift(closeRec,closeReq)
	if closeRec.Code!=200{t.Fatalf("close with denominations: %d %s",closeRec.Code,closeRec.Body.String())}
	var closed cashShiftView
	if err:=json.Unmarshal(closeRec.Body.Bytes(),&closed);err!=nil{t.Fatal(err)}
	if closed.ClosingCountedAmount==nil||*closed.ClosingCountedAmount!="70.00"||closed.VarianceAmount==nil||*closed.VarianceAmount!="0.00"{
		t.Fatalf("unexpected denomination close: %#v",closed)
	}
	var countLines,activeAssignments int
	if err:=pool.QueryRow(ctx,`SELECT count(*) FROM cash_count_lines WHERE shift_id=$1`,shift1.ID).Scan(&countLines);err!=nil{t.Fatal(err)}
	if err:=pool.QueryRow(ctx,`SELECT count(*) FROM cash_shift_users WHERE shift_id=$1 AND unassigned_at IS NULL`,shift1.ID).Scan(&activeAssignments);err!=nil{t.Fatal(err)}
	if countLines!=2||activeAssignments!=0{t.Fatalf("expected 2 count lines and released users, lines=%d assignments=%d",countLines,activeAssignments)}

	var blindRegisterID string
	if err:=pool.QueryRow(ctx,`
		INSERT INTO cash_registers(organization_id,location_id,name,blind_close,created_by)
		VALUES($1,$2,'Caja ciega',true,$3) RETURNING id
	`,s.OrganizationID,s.LocationID,s.UserID).Scan(&blindRegisterID);err!=nil{t.Fatal(err)}
	blindReq:=httptest.NewRequest("POST","/v1/admin/cash-shifts",bytes.NewReader([]byte(fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":25}`,blindRegisterID))))
	blindReq=blindReq.WithContext(context.WithValue(blindReq.Context(),scopeKey{},s3))
	blindRec:=httptest.NewRecorder()
	api.openCashShift(blindRec,blindReq)
	if blindRec.Code!=201{t.Fatalf("open blind shift: %d %s",blindRec.Code,blindRec.Body.String())}
	var blind cashShiftView
	if err:=json.Unmarshal(blindRec.Body.Bytes(),&blind);err!=nil{t.Fatal(err)}
	if blind.ExpectedVisible||blind.ExpectedAmount!=""{t.Fatalf("blind shift must hide expected cash from user without permission: %#v",blind)}
}
