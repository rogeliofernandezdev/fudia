package contracts

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/domain"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

type TaxComponent = domain.TaxComponent
type EffectiveTax = domain.EffectiveTax

// TaxResolver returns the tax that applies to a fiscal profile on a business
// date (YYYY-MM-DD); an empty date means today in the session location's timezone.
// Consumers that issue documents must store a copy of the returned components.
type TaxResolver interface {
	EffectiveTax(ctx context.Context, s requestscope.Scope, profileID, date string) (EffectiveTax, error)
}
