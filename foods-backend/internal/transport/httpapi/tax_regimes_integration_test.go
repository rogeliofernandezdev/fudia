package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type taxRegimesBody struct {
	Items []struct {
		ID        string  `json:"id"`
		ValidFrom string  `json:"validFrom"`
		ValidTo   *string `json:"validTo"`
		TotalRate string  `json:"totalRate"`
	} `json:"items"`
	General struct {
		Code string `json:"code"`
		Rate string `json:"rate"`
	} `json:"general"`
	Effective struct {
		Date       string `json:"date"`
		Source     string `json:"source"`
		TotalRate  string `json:"totalRate"`
		Components []struct {
			Code string `json:"code"`
			Rate string `json:"rate"`
		} `json:"components"`
	} `json:"effective"`
}

func TestTaxRegimesDefineDatedRatesPerFiscalProfile(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	api := New(pool)
	s := seedInventoryScope(t, pool)
	var profileID string
	if err := pool.QueryRow(ctx, `SELECT id FROM organization_fiscal_profiles WHERE organization_id=$1 AND is_default`, s.OrganizationID).Scan(&profileID); err != nil {
		t.Fatal(err)
	}

	request := func(actor scope, handler http.HandlerFunc, method, query, body, profile, regime string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "/v1/admin/fiscal-profiles/x/tax-regimes"+query, strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, actor))
		r.SetPathValue("id", profile)
		r.SetPathValue("regimeId", regime)
		w := httptest.NewRecorder()
		handler(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: want %d, got %d %s", method, query, want, w.Code, w.Body.String())
		}
		return w
	}
	list := func(date string) taxRegimesBody {
		t.Helper()
		var out taxRegimesBody
		w := request(s, api.handlers.TaxRegimes.List, "GET", "?date="+date, "", profileID, "", 200)
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	errorCode := func(w *httptest.ResponseRecorder) string {
		var body struct {
			Code string `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		return body.Code
	}
	h := api.handlers.TaxRegimes

	// Peru's MYPE restaurant regime: IGV 8% + IPM 2.5% in 2026, 12% + 3% in 2027.
	mype2026 := `{"name":"Régimen MYPE restaurantes 2026","validFrom":"2026-01-01","validTo":"2027-01-01","components":[{"code":"igv","name":"IGV","rate":"0.08"},{"code":"ipm","name":"IPM","rate":"0.025"}]}`
	created := request(s, h.Create, "POST", "", mype2026, profileID, "", 201)
	var regime2026 struct {
		ID        string `json:"id"`
		TotalRate string `json:"totalRate"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &regime2026); err != nil || regime2026.ID == "" || regime2026.TotalRate != "0.105000" {
		t.Fatalf("created regime: %s %v", created.Body.String(), err)
	}
	overlap := request(s, h.Create, "POST", "", `{"name":"Superpuesto","validFrom":"2026-06-01","components":[{"code":"igv","name":"IGV","rate":"0.18"}]}`, profileID, "", 409)
	if errorCode(overlap) != "tax_regime_overlap" {
		t.Fatalf("overlap code: %s", overlap.Body.String())
	}
	mype2027 := `{"name":"Régimen MYPE restaurantes 2027","validFrom":"2027-01-01","validTo":"2028-01-01","components":[{"code":"igv","name":"IGV","rate":"0.12"},{"code":"ipm","name":"IPM","rate":"0.02"}]}`
	var regime2027 struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(request(s, h.Create, "POST", "", mype2027, profileID, "", 201).Body.Bytes(), &regime2027); err != nil {
		t.Fatal(err)
	}
	request(s, h.Create, "POST", "", `{"name":"Inválido","validFrom":"2030-01-01","components":[{"code":"igv","name":"IGV","rate":"0.9"},{"code":"ipm","name":"IPM","rate":"0.2"}]}`, profileID, "", 400)

	got := list("2025-12-31")
	if len(got.Items) != 2 || got.Items[0].ValidFrom != "2027-01-01" || got.General.Code != "general" || got.General.Rate != "0.180000" {
		t.Fatalf("regimes must be listed newest first with the general rate: %+v", got)
	}
	if got.Effective.Source != "general" || got.Effective.TotalRate != "0.180000" {
		t.Fatalf("before any regime the general rate applies: %+v", got.Effective)
	}
	if got = list("2026-07-15"); got.Effective.Source != "regime" || got.Effective.TotalRate != "0.105000" || len(got.Effective.Components) != 2 || got.Effective.Components[1].Code != "ipm" {
		t.Fatalf("2026 regime: %+v", got.Effective)
	}
	if got = list("2028-01-01"); got.Effective.Source != "general" {
		t.Fatalf("validTo is exclusive: %+v", got.Effective)
	}
	request(s, h.List, "GET", "", "", profileID, "", 200)
	request(s, h.List, "GET", "?date=2026-02-30", "", profileID, "", 400)

	// Corrections replace the components; dates still may not overlap.
	request(s, h.Update, "PUT", "", strings.Replace(mype2027, `"0.02"`, `"0.03"`, 1), profileID, regime2027.ID, 200)
	if got = list("2027-03-01"); got.Effective.TotalRate != "0.150000" {
		t.Fatalf("updated 2027 regime: %+v", got.Effective)
	}
	request(s, h.Update, "PUT", "", strings.Replace(mype2027, `"validFrom":"2027-01-01"`, `"validFrom":"2026-12-01"`, 1), profileID, regime2027.ID, 409)

	// Another tenant cannot read or change these regimes.
	other := seedInventoryScope(t, pool)
	request(other, h.List, "GET", "", "", profileID, "", 404)
	request(other, h.Update, "PUT", "", mype2027, profileID, regime2027.ID, 404)
	request(other, h.Delete, "DELETE", "", "", profileID, regime2027.ID, 404)
	request(s, h.Delete, "DELETE", "", "", profileID, "not-a-uuid", 404)

	// An inactive profile keeps its history but cannot change it.
	if _, err := pool.Exec(ctx, `UPDATE organization_fiscal_profiles SET active=false WHERE id=$1`, profileID); err != nil {
		t.Fatal(err)
	}
	inactive := request(s, h.Delete, "DELETE", "", "", profileID, regime2027.ID, 409)
	if errorCode(inactive) != "profile_inactive" {
		t.Fatalf("inactive code: %s", inactive.Body.String())
	}
	if _, err := pool.Exec(ctx, `UPDATE organization_fiscal_profiles SET active=true WHERE id=$1`, profileID); err != nil {
		t.Fatal(err)
	}

	request(s, h.Delete, "DELETE", "", "", profileID, regime2027.ID, 204)
	request(s, h.Delete, "DELETE", "", "", profileID, regime2027.ID, 404)
	if got = list("2027-03-01"); len(got.Items) != 1 || got.Effective.Source != "general" {
		t.Fatalf("after delete: %+v", got)
	}
	var audited int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE organization_id=$1 AND entity_type='tax_regime'`, s.OrganizationID).Scan(&audited); err != nil || audited != 4 {
		t.Fatalf("create x2, update and delete must be audited: %d %v", audited, err)
	}
}
