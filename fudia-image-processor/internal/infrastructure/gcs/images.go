package gcs

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"fmt"
	"github.com/foods-platform/fudia-image-processor/internal/application"
	"github.com/foods-platform/fudia-image-processor/internal/domain"
	"google.golang.org/api/googleapi"
	"io"
	"strconv"
	"strings"
	"time"
)

type Images struct {
	client *storage.Client
	bucket string
}

func New(ctx context.Context, bucket string) (*Images, error) {
	if bucket == "" {
		return nil, errors.New("PRODUCT_IMAGES_BUCKET required")
	}
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Images{client, bucket}, nil
}
func (g *Images) Close() { _ = g.client.Close() }
func imageObject(attrs *storage.ObjectAttrs) application.ImageObject {
	return application.ImageObject{Generation: attrs.Generation, Size: attrs.Size, ContentType: attrs.ContentType, Created: attrs.Created}
}
func (g *Images) Download(ctx context.Context, key string, generation int64) ([]byte, application.ImageObject, error) {
	object := g.client.Bucket(g.bucket).Object(key).Generation(generation)
	attrs, err := object.Attrs(ctx)
	if err != nil {
		return nil, application.ImageObject{}, err
	}
	metadata := imageObject(attrs)
	if attrs.Size < 1 || attrs.Size > domain.MaxImageBytes {
		return nil, metadata, application.ErrInvalidImage
	}
	reader, err := object.NewReader(ctx)
	if err != nil {
		return nil, metadata, err
	}
	defer reader.Close()
	content, err := io.ReadAll(io.LimitReader(reader, domain.MaxImageBytes+1))
	if int64(len(content)) != attrs.Size || int64(len(content)) > domain.MaxImageBytes {
		return nil, metadata, application.ErrInvalidImage
	}
	return content, metadata, err
}
func (g *Images) WriteVariant(ctx context.Context, key string, data []byte, generation int64) error {
	object := g.client.Bucket(g.bucket).Object(key)
	writer := object.If(storage.Conditions{DoesNotExist: true}).NewWriter(ctx)
	writer.ContentType = "image/webp"
	if strings.HasSuffix(key, "/delivery.jpg") {
		writer.ContentType = "image/jpeg"
	}
	if strings.HasSuffix(key, "/logo.png") {
		writer.ContentType = "image/png"
	}
	writer.CacheControl = "private,max-age=1800,immutable"
	writer.Metadata = map[string]string{"source-generation": strconv.FormatInt(generation, 10)}
	writer.ChunkSize = 0 // bounded files, no 16 MB chunk buffer per request
	_, err := writer.Write(data)
	if err != nil {
		_ = writer.Close()
		return err
	}
	err = writer.Close()
	var apiError *googleapi.Error
	if errors.As(err, &apiError) && apiError.Code == 412 {
		attrs, inspectErr := object.Attrs(ctx)
		if inspectErr == nil && attrs.Metadata["source-generation"] == strconv.FormatInt(generation, 10) && attrs.ContentType == writer.ContentType && attrs.Size == int64(len(data)) {
			return nil
		}
	}
	return err
}
func (g *Images) ScheduleOriginalDeletion(ctx context.Context, key string, generation int64) error {
	object := g.client.Bucket(g.bucket).Object(key).Generation(generation)
	attrs, err := object.Attrs(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !attrs.CustomTime.IsZero() {
		return nil
	}
	_, err = object.If(storage.Conditions{MetagenerationMatch: attrs.Metageneration}).Update(ctx, storage.ObjectAttrsToUpdate{CustomTime: time.Now().UTC()})
	if err != nil {
		return fmt.Errorf("schedule original cleanup: %w", err)
	}
	return nil
}
