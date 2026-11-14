package application

import (
	"context"
	"testing"

	"github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
)

type pickupRepository struct {
	DispatchRepository
	pickedUp bool
}

func (r *pickupRepository) MarkPickedUp(context.Context, scope, string) error {
	r.pickedUp = true
	return nil
}

func TestPickupRequiresConfirmedPayment(t *testing.T) {
	for _, tc := range []struct {
		name, orderStatus, dispatchStatus, outcome string
		paid, transition, marked                   bool
	}{
		{"preparing unpaid", "preparando", domain.DispatchAccepted, domain.OutcomeNotReady, false, false, false},
		{"preparing paid", "preparando", domain.DispatchAccepted, domain.OutcomeNotReady, true, false, false},
		{"ready unpaid or partial", "listo", domain.DispatchAccepted, domain.OutcomePaymentPending, false, false, false},
		{"ready paid", "listo", domain.DispatchAccepted, "", true, true, false},
		{"staff dispatched unpaid", "en_camino", domain.DispatchAccepted, domain.OutcomePaymentPending, false, false, false},
		{"staff dispatched paid", "en_camino", domain.DispatchAccepted, domain.OutcomePickedUp, true, false, true},
		{"already picked up remains idempotent", "en_camino", domain.DispatchPickedUp, domain.OutcomePickedUp, false, false, false},
		{"cancelled paid", "listo", domain.DispatchCancelled, domain.OutcomeCancelled, true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &pickupRepository{}
			dispatcher := NewDispatcher(repo, nil)
			state := OfferState{OfferStatus: domain.OfferAccepted, CurrentRound: true, Assigned: true,
				OrderStatus: tc.orderStatus, DispatchStatus: tc.dispatchStatus, PaymentConfirmed: tc.paid}
			outcome, transition, err := dispatcher.checkPickup(context.Background(), scope{}, state)
			if err != nil || outcome != tc.outcome || transition != tc.transition || repo.pickedUp != tc.marked {
				t.Fatalf("got outcome=%q transition=%v marked=%v err=%v", outcome, transition, repo.pickedUp, err)
			}
		})
	}
}
