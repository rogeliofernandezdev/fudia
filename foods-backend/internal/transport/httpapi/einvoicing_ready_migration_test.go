package httpapi

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestEInvoicingReadyMigration(t *testing.T) {
	pool := integrationPool(t)
	growth, large, pilot, cancelled := seedInventoryScope(t, pool), seedInventoryScope(t, pool), seedInventoryScope(t, pool), seedInventoryScope(t, pool)
	ctx := context.Background()
	migration := func(direction string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "000099_einvoicing_ready."+direction+".sql"))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	keysOf := func(code string) []string {
		t.Helper()
		var keys []string
		if err := tx.QueryRow(ctx, `SELECT module_keys FROM subscription_plans WHERE code=$1`, code).Scan(&keys); err != nil {
			t.Fatal(err)
		}
		return keys
	}
	active := func(org string) bool {
		t.Helper()
		var value bool
		if err := tx.QueryRow(ctx, `SELECT COALESCE((SELECT active FROM organization_modules WHERE organization_id=$1 AND module_key='facturacion'),false)`, org).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	rows := func(org string) int {
		t.Helper()
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM organization_modules WHERE organization_id=$1 AND module_key='facturacion'`, org).Scan(&count); err != nil {
			t.Fatal(err)
		}
		return count
	}

	// State before the migration: no plan includes facturación; one company pilots it.
	exec(migration("down"))
	previous := map[string][]string{}
	planIDs := map[string]string{}
	for _, code := range []string{"emprende", "impulso", "escala"} {
		var id string
		if err := tx.QueryRow(ctx, `UPDATE subscription_plans SET module_keys=array_remove(module_keys,'facturacion') WHERE code=$1 RETURNING id`, code).Scan(&id); err != nil {
			t.Fatal(err)
		}
		planIDs[code] = id
		previous[code] = keysOf(code)
	}
	for _, item := range []struct {
		s    scope
		code string
	}{{growth, "impulso"}, {large, "escala"}, {pilot, "emprende"}, {cancelled, "impulso"}} {
		if err := createOrganizationSubscription(ctx, tx, item.s.OrganizationID, item.s.UserID, planIDs[item.code], "monthly", true); err != nil {
			t.Fatal(err)
		}
	}
	exec(`DELETE FROM organization_modules WHERE organization_id=$1 AND module_key='facturacion'`, growth.OrganizationID)
	exec(`INSERT INTO organization_modules(organization_id,module_key,active) VALUES($1,'facturacion',false)
		ON CONFLICT(organization_id,module_key) DO UPDATE SET active=false`, large.OrganizationID)
	exec(`INSERT INTO organization_modules(organization_id,module_key,active) VALUES($1,'facturacion',true)
		ON CONFLICT(organization_id,module_key) DO UPDATE SET active=true`, pilot.OrganizationID)
	exec(`UPDATE organization_subscriptions SET status='cancelled' WHERE organization_id=$1`, cancelled.OrganizationID)
	exec(`DELETE FROM organization_modules WHERE organization_id=$1 AND module_key='facturacion'`, cancelled.OrganizationID)

	exec(migration("up"))
	for _, code := range []string{"impulso", "escala"} {
		if got := keysOf(code); !slices.Contains(got, "facturacion") || len(got) != len(previous[code])+1 {
			t.Fatalf("%s must add facturacion only: %v", code, got)
		}
	}
	if slices.Contains(keysOf("emprende"), "facturacion") {
		t.Fatal("Emprende must not include facturacion")
	}
	if !active(growth.OrganizationID) || !active(large.OrganizationID) {
		t.Fatal("Impulso and Escala subscribers must get facturacion")
	}
	if !active(pilot.OrganizationID) || active(cancelled.OrganizationID) {
		t.Fatal("other subscriptions must stay as they were")
	}

	exec(migration("down"))
	for _, code := range []string{"impulso", "escala"} {
		if got := keysOf(code); !slices.Equal(got, previous[code]) {
			t.Fatalf("rollback changed %s modules: %v", code, got)
		}
	}
	if rows(growth.OrganizationID) != 0 || active(large.OrganizationID) || !active(pilot.OrganizationID) {
		t.Fatal("rollback must restore the previous module rows")
	}

	// Rollback must not overwrite administrative edits made afterwards.
	exec(migration("up"))
	exec(`UPDATE subscription_plans SET module_keys=array_append(module_keys,'custom-test-module'),updated_at=clock_timestamp()+interval '1 second' WHERE code='impulso'`)
	exec(`UPDATE organization_modules SET active=false,updated_at=clock_timestamp()+interval '1 second' WHERE organization_id=$1 AND module_key='facturacion'`, large.OrganizationID)
	exec(migration("down"))
	if got := keysOf("impulso"); !slices.Contains(got, "custom-test-module") || !slices.Contains(got, "facturacion") {
		t.Fatalf("rollback overwrote later plan edit: %v", got)
	}
	if rows(large.OrganizationID) != 1 || rows(growth.OrganizationID) != 0 {
		t.Fatal("rollback overwrote later module edit")
	}
}
