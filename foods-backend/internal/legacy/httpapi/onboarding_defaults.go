package httpapi

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func seedOrganizationOperationalDefaults(
	ctx context.Context,
	tx pgx.Tx,
	organizationID string,
	locationID string,
	userID string,
) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO expense_categories(organization_id,name,active)
		SELECT $1,name,active
		FROM expense_category_templates
		ON CONFLICT DO NOTHING
	`, organizationID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO zones(organization_id,location_id,name,sort_order,active)
		SELECT $1,$2,name,sort_order,active
		FROM location_zone_templates
		ON CONFLICT DO NOTHING
	`, organizationID, locationID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO cash_registers(organization_id,location_id,name,blind_close,active,created_by)
		SELECT $1,$2,name,blind_close,active,$3
		FROM cash_register_templates
		ON CONFLICT DO NOTHING
	`, organizationID, locationID, userID); err != nil {
		return err
	}

	return nil
}
