package httpapi

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestHomologatedPlanModulesMigration(t *testing.T) {
	pool := integrationPool(t)
	supplied, bare, cancelled, growth, large := seedInventoryScope(t, pool), seedInventoryScope(t, pool), seedInventoryScope(t, pool), seedInventoryScope(t, pool), seedInventoryScope(t, pool)
	ctx := context.Background()
	migration := func(direction string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "000090_homologated_plan_modules."+direction+".sql"))
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
	// Previous catalog: Emprende carried supply while Impulso lacked purchases.
	previous := map[string][]string{
		"emprende": {"reportes", "pos", "pedidos", "cocina", "mesas", "caja", "productos", "clientes", "locales", "fiscal", "usuarios", "combos", "inventario", "compras"},
		"impulso":  {"reportes", "pos", "pedidos", "cocina", "mesas", "caja", "reservas", "productos", "combos", "recetas", "inventario", "kardex", "clientes", "locales", "fiscal", "usuarios"},
		"escala":   {"reportes", "pos", "pedidos", "cocina", "mesas", "caja", "reservas", "productos", "combos", "recetas", "inventario", "kardex", "compras", "clientes", "locales", "fiscal", "usuarios", "costos"},
	}
	exec(`DROP TABLE migration_090_plan_backup`)
	exec(`DELETE FROM subscription_plans WHERE code='emprende-abastecimiento'`)
	planIDs := map[string]string{}
	for code, keys := range previous {
		var id string
		if err := tx.QueryRow(ctx, `UPDATE subscription_plans SET module_keys=$2 WHERE code=$1 RETURNING id`, code, keys).Scan(&id); err != nil {
			t.Fatal(err)
		}
		planIDs[code] = id
	}
	var plansBefore string
	if err := tx.QueryRow(ctx, `SELECT jsonb_agg(to_jsonb(p)-'module_keys'-'updated_at' ORDER BY code)::text FROM subscription_plans p WHERE code IN ('emprende','impulso','escala')`).Scan(&plansBefore); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		s    scope
		code string
	}{{supplied, "emprende"}, {bare, "emprende"}, {cancelled, "emprende"}, {growth, "impulso"}, {large, "escala"}} {
		if err := createOrganizationSubscription(ctx, tx, item.s.OrganizationID, item.s.UserID, planIDs[item.code], "monthly", true); err != nil {
			t.Fatal(err)
		}
	}
	exec(`UPDATE organization_modules SET active=false WHERE organization_id=$1 AND module_key IN ('inventario','compras')`, bare.OrganizationID)
	exec(`UPDATE organization_subscriptions SET status='cancelled' WHERE organization_id=$1`, cancelled.OrganizationID)
	exec(`UPDATE organization_subscriptions SET status='past_due' WHERE organization_id=$1`, large.OrganizationID)
	exec(`DELETE FROM organization_modules WHERE organization_id=$1 AND module_key='compras'`, growth.OrganizationID)

	active := func(org, key string) bool {
		t.Helper()
		var value bool
		if err := tx.QueryRow(ctx, `SELECT COALESCE((SELECT active FROM organization_modules WHERE organization_id=$1 AND module_key=$2),false)`, org, key).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	planOf := func(org string) string {
		t.Helper()
		var code string
		if err := tx.QueryRow(ctx, `SELECT p.code FROM organization_subscriptions s JOIN subscription_plans p ON p.id=s.plan_id WHERE s.organization_id=$1`, org).Scan(&code); err != nil {
			t.Fatal(err)
		}
		return code
	}
	keysOf := func(code string) []string {
		t.Helper()
		var keys []string
		if err := tx.QueryRow(ctx, `SELECT module_keys FROM subscription_plans WHERE code=$1`, code).Scan(&keys); err != nil {
			t.Fatal(err)
		}
		return keys
	}

	exec(migration("up"))

	emprende := []string{"reportes", "pos", "pedidos", "cocina", "mesas", "caja", "productos", "combos", "clientes", "locales", "fiscal", "usuarios", "whatsapp_bot"}
	impulso := []string{"reportes", "pos", "pedidos", "cocina", "mesas", "caja", "reservas", "productos", "combos", "recetas", "inventario", "kardex", "compras", "clientes", "locales", "costos", "fiscal", "usuarios", "whatsapp_bot"}
	if got := keysOf("emprende"); !slices.Equal(got, emprende) {
		t.Fatalf("Emprende modules: %v", got)
	}
	for _, code := range []string{"impulso", "escala"} {
		if got := keysOf(code); !slices.Equal(got, impulso) {
			t.Fatalf("%s modules: %v", code, got)
		}
	}
	// Each plan contains every module of the previous one.
	for _, key := range emprende {
		if !slices.Contains(impulso, key) {
			t.Fatalf("Impulso drops Emprende module %s", key)
		}
	}
	var plansAfter string
	if err := tx.QueryRow(ctx, `SELECT jsonb_agg(to_jsonb(p)-'module_keys'-'updated_at' ORDER BY code)::text FROM subscription_plans p WHERE code IN ('emprende','impulso','escala')`).Scan(&plansAfter); err != nil {
		t.Fatal(err)
	}
	if plansAfter != plansBefore {
		t.Fatal("changed prices, limits or descriptions")
	}
	var legacyActive bool
	var legacyPrice, emprendePrice string
	if err := tx.QueryRow(ctx, `SELECT g.active,g.monthly_price::text,e.monthly_price::text FROM subscription_plans g,subscription_plans e WHERE g.code='emprende-abastecimiento' AND e.code='emprende'`).Scan(&legacyActive, &legacyPrice, &emprendePrice); err != nil {
		t.Fatal(err)
	}
	if legacyActive || legacyPrice != emprendePrice {
		t.Fatal("grandfathered plan must be non-sellable and keep the Emprende price")
	}
	grandfathered := keysOf("emprende-abastecimiento")
	for _, key := range []string{"inventario", "compras", "combos", "whatsapp_bot"} {
		if !slices.Contains(grandfathered, key) {
			t.Fatalf("grandfathered plan missing %s", key)
		}
	}

	if planOf(supplied.OrganizationID) != "emprende-abastecimiento" || planOf(bare.OrganizationID) != "emprende" || planOf(cancelled.OrganizationID) != "emprende" {
		t.Fatal("wrong grandfathering of Emprende subscribers")
	}
	for _, key := range []string{"inventario", "compras", "whatsapp_bot"} {
		if !active(supplied.OrganizationID, key) {
			t.Fatalf("grandfathered company lost %s", key)
		}
	}
	if active(bare.OrganizationID, "inventario") || active(bare.OrganizationID, "compras") || !active(bare.OrganizationID, "whatsapp_bot") {
		t.Fatal("wrong Emprende modules")
	}
	if active(cancelled.OrganizationID, "whatsapp_bot") || !active(cancelled.OrganizationID, "inventario") {
		t.Fatal("cancelled subscription changed")
	}
	for _, key := range []string{"compras", "costos", "whatsapp_bot"} {
		if !active(growth.OrganizationID, key) {
			t.Fatalf("Impulso missing %s", key)
		}
	}
	if !active(large.OrganizationID, "whatsapp_bot") {
		t.Fatal("Escala missing Concierge")
	}

	exec(migration("down"))
	for code, keys := range previous {
		if got := keysOf(code); !slices.Equal(got, keys) {
			t.Fatalf("rollback changed %s modules: %v", code, got)
		}
	}
	var leftovers int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM subscription_plans WHERE code='emprende-abastecimiento'`).Scan(&leftovers); err != nil || leftovers != 0 {
		t.Fatal("rollback retained grandfathered plan")
	}
	if planOf(supplied.OrganizationID) != "emprende" {
		t.Fatal("rollback did not restore the subscription plan")
	}
	for _, s := range []scope{supplied, bare, growth, large} {
		if active(s.OrganizationID, "whatsapp_bot") {
			t.Fatal("rollback retained Concierge activation")
		}
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM organization_modules WHERE organization_id=$1 AND module_key='compras'`, growth.OrganizationID).Scan(&count); err != nil || count != 0 {
		t.Fatal("rollback retained inserted modules")
	}
	if !active(supplied.OrganizationID, "inventario") || active(growth.OrganizationID, "costos") {
		t.Fatal("rollback changed prior flags")
	}

	// Rollback must not overwrite administrative edits made afterwards.
	exec(migration("up"))
	exec(`UPDATE subscription_plans SET module_keys=array_append(module_keys,'custom-test-module'),updated_at=clock_timestamp()+interval '1 second' WHERE code='impulso'`)
	exec(`UPDATE organization_modules SET active=false,updated_at=clock_timestamp()+interval '1 second' WHERE organization_id=$1 AND module_key='whatsapp_bot'`, large.OrganizationID)
	exec(migration("down"))
	if got := keysOf("impulso"); !slices.Contains(got, "custom-test-module") || !slices.Contains(got, "compras") {
		t.Fatalf("rollback overwrote later plan edit: %v", got)
	}
	var updatedLater bool
	if err := tx.QueryRow(ctx, `SELECT updated_at>now() FROM organization_modules WHERE organization_id=$1 AND module_key='whatsapp_bot'`, large.OrganizationID).Scan(&updatedLater); err != nil || !updatedLater {
		t.Fatal("rollback overwrote later module edit")
	}
}
