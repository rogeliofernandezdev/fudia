package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	auditpg "github.com/foods-platform/foods-backend/internal/modules/audit/infrastructure/postgres"
	menuapp "github.com/foods-platform/foods-backend/internal/modules/menu/application"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	menupg "github.com/foods-platform/foods-backend/internal/modules/menu/infrastructure/postgres"
	menuhttp "github.com/foods-platform/foods-backend/internal/modules/menu/transport/http"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type storageImageFixture struct {
	generation int64
	size       int64
}

func (s *storageImageFixture) Authorize(_ context.Context, r domain.ImageRecord) (domain.ImageUploadAuthorization, error) {
	return domain.ImageUploadAuthorization{ImageID: r.ID, ObjectKey: r.OriginalKey, Method: "POST", ExpiresAt: r.ExpiresAt}, nil
}
func (s *storageImageFixture) Inspect(context.Context, string) (menuapp.ImageObject, error) {
	return menuapp.ImageObject{Generation: s.generation, Size: s.size, ContentType: "image/jpeg", Created: time.Now()}, nil
}
func (s *storageImageFixture) ReadURL(_ context.Context, key string, _ int64) (string, time.Time, error) {
	return "https://storage.example/" + key, time.Now().Add(time.Hour), nil
}
func TestDirectImageTenantPermissionsAsyncRaceAndSupersededEvents(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	other := seedInventoryScope(t, pool)
	category := seedProductCategory(t, pool, s.OrganizationID)
	ctx := context.Background()
	repository := menupg.NewImages(pool)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `UPDATE products SET image_id=NULL,image_key=NULL,image_status=NULL WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM product_images WHERE organization_id=$1`, s.OrganizationID)
	})
	storage := &storageImageFixture{generation: 99, size: 100}
	service := menuapp.NewCatalog(menupg.NewCatalog(pool), database.NewUnitOfWork(pool), auditpg.New(pool), nil).WithProductImages(menuapp.NewProductImages(repository, storage))
	api := New(pool)
	api.handlers.MenuCatalog = menuhttp.NewCatalog(service)
	var role string
	if err := pool.QueryRow(ctx, `INSERT INTO roles(organization_id,name,permissions) VALUES($1,'Image manager',ARRAY['menu.read']) RETURNING id`, s.OrganizationID).Scan(&role); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, s.UserID, role, s.LocationID); err != nil {
		t.Fatal(err)
	}
	token := "image-session-" + s.UserID
	hash := sha256.Sum256([]byte(token))
	if _, err := pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,organization_id,location_id,expires_at) VALUES($1,$2,$3,$4,now()+interval '1 hour')`, hash[:], s.UserID, s.OrganizationID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, hash[:]) })
	routes := api.Routes()
	call := func(auth bool, path, body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest("POST", path, strings.NewReader(body))
		if auth {
			request.AddCookie(&http.Cookie{Name: "foods_session", Value: token})
		}
		response := httptest.NewRecorder()
		routes.ServeHTTP(response, request)
		return response
	}
	metadata := `{"contentType":"image/jpeg","sizeBytes":100,"width":100,"height":50}`
	if response := call(false, "/v1/admin/product-images/uploads", metadata); response.Code != 401 {
		t.Fatal("anonymous upload")
	}
	if response := call(true, "/v1/admin/product-images/uploads", metadata); response.Code != 403 {
		t.Fatalf("reader upload %d", response.Code)
	}
	if _, err := pool.Exec(ctx, `UPDATE roles SET permissions=ARRAY['menu.read','menu.manage'] WHERE id=$1`, role); err != nil {
		t.Fatal(err)
	}
	response := call(true, "/v1/admin/product-images/uploads", metadata)
	if response.Code != 201 {
		t.Fatalf("authorize %d %s", response.Code, response.Body.String())
	}
	var grant domain.ImageUploadAuthorization
	_ = json.Unmarshal(response.Body.Bytes(), &grant)
	if grant.ImageID == "" || !strings.Contains(grant.ObjectKey, s.OrganizationID) {
		t.Fatal("UUID/object key missing")
	}
	if _, err := service.CompleteImage(ctx, other, grant.ImageID); err == nil {
		t.Fatal("cross-tenant complete")
	}
	created, err := service.CreateProduct(ctx, s, domain.ProductInput{Name: "Unconfirmed", Price: "10", SKU: "unconfirmed", CategoryID: &category, ImageID: &grant.ImageID})
	if err == nil || created.ID != "" {
		t.Fatal("attached unconfirmed upload")
	}
	if _, err = service.CompleteImage(ctx, s, grant.ImageID); err != nil {
		t.Fatal(err)
	}
	// Worker may finish BEFORE the product is saved. Attachment must preserve READY, never downgrade.
	if _, err = pool.Exec(ctx, `UPDATE product_images SET status='READY',processed_at=now() WHERE id=$1`, grant.ImageID); err != nil {
		t.Fatal(err)
	}
	product, err := service.CreateProduct(ctx, s, domain.ProductInput{Name: "Causa", Price: "10", SKU: "CAUSA", CategoryID: &category, ImageID: &grant.ImageID})
	if err != nil || product.ImageStatus == nil || *product.ImageStatus != "READY" {
		t.Fatalf("save race %#v %v", product, err)
	}
	var key, status string
	var url *string
	if err = pool.QueryRow(ctx, `SELECT image_key,image_status,image_url FROM products WHERE id=$1`, product.ID).Scan(&key, &status, &url); err != nil || status != "READY" || !strings.HasSuffix(key, "medium.webp") || url != nil {
		t.Fatal("persisted temporary URL or wrong image reference", err)
	}
	if _, err = service.GetImage(ctx, other, grant.ImageID); err == nil {
		t.Fatal("cross tenant read")
	}
	// Replacing a picture then receiving a late event for the previous picture must not touch the new reference.
	replacement, err := service.AuthorizeImage(ctx, s, domain.ImageUploadInput{ProductID: &product.ID, ContentType: "image/jpeg", SizeBytes: 100, Width: 100, Height: 50})
	if err != nil {
		t.Fatal(err)
	}
	storage.generation = 100
	if _, err = service.CompleteImage(ctx, s, replacement.ImageID); err != nil {
		t.Fatal(err)
	}
	if _, err = service.AttachImage(ctx, s, product.ID, replacement.ImageID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE products SET image_status='READY' WHERE image_id=$1`, grant.ImageID); err != nil {
		t.Fatal(err)
	}
	var current string
	if err = pool.QueryRow(ctx, `SELECT image_id::text FROM products WHERE id=$1`, product.ID).Scan(&current); err != nil || current != replacement.ImageID {
		t.Fatal("late event replaced new image")
	}
	if response = call(true, "/v1/admin/products/"+product.ID+"/image", strings.Repeat("x", 100)); response.Code != 400 || !strings.Contains(response.Body.String(), "direct_upload_required") {
		t.Fatal("legacy API accepted bytes in GCS mode")
	}
	// Object references cannot be reused for another restaurant or another product.
	if err = repository.Attach(ctx, other, product.ID, replacement.ImageID); err == nil {
		t.Fatal("foreign attachment")
	}
	if _, err = service.CreateProduct(ctx, s, domain.ProductInput{Name: "Signed", Price: "10", ImageURL: func() *string { v := "https://storage.example/?X-Goog-Signature=x"; return &v }()}); err == nil {
		t.Fatal("persisted a signed URL")
	}
	_ = fmt.Sprintf("%s", current)
}
