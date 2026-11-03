package application

import (
	"context"
	"testing"

	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

type recordTipRepo struct {
	TipRepository
	saved  *domain.TipInput
	amount float64
}

func (*recordTipRepo) TipPolicy(context.Context, requestscope.Scope, bool) (domain.TipPolicy, error) {
	return domain.TipPolicy{TipSettings: domain.TipSettings{Mode: "collect", Percentage: 10, AllowFixed: true, Channels: []string{"salon"}}, Decimals: 2}, nil
}
func (*recordTipRepo) TipOrderContext(context.Context, requestscope.Scope, string) (string, float64, error) {
	return "salon", 17, nil
}
func (*recordTipRepo) TipForOrder(context.Context, requestscope.Scope, string) (*domain.TipView, error) {
	return nil, nil
}
func (r *recordTipRepo) InsertTip(_ context.Context, _ requestscope.Scope, _, _, _ string, _ float64, in domain.TipInput, amount float64, _ bool) error {
	r.saved, r.amount = &in, amount
	return nil
}

func TestRecordTipAppliesConfiguredPercentage(t *testing.T) {
	repo := &recordTipRepo{}
	tips := NewTips(repo, nil)
	if err := tips.RecordPayment(context.Background(), requestscope.Scope{}, "order", "payment", "shift", &domain.TipInput{Kind: "percentage", Value: 50}, true); err != nil {
		t.Fatal(err)
	}
	if repo.saved == nil || repo.saved.Value != 10 || repo.amount != 1.7 {
		t.Fatalf("client percentage must be replaced by the location's: %+v %v", repo.saved, repo.amount)
	}
}

func TestRecordTipRejectsHandedToStaff(t *testing.T) {
	repo := &recordTipRepo{}
	err := NewTips(repo, nil).RecordPayment(context.Background(), requestscope.Scope{}, "order", "payment", "shift", &domain.TipInput{Kind: "fixed", Value: 3, DirectToStaff: true}, true)
	if err == nil || repo.saved != nil {
		t.Fatalf("a tip handed to staff must not be recorded: %v %+v", err, repo.saved)
	}
}
