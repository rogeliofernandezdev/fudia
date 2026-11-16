// Package application holds the Fudia Pay use cases: the pairing code that
// Cash generates, the activation of a device from its QR, the report of
// notifications and the review (associate, confirm, dismiss) by Cash. The
// registered payment is always made by the existing Payments flow; Fudia Pay
// only references it.
package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	billingcontracts "github.com/foods-platform/foods-backend/internal/modules/billing/contracts"
	noticecontracts "github.com/foods-platform/foods-backend/internal/modules/notifications/contracts"
	ordercontracts "github.com/foods-platform/foods-backend/internal/modules/orders/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/paydetect/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

type scope = requestscope.Scope

// Sentinels that the adapters translate into the corresponding HTTP statuses.
var (
	ErrNotificationNotFound = apperror.New(apperror.NotFound, "pay_notification_not_found", "No encontramos el pago detectado.")
	ErrNotificationConflict = apperror.New(apperror.Conflict, "pay_notification_processed", "El pago detectado ya se procesó; recarga la pantalla.")
	ErrDeviceNotFound       = apperror.New(apperror.NotFound, "pay_device_not_found", "No encontramos el dispositivo.")
	ErrPairingInvalid       = apperror.New(apperror.Invalid, "pay_pairing_invalid", "El código de vinculación no es válido.")
	ErrPairingUsed          = apperror.New(apperror.Conflict, "pay_pairing_used", "El código de vinculación ya se usó. Genera otro.")
	ErrPairingExpired       = apperror.New(apperror.Invalid, "pay_pairing_expired", "El código de vinculación expiró. Genera otro.")
	ErrDeviceRevoked        = apperror.New(apperror.Unauthenticated, "pay_device_revoked", "El dispositivo fue revocado o desvinculado. Pide una vinculación nueva.")
	ErrPairingUnavailable   = apperror.New(apperror.Unavailable, "paydetect_unavailable", "No pudimos completar la vinculación. Inténtalo de nuevo.")
)

func payFailure(kind apperror.Kind, code, message string, err error) error {
	if err == nil {
		return apperror.New(kind, code, message)
	}
	return apperror.Wrap(kind, code, message, err)
}

// AdminRepository is the persistence of devices, pairing codes and
// notifications, always scoped by organization and location.
type AdminRepository interface {
	CreatePairingCode(ctx context.Context, s scope, createdBy string, expires time.Time) (domain.PairingDraft, error)
	// PairingState lee el código del local; ErrPairingInvalid si no es suyo o no existe.
	PairingState(ctx context.Context, s scope, id string) (domain.PairingState, error)
	ListDevices(ctx context.Context, s scope) ([]domain.Device, error)
	RenameDevice(ctx context.Context, s scope, deviceID, name string) (domain.Device, error)
	RevokeDevice(ctx context.Context, s scope, deviceID string) (domain.Device, error)
	ListNotifications(ctx context.Context, s scope, filter domain.NotificationFilter) ([]domain.Notification, error)
	Notification(ctx context.Context, s scope, id string) (domain.Notification, error)
	AssociateNotification(ctx context.Context, s scope, id, orderID, userID string, now time.Time) (domain.Notification, error)
	LinkNotification(ctx context.Context, s scope, id, orderID, paymentID, userID string, now time.Time) (domain.Notification, error)
	DismissNotification(ctx context.Context, s scope, id, userID string, now time.Time) (domain.Notification, error)
}

// DeviceRepository is the persistence of the pairing and the device session.
type DeviceRepository interface {
	PairingByCode(ctx context.Context, code string) (domain.PairingCode, error)
	// MarkPairingScanned registra que la app leyó el QR; false si ya no es canjeable.
	MarkPairingScanned(ctx context.Context, code, model string) (bool, error)
	ActivateDevice(ctx context.Context, code string, now time.Time, in domain.PairInput, tokenHash []byte, tokenSuffix string) (domain.Activation, error)
	DeviceByTokenHash(ctx context.Context, tokenHash []byte) (domain.DeviceSession, error)
	TouchDevice(ctx context.Context, deviceID string, now time.Time) error
	ReportEvents(ctx context.Context, session domain.DeviceSession, events []domain.Event, currency string) ([]domain.EventOutcome, error)
}

// MaintenanceRepository keeps the pairing table clean.
type MaintenanceRepository interface {
	SweepExpiredCodes(ctx context.Context, now time.Time) (int64, error)
}

// OrderLooker reads an order through the contracts of the orders module:
// the order must exist within the session scope for the cashier to be able to
// link a detected payment to it.
type OrderLooker interface {
	PayOrderRef(ctx context.Context, scope requestscope.Scope, orderID string) (ordercontracts.PayOrderRef, error)
}

// PayDevices manages the devices of a location from Cash: pairing QR, list,
// rename and revoke.
type PayDevices struct {
	repo AdminRepository
}

func NewPayDevices(repo AdminRepository) *PayDevices { return &PayDevices{repo} }

// PairingCode generates a temporary code, for single use, that Cash shows as
// a QR. It never contains permanent credentials.
func (a *PayDevices) PairingCode(ctx context.Context, s scope, createdBy string, now time.Time) (domain.PairingDraft, error) {
	if s.OrganizationID == "" || s.LocationID == "" {
		return domain.PairingDraft{}, payFailure(apperror.Invalid, "pay_scope_missing", "Abre un local para vincular dispositivos.", nil)
	}
	draft, err := a.repo.CreatePairingCode(ctx, s, createdBy, now.Add(domain.PairingCodeMinutes*time.Minute))
	if err != nil {
		return domain.PairingDraft{}, payFailure(apperror.Unavailable, "pay_pairing_unavailable", "No pudimos generar el código de vinculación.", err)
	}
	return draft, nil
}

// PairingStatus tells Cash what happened with the code it is showing: still
// waiting, scanned by the app (pairing in progress), linked or expired.
func (a *PayDevices) PairingStatus(ctx context.Context, s scope, id string, now time.Time) (domain.PairingStatus, error) {
	if !isUUID(id) {
		return domain.PairingStatus{}, ErrPairingInvalid
	}
	state, err := a.repo.PairingState(ctx, s, id)
	if errors.Is(err, ErrPairingInvalid) {
		return domain.PairingStatus{}, ErrPairingInvalid
	}
	if err != nil {
		return domain.PairingStatus{}, payFailure(apperror.Unavailable, "pay_pairing_unavailable", "No pudimos leer el estado de la vinculación.", err)
	}
	return state.StatusOf(now), nil
}

// Devices lists the devices of the location, active and revoked, with their
// connection status.
func (a *PayDevices) Devices(ctx context.Context, s scope) ([]domain.Device, error) {
	devices, err := a.repo.ListDevices(ctx, s)
	if err != nil {
		return nil, payFailure(apperror.Unavailable, "pay_devices_unavailable", "No pudimos leer los dispositivos.", err)
	}
	return devices, nil
}

// Rename adjusts the administrative name of the device.
func (a *PayDevices) Rename(ctx context.Context, s scope, deviceID, name string) (domain.Device, error) {
	name, err := domain.DeviceName(name)
	if err != nil {
		return domain.Device{}, payFailure(apperror.Invalid, "pay_device_name_invalid", err.Error(), nil)
	}
	device, err := a.repo.RenameDevice(ctx, s, deviceID, name)
	if errors.Is(err, ErrDeviceNotFound) {
		return domain.Device{}, ErrDeviceNotFound
	}
	if err != nil {
		return domain.Device{}, payFailure(apperror.Unavailable, "pay_devices_unavailable", "No pudimos actualizar el dispositivo.", err)
	}
	return device, nil
}

// Revoke disables the device: its credential stops working immediately and
// the app must be paired again to return.
func (a *PayDevices) Revoke(ctx context.Context, s scope, deviceID string) (domain.Device, error) {
	device, err := a.repo.RevokeDevice(ctx, s, deviceID)
	if errors.Is(err, ErrDeviceNotFound) {
		return domain.Device{}, ErrDeviceNotFound
	}
	if err != nil {
		return domain.Device{}, payFailure(apperror.Unavailable, "pay_devices_unavailable", "No pudimos revocar el dispositivo.", err)
	}
	return device, nil
}

// Detector handles the device side: pair the QR code, read the session and
// report notifications. The token is generated here and returned in clear
// only once.
type Detector struct {
	repo    DeviceRepository
	work    unitofwork.Manager
	notices noticecontracts.Publisher
}

func NewDetector(repo DeviceRepository, work unitofwork.Manager) *Detector {
	return &Detector{repo: repo, work: work}
}

// WithNotices registra cada pago detectado en la campana de caja (mejor
// esfuerzo: un aviso fallido no bloquea la sincronización del dispositivo).
func (a *Detector) WithNotices(notices noticecontracts.Publisher) *Detector {
	a.notices = notices
	return a
}

// newDeviceToken generates the device credential: 64 hex characters. It is
// persisted only as a SHA-256 hash and returned in clear once.
func newDeviceToken() (token string, hash []byte, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", nil, err
	}
	token = hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	return token, sum[:], nil
}

// Scan records that the app read the QR, before it redeems the code, so Cash
// can replace the QR with «Vinculando…» at once. It is idempotent and never
// redeems the code: Pair does that.
func (a *Detector) Scan(ctx context.Context, code, model string, now time.Time) error {
	code = strings.TrimSpace(code)
	if len(code) != 32 || !isHex(code) {
		return ErrPairingInvalid
	}
	model = strings.TrimSpace(model)
	if runes := []rune(model); len(runes) > domain.PairingScanModelLimit {
		model = string(runes[:domain.PairingScanModelLimit])
	}
	marked, err := a.repo.MarkPairingScanned(ctx, code, model)
	if err != nil {
		return ErrPairingUnavailable
	}
	if marked {
		return nil
	}
	pairing, err := a.repo.PairingByCode(ctx, code)
	switch {
	case errors.Is(err, ErrPairingInvalid):
		return ErrPairingInvalid
	case err != nil:
		return ErrPairingUnavailable
	case pairing.Used():
		return ErrPairingUsed
	case pairing.Expired(now):
		return ErrPairingExpired
	}
	return ErrPairingUnavailable
}

// Pair exchanges the temporary code for the device credentials. The backend
// resolves the company and location from the code; the device does not send
// any of that. The exchange is atomic: two devices cannot redeem the same code.
func (a *Detector) Pair(ctx context.Context, code string, in domain.PairInput, now time.Time) (domain.Activation, error) {
	code = strings.TrimSpace(code)
	if len(code) != 32 || !isHex(code) {
		return domain.Activation{}, ErrPairingInvalid
	}
	if !domain.ValidPlatform(in.Platform) {
		return domain.Activation{}, payFailure(apperror.Invalid, "pay_platform_unknown", "La plataforma del dispositivo no está soportada.", nil)
	}
	token, tokenHash, err := newDeviceToken()
	if err != nil {
		return domain.Activation{}, payFailure(apperror.Unavailable, "paydetect_unavailable", "No pudimos generar la credencial del dispositivo.", err)
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return domain.Activation{}, ErrPairingUnavailable
	}
	defer tx.Rollback(ctx)

	pairing, err := a.repo.PairingByCode(tx.Context(), code)
	if errors.Is(err, ErrPairingInvalid) {
		return domain.Activation{}, ErrPairingInvalid
	}
	if err != nil {
		return domain.Activation{}, ErrPairingUnavailable
	}
	if pairing.Used() {
		return domain.Activation{}, ErrPairingUsed
	}
	if pairing.Expired(now) {
		return domain.Activation{}, ErrPairingExpired
	}
	activation, err := a.repo.ActivateDevice(tx.Context(), pairing.Code, now, in, tokenHash, token[len(token)-4:])
	if err != nil {
		return domain.Activation{}, ErrPairingUnavailable
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Activation{}, ErrPairingUnavailable
	}
	activation.Token = token
	return activation, nil
}

// isUUID valida la forma de un identificador antes de consultar la base.
func isUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, char := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if char != '-' {
				return false
			}
			continue
		}
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f' || char >= 'A' && char <= 'F') {
			return false
		}
	}
	return true
}

func isHex(value string) bool {
	for _, char := range value {
		switch {
		case char >= '0' && char <= '9', char >= 'a' && char <= 'f':
		default:
			return false
		}
	}
	return true
}

// Session resolves the device session from its credential and marks it as
// seen. A revoked or unpaired credential does not open a session.
func (a *Detector) Session(ctx context.Context, token string) (domain.DeviceSession, error) {
	if strings.TrimSpace(token) == "" {
		return domain.DeviceSession{}, ErrDeviceRevoked
	}
	sum := sha256.Sum256([]byte(token))
	session, err := a.repo.DeviceByTokenHash(ctx, sum[:])
	if errors.Is(err, ErrDeviceNotFound) {
		return domain.DeviceSession{}, ErrDeviceRevoked
	}
	if err != nil {
		return domain.DeviceSession{}, payFailure(apperror.Unavailable, "paydetect_unavailable", "No pudimos leer el dispositivo.", err)
	}
	if !session.Usable() {
		return domain.DeviceSession{}, ErrDeviceRevoked
	}
	return session, nil
}

// ReportMaxEvents acota cada solicitud de sincronización; la cola local de la
// app envía en lotes pequeños con reintentos.
const ReportMaxEvents = 20

// Report registers the notifications reported by a device. The stable
// identity (provider + eventKey) is the idempotency: the same event is never
// registered twice, not even concurrently, and the outcome of each event is
// reported to the device so it can clear its local queue.
func (a *Detector) Report(ctx context.Context, session domain.DeviceSession, events []domain.Event, now time.Time) (domain.ReportResult, error) {
	if !session.Usable() {
		return domain.ReportResult{}, ErrDeviceRevoked
	}
	if len(events) == 0 {
		return domain.ReportResult{}, payFailure(apperror.Invalid, "pay_events_missing", "No hay notificaciones para registrar.", nil)
	}
	if len(events) > ReportMaxEvents {
		return domain.ReportResult{}, payFailure(apperror.Invalid, "pay_events_too_many", fmt.Sprintf("Envía como máximo %d notificaciones por solicitud.", ReportMaxEvents), nil)
	}
	normalized := make([]domain.Event, 0, len(events))
	for _, event := range events {
		clean, err := domain.NormalizeEvent(event)
		if err != nil {
			return domain.ReportResult{}, payFailure(apperror.Invalid, "pay_event_invalid", fmt.Sprintf("La notificación %s no se pudo registrar: %v", event.EventKey, err), nil)
		}
		normalized = append(normalized, clean)
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return domain.ReportResult{}, payFailure(apperror.Unavailable, "paydetect_unavailable", "No pudimos registrar las notificaciones.", err)
	}
	defer tx.Rollback(ctx)
	outcomes, err := a.repo.ReportEvents(tx.Context(), session, normalized, session.Currency)
	if err != nil {
		return domain.ReportResult{}, payFailure(apperror.Unavailable, "paydetect_unavailable", "No pudimos registrar las notificaciones.", err)
	}
	if err = a.repo.TouchDevice(tx.Context(), session.DeviceID, now); err != nil {
		return domain.ReportResult{}, payFailure(apperror.Unavailable, "paydetect_unavailable", "No pudimos registrar las notificaciones.", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.ReportResult{}, payFailure(apperror.Unavailable, "paydetect_unavailable", "No pudimos registrar las notificaciones.", err)
	}
	a.announce(ctx, session, normalized, outcomes)
	result := domain.ReportResult{Events: outcomes}
	for _, outcome := range outcomes {
		if outcome.Status == "registered" {
			result.Registered++
		} else {
			result.Duplicates++
		}
	}
	return result, nil
}

// Notifications is the Cash review: lists the detected payments and advances
// their states. Associating one of them registers the real charge through the
// contracts of the billing module (same flow, no second cash logic).
type PayRegisterer interface {
	PayRegister(ctx context.Context, scope requestscope.Scope, in billingcontracts.PayRegisterInput) (billingcontracts.PayRegisterResult, error)
}

type Notifications struct {
	repo       AdminRepository
	orders     OrderLooker
	registerer PayRegisterer
}

func NewNotifications(repo AdminRepository) *Notifications { return &Notifications{repo: repo} }

func (a *Notifications) WithOrderLooker(orders OrderLooker) *Notifications {
	a.orders = orders
	return a
}

func (a *Notifications) WithPaymentRegisterer(registerer PayRegisterer) *Notifications {
	a.registerer = registerer
	return a
}

// List returns the notifications of the location, optionally filtered by
// status and provider, most recent first.
func (a *Notifications) List(ctx context.Context, s scope, filter domain.NotificationFilter) ([]domain.Notification, error) {
	switch filter.Status {
	case "", domain.NotificationDetected, domain.NotificationAssociated, domain.NotificationConfirmed, domain.NotificationDismissed:
	default:
		return nil, payFailure(apperror.Invalid, "pay_status_unknown", "El estado del filtro no es válido.", nil)
	}
	switch filter.Provider {
	case "", domain.ProviderYape, domain.ProviderPlin:
	default:
		return nil, payFailure(apperror.Invalid, "pay_provider_unknown", "Solo se filtra por Yape y Plin.", nil)
	}
	notifications, err := a.repo.ListNotifications(ctx, s, filter)
	if err != nil {
		return nil, payFailure(apperror.Unavailable, "pay_notifications_unavailable", "No pudimos leer los pagos detectados.", err)
	}
	return notifications, nil
}

// Associate links a detected payment to an order of the same local. The order
// is validated through the contracts of the orders module; tenancy is not
// trusted to the request.
func (a *Notifications) Associate(ctx context.Context, s scope, id, orderID, method, userID string, now time.Time) (domain.Notification, error) {
	orderID = strings.TrimSpace(orderID)
	method = strings.TrimSpace(method)
	if orderID == "" {
		return domain.Notification{}, payFailure(apperror.Invalid, "pay_order_missing", "Selecciona el pedido al que corresponde el pago.", nil)
	}
	if method == "" {
		return domain.Notification{}, payFailure(apperror.Invalid, "pay_method_missing", "Elige el medio de pago con el que se registrará el cobro.", nil)
	}
	notification, err := a.repo.Notification(ctx, s, id)
	if errors.Is(err, ErrNotificationNotFound) {
		return domain.Notification{}, ErrNotificationNotFound
	}
	if err != nil {
		return domain.Notification{}, payFailure(apperror.Unavailable, "pay_notifications_unavailable", "No pudimos leer el pago detectado.", err)
	}
	if notification.Status != domain.NotificationDetected {
		return domain.Notification{}, ErrNotificationConflict
	}
	if a.orders == nil || a.registerer == nil {
		return domain.Notification{}, payFailure(apperror.Unavailable, "paydetect_unavailable", "No pudimos registrar el cobro.", nil)
	}
	order, err := a.orders.PayOrderRef(ctx, s, orderID)
	if errors.Is(err, ordercontracts.ErrPayOrderNotFound) {
		return domain.Notification{}, payFailure(apperror.Invalid, "pay_order_not_found", "El pedido no existe en este local.", nil)
	}
	if err != nil {
		return domain.Notification{}, payFailure(apperror.Unavailable, "paydetect_unavailable", "No pudimos validar el pedido.", err)
	}
	amount, parseErr := strconv.ParseFloat(notification.Amount, 64)
	if parseErr != nil || amount <= 0 {
		return domain.Notification{}, payFailure(apperror.Invalid, "pay_amount_invalid", "El monto detectado no se pudo registrar.", nil)
	}
	// El pago detectado se registra con el flujo de cobros existente: turno,
	// movimientos de caja y reglas del módulo de pagos. Un importe superior al
	// saldo lo rechaza el mismo cobro con sus reglas; la UI lo explica.
	registered, err := a.registerer.PayRegister(ctx, s, billingcontracts.PayRegisterInput{
		OrderID:   order.ID,
		Method:    method,
		Amount:    amount,
		Reference: strings.TrimSpace(fmt.Sprintf("Fudia Pay · %s", notification.EventKey)),
	})
	if err != nil {
		return domain.Notification{}, err
	}
	notification, err = a.repo.LinkNotification(ctx, s, id, order.ID, registered.PaymentID, userID, now)
	if errors.Is(err, ErrNotificationNotFound) {
		return domain.Notification{}, ErrNotificationNotFound
	}
	if errors.Is(err, ErrNotificationConflict) {
		return domain.Notification{}, ErrNotificationConflict
	}
	if err != nil {
		return domain.Notification{}, payFailure(apperror.Unavailable, "pay_notifications_unavailable", "No pudimos asociar el pago.", err)
	}
	return notification, nil
}

// announce deja en la campana de caja un aviso por cada pago registrado
// ahora; los duplicados no vuelven a avisar porque ya están publicados.
func (a *Detector) announce(ctx context.Context, session domain.DeviceSession, events []domain.Event, outcomes []domain.EventOutcome) {
	if a.notices == nil {
		return
	}
	registered := make(map[string]bool, len(outcomes))
	for _, outcome := range outcomes {
		if outcome.Status == "registered" {
			registered[outcome.EventKey] = true
		}
	}
	s := requestscope.Scope{OrganizationID: session.OrganizationID, LocationID: session.LocationID}
	for _, event := range events {
		if !registered[event.EventKey] {
			continue
		}
		provider := strings.ToUpper(event.Provider[:1]) + event.Provider[1:]
		_ = a.notices.Publish(ctx, s, noticecontracts.PayDetected(provider, session.Currency, domain.FormatAmount(event.Amount), event.SenderName, event.EventKey))
	}
}

// Dismiss marks a notification as irrelevant or duplicated: it does not
// delete the event, the review keeps its trace.
func (a *Notifications) Dismiss(ctx context.Context, s scope, id, userID string, now time.Time) (domain.Notification, error) {
	notification, err := a.repo.DismissNotification(ctx, s, id, userID, now)
	if errors.Is(err, ErrNotificationNotFound) {
		return domain.Notification{}, ErrNotificationNotFound
	}
	if errors.Is(err, ErrNotificationConflict) {
		return domain.Notification{}, ErrNotificationConflict
	}
	if err != nil {
		return domain.Notification{}, payFailure(apperror.Unavailable, "pay_notifications_unavailable", "No pudimos descartar el pago.", err)
	}
	return notification, nil
}

// Maintenance keeps the pairing table clean.
type Maintenance struct {
	repo MaintenanceRepository
}

func NewMaintenance(repo MaintenanceRepository) *Maintenance { return &Maintenance{repo} }

// Sweep deletes pairing codes expired past the retention period.
func (a *Maintenance) Sweep(ctx context.Context, now time.Time) (int64, error) {
	return a.repo.SweepExpiredCodes(ctx, now)
}
