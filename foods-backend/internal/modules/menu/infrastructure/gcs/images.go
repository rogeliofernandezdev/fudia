package gcs

import (
	"cloud.google.com/go/storage"
	"context"
	"encoding/base64"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/menu/application"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"golang.org/x/sync/singleflight"
	"google.golang.org/api/iamcredentials/v1"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"time"
)

type signedImage struct {
	url     string
	expires time.Time
}
type Images struct {
	client         *storage.Client
	iam            *iamcredentials.Service
	bucket, signer string
	mu             sync.Mutex
	cache          map[string]signedImage
	signing        singleflight.Group
	permits        chan struct{}
	signOverride   func(context.Context, []byte) ([]byte, error) // test seam; production always uses IAM signBlob
}

func New(ctx context.Context, bucket, signer string) (*Images, error) {
	if !regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`).MatchString(bucket) {
		return nil, errors.New("invalid PRODUCT_IMAGES_BUCKET")
	}
	if signer != "" && !regexp.MustCompile(`^[a-z][a-z0-9-]+@[a-z][a-z0-9-]+\.iam\.gserviceaccount\.com$`).MatchString(signer) {
		return nil, errors.New("invalid PRODUCT_IMAGES_SIGNER")
	}
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	result := &Images{client: client, bucket: bucket, signer: signer, cache: map[string]signedImage{}, permits: make(chan struct{}, 8)}
	if signer != "" {
		result.iam, err = iamcredentials.NewService(ctx)
		if err != nil {
			client.Close()
			return nil, err
		}
	}
	return result, nil
}
func (g *Images) Close() { _ = g.client.Close() }
func (g *Images) sign(ctx context.Context, data []byte) ([]byte, error) {
	if g.signOverride != nil {
		return g.signOverride(ctx, data)
	}
	if g.iam == nil {
		return nil, errors.New("image signer not configured")
	}
	select {
	case g.permits <- struct{}{}:
		defer func() { <-g.permits }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	response, err := g.iam.Projects.ServiceAccounts.SignBlob("projects/-/serviceAccounts/"+g.signer, &iamcredentials.SignBlobRequest{Payload: base64.StdEncoding.EncodeToString(data)}).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(response.SignedBlob)
}
func (g *Images) Authorize(ctx context.Context, record domain.ImageRecord) (domain.ImageUploadAuthorization, error) {
	policy, err := storage.GenerateSignedPostPolicyV4(g.bucket, record.OriginalKey, &storage.PostPolicyV4Options{
		GoogleAccessID: g.signer, Expires: record.ExpiresAt,
		SignRawBytes: func(data []byte) ([]byte, error) { return g.sign(ctx, data) },
		Fields:       &storage.PolicyV4Fields{ContentType: record.ContentType, CacheControl: "private,no-store", ContentDisposition: "inline", StatusCodeOnSuccess: 204},
		Conditions:   []storage.PostPolicyV4Condition{storage.ConditionContentLengthRange(1, uint64(domain.MaxImageBytes))},
	})
	if err != nil {
		return domain.ImageUploadAuthorization{}, err
	}
	return domain.ImageUploadAuthorization{ImageID: record.ID, ObjectKey: record.OriginalKey, UploadURL: policy.URL, Method: "POST", Fields: policy.Fields, ExpiresAt: record.ExpiresAt}, nil
}
func imageObject(attrs *storage.ObjectAttrs) application.ImageObject {
	return application.ImageObject{Generation: attrs.Generation, Size: attrs.Size, ContentType: attrs.ContentType, Created: attrs.Created}
}
func (g *Images) Inspect(ctx context.Context, key string) (application.ImageObject, error) {
	attrs, err := g.client.Bucket(g.bucket).Object(key).Attrs(ctx)
	if err != nil {
		return application.ImageObject{}, err
	}
	return imageObject(attrs), nil
}
func (g *Images) ReadURL(ctx context.Context, key string, generation int64) (string, time.Time, error) {
	cacheKey := key + ":" + strconv.FormatInt(generation, 10)
	g.mu.Lock()
	cached, ok := g.cache[cacheKey]
	g.mu.Unlock()
	if ok && time.Until(cached.expires) > time.Minute {
		return cached.url, cached.expires, nil
	}
	result, err, _ := g.signing.Do(cacheKey, func() (any, error) {
		g.mu.Lock()
		cached, ok := g.cache[cacheKey]
		g.mu.Unlock()
		if ok && time.Until(cached.expires) > time.Minute {
			return cached, nil
		}
		expiry := time.Now().UTC().Add(time.Hour)
		parameters := url.Values{}
		if generation > 0 {
			parameters.Set("generation", strconv.FormatInt(generation, 10))
		}
		signed, err := storage.SignedURL(g.bucket, key, &storage.SignedURLOptions{GoogleAccessID: g.signer, Scheme: storage.SigningSchemeV4, Method: "GET", Expires: expiry, QueryParameters: parameters, SignBytes: func(data []byte) ([]byte, error) { return g.sign(ctx, data) }})
		if err != nil {
			return nil, err
		}
		value := signedImage{signed, expiry}
		g.mu.Lock()
		if len(g.cache) >= 4096 {
			for k, v := range g.cache {
				if time.Until(v.expires) <= time.Minute {
					delete(g.cache, k)
				}
			}
			if len(g.cache) >= 4096 {
				for k := range g.cache {
					delete(g.cache, k)
					break
				}
			}
		}
		g.cache[cacheKey] = value
		g.mu.Unlock()
		return value, nil
	})
	if err != nil {
		return "", time.Time{}, err
	}
	value := result.(signedImage)
	return value.url, value.expires, nil
}
