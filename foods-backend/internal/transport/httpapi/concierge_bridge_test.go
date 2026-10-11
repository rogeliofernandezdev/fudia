package httpapi

import (
	conciergedomain "github.com/foods-platform/foods-backend/internal/modules/concierge/domain"
	menudomain "github.com/foods-platform/foods-backend/internal/modules/menu/domain"
)

type conciergeMenuItem = menudomain.PublicMenuItem
type conciergeComboDetail = menudomain.PublicComboDetail
type conciergeHandoffView = conciergedomain.HandoffView
type conciergeSettingsView = conciergedomain.SettingsView
