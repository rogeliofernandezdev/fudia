package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestExpenseLifecycleIsLocationScopedAndAuditable(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)

	categoryReq := httptest.NewRequest("POST", "/v1/admin/expense-categories", bytes.NewReader([]byte(`{"name":"Servicios"}`)))
	categoryReq = categoryReq.WithContext(context.WithValue(categoryReq.Context(), scopeKey{}, s))
	categoryRec := httptest.NewRecorder()
	api.createExpenseCategory(categoryRec, categoryReq)
	if categoryRec.Code != 201 {
		t.Fatalf("expected category 201, got %d body=%s", categoryRec.Code, categoryRec.Body.String())
	}
	var category expenseCategoryView
	if err := json.Unmarshal(categoryRec.Body.Bytes(), &category); err != nil {
		t.Fatal(err)
	}
	if category.ID == "" || category.Name != "Servicios" || !category.Active {
		t.Fatalf("unexpected category: %#v", category)
	}

	expenseReq := httptest.NewRequest("POST", "/v1/admin/expenses", bytes.NewReader([]byte(`{"categoryId":"` + category.ID + `","description":"Internet del local","amount":"89.90","paymentMethod":"transfer","businessDate":"2026-09-23","reference":"OP-100","notes":"Plan mensual"}`)))
	expenseReq = expenseReq.WithContext(context.WithValue(expenseReq.Context(), scopeKey{}, s))
	expenseRec := httptest.NewRecorder()
	api.createExpense(expenseRec, expenseReq)
	if expenseRec.Code != 201 {
		t.Fatalf("expected expense 201, got %d body=%s", expenseRec.Code, expenseRec.Body.String())
	}
	var expense expenseView
	if err := json.Unmarshal(expenseRec.Body.Bytes(), &expense); err != nil {
		t.Fatal(err)
	}
	if expense.ID == "" || expense.Amount != "89.90" || expense.Status != "active" || expense.CategoryName != "Servicios" {
		t.Fatalf("unexpected expense: %#v", expense)
	}

	listReq := httptest.NewRequest("GET", "/v1/admin/expenses?q=Internet&page=1&pageSize=20", nil)
	listReq = listReq.WithContext(context.WithValue(listReq.Context(), scopeKey{}, s))
	listRec := httptest.NewRecorder()
	api.listExpenses(listRec, listReq)
	if listRec.Code != 200 {
		t.Fatalf("expected list 200, got %d body=%s", listRec.Code, listRec.Body.String())
	}
	var list struct {
		Items []expenseView `json:"items"`
		Total int           `json:"total"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].ID != expense.ID {
		t.Fatalf("unexpected list: %#v", list)
	}

	voidReq := httptest.NewRequest("POST", "/v1/admin/expenses/"+expense.ID+"/void", bytes.NewReader([]byte(`{"reason":"Registro duplicado"}`)))
	voidReq.SetPathValue("id", expense.ID)
	voidReq = voidReq.WithContext(context.WithValue(voidReq.Context(), scopeKey{}, s))
	voidRec := httptest.NewRecorder()
	api.voidExpense(voidRec, voidReq)
	if voidRec.Code != 200 {
		t.Fatalf("expected void 200, got %d body=%s", voidRec.Code, voidRec.Body.String())
	}
	if err := json.Unmarshal(voidRec.Body.Bytes(), &expense); err != nil {
		t.Fatal(err)
	}
	if expense.Status != "void" || expense.VoidReason != "Registro duplicado" || expense.VoidedAt == nil || expense.VoidedByName == "" {
		t.Fatalf("expense must preserve void audit data: %#v", expense)
	}

	var otherLocation string
	if err := pool.QueryRow(context.Background(), `INSERT INTO locations(organization_id,name,code,address) VALUES($1,'Otro local','EXP-OTHER','') RETURNING id`, s.OrganizationID).Scan(&otherLocation); err != nil {
		t.Fatal(err)
	}
	otherScope := s
	otherScope.LocationID = otherLocation
	otherReq := httptest.NewRequest("GET", "/v1/admin/expenses", nil)
	otherReq = otherReq.WithContext(context.WithValue(otherReq.Context(), scopeKey{}, otherScope))
	otherRec := httptest.NewRecorder()
	api.listExpenses(otherRec, otherReq)
	if otherRec.Code != 200 || !bytes.Contains(otherRec.Body.Bytes(), []byte(`"total":0`)) {
		t.Fatalf("expense must be location scoped, got %d body=%s", otherRec.Code, otherRec.Body.String())
	}
}
