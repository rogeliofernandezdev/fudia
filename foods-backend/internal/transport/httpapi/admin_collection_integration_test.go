package httpapi

import (
	"context"
	"encoding/json"
	"testing"
)

func TestAdministratorSelectsOpenCollectionShiftWithoutAssignment(t *testing.T) {
	api, s, role := seedIdentityAdministrator(t)
	ctx := context.Background()
	if _, err := api.db.Exec(ctx, `UPDATE roles SET system_key='administrator',permissions=ARRAY['cash.read','cash.manage','orders.read'],menu_access=ARRAY['pos','caja'] WHERE id=$1`, role); err != nil {
		t.Fatal(err)
	}
	call := companySession(t, api, s)
	check := func(method, path string, body any, status int) []byte {
		t.Helper()
		w := call(method, path, body)
		if w.Code != status {
			t.Fatalf("%s %s: want %d got %d %s", method, path, status, w.Code, w.Body.String())
		}
		return w.Body.Bytes()
	}
	var cashier string
	if err := api.db.QueryRow(ctx, `INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,'cashier-'||gen_random_uuid()||'@example.test','Madison','test') RETURNING id`, s.OrganizationID).Scan(&cashier); err != nil {
		t.Fatal(err)
	}
	opened := 0
	open := func(name string) string {
		owner := cashier
		if opened > 0 {
			if err := api.db.QueryRow(ctx, `INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,gen_random_uuid()||'@example.test','Ruth','test') RETURNING id`, s.OrganizationID).Scan(&owner); err != nil {
				t.Fatal(err)
			}
		}
		opened++
		t.Helper()
		var id string
		if err := api.db.QueryRow(ctx, `WITH r AS (INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,$3,$4) RETURNING id) INSERT INTO cash_shifts(organization_id,location_id,cash_register_id,opened_by,business_date) SELECT $1,$2,id,$4,current_date FROM r RETURNING id`, s.OrganizationID, s.LocationID, name, owner).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	order := func() string {
		t.Helper()
		var id string
		if err := api.db.QueryRow(ctx, `INSERT INTO orders(organization_id,location_id,channel,status,total,created_by) VALUES($1,$2,'recojo','listo',10,$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	access := func() map[string]any {
		t.Helper()
		var out map[string]any
		if err := json.Unmarshal(check("GET", "/v1/admin/pos/access", nil, 200), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	if a := access(); a["canChooseShift"] != true || a["shift"] != nil || len(a["shifts"].([]any)) != 0 {
		t.Fatalf("zero shifts: %+v", a)
	}
	check("POST", "/v1/admin/payments", map[string]any{"orderId": order(), "method": "cash", "amount": 10}, 409)
	first := open("Caja Madison")
	if a := access(); a["shift"].(map[string]any)["id"] != first || a["assignedShift"] != nil {
		t.Fatalf("single shift should be automatic without assignment: %+v", a)
	}
	var payment struct{ ShiftID, CreatedByName string }
	if err := json.Unmarshal(check("POST", "/v1/admin/payments", map[string]any{"orderId": order(), "method": "cash", "amount": 10}, 201), &payment); err != nil {
		t.Fatal(err)
	}
	if payment.ShiftID != first || payment.CreatedByName == "Madison" || payment.CreatedByName == "" {
		t.Fatalf("payment actor lost: %+v", payment)
	}
	second := open("Otra caja")
	if a := access(); a["shift"] != nil || len(a["shifts"].([]any)) != 2 {
		t.Fatalf("multiple shifts need selection: %+v", a)
	}
	check("POST", "/v1/admin/payments", map[string]any{"orderId": order(), "method": "cash", "amount": 10}, 409)
	check("POST", "/v1/admin/payments", map[string]any{"orderId": order(), "method": "cash", "amount": 10, "shiftId": second}, 201)
	check("POST", "/v1/operations/payments/batch", map[string]any{"orderId": order(), "shiftId": first, "payments": []map[string]any{{"method": "cash", "amount": 4}, {"method": "card", "amount": 6}}}, 201)
	foreign := seedInventoryScope(t, api.db)
	check("POST", "/v1/admin/payments", map[string]any{"orderId": order(), "method": "cash", "amount": 10, "shiftId": foreign.LocationID}, 409)
	var assignmentCount int
	var owner string
	if err := api.db.QueryRow(ctx, `SELECT opened_by::text,(SELECT count(*) FROM cash_shift_users WHERE user_id=$2 AND shift_id IN ($1,$3)) FROM cash_shifts WHERE id=$1`, first, s.UserID, second).Scan(&owner, &assignmentCount); err != nil || owner != cashier || assignmentCount != 0 {
		t.Fatalf("administrator changed ownership/team: %s %d %v", owner, assignmentCount, err)
	}
	check("POST", "/v1/admin/cash-shifts/"+first+"/close", map[string]any{"countedAmount": 14}, 403)
	if _, err := api.db.Exec(ctx, `UPDATE cash_shifts SET status='closed',closed_by=opened_by,closed_at=now(),closing_expected_amount=10,closing_counted_amount=10,variance_amount=0 WHERE id=$1`, second); err != nil {
		t.Fatal(err)
	}
	check("POST", "/v1/admin/payments", map[string]any{"orderId": order(), "method": "cash", "amount": 10, "shiftId": second}, 409)
	// cash.manage by itself must never grant the selection capability.
	if _, err := api.db.Exec(ctx, `UPDATE roles SET system_key='cashier' WHERE id=$1`, role); err != nil {
		t.Fatal(err)
	}
	if a := access(); a["canChooseShift"] != false || len(a["shifts"].([]any)) != 0 {
		t.Fatalf("cashier saw selectable shifts: %+v", a)
	}
	check("POST", "/v1/admin/payments", map[string]any{"orderId": order(), "method": "cash", "amount": 10, "shiftId": first}, 403)
}
