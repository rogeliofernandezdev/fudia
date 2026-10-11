package httpapi

import (
	menudomain "github.com/foods-platform/foods-backend/internal/modules/menu/domain"
)

type comboOptionInput = menudomain.ComboOptionInput
type comboGroupInput = menudomain.ComboGroupInput
type comboInput = menudomain.ComboInput

var validateCombo = menudomain.ValidateCombo
var normalizedSurcharge = menudomain.NormalizedSurcharge
