package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCashReadScopeIsOwnerOrReservedAdministrator(t *testing.T) {
	api, s, administratorRole := seedIdentityAdministrator(t)
	ctx := context.Background()
	if _, err := api.db.Exec(ctx, `UPDATE roles SET system_key='administrator',permissions=ARRAY['cash.read','cash.manage','cash.registers.manage','cash.expected.read'],menu_access=ARRAY['caja','pos'] WHERE id=$1`, administratorRole); err != nil {
		t.Fatal(err)
	}
	var role string
	// Broad cash permissions and a misleading name must not replace the reserved role.
	if err := api.db.QueryRow(ctx, `INSERT INTO roles(organization_id,name,system_key,permissions,menu_access)
	 VALUES($1,'Administrador de caja','cashier',ARRAY['cash.read','cash.manage','cash.expected.read','cash.registers.manage'],ARRAY['caja']) RETURNING id`, s.OrganizationID).Scan(&role); err != nil {
		t.Fatal(err)
	}
	user := func(name string) scope {
		var id string
		if err := api.db.QueryRow(ctx, `INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,$2,$3,'test') RETURNING id`, s.OrganizationID, fmt.Sprintf("cash-read-%s-%d@example.test", name, time.Now().UnixNano()), name).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if _, err := api.db.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, id, role, s.LocationID); err != nil {
			t.Fatal(err)
		}
		return scope{OrganizationID: s.OrganizationID, LocationID: s.LocationID, UserID: id, Name: name}
	}
	a, b, collaborator := user("Cajera Ana"), user("Cajero Bruno"), user("Colaboradora")
	adminCall, aCall, bCall := companySession(t, api, s), companySession(t, api, a), companySession(t, api, b)
	expect := func(rec *httptest.ResponseRecorder, status int) {
		t.Helper()
		if rec.Code != status {
			t.Fatalf("expected %d: %d %s", status, rec.Code, rec.Body.String())
		}
	}
	open := func(call func(string, string, any) *httptest.ResponseRecorder, name string, amount int) cashShiftView {
		rec := adminCall("POST", "/v1/admin/cash-registers", map[string]any{"name": name})
		expect(rec, 201)
		var register cashRegisterView
		if err := json.Unmarshal(rec.Body.Bytes(), &register); err != nil {
			t.Fatal(err)
		}
		rec = call("POST", "/v1/admin/cash-shifts", map[string]any{"cashRegisterId": register.ID, "openingAmount": amount})
		expect(rec, 201)
		var shift cashShiftView
		if err := json.Unmarshal(rec.Body.Bytes(), &shift); err != nil {
			t.Fatal(err)
		}
		return shift
	}
	first, second := open(aCall, "Caja de Ana", 100), open(bCall, "Caja de Bruno", 200)
	path := func(id string) string { return "/v1/admin/cash-shifts/" + id }
	expect(aCall("POST", path(first.ID)+"/movements", map[string]any{"movementType": "income", "amount": 10, "reason": "Ingreso de Ana"}), 201)
	expect(bCall("POST", path(second.ID)+"/movements", map[string]any{"movementType": "income", "amount": 20, "reason": "Ingreso de Bruno"}), 201)
	expect(aCall("POST", path(first.ID)+"/users", map[string]any{"userId": collaborator.UserID}), 201)
	t.Run("assignment does not expose finances", func(t *testing.T) {
		call := companySession(t, api, collaborator)
		expect(call("GET", path(first.ID), nil), 404)
		expect(call("GET", path(first.ID)+"/report", nil), 404)
		expect(call("GET", path(first.ID)+"/users", nil), 404)
		rec := call("GET", "/v1/admin/cash-shifts/current", nil)
		expect(rec, 200)
		var current struct {
			Shift *cashShiftView `json:"shift"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &current); err != nil || current.Shift != nil {
			t.Fatalf("collaborator finances leaked: %s %v", rec.Body.String(), err)
		}
	})
	expect(aCall("POST", path(first.ID)+"/close", map[string]any{"countedAmount": 110}), 200)
	third := open(aCall, "Otra caja de Ana", 50)
	t.Run("pagination search and counts use owner scope", func(t *testing.T) {
		for _, entry := range []struct {
			call  func(string, string, any) *httptest.ResponseRecorder
			total int
			owner string
		}{{aCall, 2, a.UserID}, {bCall, 1, b.UserID}, {adminCall, 3, ""}} {
			rec := entry.call("GET", "/v1/admin/cash-shifts?pageSize=1", nil)
			expect(rec, 200)
			var page struct {
				Items []cashShiftView `json:"items"`
				Total int             `json:"total"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
				t.Fatal(err)
			}
			if page.Total != entry.total || len(page.Items) != 1 {
				t.Fatalf("incorrect scoped count: %s", rec.Body.String())
			}
			if entry.owner != "" && page.Items[0].OpenedByID != entry.owner {
				t.Fatalf("foreign shift in page: %s", rec.Body.String())
			}
		}
		rec := aCall("GET", "/v1/admin/cash-shifts?q=Bruno&ownerId="+b.UserID, nil)
		expect(rec, 200)
		var page struct {
			Total int `json:"total"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil || page.Total != 0 {
			t.Fatalf("filter bypass: %s %v", rec.Body.String(), err)
		}
	})
	t.Run("details persisted reports and teams cannot bypass scope", func(t *testing.T) {
		for _, suffix := range []string{"", "/report", "/users"} {
			expect(bCall("GET", path(first.ID)+suffix, nil), 404)
			expect(aCall("GET", path(second.ID)+suffix, nil), 404)
			expect(adminCall("GET", path(first.ID)+suffix, nil), 200)
			expect(adminCall("GET", path(second.ID)+suffix, nil), 200)
			expect(aCall("GET", path(first.ID)+suffix, nil), 200)
		}
		foreign := seedInventoryScope(t, api.db)
		var foreignShift string
		if err := api.db.QueryRow(ctx, `WITH r AS (INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,'Caja ajena',$3) RETURNING id) INSERT INTO cash_shifts(organization_id,location_id,cash_register_id,opened_by,business_date) SELECT $1,$2,id,$3,current_date FROM r RETURNING id`, foreign.OrganizationID, foreign.LocationID, foreign.UserID).Scan(&foreignShift); err != nil {
			t.Fatal(err)
		}
		for _, call := range []func(string, string, any) *httptest.ResponseRecorder{adminCall, aCall} {
			expect(call("GET", path(foreignShift), nil), 404)
			expect(call("GET", path(foreignShift)+"/report", nil), 404)
		}
	})
	t.Run("register availability never includes another cashier's finances", func(t *testing.T) {
		rec := aCall("GET", "/v1/admin/cash-registers", nil)
		expect(rec, 200)
		var result struct {
			Items []cashRegisterView `json:"items"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		for _, register := range result.Items {
			if shift := register.OpenShift; shift != nil {
				if shift.ID == second.ID && (shift.DetailsVisible || shift.ExpectedVisible || shift.ExpectedAmount != "" || shift.IncomeAmount != "" || shift.OpeningAmount != "" || shift.MovementCount != 0 || len(shift.Movements) != 0 || len(shift.ActiveUserNames) != 0) {
					t.Fatalf("foreign register finances leaked: %+v", shift)
				}
				if shift.ID == third.ID && (!shift.DetailsVisible || shift.OpeningAmount != "50.00") {
					t.Fatalf("own register hidden: %+v", shift)
				}
			}
		}
		rec = adminCall("GET", "/v1/admin/cash-registers", nil)
		expect(rec, 200)
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		for _, register := range result.Items {
			if shift := register.OpenShift; shift != nil && (!shift.DetailsVisible || !shift.ExpectedVisible) {
				t.Fatalf("administrator cannot see register: %+v", shift)
			}
		}
	})
}
