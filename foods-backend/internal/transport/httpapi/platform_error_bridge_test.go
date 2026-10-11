package httpapi

import (
	platformapp "github.com/foods-platform/foods-backend/internal/modules/platformadmin/application"
	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/infrastructure/diagnostics"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

func classifyOnboardingUserError(step string, err error) (int, string, string, bool) {
	out, ok := platformapp.ClassifyOnboardingUserError(step, diagnostics.Diagnostics{}.Classify(err), err)
	if !ok {
		return 0, "", "", false
	}
	status := 409
	if out.Kind == apperror.Invalid {
		status = 400
	}
	return status, out.Code, out.Message, true
}
