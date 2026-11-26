package application

import (
	"context"
	"errors"
	"testing"

	noticecontracts "github.com/foods-platform/foods-backend/internal/modules/notifications/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

type stationTx struct{ ctx context.Context }

func (t stationTx) Context() context.Context     { return t.ctx }
func (stationTx) Commit(context.Context) error   { return nil }
func (stationTx) Rollback(context.Context) error { return nil }

type stationWork struct{}

func (stationWork) Begin(ctx context.Context, _ unitofwork.Options) (unitofwork.Transaction, error) {
	return stationTx{ctx}, nil
}

type stationPlate struct{ id, ticket, destination, status string }

// stationRepo keeps the plates of one order in memory and derives the ticket
// status exactly like the SQL: any preparando, else any confirmado, else listo.
type stationRepo struct {
	OrderRepository
	orderID string
	plates  []*stationPlate
	synced  int
}

func (r *stationRepo) LockStationOrder(_ context.Context, _ scope, id, destination string) (string, error) {
	for _, p := range r.plates {
		if p.ticket == id && p.destination == destination {
			return r.orderID, nil
		}
	}
	return "", ErrNotFound
}
func (r *stationRepo) StationStatus(_ context.Context, _ scope, _ string, id, destination string) (string, error) {
	confirmed := false
	for _, p := range r.plates {
		if p.ticket != id || p.destination != destination {
			continue
		}
		if p.status == "preparando" {
			return "preparando", nil
		}
		confirmed = confirmed || p.status == "confirmado"
	}
	if confirmed {
		return "confirmado", nil
	}
	return "listo", nil
}
func (r *stationRepo) StationPlateCounts(_ context.Context, _ scope, _ string, id, destination string) (int, int, error) {
	confirmed, preparing := 0, 0
	for _, p := range r.plates {
		if p.ticket == id && p.destination == destination {
			if p.status == "confirmado" {
				confirmed++
			}
			if p.status == "preparando" {
				preparing++
			}
		}
	}
	return confirmed, preparing, nil
}

// previousStep es el único estado desde el que un plato llega a status.
func previousStep(status string) string {
	if status == "preparando" {
		return "confirmado"
	}
	return "preparando"
}
func (r *stationRepo) UpdateStationItems(_ context.Context, _ scope, _ string, id, destination, status string) (int64, error) {
	var n int64
	for _, p := range r.plates {
		if p.ticket == id && p.destination == destination && p.status == previousStep(status) {
			p.status = status
			n++
		}
	}
	return n, nil
}
func (r *stationRepo) UpdateStationItem(_ context.Context, _ scope, _ string, id, destination, itemID, status string) (int64, error) {
	for _, p := range r.plates {
		if p.id != itemID || p.ticket != id || p.destination != destination {
			continue
		}
		if (status == "listo" && p.status == "preparando") || (status == "preparando" && (p.status == "confirmado" || p.status == "listo")) {
			p.status = status
			return 1, nil
		}
	}
	return 0, nil
}
func (r *stationRepo) UpdateStationRound(context.Context, scope, string) (int64, error) {
	return 1, nil
}
func (r *stationRepo) SyncServiceStatus(context.Context, scope, string) error {
	r.synced++
	return nil
}
func (r *stationRepo) Record(_ context.Context, _ scope, id string) (domain.Order, error) {
	return domain.Order{ID: id, Code: "PED-1", Status: "preparando"}, nil
}

type stationAudit struct{ actions []string }

func (a *stationAudit) Record(_ context.Context, _ requestscope.Scope, action, _, _ string) error {
	a.actions = append(a.actions, action)
	return nil
}

type stationNotices struct{ ready []noticecontracts.Notice }

func (n *stationNotices) Publish(_ context.Context, _ requestscope.Scope, notice noticecontracts.Notice) error {
	n.ready = append(n.ready, notice)
	return nil
}
func (n *stationNotices) PublishCustomer(context.Context, requestscope.Scope, noticecontracts.CustomerNotice) error {
	return nil
}

func stationService(plates ...*stationPlate) (*Orders, *stationRepo, *stationAudit, *stationNotices) {
	repo := &stationRepo{orderID: "order-1", plates: plates}
	audit, notices := &stationAudit{}, &stationNotices{}
	return New(repo, stationWork{}, audit, nil).WithNotices(notices), repo, audit, notices
}

func failureCode(err error) string {
	var failure *apperror.Error
	if errors.As(err, &failure) {
		return failure.Code
	}
	return ""
}

func TestKitchenItemReadyKeepsTicketOpenUntilLastPlate(t *testing.T) {
	service, repo, audit, notices := stationService(
		&stationPlate{"lomo", "round-1", "kitchen", "preparando"},
		&stationPlate{"ceviche", "round-1", "kitchen", "preparando"},
	)
	ctx := context.Background()
	if err := service.UpdateKitchenItem(ctx, scope{}, "round-1", "lomo", OrderStatusInput{Status: "listo"}); err != nil {
		t.Fatal(err)
	}
	if repo.plates[0].status != "listo" || repo.plates[1].status != "preparando" || len(notices.ready) != 0 {
		t.Fatalf("one plate ready must not close the ticket: plates=%+v notices=%d", repo.plates, len(notices.ready))
	}
	if len(audit.actions) != 1 || audit.actions[0] != "preparation.item.listo" || repo.synced != 1 {
		t.Fatalf("plate audit/sync: %v synced=%d", audit.actions, repo.synced)
	}
	if err := service.UpdateKitchenItem(ctx, scope{}, "round-1", "ceviche", OrderStatusInput{Status: "listo"}); err != nil {
		t.Fatal(err)
	}
	if len(notices.ready) != 1 || audit.actions[len(audit.actions)-1] != "preparation.listo" {
		t.Fatalf("last plate must close the ticket like «Marcar listo»: notices=%+v audit=%v", notices.ready, audit.actions)
	}
}

func TestKitchenLastPlateMatchesWholeTicketReady(t *testing.T) {
	byPlate, _, plateAudit, plateNotices := stationService(&stationPlate{"chicha", "round-1", "bar", "preparando"})
	if err := byPlate.UpdateKitchenItem(context.Background(), scope{}, "round-1~bar", "chicha", OrderStatusInput{Status: "listo"}); err != nil {
		t.Fatal(err)
	}
	byTicket, _, ticketAudit, ticketNotices := stationService(&stationPlate{"chicha", "round-1", "bar", "preparando"})
	if err := byTicket.UpdateKitchen(context.Background(), scope{}, "round-1~bar", OrderStatusInput{Status: "listo"}); err != nil {
		t.Fatal(err)
	}
	if len(plateNotices.ready) != 1 || len(ticketNotices.ready) != 1 {
		t.Fatalf("both paths notify once: plate=%d ticket=%d", len(plateNotices.ready), len(ticketNotices.ready))
	}
	if plateNotices.ready[0] != ticketNotices.ready[0] {
		t.Fatalf("ready notice differs:\nplate  %+v\nticket %+v", plateNotices.ready[0], ticketNotices.ready[0])
	}
	if got, want := plateAudit.actions[len(plateAudit.actions)-1], ticketAudit.actions[0]; got != want {
		t.Fatalf("ready audit differs: %s vs %s", got, want)
	}
}

// Cada plato empieza por sí solo y ninguno se salta un estado: el lomo de 20 min
// arranca primero, el ceviche después, y la comanda solo queda lista con ambos hechos.
func TestKitchenPlatesStartIndependentlyAndNeverSkipStates(t *testing.T) {
	service, repo, audit, notices := stationService(
		&stationPlate{"lomo", "round-1", "kitchen", "confirmado"},
		&stationPlate{"ceviche", "round-1", "kitchen", "confirmado"},
	)
	ctx := context.Background()
	if code := failureCode(service.UpdateKitchenItem(ctx, scope{}, "round-1", "lomo", OrderStatusInput{Status: "listo"})); code != "kitchen_item_unavailable" {
		t.Fatalf("a plate cannot jump from pending to ready: %q", code)
	}
	if err := service.UpdateKitchenItem(ctx, scope{}, "round-1", "lomo", OrderStatusInput{Status: "preparando"}); err != nil {
		t.Fatal(err)
	}
	if repo.plates[0].status != "preparando" || repo.plates[1].status != "confirmado" {
		t.Fatalf("only the lomo starts: %+v", repo.plates)
	}
	if audit.actions[len(audit.actions)-1] != "preparation.preparando" {
		t.Fatalf("the first plate started starts the ticket like «Iniciar»: %v", audit.actions)
	}
	if code := failureCode(service.UpdateKitchen(ctx, scope{}, "round-1", OrderStatusInput{Status: "listo"})); code != "kitchen_plates_not_started" {
		t.Fatalf("«Marcar listo» needs every plate started: %q", code)
	}
	// «Iniciar» sobre la comanda en marcha empieza solo lo que faltaba.
	if err := service.UpdateKitchen(ctx, scope{}, "round-1", OrderStatusInput{Status: "preparando"}); err != nil {
		t.Fatal(err)
	}
	if repo.plates[0].status != "preparando" || repo.plates[1].status != "preparando" {
		t.Fatalf("start the rest: %+v", repo.plates)
	}
	if code := failureCode(service.UpdateKitchen(ctx, scope{}, "round-1", OrderStatusInput{Status: "preparando"})); code != "invalid_kitchen_transition" {
		t.Fatalf("nothing left to start: %q", code)
	}
	if err := service.UpdateKitchen(ctx, scope{}, "round-1", OrderStatusInput{Status: "listo"}); err != nil {
		t.Fatal(err)
	}
	for _, p := range repo.plates {
		if p.status != "listo" {
			t.Fatalf("«Marcar listo» finishes the plates in preparation: %+v", repo.plates)
		}
	}
	if len(notices.ready) != 1 {
		t.Fatalf("ready notice: %d", len(notices.ready))
	}
}

func TestKitchenItemRejectsForeignPlatesAndClosedTickets(t *testing.T) {
	service, _, _, _ := stationService(
		&stationPlate{"lomo", "round-1", "kitchen", "listo"},
		&stationPlate{"chicha", "round-1", "bar", "preparando"},
		&stationPlate{"arroz", "round-2", "kitchen", "preparando"},
	)
	ctx := context.Background()
	cases := []struct {
		name, ticket, item, status, code string
	}{
		{"invalid status", "round-1", "lomo", "entregado", "invalid_kitchen_status"},
		{"unknown ticket", "round-9", "lomo", "listo", "kitchen_ticket_not_found"},
		{"bar plate from kitchen ticket", "round-1", "chicha", "listo", "kitchen_item_unavailable"},
		{"plate of another round", "round-1", "arroz", "listo", "kitchen_item_unavailable"},
		{"already ready", "round-1", "lomo", "listo", "kitchen_item_unavailable"},
		{"undo on a ready ticket", "round-1", "lomo", "preparando", "kitchen_ticket_ready"},
		{"undo a plate not ready", "round-1~bar", "chicha", "preparando", "kitchen_item_unavailable"},
	}
	for _, c := range cases {
		if code := failureCode(service.UpdateKitchenItem(ctx, scope{}, c.ticket, c.item, OrderStatusInput{Status: c.status})); code != c.code {
			t.Fatalf("%s: got %q want %q", c.name, code, c.code)
		}
	}
}
