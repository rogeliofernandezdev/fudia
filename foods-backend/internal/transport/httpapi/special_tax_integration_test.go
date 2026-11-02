package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFiscalProfileEnrollsInTheLegalSpecialTaxRate(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	api := New(pool)
	s := seedInventoryScope(t, pool)
	var profileID string
	if err := pool.QueryRow(ctx, `SELECT id FROM organization_fiscal_profiles WHERE organization_id=$1 AND is_default`, s.OrganizationID).Scan(&profileID); err != nil {
		t.Fatal(err)
	}
	request := func(handler http.HandlerFunc, method, body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "/v1/admin/fiscal-profiles", strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
		r.SetPathValue("id", profileID)
		w := httptest.NewRecorder()
		handler(w, r)
		if w.Code != want {
			t.Fatalf("%s: want %d, got %d %s", method, want, w.Code, w.Body.String())
		}
		return w
	}
	profile := func(country, regime string) string {
		return fmt.Sprintf(`{"country":%q,"currency":"PEN","currencyPosition":"before","taxName":"IGV","taxRate":"0.18","taxIncluded":true,"default":true,"specialTaxRegime":%s}`, country, regime)
	}
	h := api.handlers.Organizations

	// Only a regime of the platform catalog for the profile's country is accepted.
	request(h.UpdateFiscalProfile, "PATCH", profile("PE", `"no_existe"`), 400)
	request(h.UpdateFiscalProfile, "PATCH", profile("PE", `"pe_mype_restaurants"`), 204)

	var page struct {
		Items []struct {
			ID               string  `json:"id"`
			TaxRate          string  `json:"taxRate"`
			SpecialTaxRegime *string `json:"specialTaxRegime"`
			SpecialTax       *struct {
				State string  `json:"state"`
				Rate  *string `json:"rate"`
			} `json:"specialTax"`
		} `json:"items"`
		Options []struct {
			Code    string `json:"code"`
			Country string `json:"country"`
			Rates   []struct {
				Rate string `json:"rate"`
			} `json:"rates"`
		} `json:"specialTaxRegimeOptions"`
	}
	if err := json.Unmarshal(request(h.ListFiscalProfiles, "GET", "", 200).Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Options) == 0 || page.Options[0].Code != "pe_mype_restaurants" || page.Options[0].Country != "PE" || len(page.Options[0].Rates) != 3 {
		t.Fatalf("options: %+v", page.Options)
	}
	// Today's rate comes from the legal calendar (none once it ends).
	var expected string
	if err := pool.QueryRow(ctx, `SELECT COALESCE((SELECT rate::text FROM special_tax_rates WHERE regime_code='pe_mype_restaurants' AND daterange(valid_from,valid_to,'[)') @> (SELECT (now() AT TIME ZONE timezone)::date FROM organizations WHERE id=$1)),'')`, s.OrganizationID).Scan(&expected); err != nil {
		t.Fatal(err)
	}
	item := page.Items[0]
	today := ""
	if item.SpecialTax != nil && item.SpecialTax.Rate != nil {
		today = *item.SpecialTax.Rate
	}
	if item.ID != profileID || item.SpecialTaxRegime == nil || *item.SpecialTaxRegime != "pe_mype_restaurants" || item.SpecialTax == nil || today != expected {
		t.Fatalf("enrolled profile: %+v (today %q)", item, expected)
	}

	// Leaving the padrón keeps only the general rate.
	request(h.UpdateFiscalProfile, "PATCH", profile("PE", "null"), 204)
	var regime *string
	if err := pool.QueryRow(ctx, `SELECT special_tax_regime FROM organization_fiscal_profiles WHERE id=$1`, profileID).Scan(&regime); err != nil || regime != nil {
		t.Fatalf("left regime: %v %v", regime, err)
	}

	// Platform administrators keep the legal calendar: a new period applies to
	// every enrolled company and overlapping periods are rejected.
	p := api.handlers.SpecialTaxes
	call := func(handler http.HandlerFunc, method, body string, values map[string]string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "/v1/platform/special-tax-regimes", strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
		for key, value := range values {
			r.SetPathValue(key, value)
		}
		w := httptest.NewRecorder()
		handler(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: want %d, got %d %s", method, body, want, w.Code, w.Body.String())
		}
		return w
	}
	mype := map[string]string{"code": "pe_mype_restaurants"}
	call(p.CreateRate, "POST", `{"validFrom":"2027-06-01","validTo":"2028-01-01","rate":"0.12","legalBasis":"Ley de prueba"}`, mype, 409)
	call(p.CreateRate, "POST", `{"validFrom":"2028-01-01","validTo":"2029-01-01","rate":"0.12","legalBasis":""}`, mype, 400)
	call(p.CreateRate, "POST", `{"validFrom":"2090-01-01","validTo":"2091-01-01","rate":"0.12","legalBasis":"Ley"}`, map[string]string{"code": "no_existe"}, 404)
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(call(p.CreateRate, "POST", `{"validFrom":"2090-01-01","validTo":"2091-01-01","rate":"0.12","legalBasis":"Ley de prueba"}`, mype, 201).Body.Bytes(), &created)
	call(p.UpdateRate, "PATCH", `{"validFrom":"2090-01-01","validTo":"2092-01-01","rate":"0.125","legalBasis":"Ley de prueba"}`, map[string]string{"id": created.ID}, 204)
	var view struct {
		Items []struct {
			Code   string `json:"code"`
			Status struct {
				State string `json:"state"`
			} `json:"status"`
			Rates []struct {
				ID   string `json:"id"`
				Rate string `json:"rate"`
			} `json:"rates"`
		} `json:"items"`
		WarnDays int `json:"warnDays"`
	}
	_ = json.Unmarshal(call(p.List, "GET", "", nil, 200).Body.Bytes(), &view)
	last := view.Items[0].Rates[len(view.Items[0].Rates)-1]
	if view.WarnDays != 30 || view.Items[0].Code != "pe_mype_restaurants" || last.ID != created.ID || last.Rate != "0.125000" {
		t.Fatalf("platform view: %+v", view)
	}
	call(p.DeleteRate, "DELETE", "", map[string]string{"id": created.ID}, 204)
	call(p.DeleteRate, "DELETE", "", map[string]string{"id": created.ID}, 404)
}
