package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestCashShiftAttributionKeepsOpeningAndTracksTeamAndCloser(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE users SET full_name='Jimena' WHERE id=$1`, s.UserID); err != nil {
		t.Fatal(err)
	}
	var edithID, roleID string
	if err := pool.QueryRow(ctx, `INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,$2,'Edith','test') RETURNING id`, s.OrganizationID, "edith-"+s.UserID+"@example.test").Scan(&edithID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO roles(organization_id,name,permissions,menu_access) VALUES($1,'Caja',ARRAY['cash.read','cash.manage'],ARRAY['caja']) RETURNING id`, s.OrganizationID).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, edithID, roleID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	request := func(actor scope, handler http.HandlerFunc, method, path, body, shiftID, userID string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, actor))
		r.SetPathValue("id", shiftID)
		r.SetPathValue("userId", userID)
		w := httptest.NewRecorder()
		handler(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: want %d, got %d %s", method, path, want, w.Code, w.Body.String())
		}
		return w
	}
	decode := func(w *httptest.ResponseRecorder) cashShiftView {
		t.Helper()
		var shift cashShiftView
		if err := json.Unmarshal(w.Body.Bytes(), &shift); err != nil {
			t.Fatal(err)
		}
		return shift
	}
	registerResponse := request(s, api.createCashRegister, "POST", "/v1/admin/cash-registers", `{"name":"Caja atribución"}`, "", "", 201)
	var register cashRegisterView
	if err := json.Unmarshal(registerResponse.Body.Bytes(), &register); err != nil {
		t.Fatal(err)
	}
	opened := decode(request(s, api.openCashShift, "POST", "/v1/admin/cash-shifts", fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":100}`, register.ID), "", "", 201))
	if opened.OpenedByName != "Jimena" || !reflect.DeepEqual(opened.ActiveUserNames, []string{"Jimena"}) {
		t.Fatalf("opening: %#v", opened)
	}
	path := "/v1/admin/cash-shifts/" + opened.ID
	request(s, api.unassignCashShiftUser, "DELETE", path+"/users/"+s.UserID, "", opened.ID, s.UserID, 204)
	removed := decode(request(s, api.getCashShift, "GET", path, "", opened.ID, "", 200))
	if removed.OpenedByName != "Jimena" || removed.ActiveUserNames == nil || len(removed.ActiveUserNames) != 0 {
		t.Fatalf("removal must leave empty team and keep opening audit: %#v", removed)
	}
	request(s, api.assignCashShiftUser, "POST", path+"/users", fmt.Sprintf(`{"userId":%q}`, edithID), opened.ID, "", 201)
	assigned := decode(request(s, api.getCashShift, "GET", path, "", opened.ID, "", 200))
	if !reflect.DeepEqual(assigned.ActiveUserNames, []string{"Edith"}) {
		t.Fatalf("replacement team: %#v", assigned)
	}
	registersResponse := request(s, api.listCashRegisters, "GET", "/v1/admin/cash-registers", "", "", "", 200)
	var registers struct {
		Items []cashRegisterView `json:"items"`
	}
	if err := json.Unmarshal(registersResponse.Body.Bytes(), &registers); err != nil {
		t.Fatal(err)
	}
	if len(registers.Items) != 1 || registers.Items[0].OpenShift == nil || !reflect.DeepEqual(registers.Items[0].OpenShift.ActiveUserNames, []string{"Edith"}) {
		t.Fatalf("register card must expose current team: %s", registersResponse.Body.String())
	}
	for _, status := range []string{"open", "closed"} {
		if status == "closed" {
			edith := scope{OrganizationID: s.OrganizationID, LocationID: s.LocationID, UserID: edithID, Name: "Edith"}
			closed := decode(request(edith, api.closeCashShift, "POST", path+"/close", `{"countedAmount":100}`, opened.ID, "", 200))
			if closed.OpenedByName != "Jimena" || closed.ClosedByName != "Edith" || len(closed.ActiveUserNames) != 0 {
				t.Fatalf("closing must identify Edith without overwriting Jimena: %#v", closed)
			}
			var opener, closer string
			if err := pool.QueryRow(ctx, `SELECT opened_by::text,closed_by::text FROM cash_shifts WHERE id=$1`, opened.ID).Scan(&opener, &closer); err != nil {
				t.Fatal(err)
			}
			if opener != s.UserID || closer != edithID {
				t.Fatalf("persisted attribution: %s / %s", opener, closer)
			}
		}
		list := request(s, api.listCashShifts, "GET", "/v1/admin/cash-shifts?q="+url.QueryEscape("Edith")+"&status="+status, "", "", "", 200)
		var result struct {
			Items []cashShiftView `json:"items"`
			Total int             `json:"total"`
		}
		if err := json.Unmarshal(list.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Total != 1 || len(result.Items) != 1 || result.Items[0].ID != opened.ID {
			t.Fatalf("search by active team/closer (%s): %s", status, list.Body.String())
		}
		if status == "open" && !reflect.DeepEqual(result.Items[0].ActiveUserNames, []string{"Edith"}) {
			t.Fatalf("list team: %#v", result.Items[0])
		}
		if status == "closed" && result.Items[0].ClosedByName != "Edith" {
			t.Fatalf("list closer: %#v", result.Items[0])
		}
	}
	other := seedInventoryScope(t, pool)
	request(other, api.getCashShift, "GET", path, "", opened.ID, "", 404)
}
