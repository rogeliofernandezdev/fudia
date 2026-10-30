package postgres

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/application"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RestaurantProfiles struct{ pool *pgxpool.Pool }

func NewRestaurantProfiles(pool *pgxpool.Pool) *RestaurantProfiles { return &RestaurantProfiles{pool} }
func (r *RestaurantProfiles) Read(ctx context.Context, s scope) (application.ProfileRecord, error) {
	var out application.ProfileRecord
	db := database.ExecutorFor(ctx, r.pool)
	if err := db.QueryRow(ctx, `SELECT COALESCE(founded_on::text,''),logo_image_id FROM organizations WHERE id=$1 AND active`, s.OrganizationID).Scan(&out.FoundedOn, &out.LogoImageID); err != nil {
		return out, err
	}
	rows, err := db.Query(ctx, `SELECT image_id::text FROM organization_menu_images WHERE organization_id=$1 ORDER BY sort_order`, s.OrganizationID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	out.MenuImageIDs = []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return out, err
		}
		out.MenuImageIDs = append(out.MenuImageIDs, id)
	}
	return out, rows.Err()
}
func (r *RestaurantProfiles) Save(ctx context.Context, s scope, in application.RestaurantProfileInput) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var organizationID string
	if err = tx.QueryRow(ctx, `SELECT id FROM organizations WHERE id=$1 AND active FOR UPDATE`, s.OrganizationID).Scan(&organizationID); err != nil {
		return err
	}
	check := func(id, purpose string) error {
		var found string
		return tx.QueryRow(ctx, `SELECT i.id FROM product_images i WHERE i.organization_id=$1 AND i.id::text=$2 AND i.purpose=$3
   AND i.confirmed_at IS NOT NULL AND (i.status IN ('PROCESSING','READY') OR EXISTS(SELECT 1 FROM organizations o WHERE o.id=$1 AND o.logo_image_id=i.id) OR EXISTS(SELECT 1 FROM organization_menu_images m WHERE m.organization_id=$1 AND m.image_id=i.id))
   AND (i.uploaded_by=$4 OR EXISTS(SELECT 1 FROM organizations o WHERE o.id=$1 AND o.logo_image_id=i.id)
    OR EXISTS(SELECT 1 FROM organization_menu_images m WHERE m.organization_id=$1 AND m.image_id=i.id)) FOR SHARE`, s.OrganizationID, id, purpose, s.UserID).Scan(&found)
	}
	if in.LogoImageID != nil {
		if err = check(*in.LogoImageID, "restaurant_logo"); err != nil {
			return err
		}
	}
	for _, id := range in.MenuImageIDs {
		if err = check(id, "virtual_menu"); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE organizations SET founded_on=NULLIF($2,'')::date,logo_image_id=$3,updated_at=now() WHERE id=$1`, s.OrganizationID, in.FoundedOn, in.LogoImageID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM organization_menu_images WHERE organization_id=$1`, s.OrganizationID); err != nil {
		return err
	}
	for order, id := range in.MenuImageIDs {
		if _, err = tx.Exec(ctx, `INSERT INTO organization_menu_images(organization_id,image_id,sort_order) VALUES($1,$2,$3)`, s.OrganizationID, id, order); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,metadata) VALUES($1,$2,$3,'restaurant.profile_updated','organization',$1,jsonb_build_object('menuPages',$4::int))`, s.OrganizationID, s.LocationID, s.UserID, len(in.MenuImageIDs)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *RestaurantProfiles) PlatformTarget(ctx context.Context, userID, organizationID string) (requestscope.Scope, error) {
	out := requestscope.Scope{UserID: userID, OrganizationID: organizationID}
	err := r.pool.QueryRow(ctx, `SELECT l.id FROM locations l JOIN organizations o ON o.id=l.organization_id
  WHERE o.id::text=$1 AND o.active AND l.active AND EXISTS(SELECT 1 FROM users WHERE id=$2 AND active AND platform_admin)
  ORDER BY l.created_at,l.id LIMIT 1`, organizationID, userID).Scan(&out.LocationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, application.ErrNotFound
	}
	return out, err
}
