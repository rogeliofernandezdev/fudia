package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Con el módulo costos, el gasto del turno y el egreso de caja son un solo
// registro: se ve en el turno y en Gastos, y anularlo devuelve el efectivo.
func TestCashExpenseIsOneRecordInShiftAndExpenses(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	grantCashManagementForPaymentTest(t, api, s)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM expenses WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM expense_categories WHERE organization_id=$1`, s.OrganizationID)
	})
	call := func(method, body, id string, handler http.HandlerFunc) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/", strings.NewReader(body))
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		handler(rec, req)
		return rec
	}
	register := call("POST", `{"name":"Caja gastos"}`, "", api.createCashRegister)
	var cr cashRegisterView
	_ = json.Unmarshal(register.Body.Bytes(), &cr)
	opened := call("POST", fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":100}`, cr.ID), "", api.openCashShift)
	if opened.Code != 201 {
		t.Fatalf("open: %d %s", opened.Code, opened.Body.String())
	}
	var shift cashShiftView
	_ = json.Unmarshal(opened.Body.Bytes(), &shift)

	// Sin el módulo costos, el egreso conserva su motivo libre y no crea gasto.
	if rec := call("POST", `{"movementType":"expense","amount":5,"reason":"Compra menor"}`, shift.ID, api.createCashMovement); rec.Code != 201 {
		t.Fatalf("manual expense: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call("GET", "", "", api.listCashExpenseCategories); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"items":[]`) {
		t.Fatalf("categories without module: %d %s", rec.Code, rec.Body.String())
	}

	if _, err := pool.Exec(ctx, `INSERT INTO organization_modules(organization_id,module_key,active) VALUES($1,'costos',true) ON CONFLICT (organization_id,module_key) DO UPDATE SET active=true`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	var categoryID string
	if err := pool.QueryRow(ctx, `INSERT INTO expense_categories(organization_id,name) VALUES($1,'Insumos') RETURNING id::text`, s.OrganizationID).Scan(&categoryID); err != nil {
		t.Fatal(err)
	}
	if rec := call("POST", `{"movementType":"expense","amount":10,"reason":"Balón de gas"}`, shift.ID, api.createCashMovement); rec.Code != 400 || !strings.Contains(rec.Body.String(), "cash_expense_category_required") {
		t.Fatalf("expense without category must fail: %d %s", rec.Code, rec.Body.String())
	}
	rec := call("POST", fmt.Sprintf(`{"movementType":"expense","amount":10,"reason":"Balón de gas","note":"Proveedor del barrio","categoryId":%q}`, categoryID), shift.ID, api.createCashMovement)
	if rec.Code != 201 || !strings.Contains(rec.Body.String(), `"sourceType":"expense"`) {
		t.Fatalf("cash expense: %d %s", rec.Code, rec.Body.String())
	}

	var expenseID, method, description, notes, shiftID string
	if err := pool.QueryRow(ctx, `SELECT id::text,payment_method,description,notes,cash_shift_id::text FROM expenses WHERE organization_id=$1`, s.OrganizationID).Scan(&expenseID, &method, &description, &notes, &shiftID); err != nil {
		t.Fatalf("one expense expected: %v", err)
	}
	if method != "cash" || description != "Balón de gas" || notes != "Proveedor del barrio" || shiftID != shift.ID {
		t.Fatalf("unexpected expense: %s %s %s %s", method, description, notes, shiftID)
	}
	if rec := call("GET", "", "", api.listExpenses); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"cashRegisterName":"Caja gastos"`) || !strings.Contains(rec.Body.String(), `"cashPaymentMethods":["cash"]`) {
		t.Fatalf("expense list: %d %s", rec.Code, rec.Body.String())
	}
	expected := func() string {
		var value string
		if err := pool.QueryRow(ctx, `
			SELECT (cs.opening_amount
			  + COALESCE(sum(cm.amount) FILTER (WHERE cm.movement_type='income'),0)
			  - COALESCE(sum(cm.amount) FILTER (WHERE cm.movement_type='expense'),0))::numeric(12,2)::text
			FROM cash_shifts cs LEFT JOIN cash_movements cm ON cm.shift_id=cs.id
			WHERE cs.id=$1 GROUP BY cs.opening_amount`, shift.ID).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	if got := expected(); got != "85.00" {
		t.Fatalf("expected cash after expenses: %s", got)
	}

	// Desde la pestaña Gastos no se registra efectivo: sale del turno.
	if rec := call("POST", fmt.Sprintf(`{"categoryId":%q,"description":"Gas","amount":"5","paymentMethod":"cash","businessDate":"2026-11-22"}`, categoryID), "", api.createExpense); rec.Code != 400 || !strings.Contains(rec.Body.String(), "cash_expense_requires_shift") {
		t.Fatalf("cash expense outside shift must fail: %d %s", rec.Code, rec.Body.String())
	}

	// Anular con el turno abierto devuelve el efectivo al turno.
	if rec := call("POST", `{"reason":"Se registró dos veces"}`, expenseID, api.voidExpense); rec.Code != 200 {
		t.Fatalf("void: %d %s", rec.Code, rec.Body.String())
	}
	if got := expected(); got != "95.00" {
		t.Fatalf("void must return the cash: %s", got)
	}
	var refunds int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM cash_movements WHERE source_type='expense_void' AND source_id::text=$1`, expenseID).Scan(&refunds); err != nil || refunds != 1 {
		t.Fatalf("one refund movement expected: %d %v", refunds, err)
	}

	// Con el turno cerrado, el gasto en efectivo ya no se anula.
	rec = call("POST", fmt.Sprintf(`{"movementType":"expense","amount":20,"reason":"Verduras","categoryId":%q}`, categoryID), shift.ID, api.createCashMovement)
	if rec.Code != 201 {
		t.Fatalf("second cash expense: %d %s", rec.Code, rec.Body.String())
	}
	var secondID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM expenses WHERE organization_id=$1 AND status='active'`, s.OrganizationID).Scan(&secondID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE cash_shifts SET status='closed',closed_at=now(),closing_expected_amount=75,closing_counted_amount=75,variance_amount=0 WHERE id=$1`, shift.ID); err != nil {
		t.Fatal(err)
	}
	if rec := call("POST", `{"reason":"Fuera de plazo"}`, secondID, api.voidExpense); rec.Code != 409 || !strings.Contains(rec.Body.String(), "expense_cash_shift_closed") {
		t.Fatalf("void after close must fail: %d %s", rec.Code, rec.Body.String())
	}
}
