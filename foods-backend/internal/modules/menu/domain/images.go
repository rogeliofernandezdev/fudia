package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const MaxImageBytes int64 = 5 * 1024 * 1024
const MaxImageDimension = 4000

var imageUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type ImageUploadInput struct {
	Purpose     string  `json:"purpose,omitempty"`
	ProductID   *string `json:"productId,omitempty"`
	ContentType string  `json:"contentType"`
	SizeBytes   int64   `json:"sizeBytes"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
}

func (in ImageUploadInput) Validate() string {
	if in.Purpose != "" && in.Purpose != "product" && in.Purpose != "restaurant_logo" && in.Purpose != "virtual_menu" {
		return "El uso de la imagen no es válido."
	}
	if in.Purpose != "" && in.Purpose != "product" && in.ProductID != nil {
		return "La imagen del restaurante no corresponde a un producto."
	}
	if in.ContentType != "image/jpeg" && in.ContentType != "image/png" && in.ContentType != "image/webp" {
		return "Solo se aceptan imágenes JPEG, PNG o WebP."
	}
	if in.SizeBytes < 1 || in.SizeBytes > MaxImageBytes {
		return "La imagen debe pesar como máximo 5 MB."
	}
	if in.Width < 1 || in.Height < 1 || in.Width > MaxImageDimension || in.Height > MaxImageDimension {
		return "La resolución máxima es 4000 × 4000 píxeles."
	}
	if in.ProductID != nil && !imageUUID.MatchString(*in.ProductID) {
		return "El producto no es válido."
	}
	return ""
}
func ImageOriginalKey(organizationID, imageID, contentType string, purpose ...string) string {
	extension := map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp"}[contentType]
	folder := "products"
	if len(purpose) > 0 && (purpose[0] == "restaurant_logo" || purpose[0] == "virtual_menu") {
		folder = "branding"
	}
	return fmt.Sprintf("restaurants/%s/%s/%s/original.%s", organizationID, folder, imageID, extension)
}
func ImagePrefix(original string) string { return original[:strings.LastIndex(original, "/")+1] }
func ImageVariantKey(record ImageRecord, variant string) string {
	if record.Purpose == "virtual_menu" && variant == "delivery" {
		return ImagePrefix(record.OriginalKey) + "delivery.jpg"
	}
	if record.Purpose == "restaurant_logo" && variant == "medium" {
		return ImagePrefix(record.OriginalKey) + "logo.png"
	}
	return ImagePrefix(record.OriginalKey) + variant + ".webp"
}
func ParseImageOriginalKey(key string) (organizationID, imageID string, ok bool) {
	parts := strings.Split(key, "/")
	if len(parts) != 5 || parts[0] != "restaurants" || (parts[2] != "products" && parts[2] != "branding") || !imageUUID.MatchString(parts[1]) || !imageUUID.MatchString(parts[3]) {
		return "", "", false
	}
	if parts[4] != "original.jpg" && parts[4] != "original.png" && parts[4] != "original.webp" {
		return "", "", false
	}
	return parts[1], parts[3], true
}

type ImageRecord struct {
	Purpose                                                                      string
	ID, OrganizationID, LocationID, UploadedBy, OriginalKey, ContentType, Status string
	ProductID                                                                    *string
	SizeBytes                                                                    int64
	Width, Height, Attempts                                                      int
	Generation                                                                   *int64
	ConfirmedAt                                                                  *time.Time
	ExpiresAt                                                                    time.Time
}
type ImageView struct {
	ImageID        *string           `json:"imageId,omitempty"`
	ImageKey       *string           `json:"imageKey,omitempty"`
	ImageStatus    *string           `json:"imageStatus,omitempty"`
	ImageExpiresAt *time.Time        `json:"imageExpiresAt,omitempty"`
	ImageVariants  map[string]string `json:"imageVariants,omitempty"`
}
type ImageUploadAuthorization struct {
	ImageID   string            `json:"imageId"`
	ObjectKey string            `json:"objectKey"`
	UploadURL string            `json:"uploadUrl"`
	Method    string            `json:"method"`
	Fields    map[string]string `json:"fields"`
	ExpiresAt time.Time         `json:"expiresAt"`
}
type ImageResult struct {
	ImageView
	ImageURL *string `json:"imageUrl"`
}
