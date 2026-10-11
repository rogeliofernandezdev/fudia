// Package contracts exposes stable catalogs for organization onboarding and subscriptions.
package contracts

import "github.com/foods-platform/foods-backend/internal/modules/organizations/domain"

type CurrencyOption = domain.CurrencyOption
type FiscalProfileInput = domain.FiscalProfileInput

var CurrencyByCode = domain.CurrencyByCode
