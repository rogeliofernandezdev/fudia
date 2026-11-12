package postgres

import "context"

func (r *Catalog) RenameInventoryProduct(ctx context.Context, s scope, id, name string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE products SET name=$3,updated_at=now()
	  WHERE organization_id=$1 AND id=$2`, s.OrganizationID, id, name)
	return tag.RowsAffected(), err
}
