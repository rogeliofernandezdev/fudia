package postgres

import (
	"context"
	"fmt"
	"github.com/foods-platform/fudia-image-processor/internal/application"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
	"time"
)

func TestLeasesFencingAndLateEventReferenceSafety(t *testing.T) {
	connection := os.Getenv("FUDIA_IMAGE_PROCESSOR_TEST_DATABASE_URL")
	if connection == "" {
		t.Skip("FUDIA_IMAGE_PROCESSOR_TEST_DATABASE_URL required for SQL contract test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connection)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var organizationID, locationID, userID, productID, imageID, replacementID string
	if err = pool.QueryRow(ctx, `INSERT INTO organizations(legal_name,trade_name,tax_id) VALUES('Processor test','Processor test',$1) RETURNING id`, fmt.Sprintf("%011d", time.Now().UnixNano()%100000000000)).Scan(&organizationID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `UPDATE products SET image_id=NULL,image_key=NULL,image_status=NULL WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM product_images WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM products WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM locations WHERE organization_id=$1`, organizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM organizations WHERE id=$1`, organizationID)
	}()
	if err = pool.QueryRow(ctx, `INSERT INTO locations(organization_id,name,code) VALUES($1,'Images','images') RETURNING id`, organizationID).Scan(&locationID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,'images@test.invalid','Images','unused') RETURNING id`, organizationID).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price) VALUES($1,'IMAGE','Image',10) RETURNING id`, organizationID).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	reserve := func() string {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO product_images(organization_id,location_id,uploaded_by,product_id,content_type,size_bytes,width,height,expires_at) VALUES($1,$2,$3,$4,'image/png',100,100,50,now()+interval '1 hour') RETURNING id`, organizationID, locationID, userID, productID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE product_images SET original_key='restaurants/'||organization_id||'/products/'||id||'/original.png' WHERE id=$1`, id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	imageID = reserve()
	replacementID = reserve()
	repository := NewImages(pool)
	if _, err = pool.Exec(ctx, `UPDATE products SET image_id=$2,image_key='original.png',image_status='PROCESSING' WHERE id=$1`, productID, imageID); err != nil {
		t.Fatal(err)
	}
	first, claimed, err := repository.Claim(ctx, organizationID, imageID, 10)
	if err != nil || !claimed {
		t.Fatal("first lease", err)
	}
	if _, claimed, err = repository.Claim(ctx, organizationID, imageID, 10); err != application.ErrImageBusy || claimed {
		t.Fatal("concurrent event not retried")
	}
	if _, err = pool.Exec(ctx, `UPDATE product_images SET lease_until=now()-interval '1 second' WHERE id=$1`, imageID); err != nil {
		t.Fatal(err)
	}
	second, claimed, err := repository.Claim(ctx, organizationID, imageID, 10)
	if err != nil || !claimed || second.Attempts != first.Attempts+1 {
		t.Fatal("lease not reclaimed")
	}
	if err = repository.Finish(ctx, first, 10, "READY", ""); err == nil {
		t.Fatal("stale processor committed")
	}
	if _, err = pool.Exec(ctx, `UPDATE products SET image_id=$2,image_key='replacement.png',image_status='PROCESSING' WHERE id=$1`, productID, replacementID); err != nil {
		t.Fatal(err)
	}
	if err = repository.Finish(ctx, second, 10, "READY", ""); err != nil {
		t.Fatal(err)
	}
	var current, status, key string
	if err = pool.QueryRow(ctx, `SELECT image_id::text,image_status,image_key FROM products WHERE id=$1`, productID).Scan(&current, &status, &key); err != nil || current != replacementID || status != "PROCESSING" || key != "replacement.png" {
		t.Fatal("late event changed replacement", err)
	}
	if _, claimed, err = repository.Claim(ctx, organizationID, imageID, 10); err != nil || claimed {
		t.Fatal("duplicate READY processing")
	}
	if _, claimed, err = repository.Claim(ctx, organizationID, imageID, 11); err != nil || claimed {
		t.Fatal("another generation processed")
	}
}
