package postgres

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/subscriptions/application"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccessRepository struct{ pool *pgxpool.Pool }

func NewAccess(pool *pgxpool.Pool) *AccessRepository { return &AccessRepository{pool} }
func (r *AccessRepository) ActiveModules(ctx context.Context, org string) (map[string]bool, error) {
	rows, err := database.ExecutorFor(ctx, r.pool).Query(ctx, `
		SELECT m.module_key,m.active AND (
		  s.organization_id IS NULL OR (s.status<>'cancelled' AND m.module_key=ANY(p.module_keys))
		)
		FROM organization_modules m
		LEFT JOIN organization_subscriptions s ON s.organization_id=m.organization_id
		LEFT JOIN subscription_plans p ON p.id=s.plan_id
		WHERE m.organization_id=$1
	`, org)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]bool{}
	for rows.Next() {
		var key string
		var active bool
		if err = rows.Scan(&key, &active); err != nil {
			return nil, err
		}
		result[key] = active && readyModules[key]
	}
	return result, rows.Err()
}

var readyModules = map[string]bool{
	"reportes": true, "pos": true, "pedidos": true, "cocina": true, "mesas": true, "caja": true, "reservas": true,
	"productos": true, "combos": true, "recetas": true, "inventario": true, "kardex": true, "compras": true,
	"clientes": true, "locales": true, "fiscal": true, "usuarios": true, "costos": true, "whatsapp_bot": true,
}

func (r *AccessRepository) LockCapacity(ctx context.Context, org, resource string) (application.Capacity, error) {
	var v application.Capacity
	q := database.ExecutorFor(ctx, r.pool)
	var maxSQL, countSQL string
	switch resource {
	case "users":
		maxSQL = `
			SELECT p.max_users
			FROM organization_subscriptions s JOIN subscription_plans p ON p.id=s.plan_id
			WHERE s.organization_id=$1
			FOR UPDATE OF s
		`
		countSQL = `SELECT count(*) FROM users WHERE organization_id=$1 AND active AND NOT platform_admin`
	case "locations":
		maxSQL = `
			SELECT p.max_locations
			FROM organization_subscriptions s JOIN subscription_plans p ON p.id=s.plan_id
			WHERE s.organization_id=$1
			FOR UPDATE OF s
		`
		countSQL = `SELECT count(*) FROM locations WHERE organization_id=$1 AND active`
	default:
		return v, errors.New("unknown subscription resource")
	}
	if err := q.QueryRow(ctx, maxSQL, org).Scan(&v.Maximum); err != nil {
		return v, err
	}
	if v.Maximum != nil {
		if err := q.QueryRow(ctx, countSQL, org).Scan(&v.Used); err != nil {
			return v, err
		}
	}
	return v, nil
}
