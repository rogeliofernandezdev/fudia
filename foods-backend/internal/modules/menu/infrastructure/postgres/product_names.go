package postgres

import "context"

func (r *Catalog) RenameInventoryProduct(ctx context.Context, s scope, id, name string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE products SET name=$3,updated_at=now()
	  WHERE organization_id=$1 AND id=$2 AND EXISTS(
	    SELECT 1 FROM inventory_items i WHERE i.organization_id=$1 AND i.product_id=products.id AND i.active
	  )`, s.OrganizationID, id, name)
	return tag.RowsAffected(), err
}
