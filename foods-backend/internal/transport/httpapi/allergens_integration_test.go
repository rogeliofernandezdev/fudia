package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func allergenRequest(s scope, method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	return req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
}

func TestAllergenCreationRejectsDuplicatesAndStaysInOrganization(t *testing.T) {
	pool := integrationPool(t)
	owner := seedInventoryScope(t, pool)
	other := seedInventoryScope(t, pool)
	api := New(pool)
	create := func(s scope, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		api.createAllergen(rec, allergenRequest(s, "POST", "/v1/admin/allergens", body))
		return rec
	}
	list := func(s scope, q string) []string {
		t.Helper()
		rec := httptest.NewRecorder()
		api.listAllergens(rec, allergenRequest(s, "GET", "/v1/admin/allergens?q="+url.QueryEscape(q), ""))
		var out struct{ Items []string }
		if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &out) != nil {
			t.Fatalf("list allergens: %d %s", rec.Code, rec.Body.String())
		}
		return out.Items
	}

	created := create(owner, `{"name":"  Ají   amarillo "}`)
	var allergen struct{ ID, Name string }
	if created.Code != http.StatusCreated || json.Unmarshal(created.Body.Bytes(), &allergen) != nil || allergen.ID == "" || allergen.Name != "Ají amarillo" {
		t.Fatalf("create allergen: %d %s", created.Code, created.Body.String())
	}
	for _, duplicate := range []string{`{"name":"aji AMARILLO"}`, `{"name":"Sesamo"}`, `{"name":"gluten"}`} {
		if rec := create(owner, duplicate); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "allergen_conflict") {
			t.Fatalf("duplicate %s must conflict: %d %s", duplicate, rec.Code, rec.Body.String())
		}
	}
	if rec := create(owner, `{"name":"a"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("short name must be invalid: %d %s", rec.Code, rec.Body.String())
	}
	if items := list(owner, "aji"); len(items) != 1 || items[0] != "Ají amarillo" {
		t.Fatalf("owner search without accents: %v", items)
	}
	if items := list(other, "aji"); len(items) != 0 {
		t.Fatalf("other organization must not see owner allergen: %v", items)
	}
	if rec := create(other, `{"name":"Ají amarillo"}`); rec.Code != http.StatusCreated {
		t.Fatalf("other organization can register the same name: %d %s", rec.Code, rec.Body.String())
	}
	if items := list(owner, "s%"); len(items) != 0 {
		t.Fatalf("LIKE wildcards must be literal: %v", items)
	}
}
