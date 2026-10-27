package postgres

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/identity/application"
	"github.com/foods-platform/foods-backend/internal/modules/identity/domain"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) *SessionRepository { return &SessionRepository{db} }

var _ application.SessionRepository = (*SessionRepository)(nil)

func (r *SessionRepository) Candidates(ctx context.Context, email string) ([]domain.Candidate, error) {
	rows, err := r.db.Query(ctx, `SELECT id,organization_id,full_name,password_hash,platform_admin FROM users WHERE lower(email)=lower($1) AND active ORDER BY created_at,id`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Candidate{}
	for rows.Next() {
		var c domain.Candidate
		if err = rows.Scan(&c.UserID, &c.OrganizationID, &c.Name, &c.Hash, &c.PlatformAdmin); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}
func (r *SessionRepository) FirstLocation(ctx context.Context, c domain.Candidate) (string, error) {
	var id string
	var err error
	if c.PlatformAdmin {
		err = r.db.QueryRow(ctx, `SELECT id FROM locations WHERE organization_id=$1 AND active ORDER BY created_at,id LIMIT 1`, c.OrganizationID).Scan(&id)
	} else {
		err = r.db.QueryRow(ctx, `SELECT l.id FROM user_roles ur JOIN roles ro ON ro.id=ur.role_id JOIN users u ON u.id=ur.user_id AND u.organization_id=ro.organization_id AND u.active JOIN locations l ON l.organization_id=ro.organization_id AND (l.id=ur.location_id OR ro.system_key='administrator') WHERE ur.user_id=$1 AND ro.organization_id=$2 AND ro.active AND l.active ORDER BY l.created_at,l.id,ro.name LIMIT 1`, c.UserID, c.OrganizationID).Scan(&id)
	}
	return id, translate(err)
}
func (r *SessionRepository) CreateSession(ctx context.Context, s domain.Session) error {
	_, err := r.db.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,organization_id,location_id,expires_at) VALUES($1,$2,$3,$4,$5)`, s.Hash, s.UserID, s.OrganizationID, s.LocationID, s.ExpiresAt)
	return err
}
func (r *SessionRepository) RevokeSession(ctx context.Context, hash []byte) error {
	_, err := r.db.Exec(ctx, `UPDATE sessions SET revoked_at=COALESCE(revoked_at,now()) WHERE token_hash=$1`, hash)
	return err
}
func (r *SessionRepository) SessionAccess(ctx context.Context, hash []byte) (requestscope.Scope, domain.Access, error) {
	var s requestscope.Scope
	var access domain.Access
	// Company administrators retain their tenant permissions in every active local;
	// all other roles remain restricted to their assigned location.
	err := r.db.QueryRow(ctx, `
		SELECT s.user_id,s.organization_id,s.location_id,u.full_name,u.platform_admin,
		       COALESCE((
		         SELECT array_agg(DISTINCT p.permission)
		         FROM user_roles ur
		         JOIN roles ro ON ro.id=ur.role_id
		         CROSS JOIN LATERAL unnest(ro.permissions) AS p(permission)
		         WHERE ur.user_id=s.user_id AND (ur.location_id=s.location_id OR ro.system_key='administrator')
		           AND ro.organization_id=s.organization_id AND ro.active
		       ),'{}')
		FROM sessions s JOIN users u ON u.id=s.user_id
		JOIN locations l ON l.id=s.location_id AND l.organization_id=s.organization_id
		WHERE s.token_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now() AND u.active
		  AND (u.platform_admin OR u.organization_id=s.organization_id)`, hash).Scan(&s.UserID, &s.OrganizationID, &s.LocationID, &s.Name, &access.PlatformAdmin, &access.Permissions)
	return s, access, translate(err)
}
func (r *SessionRepository) IsPlatformAdmin(ctx context.Context, userID string) (bool, error) {
	var admin bool
	err := r.db.QueryRow(ctx, `SELECT platform_admin FROM users WHERE id=$1 AND active`, userID).Scan(&admin)
	return admin, err
}
func (r *SessionRepository) HasPermission(ctx context.Context, s requestscope.Scope, permission string) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id=ur.role_id JOIN users u ON u.id=ur.user_id AND u.organization_id=ro.organization_id AND u.active JOIN locations l ON l.id=$2 AND l.organization_id=ro.organization_id AND l.active WHERE ur.user_id=$1 AND (ur.location_id=l.id OR ro.system_key='administrator') AND ro.organization_id=$3 AND ro.active AND $4 = ANY(ro.permissions))`, s.UserID, s.LocationID, s.OrganizationID, permission).Scan(&allowed)
	return allowed, err
}
func translate(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	return err
}
