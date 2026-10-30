package application

import (
	"context"
	"github.com/foods-platform/fudia-image-processor/internal/domain"
	"strings"
	"testing"
	"time"
)

type brandingTransformer map[string][]byte

func (f brandingTransformer) Transform(context.Context, []byte, domain.ImageRecord) (map[string][]byte, error) {
	return f, nil
}
func TestBrandingRequiresDeliveryVariantsBeforeReady(t *testing.T) {
	for _, purpose := range []string{"restaurant_logo", "virtual_menu"} {
		for _, complete := range []bool{false, true} {
			generation := int64(9)
			key := strings.Replace(domain.ImageOriginalKey(testImageOrg, testImageID, "image/png"), "/products/", "/branding/", 1)
			repo := &processingRepoFake{record: domain.ImageRecord{ID: testImageID, OrganizationID: testImageOrg, OriginalKey: key, Purpose: purpose, SizeBytes: 4, ContentType: "image/png", Status: "PROCESSING", Generation: &generation, ExpiresAt: time.Now().Add(time.Hour), Attempts: 1}, claimed: true}
			store := &processingStoreFake{}
			variants := brandingTransformer{"thumbnail": []byte("thumb"), "medium": []byte("webp")}
			name := "logo"
			if purpose == "virtual_menu" {
				name = "delivery"
			}
			if complete {
				variants[name] = []byte("output")
			}
			err := NewImageProcessor(repo, store, variants).Process(context.Background(), key, generation)
			if complete {
				if err != nil || repo.record.Status != "READY" || store.writes != 3 || store.cleanups != 1 {
					t.Fatalf("complete %s %v %#v", purpose, err, repo)
				}
			} else {
				if err == nil || repo.record.Status != "PROCESSING" || store.cleanups != 0 {
					t.Fatalf("premature ready %s", purpose)
				}
			}
		}
	}
}
