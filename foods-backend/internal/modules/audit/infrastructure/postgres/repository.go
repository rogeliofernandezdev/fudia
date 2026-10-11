package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool} }
func (r *Repository) Record(ctx context.Context, s requestscope.Scope, action, entity, id string) error {
	_, err := database.ExecutorFor(ctx, r.pool).Exec(ctx, `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id) VALUES($1,$2,$3,$4,$5,$6)`, s.OrganizationID, s.LocationID, s.UserID, action, entity, id)
	return err
}
