package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/application"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
)

// CatalogNameTaken runs inside the caller's creation transaction. Taking the
// name lock before the existence query prevents two new articles with the
// same name from passing validation together, even from different locations.
func (r *Stock) CatalogNameTaken(ctx context.Context, s scope, name string) (bool, error) {
	if err := r.lockCatalogName(ctx, s, name); err != nil {
		return false, err
	}
	return r.catalogNameTaken(ctx, s, name, "")
}

func (r *Stock) lockCatalogName(ctx context.Context, s scope, name string) error {
	if _, err := r.db(ctx).Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(
	  $1::uuid::text||':'||lower(btrim(regexp_replace($2,'[[:space:]]+',' ','g'))),0))`, s.OrganizationID, name); err != nil {
		return err
	}
	return nil
}

func (r *Stock) catalogNameTaken(ctx context.Context, s scope, name, exceptItemID string) (bool, error) {
	var taken bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT EXISTS(
		  SELECT 1 FROM products WHERE organization_id=$1
		    AND id IS DISTINCT FROM (SELECT product_id FROM inventory_items WHERE organization_id=$1 AND id=NULLIF($3,'')::uuid)
		    AND lower(btrim(regexp_replace(name,'[[:space:]]+',' ','g')))=lower(btrim(regexp_replace($2,'[[:space:]]+',' ','g')))
		  UNION ALL
		  SELECT 1 FROM inventory_items WHERE organization_id=$1 AND product_id IS NULL
		    AND id IS DISTINCT FROM NULLIF($3,'')::uuid
		    AND lower(btrim(regexp_replace(name,'[[:space:]]+',' ','g')))=lower(btrim(regexp_replace($2,'[[:space:]]+',' ','g')))
		)`, s.OrganizationID, name, exceptItemID).Scan(&taken)
	return taken, err
}

func (r *Stock) LockItemName(ctx context.Context, s scope, id, name string) (domain.InventoryNameTarget, error) {
	var target domain.InventoryNameTarget
	if err := r.lockCatalogName(ctx, s, name); err != nil {
		return target, err
	}
	err := r.db(ctx).QueryRow(ctx, `SELECT i.product_id,COALESCE(p.name,i.name)
	  FROM inventory_items i LEFT JOIN products p ON p.organization_id=i.organization_id AND p.id=i.product_id
	  WHERE i.organization_id=$1 AND i.id=$2 AND i.active
	    AND EXISTS(SELECT 1 FROM locations WHERE organization_id=$1 AND id=$3 AND active)
	  FOR UPDATE OF i`, s.OrganizationID, id, s.LocationID).Scan(&target.ProductID, &target.Name)
	if err != nil {
		return target, stockError(err)
	}
	if target.Name != name {
		taken, err := r.catalogNameTaken(ctx, s, name, id)
		if err != nil {
			return target, err
		}
		if taken {
			return target, application.ErrNameTaken
		}
	}
	return target, nil
}

func (r *Stock) RenameItem(ctx context.Context, s scope, id, name string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE inventory_items SET name=$3,updated_at=now() WHERE organization_id=$1 AND id=$2 AND active`, s.OrganizationID, id, name)
	return tag.RowsAffected(), err
}
