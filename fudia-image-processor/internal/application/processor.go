package application

import (
	"context"
	"errors"
	"fmt"
	"github.com/foods-platform/fudia-image-processor/internal/domain"
	"time"
)

var ErrNotFound = errors.New("image reservation not found")

type ImageObject struct {
	Generation  int64
	Size        int64
	ContentType string
	Created     time.Time
}

var ErrImageBusy = errors.New("image processing lease busy")
var ErrInvalidImage = errors.New("invalid image contents")

type ImageProcessingRepository interface {
	Claim(context.Context, string, string, int64) (domain.ImageRecord, bool, error)
	Finish(context.Context, domain.ImageRecord, int64, string, string) error
}
type ImageProcessingStorage interface {
	Download(context.Context, string, int64) ([]byte, ImageObject, error)
	WriteVariant(context.Context, string, []byte, int64) error
	ScheduleOriginalDeletion(context.Context, string, int64) error
}
type ImageTransformer interface {
	Transform(context.Context, []byte, domain.ImageRecord) (map[string][]byte, error)
}
type ImageProcessor struct {
	repo        ImageProcessingRepository
	storage     ImageProcessingStorage
	transformer ImageTransformer
}

func NewImageProcessor(repo ImageProcessingRepository, storage ImageProcessingStorage, transformer ImageTransformer) *ImageProcessor {
	return &ImageProcessor{repo, storage, transformer}
}
func (p *ImageProcessor) Process(ctx context.Context, key string, generation int64) error {
	organizationID, imageID, ok := domain.ParseImageOriginalKey(key)
	if !ok {
		return nil
	} // Derived images also emit finalized events; never recurse.
	record, claimed, err := p.repo.Claim(ctx, organizationID, imageID, generation)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if record.OriginalKey != key {
		return nil
	}
	if !claimed {
		if record.Status == "READY" && record.Generation != nil && *record.Generation == generation {
			return p.storage.ScheduleOriginalDeletion(ctx, key, generation)
		}
		return nil
	}
	content, object, err := p.storage.Download(ctx, key, generation)
	if err == nil && (object.Size != record.SizeBytes || object.ContentType != record.ContentType || object.Created.After(record.ExpiresAt)) {
		err = ErrInvalidImage
	}
	var variants map[string][]byte
	if err == nil {
		variants, err = p.transformer.Transform(ctx, content, record)
	}
	if err == nil {
		names := []string{"thumbnail", "medium"}
		if record.Purpose == "restaurant_logo" {
			names = append(names, "logo")
		}
		if record.Purpose == "virtual_menu" {
			names = append(names, "delivery")
		}
		for _, name := range names {
			data, ok := variants[name]
			if !ok || len(data) == 0 {
				err = fmt.Errorf("missing %s variant", name)
				break
			}
			extension := ".webp"
			if name == "delivery" {
				extension = ".jpg"
			}
			if name == "logo" {
				extension = ".png"
			}
			if err = p.storage.WriteVariant(ctx, domain.ImagePrefix(key)+name+extension, data, generation); err != nil {
				break
			}
		}
	}
	if err != nil {
		if errors.Is(err, ErrInvalidImage) || record.Attempts >= 5 {
			code := "processing_failed"
			if errors.Is(err, ErrInvalidImage) {
				code = "invalid_image"
			}
			return p.repo.Finish(ctx, record, generation, "FAILED", code)
		}
		// Release the lease, keep PROCESSING, and let Eventarc retry transient faults.
		if releaseErr := p.repo.Finish(ctx, record, generation, "PROCESSING", ""); releaseErr != nil {
			return releaseErr
		}
		return err
	}
	if err = p.repo.Finish(ctx, record, generation, "READY", ""); err != nil {
		return err
	}
	// Custom-Time is set only after READY is durable. Lifecycle never deletes FAILED originals.
	return p.storage.ScheduleOriginalDeletion(ctx, key, generation)
}
