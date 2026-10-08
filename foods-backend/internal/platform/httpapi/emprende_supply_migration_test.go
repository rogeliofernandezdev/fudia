package httpapi

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestEmprendeInventoryPurchasesMigration(t *testing.T) {
	pool := integrationPool(t)
	existing, missing, enabled, cancelled, other := seedInventoryScope(t, pool), seedInventoryScope(t, pool), seedInventoryScope(t, pool), seedInventoryScope(t, pool), seedInventoryScope(t, pool)
	ctx := context.Background()
	migration := func(direction string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "000075_emprende_inventory_purchases."+direction+".sql"))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	for _, before := range [][]string{{}, {"inventario"}, {"compras"}, {"inventario", "compras"}} {
		t.Run("prior-"+strings.Join(before, "-"), func(t *testing.T) {
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
			exec(`DROP TABLE migration_075_module_backup`)
			exec(`UPDATE subscription_plans SET module_keys=array_remove(array_remove(module_keys,'inventario'),'compras') || $1::text[] WHERE code='emprende'`, before)
			var planID, otherPlanID, planBefore, otherPlansBefore string
			var original []string
			if err := tx.QueryRow(ctx, `SELECT id,module_keys,(to_jsonb(p)-'module_keys'-'updated_at')::text FROM subscription_plans p WHERE code='emprende'`).Scan(&planID, &original, &planBefore); err != nil {
				t.Fatal(err)
			}
			if err := tx.QueryRow(ctx, `SELECT id FROM subscription_plans WHERE code='impulso'`).Scan(&otherPlanID); err != nil {
				t.Fatal(err)
			}
			if err := tx.QueryRow(ctx, `SELECT jsonb_agg(to_jsonb(p) ORDER BY code)::text FROM subscription_plans p WHERE code<>'emprende'`).Scan(&otherPlansBefore); err != nil {
				t.Fatal(err)
			}
			for _, s := range []scope{existing, missing, enabled, cancelled, other} {
				id := planID
				if s.OrganizationID == other.OrganizationID {
					id = otherPlanID
				}
				if err := createOrganizationSubscription(ctx, tx, s.OrganizationID, s.UserID, id, "monthly", true); err != nil {
					t.Fatal(err)
				}
			}
			exec(`UPDATE organization_modules SET active=false WHERE organization_id=ANY($1::uuid[]) AND module_key IN ('inventario','compras')`, []string{existing.OrganizationID, cancelled.OrganizationID, other.OrganizationID})
			exec(`DELETE FROM organization_modules WHERE organization_id=$1 AND module_key IN ('inventario','compras')`, missing.OrganizationID)
			exec(`UPDATE organization_subscriptions SET status='active' WHERE organization_id=$1`, existing.OrganizationID)
			exec(`UPDATE organization_subscriptions SET status='past_due' WHERE organization_id=$1`, enabled.OrganizationID)
			exec(`UPDATE organization_subscriptions SET status='cancelled' WHERE organization_id=$1`, cancelled.OrganizationID)
			exec(`UPDATE organization_modules SET active=true WHERE organization_id=$1 AND module_key IN ('inventario','compras')`, enabled.OrganizationID)
			var subscriptionsBefore string
			if err := tx.QueryRow(ctx, `SELECT jsonb_agg(to_jsonb(s) ORDER BY organization_id)::text FROM organization_subscriptions s WHERE organization_id=ANY($1::uuid[])`, []string{existing.OrganizationID, missing.OrganizationID, enabled.OrganizationID, cancelled.OrganizationID, other.OrganizationID}).Scan(&subscriptionsBefore); err != nil {
				t.Fatal(err)
			}
			exec(migration("up"))
			exec(migration("up")) // Repeating the data operation cannot duplicate modules.
			var modules []string
			var planAfter, otherPlansAfter, subscriptionsAfter string
			if err := tx.QueryRow(ctx, `SELECT module_keys,(to_jsonb(p)-'module_keys'-'updated_at')::text FROM subscription_plans p WHERE id=$1`, planID).Scan(&modules, &planAfter); err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"inventario", "compras"} {
				count := 0
				for _, value := range modules {
					if value == key {
						count++
					}
				}
				if count != 1 {
					t.Fatalf("expected exactly one %s, got %v", key, modules)
				}
			}
			for _, key := range original {
				if !slices.Contains(modules, key) {
					t.Fatalf("removed existing plan module %s", key)
				}
			}
			if err := tx.QueryRow(ctx, `SELECT jsonb_agg(to_jsonb(p) ORDER BY code)::text FROM subscription_plans p WHERE code<>'emprende'`).Scan(&otherPlansAfter); err != nil {
				t.Fatal(err)
			}
			if err := tx.QueryRow(ctx, `SELECT jsonb_agg(to_jsonb(s) ORDER BY organization_id)::text FROM organization_subscriptions s WHERE organization_id=ANY($1::uuid[])`, []string{existing.OrganizationID, missing.OrganizationID, enabled.OrganizationID, cancelled.OrganizationID, other.OrganizationID}).Scan(&subscriptionsAfter); err != nil {
				t.Fatal(err)
			}
			if planAfter != planBefore || otherPlansAfter != otherPlansBefore || subscriptionsAfter != subscriptionsBefore {
				t.Fatal("changed prices, limits, other plans or subscription contracts")
			}
			active := func(org, key string) bool {
				t.Helper()
				var value bool
				if err := tx.QueryRow(ctx, `SELECT active FROM organization_modules WHERE organization_id=$1 AND module_key=$2`, org, key).Scan(&value); err != nil {
					t.Fatal(err)
				}
				return value
			}
			for _, key := range []string{"inventario", "compras"} {
				if !active(existing.OrganizationID, key) || !active(missing.OrganizationID, key) || !active(enabled.OrganizationID, key) || active(cancelled.OrganizationID, key) || active(other.OrganizationID, key) {
					t.Fatalf("wrong subscriber activation for %s", key)
				}
			}
			exec(migration("down"))
			if err := tx.QueryRow(ctx, `SELECT module_keys FROM subscription_plans WHERE id=$1`, planID).Scan(&modules); err != nil || !slices.Equal(modules, original) {
				t.Fatalf("rollback changed original plan modules: %v %v", modules, err)
			}
			for _, key := range []string{"inventario", "compras"} {
				if active(existing.OrganizationID, key) || !active(enabled.OrganizationID, key) {
					t.Fatalf("rollback changed prior flags for %s", key)
				}
			}
			var count int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM organization_modules WHERE organization_id=$1 AND module_key IN ('inventario','compras')`, missing.OrganizationID).Scan(&count); err != nil || count != 0 {
				t.Fatal("rollback retained inserted modules")
			}
			// Rollback must not overwrite administrative edits made afterwards.
			exec(migration("up"))
			exec(`UPDATE subscription_plans SET module_keys=array_append(module_keys,'custom-test-module'),updated_at=clock_timestamp()+interval '1 second' WHERE id=$1`, planID)
			exec(`UPDATE organization_modules SET active=false,updated_at=clock_timestamp()+interval '1 second' WHERE organization_id=$1 AND module_key='inventario'`, missing.OrganizationID)
			exec(migration("down"))
			if err := tx.QueryRow(ctx, `SELECT module_keys FROM subscription_plans WHERE id=$1`, planID).Scan(&modules); err != nil || !slices.Contains(modules, "custom-test-module") || !slices.Contains(modules, "inventario") || !slices.Contains(modules, "compras") || active(missing.OrganizationID, "inventario") {
				t.Fatalf("rollback overwrote later edits: %v %v", modules, err)
			}
		})
	}
}
