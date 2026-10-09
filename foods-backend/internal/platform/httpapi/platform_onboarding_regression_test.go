package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func seedOperationalWhatsAppChannel(t *testing.T, pool *pgxpool.Pool, country, suffix string) {
	t.Helper()
	phone := "+999" + suffix
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO platform_whatsapp_channels(country_code,phone_number,display_name,active)
		VALUES($1,$2,'Integration Test',true)
		ON CONFLICT(phone_number) DO UPDATE SET
			country_code=EXCLUDED.country_code,
			active=true,
			updated_at=now()
	`, country, phone); err != nil {
		t.Fatalf("seed operational WhatsApp channel for %s: %v", country, err)
	}
}

func seedInactiveWhatsAppChannel(t *testing.T, pool *pgxpool.Pool, country, suffix string) {
	t.Helper()
	phone := "+998" + suffix
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO platform_whatsapp_channels(country_code,phone_number,display_name,active)
		VALUES($1,$2,'Inactive Integration Test',false)
		ON CONFLICT(phone_number) DO UPDATE SET
			country_code=EXCLUDED.country_code,
			active=false,
			updated_at=now()
	`, country, phone); err != nil {
		t.Fatalf("seed inactive WhatsApp channel for %s: %v", country, err)
	}
}

func seededOnboardingBodyForCatalog(t *testing.T, pool *pgxpool.Pool, planCode, taxID, email, country, currency string) []byte {
	t.Helper()
	var planID string
	if err := pool.QueryRow(context.Background(), `
		SELECT id
		FROM subscription_plans
		WHERE code=$1 AND active
	`, planCode).Scan(&planID); err != nil {
		t.Fatalf("load seeded plan %s: %v", planCode, err)
	}
	return []byte(fmt.Sprintf(`{
		"legalName":"Regression Restaurant SAC",
		"tradeName":"Regression Restaurant",
		"taxId":%q,
		"timezone":"America/Lima",
		"country":%q,
		"currency":%q,
		"currencyPosition":"before",
		"taxName":"IGV",
		"taxRate":"0.18",
		"taxIncluded":false,
		"locationName":"Sede principal",
		"locationCode":"PRINCIPAL",
		"address":"Dirección de prueba",
		"adminName":"Administrador Regression",
		"adminEmail":%q,
		"adminPassword":"OwnerPass123",
		"planId":%q,
		"billingCycle":"monthly",
		"termsAccepted":true
	}`, taxID, country, currency, email, planID))
}

func seededOnboardingBody(t *testing.T, pool *pgxpool.Pool, planCode, taxID, email string) []byte {
	t.Helper()
	return seededOnboardingBodyForCatalog(t, pool, planCode, taxID, email, "PE", "PEN")
}

func TestPlatformOnboardingWorksWithSeededCommercialPlans(t *testing.T) {
	pool := integrationPool(t)
	actor := seedInventoryScope(t, pool)
	api := New(pool)
	nonce := time.Now().UnixNano()
	seedOperationalWhatsAppChannel(t, pool, "PE", fmt.Sprint(nonce))

	for i, planCode := range []string{"emprende", "impulso", "escala"} {
		t.Run(planCode, func(t *testing.T) {
			taxID := fmt.Sprintf("%011d", (nonce+int64(i+1))%100000000000)
			email := fmt.Sprintf("seeded-%s-%d@example.test", planCode, nonce+int64(i))
			body := seededOnboardingBody(t, pool, planCode, taxID, email)
			req := httptest.NewRequest("POST", "/v1/platform/organizations", bytes.NewReader(body))
			req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, actor))
			rec := httptest.NewRecorder()
			api.onboardTenant(rec, req)
			if rec.Code != 201 {
				t.Fatalf("seeded plan %s onboarding: %d %s", planCode, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestPlatformOnboardingReportsDuplicateTaxID(t *testing.T) {
	pool := integrationPool(t)
	actor := seedInventoryScope(t, pool)
	api := New(pool)
	nonce := time.Now().UnixNano()
	seedOperationalWhatsAppChannel(t, pool, "PE", fmt.Sprint(nonce))
	taxID := fmt.Sprintf("%011d", nonce%100000000000)

	create := func(email string) *httptest.ResponseRecorder {
		body := seededOnboardingBody(t, pool, "emprende", taxID, email)
		req := httptest.NewRequest("POST", "/v1/platform/organizations", bytes.NewReader(body))
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, actor))
		rec := httptest.NewRecorder()
		api.onboardTenant(rec, req)
		return rec
	}

	first := create(fmt.Sprintf("duplicate-first-%d@example.test", nonce))
	if first.Code != 201 {
		t.Fatalf("first onboarding: %d %s", first.Code, first.Body.String())
	}

	second := create(fmt.Sprintf("duplicate-second-%d@example.test", nonce))
	if second.Code != 409 || !strings.Contains(second.Body.String(), "tax_id_already_registered") {
		t.Fatalf("duplicate tax id must be descriptive: %d %s", second.Code, second.Body.String())
	}
}

func TestPlatformOnboardingAcceptsDatabaseOnlyCountryAndCurrency(t *testing.T) {
	pool := integrationPool(t)
	actor := seedInventoryScope(t, pool)
	api := New(pool)
	nonce := time.Now().UnixNano()

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO platform_currencies(code,name,symbol,decimals,active)
		VALUES('ZZZ','Moneda dinámica','Z$',2,true)
		ON CONFLICT(code) DO UPDATE SET name=EXCLUDED.name,symbol=EXCLUDED.symbol,decimals=EXCLUDED.decimals,active=true,updated_at=now()
	`); err != nil {
		t.Fatalf("seed dynamic currency: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO platform_countries(code,name,default_currency,active)
		VALUES('ZZ','País dinámico','ZZZ',true)
		ON CONFLICT(code) DO UPDATE SET name=EXCLUDED.name,default_currency=EXCLUDED.default_currency,active=true,updated_at=now()
	`); err != nil {
		t.Fatalf("seed dynamic country: %v", err)
	}
	seedOperationalWhatsAppChannel(t, pool, "ZZ", fmt.Sprint(nonce))

	taxID := fmt.Sprintf("%011d", (nonce+11)%100000000000)
	email := fmt.Sprintf("dynamic-catalog-%d@example.test", nonce)
	body := seededOnboardingBodyForCatalog(t, pool, "emprende", taxID, email, "ZZ", "ZZZ")
	req := httptest.NewRequest("POST", "/v1/platform/organizations", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, actor))
	rec := httptest.NewRecorder()
	api.onboardTenant(rec, req)
	if rec.Code != 201 {
		t.Fatalf("database-only catalog onboarding: %d %s", rec.Code, rec.Body.String())
	}
}

func TestPlatformOnboardingRejectsCountryWithoutOperationalWhatsApp(t *testing.T) {
	pool := integrationPool(t)
	actor := seedInventoryScope(t, pool)
	api := New(pool)
	nonce := time.Now().UnixNano()

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO platform_currencies(code,name,symbol,decimals,active)
		VALUES('ZYY','Moneda sin canal','Y$',2,true)
		ON CONFLICT(code) DO UPDATE SET name=EXCLUDED.name,symbol=EXCLUDED.symbol,decimals=EXCLUDED.decimals,active=true,updated_at=now()
	`); err != nil {
		t.Fatalf("seed incomplete currency: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO platform_countries(code,name,default_currency,active)
		VALUES('ZY','País sin canal operativo','ZYY',true)
		ON CONFLICT(code) DO UPDATE SET name=EXCLUDED.name,default_currency=EXCLUDED.default_currency,active=true,updated_at=now()
	`); err != nil {
		t.Fatalf("seed incomplete country: %v", err)
	}
	seedInactiveWhatsAppChannel(t, pool, "ZY", fmt.Sprint(nonce))

	taxID := fmt.Sprintf("%011d", (nonce+12)%100000000000)
	email := fmt.Sprintf("incomplete-channel-%d@example.test", nonce)
	body := seededOnboardingBodyForCatalog(t, pool, "emprende", taxID, email, "ZY", "ZYY")
	req := httptest.NewRequest("POST", "/v1/platform/organizations", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, actor))
	rec := httptest.NewRecorder()
	api.onboardTenant(rec, req)
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "invalid_fiscal_profile") {
		t.Fatalf("inactive WhatsApp channel must be rejected: %d %s", rec.Code, rec.Body.String())
	}
}

func TestPlatformOnboardingRejectsCurrencyDifferentFromCountryDefault(t *testing.T) {
	pool := integrationPool(t)
	actor := seedInventoryScope(t, pool)
	api := New(pool)
	nonce := time.Now().UnixNano()

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO platform_currencies(code,name,symbol,decimals,active)
		VALUES
			('ZZA','Moneda predeterminada','A$',2,true),
			('ZZB','Moneda no asociada','B$',2,true)
		ON CONFLICT(code) DO UPDATE SET name=EXCLUDED.name,symbol=EXCLUDED.symbol,decimals=EXCLUDED.decimals,active=true,updated_at=now()
	`); err != nil {
		t.Fatalf("seed currencies for association test: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO platform_countries(code,name,default_currency,active)
		VALUES('ZX','País con moneda asociada','ZZA',true)
		ON CONFLICT(code) DO UPDATE SET name=EXCLUDED.name,default_currency=EXCLUDED.default_currency,active=true,updated_at=now()
	`); err != nil {
		t.Fatalf("seed country for association test: %v", err)
	}
	seedOperationalWhatsAppChannel(t, pool, "ZX", fmt.Sprint(nonce))

	taxID := fmt.Sprintf("%011d", (nonce+13)%100000000000)
	email := fmt.Sprintf("wrong-currency-%d@example.test", nonce)
	body := seededOnboardingBodyForCatalog(t, pool, "emprende", taxID, email, "ZX", "ZZB")
	req := httptest.NewRequest("POST", "/v1/platform/organizations", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, actor))
	rec := httptest.NewRecorder()
	api.onboardTenant(rec, req)
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "invalid_fiscal_profile") {
		t.Fatalf("currency different from country default must be rejected: %d %s", rec.Code, rec.Body.String())
	}
}

func TestPlatformCountryCatalogReturnsTimezoneSuggestion(t *testing.T) {
	pool := integrationPool(t)
	api := New(pool)
	seedOperationalWhatsAppChannel(t, pool, "PE", fmt.Sprint(time.Now().UnixNano()))
	for _, scope := range []string{"", "?scope=all"} {
		rec := httptest.NewRecorder()
		api.getPlatformOnboardingCatalogs(rec, httptest.NewRequest("GET", "/v1/platform/onboarding/catalogs"+scope, nil))
		if rec.Code != 200 {
			t.Fatalf("catalog %s: %d %s", scope, rec.Code, rec.Body.String())
		}
		var body struct {
			CountryOptions []platformCountryOption `json:"countryOptions"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, country := range body.CountryOptions {
			if country.DefaultTimezone != "" {
				if _, err := time.LoadLocation(country.DefaultTimezone); err != nil {
					t.Errorf("%s: invalid suggestion %q: %v", country.Code, country.DefaultTimezone, err)
				}
			}
			if country.Code == "PE" {
				found = true
				if country.DefaultTimezone != "America/Lima" {
					t.Errorf("Peru suggestion: %q", country.DefaultTimezone)
				}
			}
		}
		if !found {
			t.Fatal("Peru missing from catalog")
		}
	}
}

func TestPlatformOnboardingPersistsChosenLocalTimezone(t *testing.T) {
	pool := integrationPool(t)
	actor := seedInventoryScope(t, pool)
	api := New(pool)
	nonce := time.Now().UnixNano()
	seedOperationalWhatsAppChannel(t, pool, "PE", fmt.Sprint(nonce))
	body := seededOnboardingBody(t, pool, "emprende", fmt.Sprintf("%011d", nonce%100000000000), fmt.Sprintf("timezone-%d@example.test", nonce))
	// A local may override the country's suggestion. The company initially shares it.
	body = bytes.ReplaceAll(body, []byte("America/Lima"), []byte("America/New_York"))
	req := httptest.NewRequest("POST", "/v1/platform/organizations", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, actor))
	rec := httptest.NewRecorder()
	api.onboardTenant(rec, req)
	if rec.Code != 201 {
		t.Fatalf("onboarding: %d %s", rec.Code, rec.Body.String())
	}
	var result struct {
		OrganizationID string `json:"organizationId"`
		LocationID     string `json:"locationId"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	var companyZone, localZone string
	if err := pool.QueryRow(context.Background(), `SELECT o.timezone,l.timezone FROM organizations o JOIN locations l ON l.organization_id=o.id WHERE o.id=$1 AND l.id=$2`, result.OrganizationID, result.LocationID).Scan(&companyZone, &localZone); err != nil {
		t.Fatal(err)
	}
	if companyZone != "America/New_York" || localZone != "America/New_York" {
		t.Fatalf("unexpected persisted zones: %s / %s", companyZone, localZone)
	}
}
