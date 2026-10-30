package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	auditpg "github.com/foods-platform/foods-backend/internal/modules/audit/infrastructure/postgres"
	menuapp "github.com/foods-platform/foods-backend/internal/modules/menu/application"
	"github.com/foods-platform/foods-backend/internal/modules/menu/contracts"
	menupg "github.com/foods-platform/foods-backend/internal/modules/menu/infrastructure/postgres"
	orgapp "github.com/foods-platform/foods-backend/internal/modules/organizations/application"
	orgpg "github.com/foods-platform/foods-backend/internal/modules/organizations/infrastructure/postgres"
	orghttp "github.com/foods-platform/foods-backend/internal/modules/organizations/transport/http"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRestaurantProfileTenantPermissionsOrderingAndAsyncImages(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	other := seedInventoryScope(t, pool)
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM organization_menu_images WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(ctx, `UPDATE organizations SET logo_image_id=NULL WHERE id=$1`, s.OrganizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM product_images WHERE organization_id=$1`, s.OrganizationID)
	})
	media := menuapp.NewCatalog(menupg.NewCatalog(pool), database.NewUnitOfWork(pool), auditpg.New(pool), nil).WithProductImages(menuapp.NewProductImages(menupg.NewImages(pool), &storageImageFixture{generation: 99, size: 100})).RestaurantMedia()
	profiles := orgapp.NewRestaurantProfiles(orgpg.NewRestaurantProfiles(pool), media)
	api := New(pool)
	api.handlers.RestaurantProfile = orghttp.New(nil).WithProfiles(profiles)
	var role string
	if err := pool.QueryRow(ctx, `INSERT INTO roles(organization_id,name,permissions) VALUES($1,'Restaurant reader',ARRAY['organizations.read']) RETURNING id`, s.OrganizationID).Scan(&role); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, s.UserID, role, s.LocationID); err != nil {
		t.Fatal(err)
	}
	token := "restaurant-profile-" + s.UserID
	hash := sha256.Sum256([]byte(token))
	if _, err := pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,organization_id,location_id,expires_at) VALUES($1,$2,$3,$4,now()+interval '1 hour')`, hash[:], s.UserID, s.OrganizationID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, hash[:]) })
	routes := api.Routes()
	call := func(method, path, body string, auth bool) *httptest.ResponseRecorder {
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		if auth {
			request.AddCookie(&http.Cookie{Name: "foods_session", Value: token})
		}
		response := httptest.NewRecorder()
		routes.ServeHTTP(response, request)
		return response
	}
	const path = "/v1/admin/organization/profile"
	if got := call("GET", path, "", false).Code; got != 401 {
		t.Fatalf("anonymous %d", got)
	}
	if got := call("PATCH", path, `{"foundedOn":"2020-01-01"}`, true).Code; got != 403 {
		t.Fatalf("reader write %d", got)
	}
	if _, err := pool.Exec(ctx, `UPDATE roles SET permissions=ARRAY['organizations.read','organizations.manage'] WHERE id=$1`, role); err != nil {
		t.Fatal(err)
	}
	upload := func(purpose string) string {
		result, err := media.Authorize(ctx, s, contracts.MediaMetadata{Purpose: purpose, ContentType: "image/jpeg", SizeBytes: 100, Width: 100, Height: 50})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result.ObjectKey, "/branding/") {
			t.Fatal("wrong namespace")
		}
		if _, err = media.Complete(ctx, s, result.ImageID); err != nil {
			t.Fatal(err)
		}
		return result.ImageID
	}
	logo, page1, page2 := upload("restaurant_logo"), upload("virtual_menu"), upload("virtual_menu")
	save := func(logoID string, pages []string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(orgapp.RestaurantProfileInput{FoundedOn: "2020-04-12", LogoImageID: &logoID, MenuImageIDs: pages})
		return call("PATCH", path, string(body), true)
	}
	if response := save(logo, []string{page2, page1}); response.Code != 204 {
		t.Fatal(response.Code, response.Body.String())
	}
	result, err := profiles.Read(ctx, s)
	if err != nil || result.FoundedOn != "2020-04-12" || result.Logo.Status != "PROCESSING" || result.MenuPages[0].ID != page2 {
		t.Fatalf("async/profile %#v %v", result, err)
	}
	if brand, err := profiles.Branding(ctx, s); err != nil || len(brand.VirtualMenuURLs) != 0 {
		t.Fatal("unprocessed assets sent to WhatsApp", err)
	}
	if _, err = media.Read(ctx, other, page1); err == nil {
		t.Fatal("cross tenant asset read")
	}
	if err = profiles.Save(ctx, other, orgapp.RestaurantProfileInput{LogoImageID: &logo}); err == nil {
		t.Fatal("cross tenant attachment")
	}
	if response := save(page1, []string{page2}); response.Code != 409 {
		t.Fatal("menu accepted as logo", response.Code)
	}
	if response := save(logo, []string{page1, page1}); response.Code != 400 {
		t.Fatal("duplicate pages accepted", response.Code)
	}
	if response := call("GET", "/v1/platform/organizations/"+other.OrganizationID+"/profile", "", true); response.Code != 403 {
		t.Fatal("platform route bypass")
	}
	if _, err = pool.Exec(ctx, `UPDATE product_images SET status='READY' WHERE organization_id=$1`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	brand, err := profiles.Branding(ctx, s)
	if err != nil || brand.LogoURL == nil || !strings.HasSuffix(*brand.LogoURL, "/logo.png") || len(brand.VirtualMenuURLs) != 2 || !strings.Contains(brand.VirtualMenuURLs[0], page2+"/delivery.jpg") {
		t.Fatalf("branding %#v %v", brand, err)
	}
	if response := call("GET", "/v1/admin/restaurant-branding", "", true); response.Code != 200 {
		t.Fatal("branding endpoint", response.Code)
	}
}
