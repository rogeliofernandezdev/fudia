package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"strings"
)

func (r *Stock) LockUnit(ctx context.Context, s scope, in domain.InventoryCombinationInput) (string, bool, error) {
	var unitName string
	var countable bool
	err := r.db(ctx).QueryRow(ctx, `SELECT name,countable FROM inventory_units WHERE organization_id=$1 AND code=$2 FOR UPDATE`, s.OrganizationID, in.Unit).Scan(&unitName, &countable)
	return unitName, countable, stockError(err)
}
func (r *Stock) EnsurePresentationType(ctx context.Context, s scope, in domain.InventoryCombinationInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `INSERT INTO inventory_presentation_types(organization_id,code,name) VALUES($1,$2,$3) ON CONFLICT(organization_id,code) DO NOTHING`, s.OrganizationID, in.Code, in.Name)
	return tag.RowsAffected(), stockError(err)
}
func (r *Stock) PresentationTypeName(ctx context.Context, s scope, in domain.InventoryCombinationInput) (string, error) {
	var name string
	err := r.db(ctx).QueryRow(ctx, `SELECT name FROM inventory_presentation_types WHERE organization_id=$1 AND code=$2`, s.OrganizationID, in.Code).Scan(&name)
	return name, stockError(err)
}
func (r *Stock) ClearDefaultCombination(ctx context.Context, s scope, in domain.InventoryCombinationInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE inventory_unit_combinations SET is_default=false WHERE organization_id=$1 AND unit=$2 AND is_default`, s.OrganizationID, in.Unit)
	return tag.RowsAffected(), stockError(err)
}
func (r *Stock) SaveCombination(ctx context.Context, s scope, in domain.InventoryCombinationInput) (string, error) {
	var id string
	err := r.db(ctx).QueryRow(ctx, `INSERT INTO inventory_unit_combinations(organization_id,unit,presentation_type,is_default) VALUES($1,$2,$3,$4) ON CONFLICT(organization_id,unit,presentation_type) DO UPDATE SET is_default=EXCLUDED.is_default RETURNING id`, s.OrganizationID, in.Unit, in.Code, in.IsDefault).Scan(&id)
	return id, stockError(err)
}
func (r *Stock) AuditCombination(ctx context.Context, s scope, id string, in domain.InventoryCombinationInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,metadata) VALUES($1,$2,$3,'inventory_combination.saved','inventory_combination',$4,jsonb_build_object('unit',$5::text,'type',$6::text,'default',$7::boolean))`, s.OrganizationID, s.LocationID, s.UserID, id, in.Unit, in.Code, in.IsDefault)
	return tag.RowsAffected(), stockError(err)
}
func (r *Stock) LockPresentationItem(ctx context.Context, s scope, id string) (string, error) {
	var unit string
	err := r.db(ctx).QueryRow(ctx, `SELECT unit FROM inventory_items WHERE organization_id=$1 AND id=$2 AND active FOR UPDATE`, s.OrganizationID, id).Scan(&unit)
	return unit, stockError(err)
}
func (r *Stock) AuditPresentation(ctx context.Context, s scope, id string, itemID string, in domain.InventoryPresentationInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,metadata) VALUES($1,$2,$3,'inventory_presentation.saved','inventory_presentation',$4,jsonb_build_object('itemId',$5::text,'factor',$6::numeric,'default',$7::boolean))`, s.OrganizationID, s.LocationID, s.UserID, id, itemID, in.UnitsPerPresentation, in.IsDefault)
	return tag.RowsAffected(), stockError(err)
}
func (r *Stock) Presentation(ctx context.Context, s scope, id string) (domain.InventoryPresentationOption, error) {
	var out domain.InventoryPresentationOption
	err := r.db(ctx).QueryRow(ctx, `SELECT p.id,p.presentation_type,p.units_per_presentation::text,t.name,p.is_default FROM inventory_presentations p JOIN inventory_presentation_types t ON t.organization_id=p.organization_id AND t.code=p.presentation_type WHERE p.organization_id=$1 AND p.id=$2`, s.OrganizationID, id).Scan(&out.ID, &out.PresentationType, &out.UnitsPerPresentation, &out.Name, &out.IsDefault)
	return out, stockError(err)
}
func (r *Stock) Combinations(ctx context.Context, s scope, unit string) ([]domain.InventoryCombinationView, error) {
	rows, err := r.db(ctx).Query(ctx, `
 SELECT t.code,t.name,u.code,u.name,u.countable,
  CASE WHEN t.code='unit' THEN NOT EXISTS(SELECT 1 FROM inventory_unit_combinations d WHERE d.organization_id=u.organization_id AND d.unit=u.code AND d.is_default AND d.presentation_type<>'unit') ELSE c.is_default END
 FROM inventory_units u JOIN inventory_presentation_types t ON t.organization_id=u.organization_id
 LEFT JOIN inventory_unit_combinations c ON c.organization_id=u.organization_id AND c.unit=u.code AND c.presentation_type=t.code
 WHERE u.organization_id=$1 AND ($2='' OR u.code=$2) AND (t.code='unit' OR c.id IS NOT NULL)
 ORDER BY u.sort_order,u.name,t.sort_order,t.name`, s.OrganizationID, strings.TrimSpace(unit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.InventoryCombinationView{}
	for rows.Next() {
		var x domain.InventoryCombinationView
		if err = rows.Scan(&x.Code, &x.Name, &x.Unit, &x.UnitName, &x.UnitCountable, &x.IsDefault); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

func (r *Stock) ItemExists(ctx context.Context, s scope, id string) (bool, error) {
	var exists bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM inventory_items WHERE id=$1 AND organization_id=$2 AND active)`, id, s.OrganizationID).Scan(&exists)
	return exists, stockError(err)
}
func (r *Stock) LocationSettings(ctx context.Context, s scope, id string, in domain.LocationSettings) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		INSERT INTO inventory_location_settings(organization_id,location_id,inventory_item_id,minimum_stock,reorder_point,optimal_stock)
		VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(location_id,inventory_item_id)
		DO UPDATE SET minimum_stock=EXCLUDED.minimum_stock,reorder_point=EXCLUDED.reorder_point,
		  optimal_stock=EXCLUDED.optimal_stock,updated_at=now()
	`, s.OrganizationID, s.LocationID, id, in.MinimumStock, in.ReorderPoint, in.OptimalStock)
	return tag.RowsAffected(), stockError(err)
}
