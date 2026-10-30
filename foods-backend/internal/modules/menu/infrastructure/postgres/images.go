package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/menu/application"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Images struct{ pool *pgxpool.Pool }

func NewImages(pool *pgxpool.Pool) *Images                 { return &Images{pool} }
func (r *Images) db(ctx context.Context) database.Executor { return database.ExecutorFor(ctx, r.pool) }

const imageColumns = `id,organization_id,location_id,uploaded_by,product_id,original_key,content_type,size_bytes,width,height,status,generation,confirmed_at,expires_at,attempts,purpose`

func scanImage(row pgx.Row) (domain.ImageRecord, error) {
	var v domain.ImageRecord
	err := row.Scan(&v.ID, &v.OrganizationID, &v.LocationID, &v.UploadedBy, &v.ProductID, &v.OriginalKey, &v.ContentType, &v.SizeBytes, &v.Width, &v.Height, &v.Status, &v.Generation, &v.ConfirmedAt, &v.ExpiresAt, &v.Attempts, &v.Purpose)
	return v, menuError(err)
}
func (r *Images) Reserve(ctx context.Context, s scope, in domain.ImageUploadInput, expiry time.Time) (domain.ImageRecord, error) {
	// Serialize quotas per tenant/user. PostgreSQL, not the application, generates the UUID.
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.ImageRecord{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, s.OrganizationID+":"+s.UserID); err != nil {
		return domain.ImageRecord{}, err
	}
	record, err := scanImage(tx.QueryRow(ctx, `INSERT INTO product_images(organization_id,location_id,uploaded_by,product_id,content_type,size_bytes,width,height,expires_at,purpose)
        SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,COALESCE(NULLIF($10,''),'product') FROM locations l WHERE l.id=$2 AND l.organization_id=$1 AND l.active
        AND (SELECT count(*) FROM product_images WHERE organization_id=$1 AND uploaded_by=$3 AND created_at>now()-interval '1 hour')<100
        AND (SELECT count(*) FROM product_images WHERE organization_id=$1 AND uploaded_by=$3 AND confirmed_at IS NULL AND expires_at>now())<20
        RETURNING `+imageColumns, s.OrganizationID, s.LocationID, s.UserID, in.ProductID, in.ContentType, in.SizeBytes, in.Width, in.Height, expiry, in.Purpose))
	if err != nil {
		return record, err
	}
	record.OriginalKey = domain.ImageOriginalKey(record.OrganizationID, record.ID, record.ContentType, record.Purpose)
	if _, err = tx.Exec(ctx, `UPDATE product_images SET original_key=$2 WHERE id=$1`, record.ID, record.OriginalKey); err != nil {
		return record, err
	}
	return record, tx.Commit(ctx)
}
func (r *Images) Read(ctx context.Context, s scope, id string) (domain.ImageRecord, error) {
	return scanImage(r.db(ctx).QueryRow(ctx, `SELECT `+imageColumns+` FROM product_images WHERE organization_id=$1 AND id=$2`, s.OrganizationID, id))
}
func (r *Images) Confirm(ctx context.Context, s scope, id string, generation int64) (domain.ImageRecord, error) {
	return scanImage(r.db(ctx).QueryRow(ctx, `UPDATE product_images SET confirmed_at=COALESCE(confirmed_at,now()),generation=COALESCE(generation,$4)
        WHERE organization_id=$1 AND id=$2 AND uploaded_by=$3 AND location_id=$5 AND (generation IS NULL OR generation=$4)
        RETURNING `+imageColumns, s.OrganizationID, id, s.UserID, generation, s.LocationID))
}
func (r *Images) Attach(ctx context.Context, s scope, productID, imageID string) error {
	// Product is locked by its save transaction; lock the asset to serialize worker/attachment races.
	record, err := scanImage(r.db(ctx).QueryRow(ctx, `SELECT `+imageColumns+` FROM product_images
        WHERE organization_id=$1 AND id=$2 AND uploaded_by=$3 AND location_id=$5 AND confirmed_at IS NOT NULL
        AND purpose='product' AND (product_id IS NULL OR product_id=$4) FOR UPDATE`, s.OrganizationID, imageID, s.UserID, productID, s.LocationID))
	if err != nil {
		return err
	}
	key := record.OriginalKey
	if record.Status == "READY" {
		key = domain.ImagePrefix(key) + "medium.webp"
	}
	if _, err = r.db(ctx).Exec(ctx, `UPDATE product_images SET product_id=$3 WHERE organization_id=$1 AND id=$2`, s.OrganizationID, imageID, productID); err != nil {
		return err
	}
	tag, err := r.db(ctx).Exec(ctx, `UPDATE products SET image_id=$3,image_key=$4,image_status=$5,image_url=NULL,updated_at=now() WHERE organization_id=$1 AND id=$2`, s.OrganizationID, productID, imageID, key, record.Status)
	if err == nil && tag.RowsAffected() != 1 {
		return application.ErrNotFound
	}
	return err
}
func (r *Images) Detach(ctx context.Context, s scope, id string) error {
	_, err := r.db(ctx).Exec(ctx, `UPDATE products SET image_id=NULL,image_key=NULL,image_status=NULL,image_url=NULL,updated_at=now() WHERE organization_id=$1 AND id=$2`, s.OrganizationID, id)
	return err
}
func (r *Images) ForProducts(ctx context.Context, s scope, ids []string) (map[string]domain.ImageRecord, error) {
	// UUIDs come from the scoped catalogue; compare text to keep malformed external IDs harmless.
	rows, err := r.db(ctx).Query(ctx, `SELECT p.id::text,i.id,i.organization_id,i.location_id,i.uploaded_by,i.product_id,i.original_key,i.content_type,i.size_bytes,i.width,i.height,i.status,i.generation,i.confirmed_at,i.expires_at,i.attempts,i.purpose
        FROM products p JOIN product_images i ON i.id=p.image_id AND i.organization_id=p.organization_id
        WHERE p.organization_id=$1 AND p.id::text=ANY($2::text[])`, s.OrganizationID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]domain.ImageRecord{}
	for rows.Next() {
		var productID string
		var v domain.ImageRecord
		if err = rows.Scan(&productID, &v.ID, &v.OrganizationID, &v.LocationID, &v.UploadedBy, &v.ProductID, &v.OriginalKey, &v.ContentType, &v.SizeBytes, &v.Width, &v.Height, &v.Status, &v.Generation, &v.ConfirmedAt, &v.ExpiresAt, &v.Attempts, &v.Purpose); err != nil {
			return nil, err
		}
		out[productID] = v
	}
	return out, rows.Err()
}
