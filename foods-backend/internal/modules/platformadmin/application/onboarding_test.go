package application

import (
	"context"
	"errors"
	"reflect"
	"testing"

	identitycontracts "github.com/foods-platform/foods-backend/internal/modules/identity/contracts"
	orgcontracts "github.com/foods-platform/foods-backend/internal/modules/organizations/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

type onboardingTxKey struct{}
type onboardingTx struct {
	ctx    context.Context
	events *[]string
}

func (t *onboardingTx) Context() context.Context { return t.ctx }
func (t *onboardingTx) Commit(context.Context) error {
	*t.events = append(*t.events, "commit")
	return nil
}
func (t *onboardingTx) Rollback(context.Context) error {
	*t.events = append(*t.events, "rollback")
	return nil
}

type onboardingWork struct{ events *[]string }

func (w onboardingWork) Begin(ctx context.Context, _ unitofwork.Options) (unitofwork.Transaction, error) {
	*w.events = append(*w.events, "begin")
	return &onboardingTx{context.WithValue(ctx, onboardingTxKey{}, true), w.events}, nil
}

type onboardingRepository struct {
	Repository
	t         *testing.T
	events    *[]string
	defaults  *domain.TenantOperationalDefaults
	failSeed  bool
	tenantErr error
	assigned  string
}

func (r *onboardingRepository) record(ctx context.Context, event string) {
	r.t.Helper()
	if ctx.Value(onboardingTxKey{}) != true {
		r.t.Fatalf("%s escaped onboarding transaction", event)
	}
	*r.events = append(*r.events, event)
}
func (r *onboardingRepository) CreateTenant(ctx context.Context, _ scope, _ domain.TenantOnboardingInput, _ orgcontracts.CurrencyOption, hash string) (domain.TenantRecords, error) {
	r.record(ctx, "tenant")
	if hash != "hash" {
		r.t.Fatalf("tenant must receive the hashed password, got %q", hash)
	}
	if r.tenantErr != nil {
		return domain.TenantRecords{}, r.tenantErr
	}
	return domain.TenantRecords{OrganizationID: "org-id", FiscalProfileID: "profile-id", LocationID: "location-id", AdministratorID: "user-id"}, nil
}
func (r *onboardingRepository) SeedRoles(ctx context.Context, _ string, seeds []identitycontracts.OrganizationRoleSeed) ([]string, error) {
	r.record(ctx, "roles")
	ids := make([]string, len(seeds))
	for i, seed := range seeds {
		ids[i] = "role-" + seed.SystemKey
	}
	return ids, nil
}
func (r *onboardingRepository) SeedPaymentMethods(ctx context.Context, _ string) error {
	r.record(ctx, "payment_methods")
	return nil
}
func (r *onboardingRepository) SeedOperationalDefaults(ctx context.Context, organizationID, locationID, userID string, defaults domain.TenantOperationalDefaults) error {
	r.record(ctx, "operational_defaults")
	if organizationID != "org-id" || locationID != "location-id" || userID != "user-id" {
		r.t.Fatalf("seed received wrong scope: %s %s %s", organizationID, locationID, userID)
	}
	r.defaults = &defaults
	if r.failSeed {
		return errors.New("storage failure")
	}
	return nil
}
func (r *onboardingRepository) AssignAdministrator(ctx context.Context, _ scope, userID string, roleID string, locationID string) (int64, error) {
	r.record(ctx, "assign")
	r.assigned = userID + "/" + roleID + "/" + locationID
	return 1, nil
}
func (r *onboardingRepository) AuditOnboarded(ctx context.Context, _ scope, _ string, _ string, _ string) (int64, error) {
	r.record(ctx, "audit")
	return 1, nil
}

type onboardingCatalog struct{}

func (onboardingCatalog) ValidateFiscal(context.Context, *orgcontracts.FiscalProfileInput, bool) (orgcontracts.CurrencyOption, bool, error) {
	return orgcontracts.CurrencyOption{Code: "PEN", Symbol: "S/", Decimals: 2}, true, nil
}

type onboardingSubscriptions struct {
	t      *testing.T
	events *[]string
}

func (s onboardingSubscriptions) Create(ctx context.Context, _, _, _, _ string, _ bool) error {
	if ctx.Value(onboardingTxKey{}) != true {
		s.t.Fatal("subscription escaped onboarding transaction")
	}
	*s.events = append(*s.events, "subscription")
	return nil
}

type onboardingPasswords struct{}

func (onboardingPasswords) Hash(string) (string, error) { return "hash", nil }

var errLocationCodeTaken = errors.New("locations_organization_id_code_key")

type onboardingFailures struct{}

func (onboardingFailures) Classify(err error) FailureCategory {
	if errors.Is(err, errLocationCodeTaken) {
		return FailureLocationCode
	}
	return FailureUnknown
}

type onboardingFlow struct{ events *[]string }

func (f onboardingFlow) Log(_ context.Context, event string, step string, _ error, _ ...any) {
	if f.events != nil {
		*f.events = append(*f.events, event+":"+step)
	}
}

func onboardingInput() domain.TenantOnboardingInput {
	return domain.TenantOnboardingInput{
		LegalName: "Restaurante SAC", TradeName: "Restaurante", TaxID: "20123456789",
		Timezone: "UTC", Country: "PE", Currency: "PEN", CurrencyPosition: "before",
		TaxName: "IGV", TaxRate: "0.18", LocationName: "Local principal", LocationCode: "PRINCIPAL",
		AdminName: "Propietaria", AdminEmail: "owner@example.test", AdminPassword: "OwnerPass123",
		PlanID: "plan-id", BillingCycle: "monthly", TermsAccepted: true,
	}
}

func TestOnboardingSeedsRestaurantDefaultsInsideTransaction(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "rollback"}[failure], func(t *testing.T) {
			events := []string{}
			repo := &onboardingRepository{t: t, events: &events, failSeed: failure}
			service := New(repo, onboardingWork{&events}, onboardingCatalog{}, onboardingSubscriptions{t, &events}, onboardingPasswords{}, onboardingFailures{}, onboardingFlow{})
			_, err := service.Onboard(context.Background(), scope{UserID: "platform-admin"}, onboardingInput())
			if (err != nil) != failure {
				t.Fatalf("unexpected error: %v", err)
			}
			if repo.defaults == nil || !reflect.DeepEqual(*repo.defaults, domain.NewTenantOperationalDefaults()) {
				t.Fatalf("seed must receive the tenant operational defaults, got %#v", repo.defaults)
			}
			seed := -1
			for i, event := range events {
				if event == "operational_defaults" {
					seed = i
				}
			}
			if seed < 0 || events[seed-1] != "payment_methods" {
				t.Fatalf("operational defaults must follow payment methods: %v", events)
			}
			if want := []string{"begin", "tenant", "roles", "payment_methods"}; !reflect.DeepEqual(events[:4], want) {
				t.Fatalf("tenant and roles must be one call each: %v", events)
			}
			last := events[len(events)-2:]
			if failure {
				var appErr *apperror.Error
				if !errors.As(err, &appErr) || appErr.Code != "onboarding_failed" {
					t.Fatalf("seed failure must surface as onboarding_failed, got %v", err)
				}
				if events[len(events)-1] != "rollback" || contains(events, "commit") || contains(events, "subscription") {
					t.Fatalf("failed seed must roll back without later steps: %v", events)
				}
				return
			}
			if !reflect.DeepEqual(last, []string{"commit", "rollback"}) || !contains(events, "subscription") {
				t.Fatalf("successful onboarding must commit after every step: %v", events)
			}
			if repo.assigned != "user-id/role-administrator/location-id" {
				t.Fatalf("administrator must receive the batched administrator role: %q", repo.assigned)
			}
		})
	}
}

func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func TestOnboardingReportsTheFailedStepInsideTheTenantBatch(t *testing.T) {
	events, flow := []string{}, []string{}
	repo := &onboardingRepository{t: t, events: &events, tenantErr: &TenantCreateError{Step: TenantStepLocation, Err: errLocationCodeTaken}}
	service := New(repo, onboardingWork{&events}, onboardingCatalog{}, onboardingSubscriptions{t, &events}, onboardingPasswords{}, onboardingFailures{}, onboardingFlow{&flow})
	_, err := service.Onboard(context.Background(), scope{UserID: "platform-admin"}, onboardingInput())
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Code != "location_code_already_registered" {
		t.Fatalf("batched location failure must keep its user message, got %v", err)
	}
	if !contains(flow, "application_flow_rejected:create_location") {
		t.Fatalf("failure must be attributed to create_location: %v", flow)
	}
	if contains(events, "roles") || contains(events, "commit") || events[len(events)-1] != "rollback" {
		t.Fatalf("failed tenant batch must stop and roll back: %v", events)
	}
}
