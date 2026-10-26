package application

import (
	"context"
	"errors"
	"testing"

	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
)

type presentationRuleRepository struct {
	StockRepository
	rule  domain.PresentationRule
	saved []float64
}

func (r *presentationRuleRepository) PresentationRule(context.Context, scope, string, string) (domain.PresentationRule, error) {
	return r.rule, nil
}
func (r *presentationRuleRepository) EnsurePresentation(_ context.Context, _ scope, _ string, _ string, factor float64) (string, error) {
	r.saved = append(r.saved, factor)
	return "presentation-id", nil
}

func TestEnsurePresentationRejectsFractionalContentForCountableUnits(t *testing.T) {
	repo := &presentationRuleRepository{rule: domain.PresentationRule{Allowed: true, Countable: true, UnitName: "Botella"}}
	stock := &Stock{repo: repo}
	_, err := stock.ensurePresentation(context.Background(), scope{}, "item-1", "package", 0.001)
	var invalid *inventoryCombinationError
	if !errors.As(err, &invalid) || invalid.Code() != "invalid_presentation_content" {
		t.Fatalf("expected invalid_presentation_content, got %v", err)
	}
	if len(repo.saved) != 0 {
		t.Fatalf("rejected presentation must not be saved: %v", repo.saved)
	}
	if _, err = stock.ensurePresentation(context.Background(), scope{}, "item-1", "package", 6); err != nil || len(repo.saved) != 1 {
		t.Fatalf("a pack of 6 bottles must be saved: err=%v saved=%v", err, repo.saved)
	}

	measured := &presentationRuleRepository{rule: domain.PresentationRule{Allowed: true, Countable: false, UnitName: "Kilogramo"}}
	if _, err = (&Stock{repo: measured}).ensurePresentation(context.Background(), scope{}, "item-2", "package", 0.5); err != nil {
		t.Fatalf("measured units keep accepting fractions: %v", err)
	}

	notConfigured := &presentationRuleRepository{rule: domain.PresentationRule{Allowed: false, Countable: true}}
	_, err = (&Stock{repo: notConfigured}).ensurePresentation(context.Background(), scope{}, "item-3", "box", 0.5)
	if !errors.As(err, &invalid) || invalid.Code() != "invalid_inventory_combination" {
		t.Fatalf("an unconfigured combination keeps its own error, got %v", err)
	}
}
