package application

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/menu/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

type restaurantMedia struct{ catalog *Catalog }

func (a *Catalog) RestaurantMedia() contracts.RestaurantMedia { return restaurantMedia{a} }
func (g restaurantMedia) Authorize(ctx context.Context, s scope, in contracts.MediaMetadata) (contracts.MediaAuthorization, error) {
	if in.Purpose != "restaurant_logo" && in.Purpose != "virtual_menu" {
		return contracts.MediaAuthorization{}, menuFailure(apperror.Invalid, "invalid_image_purpose", "Selecciona logo o carta virtual.")
	}
	result, err := g.catalog.AuthorizeImage(ctx, s, domain.ImageUploadInput{Purpose: in.Purpose, ContentType: in.ContentType, SizeBytes: in.SizeBytes, Width: in.Width, Height: in.Height})
	return contracts.MediaAuthorization{ImageID: result.ImageID, ObjectKey: result.ObjectKey, UploadURL: result.UploadURL, Method: result.Method, Fields: result.Fields, ExpiresAt: result.ExpiresAt}, err
}
func (g restaurantMedia) record(ctx context.Context, s scope, id string) (domain.ImageRecord, error) {
	if g.catalog.direct == nil {
		return domain.ImageRecord{}, menuFailure(apperror.Unavailable, "direct_upload_unavailable", "La carga de imágenes requiere configurar Storage.")
	}
	image, err := g.catalog.direct.repo.Read(ctx, s, id)
	if err != nil || (image.Purpose != "restaurant_logo" && image.Purpose != "virtual_menu") || (image.ConfirmedAt == nil && image.UploadedBy != s.UserID) {
		return image, menuFailure(apperror.NotFound, "image_not_found", "La imagen no existe en este restaurante.")
	}
	return image, nil
}
func (g restaurantMedia) Complete(ctx context.Context, s scope, id string) (contracts.RestaurantImage, error) {
	image, err := g.record(ctx, s, id)
	if err != nil {
		return contracts.RestaurantImage{}, err
	}
	result, err := g.catalog.CompleteImage(ctx, s, id)
	return contracts.RestaurantImage{ID: id, Purpose: image.Purpose, URL: result.ImageURL, Status: valueOr(result.ImageStatus, image.Status), ExpiresAt: result.ImageExpiresAt}, err
}
func (g restaurantMedia) Read(ctx context.Context, s scope, id string) (contracts.RestaurantImage, error) {
	image, err := g.record(ctx, s, id)
	if err != nil {
		return contracts.RestaurantImage{}, err
	}
	result, err := g.catalog.direct.view(ctx, image, "medium", false)
	out := contracts.RestaurantImage{ID: id, Purpose: image.Purpose, URL: result.ImageURL, Status: image.Status, ExpiresAt: result.ImageExpiresAt}
	if err == nil && image.Status == "READY" {
		if image.Purpose == "virtual_menu" {
			url, _, readErr := g.catalog.direct.storage.ReadURL(ctx, domain.ImageVariantKey(image, "delivery"), 0)
			if readErr == nil {
				out.DeliveryURL = &url
			}
		} else {
			out.DeliveryURL = out.URL
		}
	}
	return out, err
}
func valueOr(value *string, fallback string) string {
	if value != nil {
		return *value
	}
	return fallback
}
