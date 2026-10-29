package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"

	billing "github.com/foods-platform/foods-backend/internal/modules/billing/domain"
)

func TestTipSettingsTargetLocationWithoutSwitchingSession(t *testing.T) {
	api, s, role := seedIdentityAdministrator(t)
	ctx := context.Background()
	if _, err := api.db.Exec(ctx, `UPDATE roles SET system_key='administrator',permissions=ARRAY['organizations.read','organizations.manage'],menu_access=ARRAY['locales'] WHERE id=$1`, role); err != nil {
		t.Fatal(err)
	}
	var target string
	if err := api.db.QueryRow(ctx, `INSERT INTO locations(organization_id,name,code,fiscal_profile_id) SELECT organization_id,'Local propinas','TIPS-TARGET',fiscal_profile_id FROM locations WHERE id=$1 RETURNING id`, s.LocationID).Scan(&target); err != nil {
		t.Fatal(err)
	}
	call := companySession(t, api, s)
	path := "/v1/admin/tip-settings?scope=location&locationId=" + target
	w := call("PUT", path, map[string]any{"mode": "collect", "percentage": 15, "allowFixed": true, "showSuggestion": true, "channels": []string{"salon"}, "inherit": false})
	if w.Code != 200 {
		t.Fatalf("target save: %d %s", w.Code, w.Body.String())
	}
	var selected billing.TipPolicy
	_ = json.Unmarshal(w.Body.Bytes(), &selected)
	if selected.Inherited || selected.Percentage != 15 || selected.Mode != "collect" {
		t.Fatalf("incorrect target policy: %s", w.Body.String())
	}
	w = call("GET", "/v1/admin/tip-settings", nil)
	var active billing.TipPolicy
	_ = json.Unmarshal(w.Body.Bytes(), &active)
	if w.Code != 200 || active.Mode != "disabled" {
		t.Fatalf("target save changed active local: %s", w.Body.String())
	}
	foreign := seedInventoryScope(t, api.db)
	w = call("GET", "/v1/admin/tip-settings?scope=location&locationId="+foreign.LocationID, nil)
	if w.Code != 404 {
		t.Fatalf("foreign location exposed: %d %s", w.Code, w.Body.String())
	}
	if _, err := api.db.Exec(ctx, `UPDATE roles SET system_key='cashier' WHERE id=$1`, role); err != nil {
		t.Fatal(err)
	}
	w = call("GET", path, nil)
	if w.Code != 403 {
		t.Fatalf("target location permission bypass: %d %s", w.Code, w.Body.String())
	}
}

func TestTipsCollectionPayoutIsolationAndClosingSnapshot(t *testing.T) {
	api, s, role := seedIdentityAdministrator(t)
	ctx := context.Background()
	if _, err := api.db.Exec(ctx, `UPDATE roles SET system_key='administrator',permissions=ARRAY['cash.read','cash.manage','cash.registers.manage','cash.expected.read','orders.read','orders.manage','organizations.read','organizations.manage'],menu_access=ARRAY['caja','pos','empresa'] WHERE id=$1`, role); err != nil {
		t.Fatal(err)
	}
	call := companySession(t, api, s)
	expect := func(w *httptest.ResponseRecorder, status int) {
		t.Helper()
		if w.Code != status {
			t.Fatalf("expected %d, got %d: %s", status, w.Code, w.Body.String())
		}
	}
	settings := map[string]any{"mode": "collect", "percentage": 10, "showSuggestion": true, "allowFixed": true, "channels": []string{"recojo", "salon", "delivery"}}
	expect(call("PUT", "/v1/admin/tip-settings?scope=organization", settings), 200)
	policy := call("GET", "/v1/admin/tip-settings", nil)
	expect(policy, 200)
	var p billing.TipPolicy
	_ = json.Unmarshal(policy.Body.Bytes(), &p)
	if !p.Inherited || p.Mode != "collect" {
		t.Fatalf("inheritance: %s", policy.Body.String())
	}
	var recipient string
	if err := api.db.QueryRow(ctx, `INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,$2,'Mozo propinas','test') RETURNING id`, s.OrganizationID, fmt.Sprintf("tips-%s@example.test", s.UserID)).Scan(&recipient); err != nil {
		t.Fatal(err)
	}
	if _, err := api.db.Exec(ctx, `WITH r AS (INSERT INTO roles(organization_id,name,system_key,permissions,menu_access) VALUES($1,'Mozo propinas','waiter',ARRAY['orders.read'],ARRAY['mesas']) RETURNING id) INSERT INTO user_roles(user_id,role_id,location_id) SELECT $2,id,$3 FROM r`, s.OrganizationID, recipient, s.LocationID); err != nil {
		t.Fatal(err)
	}
	reg := call("POST", "/v1/admin/cash-registers", map[string]any{"name": "Caja propinas"})
	expect(reg, 201)
	var cr cashRegisterView
	_ = json.Unmarshal(reg.Body.Bytes(), &cr)
	open := func() cashShiftView {
		w := call("POST", "/v1/admin/cash-shifts", map[string]any{"cashRegisterId": cr.ID, "openingAmount": 100})
		expect(w, 201)
		var v cashShiftView
		_ = json.Unmarshal(w.Body.Bytes(), &v)
		return v
	}
	shift := open()
	makeOrder := func(total, delivery float64) string {
		t.Helper()
		var id string
		if err := api.db.QueryRow(ctx, `INSERT INTO orders(organization_id,location_id,channel,status,subtotal,total,delivery_fee,created_by) VALUES($1,$2,'delivery','listo',$3,$4,$5,$6) RETURNING id`, s.OrganizationID, s.LocationID, total-delivery, total, delivery, s.UserID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	pay := func(order, method string, amount float64, tip any) *httptest.ResponseRecorder {
		in := map[string]any{"orderId": order, "method": method, "amount": amount}
		if tip != nil {
			in["tip"] = tip
		}
		return call("POST", "/v1/admin/payments", in)
	}
	order := makeOrder(110, 10)
	expect(pay(order, "cash", 50, map[string]any{"kind": "percentage", "value": 10}), 201)
	// Partial consumption payments cannot accidentally collect a second tip.
	expect(pay(order, "cash", 60, map[string]any{"kind": "fixed", "value": 5}), 409)
	expect(pay(order, "card", 60, nil), 201)
	expect(pay(makeOrder(20, 0), "card", 20, map[string]any{"kind": "fixed", "value": 5}), 201)
	expect(pay(makeOrder(10, 0), "cash", 10, map[string]any{"kind": "fixed", "value": 3, "directToStaff": true}), 201)
	detail := call("GET", "/v1/admin/pos/orders/"+order, nil)
	expect(detail, 200)
	var d struct {
		PaidAmount string           `json:"paidAmount"`
		Tip        *billing.TipView `json:"tip"`
	}
	_ = json.Unmarshal(detail.Body.Bytes(), &d)
	if d.PaidAmount != "110.00" || d.Tip == nil || d.Tip.Amount != "10.00" {
		t.Fatalf("consumption/delivery separation: %s", detail.Body.String())
	}
	workspace := func(id string) struct {
		Report    billing.TipReport `json:"report"`
		Available string            `json:"available"`
	} {
		w := call("GET", "/v1/admin/cash-shifts/"+id+"/tips", nil)
		expect(w, 200)
		var out struct {
			Report    billing.TipReport `json:"report"`
			Available string            `json:"available"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return out
	}
	w := workspace(shift.ID)
	if w.Report.CashCollected != "10.00" || w.Report.OtherCollected != "5.00" || w.Report.Direct != "3.00" || w.Available != "15.00" {
		t.Fatalf("report: %+v", w)
	}
	payout := func(id, key, method string, amount float64) *httptest.ResponseRecorder {
		return call("POST", "/v1/admin/cash-shifts/"+id+"/tip-payouts", map[string]any{"recipientId": recipient, "amount": amount, "method": method, "requestKey": key})
	}
	expect(payout(shift.ID, "payout-cash-1", "cash", 6), 201)
	expect(payout(shift.ID, "payout-cash-1", "cash", 6), 201)
	expect(payout(shift.ID, "payout-cash-1", "cash", 7), 409)
	expect(payout(shift.ID, "payout-transfer-1", "transfer", 2), 201)
	report := call("GET", "/v1/admin/cash-shifts/"+shift.ID+"/report", nil)
	expect(report, 200)
	var r cashShiftReport
	_ = json.Unmarshal(report.Body.Bytes(), &r)
	if r.CollectedAmount != "140.00" || r.Shift.ExpectedAmount != "160.00" || r.Tips.Pending != "7.00" {
		t.Fatalf("cash/report: %s", report.Body.String())
	}
	// Assigned waiters cannot use the cashier's financial tip workspace.
	waiterCall := companySession(t, api, scope{OrganizationID: s.OrganizationID, LocationID: s.LocationID, UserID: recipient})
	expect(waiterCall("GET", "/v1/admin/cash-shifts/"+shift.ID+"/tips", nil), 403)
	// Legacy tip ledger entries must also stay outside operational balances.
	if _, err := api.db.Exec(ctx, "INSERT INTO cash_movements(organization_id,location_id,shift_id,movement_type,source_type,amount,reason,created_by) VALUES($1,$2,$3,'income','cash_tip',90,'Legacy tip',$4),($1,$2,$3,'expense','tip_payout',20,'Legacy payout',$4)", s.OrganizationID, s.LocationID, shift.ID, s.UserID); err != nil {
		t.Fatal(err)
	}
	payoutAtClose := map[string]any{"recipientId": recipient, "amount": 2, "method": "cash", "requestKey": "closing-tip-payout"}
	closeInput := map[string]any{"countedAmount": 160, "tipPayout": payoutAtClose}
	// Fail after the payout is written: the whole transaction must roll back.
	if _, err := api.db.Exec(ctx, fmt.Sprintf("CREATE FUNCTION fail_tip_close() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.shift_id='%s'::uuid THEN RAISE EXCEPTION 'test report failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER fail_tip_close BEFORE INSERT ON cash_shift_reports FOR EACH ROW EXECUTE FUNCTION fail_tip_close()", shift.ID)); err != nil {
		t.Fatal(err)
	}
	expect(call("POST", "/v1/admin/cash-shifts/"+shift.ID+"/close", closeInput), 503)
	if workspace(shift.ID).Available != "7.00" {
		t.Fatal("failed close left a tip delivery")
	}
	if _, err := api.db.Exec(ctx, "DROP TRIGGER fail_tip_close ON cash_shift_reports; DROP FUNCTION fail_tip_close()"); err != nil {
		t.Fatal(err)
	}
	expect(call("POST", "/v1/admin/cash-shifts/"+shift.ID+"/close", closeInput), 200)
	expect(call("POST", "/v1/admin/cash-shifts/"+shift.ID+"/close", closeInput), 409)
	closed := call("GET", "/v1/admin/cash-shifts/"+shift.ID+"/report", nil)
	expect(closed, 200)
	var before cashShiftReport
	_ = json.Unmarshal(closed.Body.Bytes(), &before)
	next := open()
	expect(payout(next.ID, "payout-cash-1", "cash", 6), 409)
	// Two simultaneous payouts may not overdraw the remaining pooled tip balance.
	var wg sync.WaitGroup
	codes := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes <- payout(next.ID, fmt.Sprintf("concurrent-tip-%d", i), "transfer", 4).Code
		}(i)
	}
	wg.Wait()
	close(codes)
	success, conflict := 0, 0
	for code := range codes {
		if code == 201 {
			success++
		} else if code == 409 {
			conflict++
		} else {
			t.Fatalf("concurrent payout status %d", code)
		}
	}
	if success != 1 || conflict != 1 || workspace(next.ID).Available != "1.00" {
		t.Fatal("concurrent payouts overdrew tips")
	}
	after := call("GET", "/v1/admin/cash-shifts/"+shift.ID+"/report", nil)
	expect(after, 200)
	var frozen cashShiftReport
	_ = json.Unmarshal(after.Body.Bytes(), &frozen)
	if !frozen.Persisted || frozen.Tips.Pending != before.Tips.Pending || frozen.Tips.Pending != "5.00" {
		t.Fatal("a later payout changed the stored closing snapshot")
	}
	expect(payout(shift.ID, "closed-tip-payout", "cash", 1), 403)

	// Server-side configuration checks are atomic with the consumption payment.
	settings["allowFixed"] = false
	expect(call("PUT", "/v1/admin/tip-settings", settings), 200)
	badOrder := makeOrder(10, 0)
	expect(pay(badOrder, "cash", 10, map[string]any{"kind": "fixed", "value": 5}), 400)
	var count int
	if err := api.db.QueryRow(ctx, `SELECT count(*) FROM payments WHERE order_id=$1`, badOrder).Scan(&count); err != nil || count != 0 {
		t.Fatalf("rejected tip left a consumption payment: %d %v", count, err)
	}
	settings["allowFixed"] = true
	settings["mode"] = "suggest"
	expect(call("PUT", "/v1/admin/tip-settings", settings), 200)
	expect(pay(badOrder, "cash", 10, map[string]any{"kind": "percentage", "value": 10}), 400)
	settings["mode"] = "collect"
	expect(call("PUT", "/v1/admin/tip-settings", settings), 200)
	expect(pay(badOrder, "cash", 10, map[string]any{"kind": "fixed", "value": 5.001}), 400)
	// Removing the local override restores the organization policy.
	expect(call("PUT", "/v1/admin/tip-settings", map[string]any{"inherit": true}), 200)

	var cashier string
	if err := api.db.QueryRow(ctx, `INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,$2,'Otra cajera','test') RETURNING id`, s.OrganizationID, fmt.Sprintf("tips-cashier-%s@example.test", s.UserID)).Scan(&cashier); err != nil {
		t.Fatal(err)
	}
	if _, err := api.db.Exec(ctx, `WITH r AS (INSERT INTO roles(organization_id,name,system_key,permissions,menu_access) VALUES($1,'Caja propinas','cashier',ARRAY['cash.read','cash.manage','organizations.manage','organizations.read'],ARRAY['caja','pos']) RETURNING id) INSERT INTO user_roles(user_id,role_id,location_id) SELECT $2,id,$3 FROM r`, s.OrganizationID, cashier, s.LocationID); err != nil {
		t.Fatal(err)
	}
	cashierCall := companySession(t, api, scope{OrganizationID: s.OrganizationID, LocationID: s.LocationID, UserID: cashier})
	expect(cashierCall("PUT", "/v1/admin/tip-settings", settings), 403)
	expect(cashierCall("GET", "/v1/admin/cash-shifts/"+next.ID+"/tips", nil), 404)
	reg2 := call("POST", "/v1/admin/cash-registers", map[string]any{"name": "Caja otra cajera"})
	expect(reg2, 201)
	var cr2 cashRegisterView
	_ = json.Unmarshal(reg2.Body.Bytes(), &cr2)
	opened2 := cashierCall("POST", "/v1/admin/cash-shifts", map[string]any{"cashRegisterId": cr2.ID, "openingAmount": 0})
	expect(opened2, 201)
	var shift2 cashShiftView
	_ = json.Unmarshal(opened2.Body.Bytes(), &shift2)
	private := cashierCall("GET", "/v1/admin/cash-shifts/"+shift2.ID+"/tips", nil)
	expect(private, 200)
	var privateBalance struct {
		Available string `json:"available"`
	}
	_ = json.Unmarshal(private.Body.Bytes(), &privateBalance)
	if privateBalance.Available != "0" {
		t.Fatalf("other cashier's pending tips leaked: %s", private.Body.String())
	}
	expect(cashierCall("POST", "/v1/admin/cash-shifts/"+shift2.ID+"/tip-payouts", map[string]any{"recipientId": recipient, "amount": 1, "method": "transfer", "requestKey": "foreign-source-payout"}), 409)
	foreign := seedInventoryScope(t, api.db)
	expect(call("POST", "/v1/admin/cash-shifts/"+next.ID+"/tip-payouts", map[string]any{"recipientId": foreign.UserID, "amount": 1, "method": "transfer", "requestKey": "foreign-recipient"}), 400)
	expect(call("GET", "/v1/admin/cash-shifts/"+foreign.UserID+"/tips", nil), 404)
}
