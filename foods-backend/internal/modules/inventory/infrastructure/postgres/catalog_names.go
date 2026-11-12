package postgres

import "context"

// CatalogNameTaken runs inside the caller's creation transaction. Taking the
// name lock before the existence query prevents two new articles with the
// same name from passing validation together, even from different locations.
func (r *Stock) CatalogNameTaken(ctx context.Context, s scope, name string) (bool, error) {
	if _, err := r.db(ctx).Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(
	  $1::text||':'||lower(btrim(regexp_replace($2,'[[:space:]]+',' ','g'))),0))`, s.OrganizationID, name); err != nil {
		return false, err
	}
	var taken bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT EXISTS(
		  SELECT 1 FROM products WHERE organization_id=$1
		    AND lower(btrim(regexp_replace(name,'[[:space:]]+',' ','g')))=lower(btrim(regexp_replace($2,'[[:space:]]+',' ','g')))
		  UNION ALL
		  SELECT 1 FROM inventory_items WHERE organization_id=$1 AND product_id IS NULL
		    AND lower(btrim(regexp_replace(name,'[[:space:]]+',' ','g')))=lower(btrim(regexp_replace($2,'[[:space:]]+',' ','g')))
		)`, s.OrganizationID, name).Scan(&taken)
	return taken, err
}
