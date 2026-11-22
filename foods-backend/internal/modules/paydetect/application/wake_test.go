package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/paydetect/domain"
)

type wakeRepo struct {
	targets  []domain.WakeTarget
	interval time.Duration
	cleared  []string
}

func (r *wakeRepo) ClaimWakeTargets(_ context.Context, _ scope, _ time.Time, interval time.Duration) ([]domain.WakeTarget, error) {
	r.interval = interval
	return r.targets, nil
}

func (r *wakeRepo) CountPushDevices(context.Context, scope) (int, error) { return len(r.targets), nil }

func (r *wakeRepo) ClearPushToken(_ context.Context, deviceID, _ string) error {
	r.cleared = append(r.cleared, deviceID)
	return nil
}

type wakePusher map[string]error

func (p wakePusher) Wake(_ context.Context, token string) error { return p[token] }

func TestWakeSendsToClaimedDevicesAndForgetsGoneTokens(t *testing.T) {
	repo := &wakeRepo{targets: []domain.WakeTarget{{DeviceID: "a", Token: "ok"}, {DeviceID: "b", Token: "gone"}, {DeviceID: "c", Token: "down"}}}
	pusher := wakePusher{"gone": ErrPushTokenGone, "down": errors.New("fcm caído")}
	result, err := NewWaker(repo, pusher).Wake(context.Background(), scope{OrganizationID: "o", LocationID: "l"}, time.Now())
	if err != nil || result.Sent != 1 || result.Devices != 3 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if repo.interval != domain.WakeInterval {
		t.Fatalf("intervalo %v", repo.interval)
	}
	if len(repo.cleared) != 1 || repo.cleared[0] != "b" {
		t.Fatalf("solo el token vencido se borra: %v", repo.cleared)
	}
}

func TestWakeWithoutPushOrLocationIsNoop(t *testing.T) {
	repo := &wakeRepo{targets: []domain.WakeTarget{{DeviceID: "a", Token: "ok"}}}
	if result, err := NewWaker(repo, nil).Wake(context.Background(), scope{OrganizationID: "o", LocationID: "l"}, time.Now()); result.Sent != 0 || err != nil {
		t.Fatalf("sin FCM no se envía: %+v %v", result, err)
	}
	if result, _ := NewWaker(repo, wakePusher{}).Wake(context.Background(), scope{OrganizationID: "o"}, time.Now()); result.Sent != 0 || repo.interval != 0 {
		t.Fatal("sin local no se reservan dispositivos")
	}
}

func TestNormalizePushToken(t *testing.T) {
	if token, ok := domain.NormalizePushToken("  abc:DEF_1-2 "); !ok || token != "abc:DEF_1-2" {
		t.Fatalf("token válido rechazado: %q", token)
	}
	if token, ok := domain.NormalizePushToken(""); !ok || token != "" {
		t.Fatal("vacío borra el token")
	}
	if _, ok := domain.NormalizePushToken("con espacio"); ok {
		t.Fatal("token con espacios aceptado")
	}
}
