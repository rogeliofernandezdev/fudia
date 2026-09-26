package httpapi

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func seededOnboardingBody(t *testing.T, pool *pgxpool.Pool, planCode, taxID, email string) []byte {
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
		"country":"PE",
		"currency":"PEN",
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
	}`, taxID, email, planID))
}

func TestPlatformOnboardingWorksWithSeededCommercialPlans(t *testing.T) {
	pool := integrationPool(t)
	actor := seedInventoryScope(t, pool)
	api := New(pool)
	nonce := time.Now().UnixNano()

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
