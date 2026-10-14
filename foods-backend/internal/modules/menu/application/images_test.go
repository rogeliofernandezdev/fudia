package application

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"strings"
	"testing"
	"time"
)

const testImageID = "00000000-0000-0000-0000-000000000002"
const testImageOrg = "00000000-0000-0000-0000-000000000001"

type imageRepoFake struct {
	ImageRepository
	record                       domain.ImageRecord
	reserved, confirmed, batches int
	ownerMissing                 bool
}

func (r *imageRepoFake) Reserve(_ context.Context, s scope, in domain.ImageUploadInput, expiry time.Time) (domain.ImageRecord, error) {
	r.reserved++
	r.record = domain.ImageRecord{ID: testImageID, OrganizationID: s.OrganizationID, LocationID: s.LocationID, UploadedBy: s.UserID, OriginalKey: domain.ImageOriginalKey(s.OrganizationID, testImageID, in.ContentType), ContentType: in.ContentType, SizeBytes: in.SizeBytes, Width: in.Width, Height: in.Height, Status: "PROCESSING", ExpiresAt: expiry}
	return r.record, nil
}
func (r *imageRepoFake) Read(_ context.Context, s scope, _ string) (domain.ImageRecord, error) {
	if r.ownerMissing || s.OrganizationID != r.record.OrganizationID {
		return domain.ImageRecord{}, ErrNotFound
	}
	return r.record, nil
}
func (r *imageRepoFake) Confirm(_ context.Context, _ scope, _ string, g int64) (domain.ImageRecord, error) {
	r.confirmed++
	r.record.Generation = &g
	now := time.Now()
	r.record.ConfirmedAt = &now
	return r.record, nil
}
func (r *imageRepoFake) ForProducts(_ context.Context, _ scope, ids []string) (map[string]domain.ImageRecord, error) {
	r.batches++
	out := map[string]domain.ImageRecord{}
	for _, id := range ids {
		out[id] = r.record
	}
	return out, nil
}

type imageStorageFake struct {
	object            ImageObject
	inspected, signed int
	keys              []string
}

func (s *imageStorageFake) Authorize(_ context.Context, r domain.ImageRecord) (domain.ImageUploadAuthorization, error) {
	return domain.ImageUploadAuthorization{ImageID: r.ID, ObjectKey: r.OriginalKey, Method: "POST"}, nil
}
func (s *imageStorageFake) Inspect(context.Context, string) (ImageObject, error) {
	s.inspected++
	return s.object, nil
}
func (s *imageStorageFake) ReadURL(_ context.Context, key string, _ int64) (string, time.Time, error) {
	s.signed++
	s.keys = append(s.keys, key)
	return "https://storage.example/" + key, time.Now().Add(time.Hour), nil
}

type ownedProductFake struct {
	CatalogRepository
	exists bool
}

func (r ownedProductFake) ProductExists(context.Context, scope, string) (bool, error) {
	return r.exists, nil
}
func TestAuthorizeImageMetadataAndOwnershipBeforeReservation(t *testing.T) {
	repo := &imageRepoFake{}
	store := &imageStorageFake{}
	service := NewCatalog(ownedProductFake{exists: false}, nil, nil, nil).WithProductImages(NewProductImages(repo, store))
	s := scope{OrganizationID: testImageOrg, UserID: "user", LocationID: "location"}
	valid := domain.ImageUploadInput{ContentType: "image/png", SizeBytes: 100, Width: 100, Height: 100}
	for _, invalid := range []domain.ImageUploadInput{{ContentType: "image/svg+xml", SizeBytes: 10, Width: 1, Height: 1}, {ContentType: "image/png", SizeBytes: domain.MaxImageBytes + 1, Width: 1, Height: 1}, {ContentType: "image/png", SizeBytes: 10, Width: 4001, Height: 1}} {
		if _, err := service.AuthorizeImage(context.Background(), s, invalid); err == nil {
			t.Fatal("accepted invalid metadata")
		}
	}
	valid.ProductID = new(string)
	*valid.ProductID = testImageID
	if _, err := service.AuthorizeImage(context.Background(), s, valid); err == nil {
		t.Fatal("authorized another tenant's product")
	}
	if repo.reserved != 0 {
		t.Fatal("reserved before checking permission/metadata")
	}
	valid.ProductID = nil
	out, err := service.AuthorizeImage(context.Background(), s, valid)
	if err != nil || out.ImageID != testImageID || !strings.Contains(out.ObjectKey, testImageOrg) {
		t.Fatalf("reservation: %#v %v", out, err)
	}
}
func TestCompleteImageVerifiesStorageMetadataAndCannotCrossTenant(t *testing.T) {
	s := scope{OrganizationID: testImageOrg, UserID: "user", LocationID: "location"}
	repo := &imageRepoFake{}
	store := &imageStorageFake{}
	service := NewCatalog(nil, nil, nil, nil).WithProductImages(NewProductImages(repo, store))
	in := domain.ImageUploadInput{ContentType: "image/png", SizeBytes: 100, Width: 100, Height: 100}
	_, _ = service.AuthorizeImage(context.Background(), s, in)
	if _, err := service.CompleteImage(context.Background(), scope{OrganizationID: testImageID, UserID: s.UserID}, testImageID); err == nil {
		t.Fatal("cross tenant confirmation")
	}
	if store.inspected != 0 {
		t.Fatal("Storage accessed before authorization")
	}
	store.object = ImageObject{Generation: 3, Size: 101, ContentType: "image/png", Created: time.Now()}
	if _, err := service.CompleteImage(context.Background(), s, testImageID); err == nil || repo.confirmed != 0 {
		t.Fatal("metadata mismatch accepted")
	}
	store.object.Size = 100
	out, err := service.CompleteImage(context.Background(), s, testImageID)
	if err != nil || *out.ImageStatus != "PROCESSING" || out.ImageURL == nil {
		t.Fatalf("processing fallback: %#v %v", out, err)
	}
	_, err = service.CompleteImage(context.Background(), s, testImageID)
	if err != nil || repo.confirmed != 1 || store.inspected != 2 {
		t.Fatal("completion is not idempotent")
	}
}
func TestImageViewsChooseOriginalThenThumbnailAndMedium(t *testing.T) {
	g := int64(4)
	record := domain.ImageRecord{ID: testImageID, OrganizationID: testImageOrg, OriginalKey: domain.ImageOriginalKey(testImageOrg, testImageID, "image/jpeg"), Status: "PROCESSING", Generation: &g}
	store := &imageStorageFake{}
	images := NewProductImages(nil, store)
	for _, status := range []string{"PROCESSING", "FAILED", "READY"} {
		record.Status = status
		view, err := images.view(context.Background(), record, "thumbnail", true)
		if err != nil {
			t.Fatal(err)
		}
		if status == "READY" {
			if !strings.HasSuffix(*view.ImageKey, "thumbnail.webp") || len(view.ImageVariants) != 2 {
				t.Fatal("wrong ready variants")
			}
		} else if !strings.HasSuffix(*view.ImageKey, "original.jpg") {
			t.Fatal("missing original fallback")
		}
	}
}
