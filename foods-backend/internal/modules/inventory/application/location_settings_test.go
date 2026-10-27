package application

import (
	"context"
	"errors"
	"testing"

	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

type settingsRepository struct {
	StockRepository
	countable bool
	saved     []domain.LocationSettings
	ruleError error
}

func (r *settingsRepository) ItemExists(context.Context, scope, string) (bool, error) {
	return true, nil
}
func (r *settingsRepository) PresentationRule(context.Context, scope, string, string) (domain.PresentationRule, error) {
	return domain.PresentationRule{Allowed: true, Countable: r.countable}, r.ruleError
}
func (r *settingsRepository) LocationSettings(_ context.Context, _ scope, _ string, in domain.LocationSettings) (int64, error) {
	r.saved = append(r.saved, in)
	return 1, nil
}

func TestLocationSettingsRejectsFractionsBeforeNormalizingCountableUnits(t *testing.T) {
	for _, in := range []domain.LocationSettings{
		{MinimumStock: 99.993, ReorderPoint: 100, OptimalStock: 100},
		{MinimumStock: 10, ReorderPoint: 0.5, OptimalStock: 20},
		{MinimumStock: 10, ReorderPoint: 20, OptimalStock: 0.5},
	} {
		repo := &settingsRepository{countable: true}
		stock := &Stock{repo: repo, audit: noAudit{}}
		err := stock.LocationSettings(context.Background(), scope{}, "item", in)
		var invalid *apperror.Error
		if !errors.As(err, &invalid) || invalid.Code != "fractional_inventory_settings" || len(repo.saved) != 0 {
			t.Fatalf("fraction was not rejected: err=%v saved=%v", err, repo.saved)
		}
	}
}

func TestLocationSettingsMeasuredUnitsAndWholeNormalization(t *testing.T) {
	for _, countable := range []bool{false, true} {
		repo := &settingsRepository{countable: countable}
		stock := &Stock{repo: repo, audit: noAudit{}}
		minimum := 0.5
		if countable {
			minimum = 5
		}
		err := stock.LocationSettings(context.Background(), scope{}, "item", domain.LocationSettings{MinimumStock: minimum})
		if err != nil || len(repo.saved) != 1 || repo.saved[0].ReorderPoint != minimum || repo.saved[0].OptimalStock != minimum {
			t.Fatalf("normalization countable=%v: err=%v saved=%v", countable, err, repo.saved)
		}
	}
	repo := &settingsRepository{ruleError: errors.New("catalog unavailable")}
	if err := (&Stock{repo: repo, audit: noAudit{}}).LocationSettings(context.Background(), scope{}, "item", domain.LocationSettings{}); err == nil || len(repo.saved) != 0 {
		t.Fatalf("must not save without unit metadata: %v", err)
	}
}
