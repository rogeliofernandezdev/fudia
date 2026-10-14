package application

import (
	"context"
	"errors"
	"github.com/foods-platform/fudia-image-processor/internal/domain"
	"testing"
	"time"
)

const testImageID = "00000000-0000-0000-0000-000000000002"
const testImageOrg = "00000000-0000-0000-0000-000000000001"

type processingRepoFake struct {
	record   domain.ImageRecord
	claimed  bool
	finishes []string
}

func (r *processingRepoFake) Claim(context.Context, string, string, int64) (domain.ImageRecord, bool, error) {
	return r.record, r.claimed, nil
}
func (r *processingRepoFake) Finish(_ context.Context, _ domain.ImageRecord, _ int64, status, _ string) error {
	r.finishes = append(r.finishes, status)
	r.record.Status = status
	r.claimed = status == "PROCESSING"
	return nil
}

type processingStoreFake struct {
	writeFail        bool
	downloadFail     bool
	writes, cleanups int
}

func (s *processingStoreFake) Download(context.Context, string, int64) ([]byte, ImageObject, error) {
	if s.downloadFail {
		return nil, ImageObject{}, errors.New("network")
	}
	return []byte("data"), ImageObject{Size: 4, ContentType: "image/png", Created: time.Now()}, nil
}
func (s *processingStoreFake) WriteVariant(context.Context, string, []byte, int64) error {
	s.writes++
	if s.writeFail {
		return errors.New("network")
	}
	return nil
}
func (s *processingStoreFake) ScheduleOriginalDeletion(context.Context, string, int64) error {
	s.cleanups++
	return nil
}

type transformerFake struct {
	err   error
	calls int
}

func (f *transformerFake) Transform(context.Context, []byte, domain.ImageRecord) (map[string][]byte, error) {
	f.calls++
	return map[string][]byte{"thumbnail": []byte("thumb"), "medium": []byte("medium")}, f.err
}
func TestProcessorIdempotenceFailuresAndCleanupOnlyAfterReady(t *testing.T) {
	for _, scenario := range []string{"ready", "invalid", "retry", "exhausted"} {
		t.Run(scenario, func(t *testing.T) {
			g := int64(8)
			key := domain.ImageOriginalKey(testImageOrg, testImageID, "image/png")
			repo := &processingRepoFake{record: domain.ImageRecord{ID: testImageID, OrganizationID: testImageOrg, OriginalKey: key, SizeBytes: 4, ContentType: "image/png", Status: "PROCESSING", Generation: &g, ExpiresAt: time.Now().Add(time.Hour), Attempts: 1}, claimed: true}
			store := &processingStoreFake{}
			transform := &transformerFake{}
			if scenario == "invalid" {
				transform.err = ErrInvalidImage
			}
			if scenario == "retry" || scenario == "exhausted" {
				store.writeFail = true
			}
			if scenario == "exhausted" {
				repo.record.Attempts = 5
			}
			processor := NewImageProcessor(repo, store, transform)
			err := processor.Process(context.Background(), key, g)
			want := map[string]string{"ready": "READY", "invalid": "FAILED", "retry": "PROCESSING", "exhausted": "FAILED"}[scenario]
			if len(repo.finishes) != 1 || repo.finishes[0] != want {
				t.Fatalf("finishes %v", repo.finishes)
			}
			if (err != nil) != (scenario == "retry") {
				t.Fatalf("retry result %v", err)
			}
			if scenario == "ready" {
				if store.writes != 2 || store.cleanups != 1 {
					t.Fatal("incomplete ready")
				}
				if err = processor.Process(context.Background(), key, g); err != nil || transform.calls != 1 {
					t.Fatal("duplicate event reprocessed")
				}
			} else if store.cleanups != 0 {
				t.Fatal("deleted failed/processing original")
			}
			calls := transform.calls
			if err = processor.Process(context.Background(), domain.ImagePrefix(key)+"medium.webp", g); err != nil || calls != transform.calls {
				t.Fatal("recursive processing")
			}
		})
	}
}
