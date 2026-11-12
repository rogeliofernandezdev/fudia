package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDeliveryGeographyCatalogAndPlatformOwnership(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	token := fmt.Sprintf("geography-session-%d", time.Now().UnixNano())
	hash := sha256.Sum256([]byte(token))
	if _, err := pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,organization_id,location_id,expires_at) VALUES($1,$2,$3,$4,now()+interval '1 hour')`, hash[:], s.UserID, s.OrganizationID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE token_hash=$1`, hash[:]) })
	request := func(method, path, body string, authenticated bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if authenticated {
			req.AddCookie(&http.Cookie{Name: "foods_session", Value: token})
		}
		rec := httptest.NewRecorder()
		api.Routes().ServeHTTP(rec, req)
		return rec
	}
	if rec := request("GET", "/v1/admin/geographic-areas", "", false); rec.Code != 401 {
		t.Fatalf("anonymous geography: %d", rec.Code)
	}
	var result struct {
		Items []struct{ Code, ParentCode, Name, Label string }
	}
	rec := request("GET", "/v1/admin/geographic-areas", "", true)
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &result) != nil || len(result.Items) != 25 {
		t.Fatalf("departments: %d %s", rec.Code, rec.Body.String())
	}
	rec = request("GET", "/v1/admin/geographic-areas?parentCode=1501", "", true)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"code":"150101"`) || !strings.Contains(rec.Body.String(), `"parentCode":"1501"`) {
		t.Fatalf("districts: %d %s", rec.Code, rec.Body.String())
	}
	if rec := request("GET", "/v1/admin/geographic-areas?parentCode=150101", "", true); rec.Code != 400 {
		t.Fatalf("invalid parent: %d", rec.Code)
	}
	for _, input := range []struct{ Method, Path, Body string }{
		{"GET", "/v1/platform/delivery-partners", ""},
		{"POST", "/v1/platform/delivery-partners", `{"name":"Central ajena","whatsappPhone":"51987654321","coverageCodes":["150101"]}`},
		{"PATCH", "/v1/platform/delivery-partners/00000000-0000-0000-0000-000000000000", `{"active":false}`},
		{"PUT", "/v1/platform/delivery-partners/order", `{"partnerIds":[]}`},
		{"PUT", "/v1/platform/organizations/" + s.OrganizationID + "/delivery", `{"enabled":true}`},
	} {
		if rec := request(input.Method, input.Path, input.Body, true); rec.Code != 403 {
			t.Fatalf("company can administer Fudia coverage: %s %s: %d %s", input.Method, input.Path, rec.Code, rec.Body.String())
		}
	}
	// A geographic code can be supplied only from the INEI district catalog.
	var profileID, locationCode string
	if err := pool.QueryRow(ctx, `SELECT fiscal_profile_id::text,code FROM locations WHERE id=$1`, s.LocationID).Scan(&profileID, &locationCode); err != nil {
		t.Fatal(err)
	}
	update := func(code string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("PATCH", "/", strings.NewReader(fmt.Sprintf(`{"name":"Principal","code":%q,"address":"Av. Perú 100","timezone":"America/Lima","fiscalProfileId":%q,"districtCode":%q}`, locationCode, profileID, code)))
		req.SetPathValue("id", s.LocationID)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		out := httptest.NewRecorder()
		api.updateLocation(out, req)
		return out
	}
	if rec := update("150101"); rec.Code != 204 {
		t.Fatalf("location district: %d %s", rec.Code, rec.Body.String())
	}
	if rec := update("999999"); rec.Code < 400 {
		t.Fatalf("unknown district accepted: %s", rec.Body.String())
	}
	if rec := update("1501"); rec.Code != 400 {
		t.Fatalf("province accepted as district: %d", rec.Code)
	}
	var district string
	if err := pool.QueryRow(ctx, `SELECT district_code FROM locations WHERE id=$1`, s.LocationID).Scan(&district); err != nil || district != "150101" {
		t.Fatalf("invalid update changed district: %s %v", district, err)
	}
}
