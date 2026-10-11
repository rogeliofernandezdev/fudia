package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/identity/application"
	"github.com/foods-platform/foods-backend/internal/modules/identity/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdministrationRepository struct{ pool *pgxpool.Pool }

func NewAdministration(pool *pgxpool.Pool) *AdministrationRepository {
	return &AdministrationRepository{pool}
}

var _ application.AdministrationRepository = (*AdministrationRepository)(nil)

func (r *AdministrationRepository) db(ctx context.Context) database.Executor {
	return database.ExecutorFor(ctx, r.pool)
}
func administrationError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	return err
}
func scanRole(row pgx.Row) (domain.RoleView, error) {
	var v domain.RoleView
	err := row.Scan(&v.ID, &v.Name, &v.SystemKey, &v.Description, &v.MenuAccess, &v.Permissions, &v.Active, &v.UserCount)
	return v, administrationError(err)
}
func (r *AdministrationRepository) Roles(ctx context.Context, org string) ([]domain.RoleView, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT r.id,r.name,r.system_key,r.description,r.menu_access,r.permissions,r.active,count(DISTINCT ur.user_id) FROM roles r LEFT JOIN user_roles ur ON ur.role_id=r.id WHERE r.organization_id=$1 GROUP BY r.id ORDER BY (r.system_key='administrator') DESC,r.name`, org)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.RoleView{}
	for rows.Next() {
		v, e := scanRole(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, v)
	}
	return items, rows.Err()
}
func (r *AdministrationRepository) SaveRole(ctx context.Context, org, id string, in domain.RoleInput, menus, permissions []string) (domain.RoleView, error) {
	if id != "" {
		return scanRole(r.db(ctx).QueryRow(ctx, `
			UPDATE roles
			SET name=CASE WHEN system_key IS NULL THEN $3 ELSE name END,
			    description=CASE WHEN system_key IS NULL THEN $4 ELSE description END,
			    menu_access=$5::text[] || ARRAY(SELECT v FROM unnest(menu_access) AS v WHERE v<>'*' AND NOT (v=ANY($7::text[]))),
			    permissions=$6::text[] || ARRAY(SELECT v FROM unnest(permissions) AS v WHERE v<>'*' AND NOT (v=ANY($8::text[]))),updated_at=now()
			WHERE id=$1 AND organization_id=$2 AND COALESCE(system_key,'')<>'administrator'
			RETURNING id,name,system_key,description,menu_access,permissions,active,0
		`, id, org, in.Name, in.Description, in.MenuAccess, in.Permissions, menus, permissions))
	}
	return scanRole(r.db(ctx).QueryRow(ctx, `
			INSERT INTO roles(organization_id,name,description,menu_access,permissions)
			VALUES($1,$2,$3,$4,$5)
			RETURNING id,name,system_key,description,menu_access,permissions,active,0
		`, org, in.Name, in.Description, in.MenuAccess, in.Permissions))
}
func (r *AdministrationRepository) SetRoleActive(ctx context.Context, org, id string, active bool) (bool, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE roles SET active=$3,updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND COALESCE(system_key,'')<>'administrator'
	`, id, org, active)
	return tag.RowsAffected() > 0, err
}
func (r *AdministrationRepository) HasAdministrator(ctx context.Context, org string) (bool, error) {
	var ok bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM users u
			JOIN user_roles ur ON ur.user_id=u.id
			JOIN roles r ON r.id=ur.role_id AND r.organization_id=u.organization_id
			JOIN locations l ON l.id=ur.location_id AND l.organization_id=u.organization_id
			WHERE u.organization_id=$1
			  AND u.active
			  AND NOT u.platform_admin
			  AND r.active
			  AND l.active
			  AND r.permissions @> ARRAY['users.manage']::text[]
		)
	`, org).Scan(&ok)
	return ok, err
}
func (r *AdministrationRepository) PlatformAdmin(ctx context.Context, id string) (bool, error) {
	var ok bool
	err := r.db(ctx).QueryRow(ctx, `SELECT platform_admin FROM users WHERE id=$1 AND active`, id).Scan(&ok)
	return ok, administrationError(err)
}
func (r *AdministrationRepository) CatalogAdministrator(ctx context.Context, s requestscope.Scope) (bool, error) {
	var ok bool
	err := r.db(ctx).QueryRow(ctx, `SELECT platform_admin FROM users WHERE id=$1 AND active AND (organization_id=$2 OR platform_admin)`, s.UserID, s.OrganizationID).Scan(&ok)
	return ok, administrationError(err)
}
func (r *AdministrationRepository) Users(ctx context.Context, org string, f application.UserFilter) (application.UserPage, error) {
	out := application.UserPage{Items: []domain.UserView{}, Page: f.Page, PageSize: f.Size}
	q := r.db(ctx)
	if err := q.QueryRow(ctx, `
		SELECT count(*)
		FROM users
		WHERE organization_id=$1
		  AND ($2='' OR full_name ILIKE '%'||$2||'%' OR email ILIKE '%'||$2||'%')
		  AND ($3='' OR ($3='active' AND active) OR ($3='inactive' AND NOT active))
	`, org, f.Query, f.Status).Scan(&out.Total); err != nil {
		return out, err
	}
	rows, err := q.Query(ctx, `
		SELECT u.id,u.full_name,u.email,u.active,u.platform_admin,
		       COALESCE(
		         jsonb_agg(
		           jsonb_build_object(
		             'roleId',ur.role_id::text,
		             'roleName',r.name,
		             'locationId',ur.location_id::text,
		             'locationName',l.name
		           )
		           ORDER BY l.name,r.name
		         ) FILTER (WHERE ur.role_id IS NOT NULL AND r.id IS NOT NULL AND l.id IS NOT NULL),
		         '[]'::jsonb
		       )
		FROM (
		  SELECT id,full_name,email,active,platform_admin
		  FROM users
		  WHERE organization_id=$1
		    AND ($2='' OR full_name ILIKE '%'||$2||'%' OR email ILIKE '%'||$2||'%')
		    AND ($3='' OR ($3='active' AND active) OR ($3='inactive' AND NOT active))
		  ORDER BY full_name,id
		  LIMIT $4 OFFSET $5
		) u
		LEFT JOIN user_roles ur ON ur.user_id=u.id
		LEFT JOIN roles r ON r.id=ur.role_id AND r.organization_id=$1
		LEFT JOIN locations l ON l.id=ur.location_id AND l.organization_id=$1
		GROUP BY u.id,u.full_name,u.email,u.active,u.platform_admin
		ORDER BY u.full_name,u.id
	`, org, f.Query, f.Status, f.Size, (f.Page-1)*f.Size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var v domain.UserView
		var data []byte
		if err = rows.Scan(&v.ID, &v.FullName, &v.Email, &v.Active, &v.PlatformAdmin, &data); err != nil {
			return out, err
		}
		v.Assignments = []domain.UserAssignment{}
		if len(data) > 0 {
			if err = json.Unmarshal(data, &v.Assignments); err != nil {
				return out, err
			}
		}
		out.Items = append(out.Items, v)
	}
	return out, rows.Err()
}
func (r *AdministrationRepository) LockUser(ctx context.Context, org, id string) (application.UserState, error) {
	var v application.UserState
	err := r.db(ctx).QueryRow(ctx, `
		SELECT platform_admin,active FROM users
		WHERE id=$1 AND organization_id=$2
		FOR UPDATE
	`, id, org).Scan(&v.PlatformAdmin, &v.Active)
	return v, administrationError(err)
}
func (r *AdministrationRepository) AssignmentValid(ctx context.Context, org string, v domain.UserAssignment) (bool, error) {
	var ok bool
	err := r.db(ctx).QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM roles r
				JOIN locations l ON l.organization_id=r.organization_id
				WHERE r.id=$1 AND l.id=$2 AND r.organization_id=$3 AND r.active AND l.active
			)
		`, v.RoleID, v.LocationID, org).Scan(&ok)
	return ok, err
}
func (r *AdministrationRepository) SaveUser(ctx context.Context, org, id string, in domain.UserInput, hash string) (string, error) {
	var saved string
	var err error
	q := r.db(ctx)
	if id == "" {
		err = q.QueryRow(ctx, `
			INSERT INTO users(organization_id,email,full_name,password_hash)
			VALUES($1,$2,$3,$4) RETURNING id
		`, org, in.Email, in.FullName, hash).Scan(&saved)
	} else if hash != "" {
		err = q.QueryRow(ctx, `
				UPDATE users SET full_name=$3,email=$4,password_hash=$5,updated_at=now()
				WHERE id=$1 AND organization_id=$2 RETURNING id
			`, id, org, in.FullName, in.Email, hash).Scan(&saved)
	} else {
		err = q.QueryRow(ctx, `
				UPDATE users SET full_name=$3,email=$4,updated_at=now()
				WHERE id=$1 AND organization_id=$2 RETURNING id
			`, id, org, in.FullName, in.Email).Scan(&saved)
	}
	return saved, administrationError(err)
}
func (r *AdministrationRepository) ReplaceAssignments(ctx context.Context, id string, items []domain.UserAssignment) error {
	q := r.db(ctx)
	if _, err := q.Exec(ctx, `DELETE FROM user_roles WHERE user_id=$1`, id); err != nil {
		return err
	}
	for _, v := range items {
		tag, err := q.Exec(ctx, `
			INSERT INTO user_roles(user_id,role_id,location_id)
			VALUES($1,$2,$3)
		`, id, v.RoleID, v.LocationID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return errors.New("assignment was not inserted")
		}
	}
	return nil
}
func (r *AdministrationRepository) SetUserActive(ctx context.Context, org, id string, active bool) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE users SET active=$3,updated_at=now()
		WHERE id=$1 AND organization_id=$2
	`, id, org, active)
	return err
}
func (r *AdministrationRepository) RevokeUserSessions(ctx context.Context, id string, except []byte) error {
	if len(except) > 0 {
		_, err := r.db(ctx).Exec(ctx, `
				UPDATE sessions
				SET revoked_at=COALESCE(revoked_at,now())
				WHERE user_id=$1 AND revoked_at IS NULL AND token_hash<>$2
			`, id, except)
		return err
	}
	_, err := r.db(ctx).Exec(ctx, `
			UPDATE sessions SET revoked_at=COALESCE(revoked_at,now())
			WHERE user_id=$1 AND revoked_at IS NULL
		`, id)
	return err
}
func (r *AdministrationRepository) Profile(ctx context.Context, s requestscope.Scope) (domain.MyProfileView, error) {
	var out domain.MyProfileView
	var plan domain.MyProfilePlanView
	err := r.db(ctx).QueryRow(ctx, `
		SELECT u.full_name,u.email,u.platform_admin,
		       COALESCE((
		         SELECT array_agg(DISTINCT r.name ORDER BY r.name)
		         FROM user_roles ur
		         JOIN roles r ON r.id=ur.role_id
		         WHERE ur.user_id=u.id
		           AND ur.location_id=$2
		           AND r.organization_id=$3
		           AND r.active
		       ),ARRAY[]::text[]),
		       COALESCE((
		         SELECT array_agg(DISTINCT permission ORDER BY permission)
		         FROM user_roles ur
		         JOIN roles r ON r.id=ur.role_id
		         CROSS JOIN LATERAL unnest(r.permissions) permission
		         WHERE ur.user_id=u.id
		           AND ur.location_id=$2
		           AND r.organization_id=$3
		           AND r.active
		       ),ARRAY[]::text[]),
		       COALESCE(sp.id::text,''),COALESCE(sp.code,''),COALESCE(sp.name,''),COALESCE(os.status,'')
		FROM users u
		LEFT JOIN organization_subscriptions os ON os.organization_id=$3
		LEFT JOIN subscription_plans sp ON sp.id=os.plan_id
		WHERE u.id=$1 AND u.organization_id=$3 AND u.active
		LIMIT 1
	`, s.UserID, s.LocationID, s.OrganizationID).Scan(&out.FullName, &out.Email, &out.PlatformAdmin, &out.RoleNames, &out.Permissions, &plan.ID, &plan.Code, &plan.Name, &plan.Status)
	if plan.ID != "" {
		out.CurrentPlan = &plan
	}
	return out, administrationError(err)
}
func (r *AdministrationRepository) LockProfile(ctx context.Context, id string) (application.ProfileCredentials, error) {
	var v application.ProfileCredentials
	err := r.db(ctx).QueryRow(ctx, `
		SELECT email,password_hash
		FROM users
		WHERE id=$1 AND active
		FOR UPDATE
	`, id).Scan(&v.Email, &v.Hash)
	return v, administrationError(err)
}
func (r *AdministrationRepository) UpdateProfile(ctx context.Context, id, name, hash string) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE users
		SET full_name=$2,password_hash=$3,updated_at=now()
		WHERE id=$1
	`, id, name, hash)
	return err
}
func (r *AdministrationRepository) Organizations(ctx context.Context) ([]domain.OrgSummary, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT o.id, o.trade_name, o.active, count(l.id)
		FROM organizations o
		LEFT JOIN locations l ON l.organization_id=o.id AND l.active
		WHERE o.active
		GROUP BY o.id, o.trade_name, o.active
		ORDER BY o.trade_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.OrgSummary{}
	for rows.Next() {
		var v domain.OrgSummary
		if err = rows.Scan(&v.ID, &v.Name, &v.Active, &v.LocationCount); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}
func locations(rows pgx.Rows, err error) ([]domain.LocationSummary, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.LocationSummary{}
	for rows.Next() {
		var v domain.LocationSummary
		if err = rows.Scan(&v.ID, &v.Name, &v.Code, &v.Active); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}
func (r *AdministrationRepository) Locations(ctx context.Context, id string) ([]domain.LocationSummary, error) {
	return locations(r.db(ctx).Query(ctx, `
		SELECT id, name, code, active
		FROM locations
		WHERE organization_id=$1 AND active
		ORDER BY name`, id))
}
func (r *AdministrationRepository) AvailableLocations(ctx context.Context, s requestscope.Scope) ([]domain.LocationSummary, error) {
	return locations(r.db(ctx).Query(ctx, `
		SELECT l.id, l.name, l.code, l.active
		FROM locations l
		WHERE l.organization_id=$1 AND l.active
		  AND (
		    EXISTS(SELECT 1 FROM users u WHERE u.id=$2 AND u.platform_admin)
		    OR EXISTS(SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id=ur.role_id
		              WHERE ur.user_id=$2 AND ur.location_id=l.id AND ro.active)
		  )
		ORDER BY l.name`, s.OrganizationID, s.UserID))
}
func (r *AdministrationRepository) ContextDestination(ctx context.Context, org, loc string) (application.ContextDestination, error) {
	var v application.ContextDestination
	err := r.db(ctx).QueryRow(ctx, `
		SELECT o.trade_name, l.name, p.country_code, l.timezone
		FROM organizations o
		JOIN locations l ON l.organization_id=o.id AND l.active
		JOIN organization_fiscal_profiles p ON p.id=l.fiscal_profile_id AND p.organization_id=l.organization_id AND p.active
		WHERE o.id=$1 AND l.id=$2 AND o.active`, org, loc).Scan(&v.OrganizationName, &v.LocationName, &v.Country, &v.Timezone)
	return v, administrationError(err)
}
func (r *AdministrationRepository) HasLocationRole(ctx context.Context, id, loc string) (bool, error) {
	var ok bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id=ur.role_id WHERE ur.user_id=$1 AND ur.location_id=$2 AND ro.active)`, id, loc).Scan(&ok)
	return ok, err
}
func (r *AdministrationRepository) SwitchSessionContext(ctx context.Context, hash []byte, in domain.SwitchContextInput) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE sessions SET organization_id=$1, location_id=$2 WHERE token_hash=$3 AND revoked_at IS NULL`, in.OrganizationID, in.LocationID, hash)
	return err
}

func (r *AdministrationRepository) CurrentContext(ctx context.Context, s requestscope.Scope) (domain.ContextRecord, error) {
	var v domain.ContextRecord
	err := r.db(ctx).QueryRow(ctx, `
		SELECT o.trade_name,l.name,p.country_code,l.timezone,u.platform_admin,
		       CASE WHEN u.platform_admin THEN ARRAY['*']::text[] ELSE COALESCE((
		         SELECT array_agg(DISTINCT permission ORDER BY permission)
		         FROM user_roles ur
		         JOIN roles ro ON ro.id=ur.role_id AND ro.organization_id=$1 AND ro.active
		         CROSS JOIN LATERAL unnest(ro.permissions) permission
		         WHERE ur.user_id=$3 AND ur.location_id=$2
		       ),ARRAY[]::text[]) END,
		       CASE WHEN u.platform_admin THEN ARRAY['*']::text[] ELSE COALESCE((
		         SELECT array_agg(DISTINCT access_key ORDER BY access_key)
		         FROM user_roles ur
		         JOIN roles ro ON ro.id=ur.role_id AND ro.organization_id=$1 AND ro.active
		         CROSS JOIN LATERAL unnest(ro.menu_access) access_key
		         WHERE ur.user_id=$3 AND ur.location_id=$2
		       ),ARRAY[]::text[]) END
		FROM organizations o
		JOIN locations l ON l.organization_id=o.id
		JOIN organization_fiscal_profiles p ON p.id=l.fiscal_profile_id AND p.organization_id=l.organization_id AND p.active
		JOIN users u ON u.id=$3 AND u.active AND (u.platform_admin OR u.organization_id=o.id)
		WHERE o.id=$1 AND l.id=$2 AND o.active AND l.active
	`, s.OrganizationID, s.LocationID, s.UserID).Scan(&v.OrganizationName, &v.LocationName, &v.Country, &v.Timezone, &v.PlatformAdmin, &v.Permissions, &v.MenuAccess)
	return v, administrationError(err)
}
func (r *AdministrationRepository) SetupRequired(ctx context.Context, org string) (bool, error) {
	var required bool
	err := r.db(ctx).QueryRow(ctx, `
			SELECT completed_at IS NULL
			FROM organization_operational_setup
			WHERE organization_id=$1
		`, org).Scan(&required)
	return required, administrationError(err)
}

// Cash reporting retains its existing wildcard and organization-wide-role policy.
func (r *AdministrationRepository) CashExpectedVisible(ctx context.Context, s requestscope.Scope) (bool, error) {
	var platform bool
	if err := r.db(ctx).QueryRow(ctx, `SELECT platform_admin FROM users WHERE id=$1 AND active`, s.UserID).Scan(&platform); err == nil && platform {
		return true, nil
	}
	var allowed bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT EXISTS(
		  SELECT 1
		  FROM user_roles ur
		  JOIN roles ro ON ro.id=ur.role_id
		  WHERE ur.user_id=$1
		    AND (ur.location_id IS NULL OR ur.location_id=$2)
		    AND ro.active
		    AND ro.organization_id=$3
		    AND (
		      ro.permissions @> ARRAY['*']::text[]
		      OR ro.permissions @> ARRAY['cash.expected.read']::text[]
		    )
		)
	`, s.UserID, s.LocationID, s.OrganizationID).Scan(&allowed)
	return allowed, err
}
