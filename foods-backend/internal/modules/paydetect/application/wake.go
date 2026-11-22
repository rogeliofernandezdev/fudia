package application

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/paydetect/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

// ErrPushTokenGone lo devuelve el Pusher cuando el servicio de push ya no
// reconoce el token (app desinstalada o token renovado): se borra del
// dispositivo para no volver a intentarlo.
var ErrPushTokenGone = errors.New("push token no longer registered")

// Pusher envía el aviso silencioso que despierta la app. No lleva datos del
// restaurante ni del pago: solo pide a la app procesar su bandeja y su cola.
type Pusher interface {
	Wake(ctx context.Context, token string) error
}

// WakeRepository reserva los dispositivos a despertar respetando el intervalo
// mínimo (de forma atómica en la base) y olvida los tokens vencidos.
type WakeRepository interface {
	// ClaimWakeTargets devuelve cuántos dispositivos activos del local tienen
	// token de push y reserva los que ya cumplieron el intervalo.
	ClaimWakeTargets(ctx context.Context, s scope, now time.Time, interval time.Duration) (int, []domain.WakeTarget, error)
	ClearPushToken(ctx context.Context, deviceID, token string) error
}

// Waker despierta los teléfonos de Fudia Pay del local. Algunos fabricantes
// (HONOR/MagicOS) congelan la app aunque tenga permisos y solo la descongelan
// al recibir un push: Caja lo pide mientras está abierta y al cobrar con un
// medio que no es efectivo. Es de mejor esfuerzo: un push fallido no afecta a
// Caja y la app conserva su alarma de respaldo.
type Waker struct {
	repo WakeRepository
	push Pusher
	// sendTimeout acota cada envío: la petición de Caja no espera a la red.
	sendTimeout time.Duration
}

func NewWaker(repo WakeRepository, push Pusher) *Waker {
	return &Waker{repo: repo, push: push, sendTimeout: 4 * time.Second}
}

// Enabled dice si hay servicio de push configurado.
func (w *Waker) Enabled() bool { return w != nil && w.push != nil }

// Wake envía el aviso a los dispositivos activos del local que no lo
// recibieron en los últimos domain.WakeInterval.
func (w *Waker) Wake(ctx context.Context, s scope, now time.Time) (domain.WakeResult, error) {
	if !w.Enabled() || s.OrganizationID == "" || s.LocationID == "" {
		return domain.WakeResult{}, nil
	}
	unavailable := func(err error) (domain.WakeResult, error) {
		return domain.WakeResult{}, payFailure(apperror.Unavailable, "pay_wake_unavailable", "No pudimos avisar a los dispositivos de pago.", err)
	}
	devices, targets, err := w.repo.ClaimWakeTargets(ctx, s, now, domain.WakeInterval)
	if err != nil {
		return unavailable(err)
	}
	// Los envíos van en paralelo: con varios teléfonos, Caja espera a lo sumo
	// un sendTimeout y no uno por teléfono.
	results := make([]error, len(targets))
	var wg sync.WaitGroup
	for i, target := range targets {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendCtx, cancel := context.WithTimeout(ctx, w.sendTimeout)
			defer cancel()
			results[i] = w.push.Wake(sendCtx, target.Token)
		}()
	}
	wg.Wait()
	sent := 0
	for i, err := range results {
		switch {
		case err == nil:
			sent++
		case errors.Is(err, ErrPushTokenGone):
			// El token ya no existe: la app registrará uno nuevo al abrir o en su barrido.
			_ = w.repo.ClearPushToken(ctx, targets[i].DeviceID, targets[i].Token)
		}
	}
	return domain.WakeResult{Devices: devices, Sent: sent}, nil
}

// RegisterPush guarda (o borra, si llega vacío) el token de push del
// dispositivo autenticado.
func (a *Detector) RegisterPush(ctx context.Context, session domain.DeviceSession, raw string, now time.Time) error {
	if !session.Usable() {
		return ErrDeviceRevoked
	}
	token, ok := domain.NormalizePushToken(raw)
	if !ok {
		return payFailure(apperror.Invalid, "pay_push_token_invalid", "El token de notificaciones no es válido.", nil)
	}
	if err := a.repo.SavePushToken(ctx, session.DeviceID, token, now); err != nil {
		return payFailure(apperror.Unavailable, "paydetect_unavailable", "No pudimos guardar el token de notificaciones.", err)
	}
	return nil
}
