package postgres

import (
	"context"
	"fmt"
	"github.com/foods-platform/fudia-image-processor/internal/application"
	"github.com/foods-platform/fudia-image-processor/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Images struct{ pool *pgxpool.Pool }

func NewImages(pool *pgxpool.Pool) *Images           { return &Images{pool} }
func (r *Images) db(_ context.Context) *pgxpool.Pool { return r.pool }
func imageError(err error) error {
	if err == pgx.ErrNoRows {
		return application.ErrNotFound
	}
	return err
}

const imageColumns = `id,organization_id,location_id,uploaded_by,product_id,original_key,content_type,size_bytes,width,height,status,generation,confirmed_at,expires_at,attempts,purpose`

func scanImage(row pgx.Row) (domain.ImageRecord, error) {
	var v domain.ImageRecord
	err := row.Scan(&v.ID, &v.OrganizationID, &v.LocationID, &v.UploadedBy, &v.ProductID, &v.OriginalKey, &v.ContentType, &v.SizeBytes, &v.Width, &v.Height, &v.Status, &v.Generation, &v.ConfirmedAt, &v.ExpiresAt, &v.Attempts, &v.Purpose)
	return v, imageError(err)
}
func (r *Images) Claim(ctx context.Context, organizationID, id string, generation int64) (domain.ImageRecord, bool, error) {
	record, err := scanImage(r.db(ctx).QueryRow(ctx, `UPDATE product_images SET generation=COALESCE(generation,$3),lease_until=now()+interval '3 minutes',attempts=attempts+1
        WHERE organization_id=$1 AND id=$2 AND status='PROCESSING' AND (generation IS NULL OR generation=$3)
        AND (lease_until IS NULL OR lease_until<now()) RETURNING `+imageColumns, organizationID, id, generation))
	if err == nil {
		return record, true, nil
	}
	if err != application.ErrNotFound {
		return record, false, err
	}
	record, err = scanImage(r.db(ctx).QueryRow(ctx, `SELECT `+imageColumns+` FROM product_images WHERE organization_id=$1 AND id=$2`, organizationID, id))
	if err == nil && record.Status == "PROCESSING" && (record.Generation == nil || *record.Generation == generation) {
		return record, false, application.ErrImageBusy
	}
	return record, false, err
}
func (r *Images) Finish(ctx context.Context, record domain.ImageRecord, generation int64, status, code string) error {
	key := record.OriginalKey
	if status == "READY" {
		key = domain.ImagePrefix(key) + "medium.webp"
	}
	var finished string
	// Lock product before asset, matching the product save lock order (avoids deadlocks).
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT id FROM products WHERE organization_id=$1 AND image_id=$2 FOR UPDATE`, record.OrganizationID, record.ID); err != nil {
		return err
	}
	err = tx.QueryRow(ctx, `UPDATE product_images SET status=$4,lease_until=NULL,failure_code=NULLIF($5,''),processed_at=CASE WHEN $4='PROCESSING' THEN NULL ELSE now() END
        WHERE organization_id=$1 AND id=$2 AND generation=$3 AND attempts=$6 AND status='PROCESSING' RETURNING id`, record.OrganizationID, record.ID, generation, status, code, record.Attempts).Scan(&finished)
	if err != nil {
		return fmt.Errorf("image finish lost lease: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE products SET image_status=$3,image_key=$4,updated_at=now() WHERE organization_id=$1 AND image_id=$2`, record.OrganizationID, record.ID, status, key); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
