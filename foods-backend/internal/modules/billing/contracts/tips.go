package contracts

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

type TipSettings = domain.TipSettings
type InitialTipConfigurator interface {
	ConfigureInitialTips(context.Context, requestscope.Scope, TipSettings) error
}

// LocationTipConfigurator saves a location's own tip settings inside the
// caller's transaction; the scope's LocationID is the target location. Only
// the reserved company administrator may change them.
type LocationTipConfigurator interface {
	ConfigureLocationTips(context.Context, requestscope.Scope, TipSettings) error
}

var ValidTipSettings = domain.ValidTipSettings

func OnboardingTipOptions() map[string]any {
	defaults := TipSettings{Mode: "disabled", Percentage: 10, AllowFixed: true, Channels: []string{"salon"}}
	chile := defaults
	chile.Mode = "suggest"
	chile.ShowSuggestion = true
	return map[string]any{"defaults": defaults, "countryDefaults": map[string]TipSettings{"CL": chile}, "channels": domain.TipChannelOptions()}
}
