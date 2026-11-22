// Package postgres adapts the Fudia Pay persistence: devices, temporary
// pairing codes and detected payment notifications. Every query is scoped by
// organization and location resolved in the session; nothing is trusted to
// the client.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/paydetect/application"
	"github.com/foods-platform/foods-backend/internal/modules/paydetect/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type scope = requestscope.Scope

type Pay struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Pay { return &Pay{pool} }

func (r *Pay) db(ctx context.Context) database.Executor { return database.ExecutorFor(ctx, r.pool) }

var (
	_ application.AdminRepository       = (*Pay)(nil)
	_ application.DeviceRepository      = (*Pay)(nil)
	_ application.MaintenanceRepository = (*Pay)(nil)
	_ application.WakeRepository        = (*Pay)(nil)
)

const deviceColumns = `d.id::text, d.name, d.platform, d.model, d.token_suffix, d.active, d.last_seen_at, d.created_at, d.revoked_at`

// deviceReturning es la lista de columnas sin alias de tabla, para el
// RETURNING de los INSERT que no pueden nombrar la tabla.
const deviceReturning = `id::text, name, platform, model, token_suffix, active, last_seen_at, created_at, revoked_at`

func scanDevice(row pgx.Row) (domain.Device, error) {
	var device domain.Device
	var lastSeen, revoked *time.Time
	var created time.Time
	if err := row.Scan(&device.ID, &device.Name, &device.Platform, &device.Model, &device.TokenSuffix, &device.Active, &lastSeen, &created, &revoked); err != nil {
		return domain.Device{}, err
	}
	if lastSeen != nil {
		device.LastSeenAt = lastSeen.Format(time.RFC3339)
	}
	if revoked != nil {
		device.RevokedAt = revoked.Format(time.RFC3339)
	}
	device.CreatedAt = created.Format(time.RFC3339)
	return device, nil
}

const notificationColumns = `
	n.id::text, n.device_id::text, d.name, n.provider, n.event_key, n.operation_number, n.sender_name,
	n.amount::text, n.currency, n.occurred_at, n.status,
	COALESCE(n.order_id::text, ''), COALESCE(n.payment_id::text, ''),
	COALESCE(n.confirmed_by::text, ''), n.confirmed_at, n.created_at`

func scanNotification(row pgx.Row) (domain.Notification, error) {
	var notification domain.Notification
	var occurred, created time.Time
	var confirmed *time.Time
	if err := row.Scan(
		&notification.ID, &notification.DeviceID, &notification.DeviceName, &notification.Provider,
		&notification.EventKey, &notification.OperationNumber, &notification.SenderName, &notification.Amount, &notification.Currency,
		&occurred, &notification.Status, &notification.OrderID, &notification.PaymentID,
		&notification.ConfirmedBy, &confirmed, &created,
	); err != nil {
		return domain.Notification{}, err
	}
	notification.OccurredAt = occurred.Format(time.RFC3339)
	notification.CreatedAt = created.Format(time.RFC3339)
	if confirmed != nil {
		notification.ConfirmedAt = confirmed.Format(time.RFC3339)
	}
	return notification, nil
}

// CreatePairingCode generates the temporary, single-use code. PostgreSQL
// generates the code itself; the clear token is never stored.
func (r *Pay) CreatePairingCode(ctx context.Context, s scope, createdBy string, expires time.Time) (domain.PairingDraft, error) {
	var draft domain.PairingDraft
	var expiresAt time.Time
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO pay_pairing_codes (organization_id, location_id, code, created_by, expires_at)
		VALUES ($1, $2, replace(gen_random_uuid()::text, '-', ''), $3, $4)
		RETURNING id::text, code, expires_at
	`, s.OrganizationID, s.LocationID, createdBy, expires).Scan(&draft.ID, &draft.Code, &expiresAt)
	if err != nil {
		return domain.PairingDraft{}, err
	}
	draft.ExpiresAt = expiresAt.Format(time.RFC3339)
	return draft, nil
}

// PairingState reads the code Cash is showing, within its location, with the
// name of the device it linked.
func (r *Pay) PairingState(ctx context.Context, s scope, id string) (domain.PairingState, error) {
	var state domain.PairingState
	var model, deviceName *string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT p.expires_at, p.used_at, p.scanned_at, p.scanned_model, d.name
		FROM pay_pairing_codes p
		LEFT JOIN pay_devices d ON d.id=p.device_id
		WHERE p.id=$1 AND p.organization_id=$2 AND p.location_id=$3
	`, id, s.OrganizationID, s.LocationID).Scan(&state.ExpiresAt, &state.UsedAt, &state.ScannedAt, &model, &deviceName)
	if errors.Is(err, pgx.ErrNoRows) {
		return state, application.ErrPairingInvalid
	}
	if model != nil {
		state.Model = *model
	}
	if deviceName != nil {
		state.DeviceName = *deviceName
	}
	return state, err
}

// MarkPairingScanned records that the app read the QR. The first scan keeps
// its time; a later scan only fills the model if it was missing.
func (r *Pay) MarkPairingScanned(ctx context.Context, code, model string) (bool, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE pay_pairing_codes
		SET scanned_at=COALESCE(scanned_at, now()), scanned_model=COALESCE(scanned_model, NULLIF($2, ''))
		WHERE code=$1 AND used_at IS NULL AND expires_at > now()
	`, code, model)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// LocationExists confirms the location belongs to the session organization.
func (r *Pay) LocationExists(ctx context.Context, s scope, locationID string) (bool, error) {
	var exists bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM locations WHERE id=$1 AND organization_id=$2)`, locationID, s.OrganizationID).Scan(&exists)
	return exists, err
}

func deviceWhere(s scope, filter domain.DeviceFilter) (string, []any) {
	where := ` WHERE d.organization_id=$1 AND d.location_id=$2`
	switch filter.Status {
	case domain.DeviceStatusActive:
		where += ` AND d.active`
	case domain.DeviceStatusRevoked:
		where += ` AND NOT d.active`
	}
	return where, []any{s.OrganizationID, s.LocationID}
}

// CountDevices counts the devices of the location that match the filter.
func (r *Pay) CountDevices(ctx context.Context, s scope, filter domain.DeviceFilter) (int, error) {
	where, args := deviceWhere(s, filter)
	var total int
	err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM pay_devices d`+where, args...).Scan(&total)
	return total, err
}

// ListDevices pages the devices of the location: active first, then the most
// recently paired.
func (r *Pay) ListDevices(ctx context.Context, s scope, filter domain.DeviceFilter) ([]domain.Device, error) {
	where, args := deviceWhere(s, filter)
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	args = append(args, limit, max(filter.Offset, 0))
	rows, err := r.db(ctx).Query(ctx, `
		SELECT `+deviceColumns+`
		FROM pay_devices d`+where+`
		ORDER BY d.active DESC, d.created_at DESC, d.id
		LIMIT $3 OFFSET $4
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	devices := []domain.Device{}
	for rows.Next() {
		device, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}
	return devices, rows.Err()
}

// RevokeDevice disables the credential: the device stops being usable from
// that moment, even with the token in hand.
func (r *Pay) RevokeDevice(ctx context.Context, s scope, deviceID string) (domain.Device, error) {
	device, err := scanDevice(r.db(ctx).QueryRow(ctx, `
		UPDATE pay_devices SET active=false, revoked_at=COALESCE(revoked_at, now()), push_token=NULL, updated_at=now()
		WHERE id=$2 AND organization_id=$1 AND location_id=$3
		RETURNING `+deviceReturning,
		s.OrganizationID, deviceID, s.LocationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Device{}, application.ErrDeviceNotFound
	}
	return device, err
}

// ListNotifications returns the detected payments of the location, most
// recent first, with optional status and provider filters.
func notificationWhere(s scope, filter domain.NotificationFilter) (string, []any) {
	where := ` WHERE n.organization_id=$1 AND n.location_id=$2`
	args := []any{s.OrganizationID, s.LocationID}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where += fmt.Sprintf(" AND n.status=$%d", len(args))
	}
	if filter.Provider != "" {
		args = append(args, filter.Provider)
		where += fmt.Sprintf(" AND n.provider=$%d", len(args))
	}
	if filter.Query != "" {
		// Remitente sin tildes ni mayúsculas; el monto, si lo escrito es una
		// cifra, por prefijo de su texto («25» → 25.00, 25.50, 250.00).
		args = append(args, filter.Query)
		name := len(args)
		match := fmt.Sprintf("translate(lower(n.sender_name),'áéíóúüñ','aeiouun') LIKE '%%'||translate(lower($%d),'áéíóúüñ','aeiouun')||'%%'", name)
		if amount := domain.NotificationAmountPrefix(filter.Query); amount != "" {
			args = append(args, amount)
			match += fmt.Sprintf(" OR n.amount::text LIKE $%d||'%%'", len(args))
		}
		where += " AND (" + match + ")"
	}
	return where, args
}

// CountNotifications counts the detected payments that match the filters.
func (r *Pay) CountNotifications(ctx context.Context, s scope, filter domain.NotificationFilter) (int, error) {
	where, args := notificationWhere(s, filter)
	var total int
	err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM pay_notifications n`+where, args...).Scan(&total)
	return total, err
}

func (r *Pay) ListNotifications(ctx context.Context, s scope, filter domain.NotificationFilter) ([]domain.Notification, error) {
	where, args := notificationWhere(s, filter)
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	args = append(args, limit)
	rows, err := r.db(ctx).Query(ctx, `
		SELECT `+notificationColumns+`
		FROM pay_notifications n
		JOIN pay_devices d ON d.id=n.device_id
		`+where+`
		ORDER BY n.created_at DESC, n.id
		LIMIT $`+fmt.Sprint(len(args))+fmt.Sprintf(" OFFSET %d", max(filter.Offset, 0)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	notifications := []domain.Notification{}
	for rows.Next() {
		notification, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	return notifications, rows.Err()
}

// Notification reads one notification of the location.
func (r *Pay) Notification(ctx context.Context, s scope, id string) (domain.Notification, error) {
	notification, err := scanNotification(r.db(ctx).QueryRow(ctx, `
		SELECT `+notificationColumns+`
		FROM pay_notifications n
		JOIN pay_devices d ON d.id=n.device_id
		WHERE n.id=$1::uuid AND n.organization_id=$2 AND n.location_id=$3
	`, database.UUIDArg(id), s.OrganizationID, s.LocationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Notification{}, application.ErrNotificationNotFound
	}
	return notification, err
}

// AssociateNotification links a detected payment to an order. Only a
// notification still in detected state can be linked.
func (r *Pay) AssociateNotification(ctx context.Context, s scope, id, orderID, userID string, now time.Time) (domain.Notification, error) {
	if err := r.guardedTransition(ctx, s, id, `SET status='associated', order_id=$4, updated_at=now()`, "AND status='detected'", []any{orderID}); err != nil {
		return domain.Notification{}, err
	}
	return r.Notification(ctx, s, id)
}

// LinkNotification ties the notification to the order and to the real payment
// registered by the payments flow. Only a notification still in detected
// state can be linked: two cashiers cannot use the same movement.
func (r *Pay) LinkNotification(ctx context.Context, s scope, id, orderID, paymentID, userID string, now time.Time) (domain.Notification, error) {
	if err := r.guardedTransition(ctx, s, id, `
		SET status='associated', order_id=$4, payment_id=$5, updated_at=now()`,
		"AND status='detected'", []any{orderID, paymentID}); err != nil {
		return domain.Notification{}, err
	}
	return r.Notification(ctx, s, id)
}

// DismissNotification marks the event as reviewed and irrelevant without
// deleting it.
func (r *Pay) DismissNotification(ctx context.Context, s scope, id, userID string, now time.Time) (domain.Notification, error) {
	if err := r.guardedTransition(ctx, s, id, `SET status='dismissed', confirmed_by=$4::uuid, updated_at=now()`, "AND status IN ('detected','associated')", []any{userID}); err != nil {
		return domain.Notification{}, err
	}
	return r.Notification(ctx, s, id)
}

// guardedTransition applies a state change guarded by the source states and
// distinguishes «no existe» from «ya se procesó»: la fila de notificación no
// se une a otras tablas, así que el id devuelto alimenta la relectura.
func (r *Pay) guardedTransition(ctx context.Context, s scope, id string, assignments, guard string, args []any) error {
	var updated string
	row := r.db(ctx).QueryRow(ctx, `
		UPDATE pay_notifications `+assignments+`
		WHERE id=$1 AND organization_id=$2 AND location_id=$3 `+guard+`
		RETURNING id::text
	`, append([]any{id, s.OrganizationID, s.LocationID}, args...)...).Scan(&updated)
	if errors.Is(row, pgx.ErrNoRows) {
		_, err := r.resolveTransition(ctx, s, id)
		return err
	}
	return row
}

// resolveTransition distinguishes a missing notification from one that was
// already processed by another operator.
func (r *Pay) resolveTransition(ctx context.Context, s scope, id string) (domain.Notification, error) {
	_, err := r.Notification(ctx, s, id)
	if errors.Is(err, application.ErrNotificationNotFound) {
		return domain.Notification{}, application.ErrNotificationNotFound
	}
	if err != nil {
		return domain.Notification{}, err
	}
	return domain.Notification{}, application.ErrNotificationConflict
}

// PairingByCode reads a pairing code by its content; the state validation is
// done by the application with its own clock.
func (r *Pay) PairingByCode(ctx context.Context, code string) (domain.PairingCode, error) {
	var pairing domain.PairingCode
	var expires, created time.Time
	var used *time.Time
	var device *string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT id::text, organization_id::text, location_id::text, code, created_by::text, expires_at, used_at, device_id::text, created_at
		FROM pay_pairing_codes
		WHERE code=$1
	`, code).Scan(&pairing.ID, &pairing.OrganizationID, &pairing.LocationID, &pairing.Code, &pairing.CreatedBy, &expires, &used, &device, &created)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PairingCode{}, application.ErrPairingInvalid
	}
	if err != nil {
		return domain.PairingCode{}, err
	}
	pairing.ExpiresAt, pairing.CreatedAt = expires, created
	pairing.UsedAt = used
	if device != nil {
		pairing.DeviceID = *device
	}
	return pairing, nil
}

// ActivateDevice marks the code as used and creates the device with its
// credential hash. It runs inside the transaction opened by the application:
// two devices cannot redeem the same code, not even concurrently.
func (r *Pay) ActivateDevice(ctx context.Context, code string, now time.Time, in domain.PairInput, tokenHash []byte, tokenSuffix string) (domain.Activation, error) {
	var pairing struct {
		id, organizationID, locationID string
	}
	err := r.db(ctx).QueryRow(ctx, `
		UPDATE pay_pairing_codes SET used_at=now()
		WHERE code=$1 AND used_at IS NULL AND expires_at > now()
		RETURNING id::text, organization_id::text, location_id::text
	`, code).Scan(&pairing.id, &pairing.organizationID, &pairing.locationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Activation{}, application.ErrPairingUsed
	}
	if err != nil {
		return domain.Activation{}, err
	}
	name := strings.TrimSpace(in.Model)
	if name == "" {
		name = "Fudia Pay"
	}
	if length := len([]rune(name)); length > domain.DeviceNameLimit {
		runes := []rune(name)
		name = string(runes[:domain.DeviceNameLimit])
	}
	var device domain.Device
	err = func() error {
		device, err = scanDevice(r.db(ctx).QueryRow(ctx, `
			INSERT INTO pay_devices (organization_id, location_id, name, platform, model, token_hash, token_suffix)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING `+deviceReturning,
			pairing.organizationID, pairing.locationID, name, in.Platform, strings.TrimSpace(in.Model), tokenHash, tokenSuffix))
		return err
	}()
	if err != nil {
		return domain.Activation{}, err
	}
	// The code remembers the device it linked: Cash shows its name when done.
	if _, err = r.db(ctx).Exec(ctx, `UPDATE pay_pairing_codes SET device_id=$2 WHERE id=$1`, pairing.id, device.ID); err != nil {
		return domain.Activation{}, err
	}
	var organizationName, locationName string
	err = r.db(ctx).QueryRow(ctx, `
		SELECT COALESCE(NULLIF(btrim(o.trade_name), ''), o.legal_name), l.name
		FROM organizations o
		JOIN locations l ON l.id=$2 AND l.organization_id=o.id
		WHERE o.id=$1
	`, pairing.organizationID, pairing.locationID).Scan(&organizationName, &locationName)
	if err != nil {
		return domain.Activation{}, err
	}
	return domain.Activation{Device: device, OrganizationName: organizationName, LocationName: locationName}, nil
}

// DeviceByTokenHash resolves the session from the SHA-256 of the credential,
// including the tenancy and the currency of the restaurant.
func (r *Pay) DeviceByTokenHash(ctx context.Context, tokenHash []byte) (domain.DeviceSession, error) {
	var session domain.DeviceSession
	var revoked *time.Time
	err := r.db(ctx).QueryRow(ctx, `
		SELECT d.id::text, d.name, d.active, d.revoked_at, d.organization_id::text, d.location_id::text, o.currency, COALESCE(NULLIF(btrim(o.trade_name), ''), o.legal_name), l.name
		FROM pay_devices d
		JOIN organizations o ON o.id=d.organization_id
		JOIN locations l ON l.id=d.location_id
		WHERE d.token_hash=$1
	`, tokenHash).Scan(&session.DeviceID, &session.DeviceName, &session.Active, &revoked, &session.OrganizationID, &session.LocationID, &session.Currency, &session.OrganizationName, &session.LocationName)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.DeviceSession{}, application.ErrDeviceNotFound
	}
	if err != nil {
		return domain.DeviceSession{}, err
	}
	session.RevokedAt = revoked
	return session, nil
}

// SavePushToken guarda el token de push del dispositivo; vacío lo borra. Un
// dispositivo revocado no lo recupera.
func (r *Pay) SavePushToken(ctx context.Context, deviceID, token string, now time.Time) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE pay_devices SET push_token=NULLIF($2, ''), push_token_updated_at=$3, last_seen_at=$3
		WHERE id=$1 AND active`, deviceID, token, now)
	return err
}

// ClaimWakeTargets cuenta los dispositivos activos del local con token de push
// (índice parcial pay_devices_push_location_idx) y reserva, en la misma
// sentencia, los que no recibieron un aviso en el intervalo: dos pestañas o
// instancias simultáneas no duplican el push. Caja lo llama cada 10–30 s, por
// eso es un solo viaje a la base. Siempre devuelve al menos una fila: sin
// dispositivos que reservar, la fila trae solo el total.
func (r *Pay) ClaimWakeTargets(ctx context.Context, s scope, now time.Time, interval time.Duration) (int, []domain.WakeTarget, error) {
	rows, err := r.db(ctx).Query(ctx, `
		WITH claimed AS (
			UPDATE pay_devices SET woken_at=$3
			WHERE organization_id=$1 AND location_id=$2 AND active AND push_token IS NOT NULL
			  AND (woken_at IS NULL OR woken_at <= $3::timestamptz - make_interval(secs => $4::float8))
			RETURNING id::text AS id, push_token
		)
		SELECT (SELECT count(*)::int FROM pay_devices
		        WHERE organization_id=$1 AND location_id=$2 AND active AND push_token IS NOT NULL),
		       claimed.id, claimed.push_token
		FROM (SELECT 1) AS one LEFT JOIN claimed ON true`,
		s.OrganizationID, s.LocationID, now, interval.Seconds())
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	devices := 0
	var targets []domain.WakeTarget
	for rows.Next() {
		var id, token *string
		if err := rows.Scan(&devices, &id, &token); err != nil {
			return 0, nil, err
		}
		if id != nil && token != nil {
			targets = append(targets, domain.WakeTarget{DeviceID: *id, Token: *token})
		}
	}
	return devices, targets, rows.Err()
}

// ClearPushToken olvida un token que el servicio de push ya no reconoce,
// salvo que la app ya lo haya reemplazado por otro.
func (r *Pay) ClearPushToken(ctx context.Context, deviceID, token string) error {
	_, err := r.db(ctx).Exec(ctx, `UPDATE pay_devices SET push_token=NULL WHERE id=$1 AND push_token=$2`, deviceID, token)
	return err
}

// TouchDevice records the last contact of the device.
func (r *Pay) TouchDevice(ctx context.Context, deviceID string, now time.Time) error {
	_, err := r.db(ctx).Exec(ctx, `UPDATE pay_devices SET last_seen_at=$2 WHERE id=$1`, deviceID, now)
	return err
}

// ReportEvents inserts the reported notifications; the idempotency lives in
// the unique (device_id, event_key): an event already registered stays as
// duplicate and the device clears it from its local queue.
func (r *Pay) ReportEvents(ctx context.Context, session domain.DeviceSession, events []domain.Event, currency string) ([]domain.EventOutcome, error) {
	outcomes := make([]domain.EventOutcome, 0, len(events))
	for _, event := range events {
		var id *string
		err := r.db(ctx).QueryRow(ctx, `
			INSERT INTO pay_notifications (device_id, organization_id, location_id, provider, event_key, sender_name, amount, currency, occurred_at, operation_number)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (device_id, event_key) DO NOTHING
			RETURNING id::text
		`, session.DeviceID, session.OrganizationID, session.LocationID, event.Provider, event.EventKey, event.SenderName, event.Amount, currency, event.OccurredAt, event.OperationNumber).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			outcomes = append(outcomes, domain.EventOutcome{EventKey: event.EventKey, Status: "duplicate"})
			continue
		}
		if err != nil {
			return nil, err
		}
		outcomes = append(outcomes, domain.EventOutcome{EventKey: event.EventKey, Status: "registered"})
	}
	return outcomes, nil
}

// SweepExpiredCodes deletes pairing codes expired past the retention period.
func (r *Pay) SweepExpiredCodes(ctx context.Context, now time.Time) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `DELETE FROM pay_pairing_codes WHERE expires_at < $1`, now.Add(-domain.PairingCodeRetentionDays*24*time.Hour))
	return tag.RowsAffected(), err
}
