package httpapi

import (
	"context"
	"os"
	"testing"
)

func TestWhatsAppMigrationPreservesAndReconcilesDuplicateCountries(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	// A private transactional schema models the real pre-120 schema after 067
	// removed phone_number_id. No application data is touched.
	_, err = tx.Exec(ctx, `CREATE SCHEMA whatsapp_migration_check;
 SET LOCAL search_path=whatsapp_migration_check;
 CREATE TABLE platform_countries(code char(2) PRIMARY KEY);
 INSERT INTO platform_countries VALUES('PE'),('CL');
 CREATE TABLE users(id uuid PRIMARY KEY DEFAULT gen_random_uuid());
 CREATE TABLE platform_whatsapp_channels(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),country_code char(2) NOT NULL REFERENCES platform_countries(code),phone_number text NOT NULL UNIQUE,display_name text,active boolean NOT NULL DEFAULT true,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now());
 INSERT INTO platform_whatsapp_channels(country_code,phone_number,display_name,active) VALUES('PE','+51911111111','Original',true),('PE','+51922222222','Duplicado',true),('CL','+56911111111','Único',true);`)
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../../migrations/000120_whatsapp_providers.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	var total, current, active, imported int
	if err = tx.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE superseded_by IS NULL),count(*) FILTER(WHERE active) FROM platform_whatsapp_channels`).Scan(&total, &current, &active); err != nil {
		t.Fatal(err)
	}
	if total != 3 || current != 2 || active != 0 {
		t.Fatalf("unsafe reconciliation: total=%d current=%d active=%d", total, current, active)
	}
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM platform_whatsapp_channel_history WHERE action='migration.imported' AND (after_config->>'active')::boolean`).Scan(&imported); err != nil || imported != 3 {
		t.Fatal("original operational state lost from audit")
	}
	if _, err = tx.Exec(ctx, `INSERT INTO platform_whatsapp_channels(country_code,phone_number,display_name,active) VALUES('PE','+51933333333','Otro',false)`); err == nil {
		t.Fatal("duplicate country allowed")
	}
}
