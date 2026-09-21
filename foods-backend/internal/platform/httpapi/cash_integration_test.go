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

func TestCashRegisterAndShiftLifecycleTracksExpectedAndVariance(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)

	registerReq := httptest.NewRequest("POST", "/v1/admin/cash-registers", bytes.NewReader([]byte(`{"name":"Caja principal"}`)))
	registerReq = registerReq.WithContext(context.WithValue(registerReq.Context(), scopeKey{}, s))
	registerRec := httptest.NewRecorder()
	api.createCashRegister(registerRec, registerReq)
	if registerRec.Code != 201 {
		t.Fatalf("expected cash register 201, got %d body=%s", registerRec.Code, registerRec.Body.String())
	}
	var register cashRegisterView
	if err := json.Unmarshal(registerRec.Body.Bytes(), &register); err != nil {
		t.Fatal(err)
	}
	if register.ID == "" || register.Name != "Caja principal" || !register.Active {
		t.Fatalf("unexpected cash register: %#v", register)
	}

	openBody := []byte(fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":100,"note":"Fondo inicial"}`, register.ID))
	openReq := httptest.NewRequest("POST", "/v1/admin/cash-shifts", bytes.NewReader(openBody))
	openReq = openReq.WithContext(context.WithValue(openReq.Context(), scopeKey{}, s))
	openRec := httptest.NewRecorder()
	api.openCashShift(openRec, openReq)
	if openRec.Code != 201 {
		t.Fatalf("expected open shift 201, got %d body=%s", openRec.Code, openRec.Body.String())
	}
	var opened cashShiftView
	if err := json.Unmarshal(openRec.Body.Bytes(), &opened); err != nil {
		t.Fatal(err)
	}
	if opened.Status != "open" || opened.CashRegisterID != register.ID || opened.CashRegisterName != register.Name || opened.OpeningAmount != "100.00" || opened.ExpectedAmount != "100.00" {
		t.Fatalf("unexpected opened shift: %#v", opened)
	}

	listReq := httptest.NewRequest("GET", "/v1/admin/cash-registers", nil)
	listReq = listReq.WithContext(context.WithValue(listReq.Context(), scopeKey{}, s))
	listRec := httptest.NewRecorder()
	api.listCashRegisters(listRec, listReq)
	if listRec.Code != 200 {
		t.Fatalf("expected register list 200, got %d body=%s", listRec.Code, listRec.Body.String())
	}
	var registers struct {
		Items []cashRegisterView `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &registers); err != nil {
		t.Fatal(err)
	}
	if len(registers.Items) != 1 || registers.Items[0].OpenShift == nil || registers.Items[0].OpenShift.ID != opened.ID {
		t.Fatalf("register must expose its open shift: %#v", registers.Items)
	}

	duplicateReq := httptest.NewRequest("POST", "/v1/admin/cash-shifts", bytes.NewReader(openBody))
	duplicateReq = duplicateReq.WithContext(context.WithValue(duplicateReq.Context(), scopeKey{}, s))
	duplicateRec := httptest.NewRecorder()
	api.openCashShift(duplicateRec, duplicateReq)
	if duplicateRec.Code != 409 {
		t.Fatalf("expected duplicate open shift 409, got %d body=%s", duplicateRec.Code, duplicateRec.Body.String())
	}

	incomeBody := []byte(`{"movementType":"income","amount":50,"reason":"Cambio recibido","note":"Reposición"}`)
	incomeReq := httptest.NewRequest("POST", "/v1/admin/cash-shifts/"+opened.ID+"/movements", bytes.NewReader(incomeBody))
	incomeReq.SetPathValue("id", opened.ID)
	incomeReq = incomeReq.WithContext(context.WithValue(incomeReq.Context(), scopeKey{}, s))
	incomeRec := httptest.NewRecorder()
	api.createCashMovement(incomeRec, incomeReq)
	if incomeRec.Code != 201 {
		t.Fatalf("expected income 201, got %d body=%s", incomeRec.Code, incomeRec.Body.String())
	}

	expenseBody := []byte(`{"movementType":"expense","amount":20,"reason":"Compra menor","note":"Caja chica"}`)
	expenseReq := httptest.NewRequest("POST", "/v1/admin/cash-shifts/"+opened.ID+"/movements", bytes.NewReader(expenseBody))
	expenseReq.SetPathValue("id", opened.ID)
	expenseReq = expenseReq.WithContext(context.WithValue(expenseReq.Context(), scopeKey{}, s))
	expenseRec := httptest.NewRecorder()
	api.createCashMovement(expenseRec, expenseReq)
	if expenseRec.Code != 201 {
		t.Fatalf("expected expense 201, got %d body=%s", expenseRec.Code, expenseRec.Body.String())
	}

	currentReq := httptest.NewRequest("GET", "/v1/admin/cash-shifts/current", nil)
	currentReq = currentReq.WithContext(context.WithValue(currentReq.Context(), scopeKey{}, s))
	currentRec := httptest.NewRecorder()
	api.getCurrentCashShift(currentRec, currentReq)
	if currentRec.Code != 200 {
		t.Fatalf("expected current shift 200, got %d body=%s", currentRec.Code, currentRec.Body.String())
	}
	var current struct {
		Shift *cashShiftView `json:"shift"`
	}
	if err := json.Unmarshal(currentRec.Body.Bytes(), &current); err != nil {
		t.Fatal(err)
	}
	if current.Shift == nil || current.Shift.IncomeAmount != "50.00" || current.Shift.ExpenseAmount != "20.00" || current.Shift.ExpectedAmount != "130.00" || len(current.Shift.Movements) != 2 {
		t.Fatalf("unexpected current shift totals: %#v", current.Shift)
	}

	closeBody := []byte(`{"countedAmount":128,"note":"Diferencia revisada"}`)
	closeReq := httptest.NewRequest("POST", "/v1/admin/cash-shifts/"+opened.ID+"/close", bytes.NewReader(closeBody))
	closeReq.SetPathValue("id", opened.ID)
	closeReq = closeReq.WithContext(context.WithValue(closeReq.Context(), scopeKey{}, s))
	closeRec := httptest.NewRecorder()
	api.closeCashShift(closeRec, closeReq)
	if closeRec.Code != 200 {
		t.Fatalf("expected close 200, got %d body=%s", closeRec.Code, closeRec.Body.String())
	}
	var closed cashShiftView
	if err := json.Unmarshal(closeRec.Body.Bytes(), &closed); err != nil {
		t.Fatal(err)
	}
	if closed.Status != "closed" || closed.ClosingExpectedAmount == nil || *closed.ClosingExpectedAmount != "130.00" || closed.ClosingCountedAmount == nil || *closed.ClosingCountedAmount != "128.00" || closed.VarianceAmount == nil || *closed.VarianceAmount != "-2.00" {
		t.Fatalf("unexpected closed shift: %#v", closed)
	}

	lateBody := []byte(`{"movementType":"income","amount":1,"reason":"No permitido"}`)
	lateReq := httptest.NewRequest("POST", "/v1/admin/cash-shifts/"+opened.ID+"/movements", bytes.NewReader(lateBody))
	lateReq.SetPathValue("id", opened.ID)
	lateReq = lateReq.WithContext(context.WithValue(lateReq.Context(), scopeKey{}, s))
	lateRec := httptest.NewRecorder()
	api.createCashMovement(lateRec, lateReq)
	if lateRec.Code != 409 {
		t.Fatalf("expected movement after close 409, got %d body=%s", lateRec.Code, lateRec.Body.String())
	}

	currentAfterReq := httptest.NewRequest("GET", "/v1/admin/cash-shifts/current", nil)
	currentAfterReq = currentAfterReq.WithContext(context.WithValue(currentAfterReq.Context(), scopeKey{}, s))
	currentAfterRec := httptest.NewRecorder()
	api.getCurrentCashShift(currentAfterRec, currentAfterReq)
	if currentAfterRec.Code != 200 {
		t.Fatalf("expected empty current shift response, got %d body=%s", currentAfterRec.Code, currentAfterRec.Body.String())
	}
	if err := json.Unmarshal(currentAfterRec.Body.Bytes(), &current); err != nil {
		t.Fatal(err)
	}
	if current.Shift != nil {
		t.Fatalf("expected no current shift after closing, got %#v", current.Shift)
	}
}

func TestCashShiftAndRegisterAreScopedToLocation(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()

	var otherLocationID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO locations(organization_id,name,code,address)
		VALUES($1,'Caja alterna',$2,'')
		RETURNING id
	`, s.OrganizationID, fmt.Sprintf("CASH-%d", time.Now().UnixNano())).Scan(&otherLocationID); err != nil {
		t.Fatal(err)
	}

	var registerID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO cash_registers(organization_id,location_id,name,created_by)
		VALUES($1,$2,'Caja externa',$3)
		RETURNING id
	`, s.OrganizationID, otherLocationID, s.UserID).Scan(&registerID); err != nil {
		t.Fatal(err)
	}

	var shiftID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO cash_shifts(organization_id,location_id,cash_register_id,opening_amount,opened_by)
		VALUES($1,$2,$3,10,$4)
		RETURNING id
	`, s.OrganizationID, otherLocationID, registerID, s.UserID).Scan(&shiftID); err != nil {
		t.Fatal(err)
	}

	api := New(pool)
	req := httptest.NewRequest("GET", "/v1/admin/cash-shifts/"+shiftID, nil)
	req.SetPathValue("id", shiftID)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.getCashShift(rec, req)
	if rec.Code != 404 {
		t.Fatalf("expected location-scoped 404, got %d body=%s", rec.Code, rec.Body.String())
	}
}
