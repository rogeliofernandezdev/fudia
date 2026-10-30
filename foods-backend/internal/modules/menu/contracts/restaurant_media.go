package contracts

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"time"
)

type MediaMetadata struct {
	Purpose     string `json:"purpose"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}
type MediaAuthorization struct {
	ImageID   string            `json:"imageId"`
	ObjectKey string            `json:"objectKey"`
	UploadURL string            `json:"uploadUrl"`
	Method    string            `json:"method"`
	Fields    map[string]string `json:"fields"`
	ExpiresAt time.Time         `json:"expiresAt"`
}
type RestaurantImage struct {
	DeliveryURL *string    `json:"deliveryUrl,omitempty"`
	ID          string     `json:"id"`
	Purpose     string     `json:"purpose"`
	URL         *string    `json:"url"`
	Status      string     `json:"status"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
}
type RestaurantMedia interface {
	Authorize(context.Context, requestscope.Scope, MediaMetadata) (MediaAuthorization, error)
	Complete(context.Context, requestscope.Scope, string) (RestaurantImage, error)
	Read(context.Context, requestscope.Scope, string) (RestaurantImage, error)
}
