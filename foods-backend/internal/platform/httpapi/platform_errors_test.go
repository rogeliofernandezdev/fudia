package httpapi

import (
	"errors"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestClassifyOnboardingUserError(t *testing.T) {
	tests := []struct {
		name       string
		step       string
		err        error
		wantStatus int
		wantCode   string
		wantOK     bool
	}{
		{
			name:       "duplicate tax id",
			step:       onboardingStepCreateOrganization,
			err:        &pgconn.PgError{Code: "23505", ConstraintName: "organizations_tax_id_key"},
			wantStatus: http.StatusConflict,
			wantCode:   "tax_id_already_registered",
			wantOK:     true,
		},
		{
			name:       "duplicate administrator email",
			step:       onboardingStepCreateAdministrator,
			err:        &pgconn.PgError{Code: "23505", ConstraintName: "users_organization_id_email_key"},
			wantStatus: http.StatusConflict,
			wantCode:   "admin_email_already_registered",
			wantOK:     true,
		},
		{
			name:       "plan no longer available",
			step:       onboardingStepCreateSubscription,
			err:        pgx.ErrNoRows,
			wantStatus: http.StatusConflict,
			wantCode:   "plan_unavailable",
			wantOK:     true,
		},
		{
			name:       "invalid plan uuid",
			step:       onboardingStepCreateSubscription,
			err:        &pgconn.PgError{Code: "22P02"},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_plan",
			wantOK:     true,
		},
		{
			name:   "unexpected database error is internal",
			step:   onboardingStepConfigureFiscal,
			err:    errors.New("unexpected"),
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, code, _, ok := classifyOnboardingUserError(tt.step, tt.err)
			if ok != tt.wantOK {
				t.Fatalf("expected ok=%v, got %v", tt.wantOK, ok)
			}
			if !tt.wantOK {
				return
			}
			if status != tt.wantStatus || code != tt.wantCode {
				t.Fatalf("expected %d/%s, got %d/%s", tt.wantStatus, tt.wantCode, status, code)
			}
		})
	}
}
