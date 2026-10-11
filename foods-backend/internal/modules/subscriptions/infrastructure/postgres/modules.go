package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/platform/database"
)

func (r *AccessRepository) Configured(ctx context.Context, org string) (map[string]bool, error) {
	rows, err := database.ExecutorFor(ctx, r.pool).Query(ctx, `SELECT module_key, active FROM organization_modules WHERE organization_id=$1`, org)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var key string
		var active bool
		if err = rows.Scan(&key, &active); err != nil {
			return nil, err
		}
		out[key] = active
	}
	return out, rows.Err()
}
func (r *AccessRepository) SetActive(ctx context.Context, org, key string, active bool) error {
	_, err := database.ExecutorFor(ctx, r.pool).Exec(ctx, `INSERT INTO organization_modules(organization_id, module_key, active) VALUES($1,$2,$3) ON CONFLICT(organization_id, module_key) DO UPDATE SET active=EXCLUDED.active, updated_at=now()`, org, key, active)
	return err
}
