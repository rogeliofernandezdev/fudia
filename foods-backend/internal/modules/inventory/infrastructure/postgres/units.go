package postgres

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/application"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UnitRepository struct{ db *pgxpool.Pool }

func NewUnits(db *pgxpool.Pool) *UnitRepository { return &UnitRepository{db} }

var _ application.UnitRepository = (*UnitRepository)(nil)

func (r *UnitRepository) List(ctx context.Context, organizationID string) ([]domain.Unit, error) {
	rows, err := r.db.Query(ctx, `SELECT id,code,name,countable FROM inventory_units WHERE organization_id=$1 ORDER BY sort_order,name,code`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Unit{}
	for rows.Next() {
		var item domain.Unit
		if err = rows.Scan(&item.ID, &item.Code, &item.Name, &item.Countable); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r *UnitRepository) Create(ctx context.Context, s requestscope.Scope, in domain.UnitInput) (domain.Unit, error) {
	var out domain.Unit
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `INSERT INTO inventory_units(organization_id,code,name,sort_order) VALUES($1,$2,$3,1000) RETURNING id,code,name`, s.OrganizationID, in.Code, in.Name).Scan(&out.ID, &out.Code, &out.Name)
	if err != nil {
		var dbErr *pgconn.PgError
		if errors.As(err, &dbErr) && dbErr.Code == "23505" {
			return out, application.ErrUnitConflict
		}
		return out, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO inventory_unit_combinations(organization_id,unit,presentation_type) SELECT organization_id,$2,code FROM inventory_presentation_types WHERE organization_id=$1 AND code IN ('package','box')`, s.OrganizationID, out.Code); err != nil {
		return out, application.ErrUnitPresentations
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,metadata) VALUES($1,$2,$3,'inventory_unit.created','inventory_unit',$4,jsonb_build_object('code',$5::text,'name',$6::text))`, s.OrganizationID, s.LocationID, s.UserID, out.ID, out.Code, out.Name); err != nil {
		return out, err
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return out, nil
}
