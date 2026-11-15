package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/delivery/application"
	"github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	"github.com/jackc/pgx/v5"
)

// Operadores de las centrales, sus sesiones y el registro de sus acciones.

var _ application.OperatorRepository = (*Repository)(nil)

const operatorColumns = `id::text,partner_id::text,full_name,email,active,must_change_password,last_login_at,created_at`

func scanOperator(row pgx.Row) (domain.Operator, error) {
	var o domain.Operator
	var lastLogin *time.Time
	var created time.Time
	err := row.Scan(&o.ID, &o.PartnerID, &o.FullName, &o.Email, &o.Active, &o.MustChangePassword, &lastLogin, &created)
	if errors.Is(err, pgx.ErrNoRows) {
		return o, application.ErrNotFound
	}
	if uniqueViolation(err) {
		return o, application.ErrDuplicateEmail
	}
	o.LastLoginAt = optionalStamp(lastLogin)
	o.CreatedAt = stamp(created)
	return o, err
}

func (r *Repository) PartnerExists(ctx context.Context, partnerID string) (bool, error) {
	var exists bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM delivery_partners WHERE id=$1)`, partnerID).Scan(&exists)
	return exists, err
}

func (r *Repository) ListOperators(ctx context.Context, partnerID string) ([]domain.Operator, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT `+operatorColumns+` FROM delivery_partner_operators
		WHERE partner_id=$1 ORDER BY active DESC, lower(full_name), created_at`, partnerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Operator{}
	for rows.Next() {
		operator, err := scanOperator(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, operator)
	}
	return out, rows.Err()
}

func (r *Repository) CreateOperator(ctx context.Context, partnerID string, in domain.OperatorInput, passwordHash string) (domain.Operator, error) {
	return scanOperator(r.db(ctx).QueryRow(ctx, `
		INSERT INTO delivery_partner_operators(partner_id,full_name,email,password_hash)
		VALUES($1,$2,$3,$4) RETURNING `+operatorColumns, partnerID, in.FullName, in.Email, passwordHash))
}

func (r *Repository) UpdateOperator(ctx context.Context, partnerID, id string, in domain.OperatorInput) (domain.Operator, error) {
	return scanOperator(r.db(ctx).QueryRow(ctx, `
		UPDATE delivery_partner_operators SET full_name=$3,email=$4,active=COALESCE($5,active),updated_at=now()
		WHERE id=$2 AND partner_id=$1 RETURNING `+operatorColumns, partnerID, id, in.FullName, in.Email, in.Active))
}

func (r *Repository) ResetOperatorPassword(ctx context.Context, partnerID, id, passwordHash string) (domain.Operator, error) {
	return scanOperator(r.db(ctx).QueryRow(ctx, `
		UPDATE delivery_partner_operators
		SET password_hash=$3,must_change_password=true,failed_logins=0,locked_until=NULL,updated_at=now()
		WHERE id=$2 AND partner_id=$1 RETURNING `+operatorColumns, partnerID, id, passwordHash))
}

func (r *Repository) RevokeOperatorSessions(ctx context.Context, operatorID string, keep []byte) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE delivery_partner_sessions SET revoked_at=now()
		WHERE operator_id=$1 AND revoked_at IS NULL AND ($2::bytea IS NULL OR token_hash<>$2)`, operatorID, keep)
	return err
}

func (r *Repository) OperatorForLogin(ctx context.Context, email string) (domain.OperatorLogin, error) {
	var out domain.OperatorLogin
	err := r.db(ctx).QueryRow(ctx, `
		SELECT id::text,password_hash,active,COALESCE(locked_until>now(),false)
		FROM delivery_partner_operators WHERE email=$1`, email).Scan(&out.ID, &out.PasswordHash, &out.Active, &out.Locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, application.ErrNotFound
	}
	return out, err
}

// RecordFailedLogin suma el intento; tras un bloqueo vencido la cuenta empieza de nuevo.
func (r *Repository) RecordFailedLogin(ctx context.Context, operatorID string, max int, lock time.Duration) error {
	_, err := r.db(ctx).Exec(ctx, `
		WITH next AS (
			SELECT id, CASE WHEN locked_until IS NOT NULL AND locked_until<=now() THEN 1 ELSE failed_logins+1 END AS attempts
			FROM delivery_partner_operators WHERE id=$1 FOR UPDATE
		)
		UPDATE delivery_partner_operators o
		SET failed_logins=next.attempts,
		    locked_until=CASE WHEN next.attempts>=$2 THEN now()+make_interval(secs => $3) ELSE NULL END
		FROM next WHERE o.id=next.id`, operatorID, max, lock.Seconds())
	return err
}

func (r *Repository) StartOperatorSession(ctx context.Context, operatorID string, tokenHash []byte, expiresAt time.Time) (domain.OperatorSession, error) {
	var out domain.OperatorSession
	err := r.db(ctx).QueryRow(ctx, `
		WITH signed AS (
			UPDATE delivery_partner_operators SET failed_logins=0,locked_until=NULL,last_login_at=now()
			WHERE id=$1 AND active RETURNING id,partner_id,full_name,email,must_change_password
		), started AS (
			INSERT INTO delivery_partner_sessions(token_hash,operator_id,expires_at)
			SELECT $2,id,$3 FROM signed RETURNING operator_id
		)
		SELECT s.id::text,s.full_name,s.email,s.must_change_password,p.id::text,p.name,p.active
		FROM signed s JOIN started ON started.operator_id=s.id
		JOIN delivery_partners p ON p.id=s.partner_id`, operatorID, tokenHash, expiresAt).
		Scan(&out.OperatorID, &out.OperatorName, &out.Email, &out.MustChangePassword, &out.PartnerID, &out.PartnerName, &out.PartnerActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, application.ErrNotFound
	}
	return out, err
}

func (r *Repository) OperatorSession(ctx context.Context, tokenHash []byte) (domain.OperatorSession, error) {
	var out domain.OperatorSession
	err := r.db(ctx).QueryRow(ctx, `
		SELECT o.id::text,o.full_name,o.email,o.must_change_password,p.id::text,p.name,p.active
		FROM delivery_partner_sessions s
		JOIN delivery_partner_operators o ON o.id=s.operator_id
		JOIN delivery_partners p ON p.id=o.partner_id
		WHERE s.token_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now() AND o.active`, tokenHash).
		Scan(&out.OperatorID, &out.OperatorName, &out.Email, &out.MustChangePassword, &out.PartnerID, &out.PartnerName, &out.PartnerActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, application.ErrNotFound
	}
	return out, err
}

func (r *Repository) RevokeOperatorSession(ctx context.Context, tokenHash []byte) error {
	_, err := r.db(ctx).Exec(ctx, `UPDATE delivery_partner_sessions SET revoked_at=COALESCE(revoked_at,now()) WHERE token_hash=$1`, tokenHash)
	return err
}

func (r *Repository) OperatorPasswordHash(ctx context.Context, operatorID string) (string, error) {
	var hash string
	err := r.db(ctx).QueryRow(ctx, `SELECT password_hash FROM delivery_partner_operators WHERE id=$1`, operatorID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return hash, application.ErrNotFound
	}
	return hash, err
}

func (r *Repository) SetOperatorPassword(ctx context.Context, operatorID, passwordHash string) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE delivery_partner_operators SET password_hash=$2,must_change_password=false,updated_at=now() WHERE id=$1`, operatorID, passwordHash)
	return err
}

func (r *Repository) RecordOperatorAction(ctx context.Context, partnerID, operatorID, action, outcome, orderCode string) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO delivery_partner_operator_actions(partner_id,operator_id,action,outcome,order_code)
		VALUES($1,$2,$3,$4,$5)`, partnerID, operatorID, action, outcome, orderCode)
	return err
}

func (r *Repository) PurgeOperatorSessions(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		DELETE FROM delivery_partner_sessions WHERE expires_at<$1 OR revoked_at<$1`, before)
	return tag.RowsAffected(), err
}
