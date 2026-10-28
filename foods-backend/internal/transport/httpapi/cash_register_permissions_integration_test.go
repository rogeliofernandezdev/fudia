package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	identitydomain "github.com/foods-platform/foods-backend/internal/modules/identity/domain"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestCashRegisterAdministrationIsSeparateFromShiftOperation(t *testing.T) {
	api, s, administratorRole := seedIdentityAdministrator(t)
	ctx := context.Background()
	if _, err := api.db.Exec(ctx, `UPDATE roles SET system_key='administrator',permissions=$2,menu_access=$3 WHERE id=$1`, administratorRole, identitydomain.CompanyAdministratorPermissions, identitydomain.CompanyAdministratorMenuAccess); err != nil {
		t.Fatal(err)
	}
	var cashierRole, cashierID string
	if err := api.db.QueryRow(ctx, `INSERT INTO roles(organization_id,name,system_key,permissions,menu_access) VALUES($1,'Cajero','cashier',ARRAY['cash.read','cash.manage','orders.read'],ARRAY['caja','pos']) RETURNING id`, s.OrganizationID).Scan(&cashierRole); err != nil {
		t.Fatal(err)
	}
	if err := api.db.QueryRow(ctx, `INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,$2,'Cajero','test') RETURNING id`, s.OrganizationID, fmt.Sprintf("cash-register-%d@example.test", time.Now().UnixNano())).Scan(&cashierID); err != nil {
		t.Fatal(err)
	}
	if _, err := api.db.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, cashierID, cashierRole, s.LocationID); err != nil {
		t.Fatal(err)
	}
	cashier := scope{OrganizationID: s.OrganizationID, LocationID: s.LocationID, UserID: cashierID, Name: "Cajero"}
	administratorCall := companySession(t, api, s)
	cashierCall := companySession(t, api, cashier)
	expect := func(rec *httptest.ResponseRecorder, status int) {
		t.Helper()
		if rec.Code != status {
			t.Fatalf("want %d got %d: %s", status, rec.Code, rec.Body.String())
		}
	}
	registered := administratorCall("POST", "/v1/admin/cash-registers", map[string]any{"name": "Caja del administrador", "blindClose": false})
	expect(registered, 201)
	var register struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(registered.Body.Bytes(), &register); err != nil {
		t.Fatal(err)
	}
	expect(cashierCall("POST", "/v1/admin/cash-registers", map[string]any{"name": "Caja no autorizada"}), 403)
	expect(cashierCall("PATCH", "/v1/admin/cash-registers/"+register.ID, map[string]any{"name": "Cambio no autorizado"}), 403)
	expect(cashierCall("PATCH", "/v1/admin/cash-registers/"+register.ID+"/status", map[string]any{"active": false}), 403)
	var count int
	var name string
	var active bool
	if err := api.db.QueryRow(ctx, `SELECT (SELECT count(*) FROM cash_registers WHERE organization_id=$1),name,active FROM cash_registers WHERE id=$2`, s.OrganizationID, register.ID).Scan(&count, &name, &active); err != nil || count != 1 || name != "Caja del administrador" || !active {
		t.Fatalf("cashier modified registers: %d %s %v %v", count, name, active, err)
	}
	expect(administratorCall("PATCH", "/v1/admin/cash-registers/"+register.ID, map[string]any{"name": "Caja configurada", "blindClose": false}), 200)
	expect(administratorCall("PATCH", "/v1/admin/cash-registers/"+register.ID+"/status", map[string]any{"active": false}), 204)
	expect(administratorCall("PATCH", "/v1/admin/cash-registers/"+register.ID+"/status", map[string]any{"active": true}), 204)
	expect(cashierCall("GET", "/v1/admin/cash-registers", nil), 200)
	opened := cashierCall("POST", "/v1/admin/cash-shifts", map[string]any{"cashRegisterId": register.ID, "openingAmount": 0})
	expect(opened, 201)
	var shift struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(opened.Body.Bytes(), &shift); err != nil {
		t.Fatal(err)
	}
	expect(administratorCall("PATCH", "/v1/admin/cash-registers/"+register.ID+"/status", map[string]any{"active": false}), 409)
	expect(cashierCall("POST", "/v1/admin/cash-shifts/"+shift.ID+"/close", map[string]any{"countedAmount": 0}), 200)
	var permissions []string
	if err := api.db.QueryRow(ctx, `SELECT permissions FROM roles WHERE id=$1`, cashierRole).Scan(&permissions); err != nil || containsString(permissions, "cash.registers.manage") || !containsString(permissions, "cash.manage") {
		t.Fatalf("cashier permissions changed: %v %v", permissions, err)
	}
}

func TestCashRegisterPermissionMigrationOnlyGrantsReservedAdministrators(t *testing.T) {
	api, s, administratorRole := seedIdentityAdministrator(t)
	ctx := context.Background()
	tx, err := api.db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE roles SET system_key='administrator',permissions=ARRAY['cash.read','cash.manage'] WHERE id=$1`, administratorRole); err != nil {
		t.Fatal(err)
	}
	var custom string
	if err := tx.QueryRow(ctx, `INSERT INTO roles(organization_id,name,permissions) VALUES($1,'Administrador de empresa',ARRAY['cash.manage','users.manage']) RETURNING id`, s.OrganizationID).Scan(&custom); err != nil {
		t.Fatal(err)
	}
	up, err := os.ReadFile("../../../migrations/000094_cash_register_administration.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../../migrations/000094_cash_register_administration.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		if _, err := tx.Exec(ctx, string(up)); err != nil {
			t.Fatal(err)
		}
	}
	var administrators, customGrants int
	if err := tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM unnest(permissions) AS p(permission) WHERE p.permission='cash.registers.manage') FROM roles WHERE id=$1`, administratorRole).Scan(&administrators); err != nil || administrators != 1 {
		t.Fatalf("administrator grant not idempotent: %d %v", administrators, err)
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM roles WHERE id=$1 AND 'cash.registers.manage'=ANY(permissions)`, custom).Scan(&customGrants); err != nil || customGrants != 0 {
		t.Fatalf("custom role gained administration: %d %v", customGrants, err)
	}
	if _, err := tx.Exec(ctx, string(down)); err != nil {
		t.Fatal(err)
	}
	var retained, removed bool
	if err := tx.QueryRow(ctx, `SELECT 'cash.manage'=ANY(permissions),'cash.registers.manage'=ANY(permissions) FROM roles WHERE id=$1`, administratorRole).Scan(&retained, &removed); err != nil || !retained || removed {
		t.Fatalf("rollback changed operational authority: %v %v %v", retained, removed, err)
	}
}
