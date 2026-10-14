package application

import (
	"context"
	"errors"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

type ImageObject struct {
	Generation  int64
	Size        int64
	ContentType string
	Created     time.Time
}
type ImageRepository interface {
	Reserve(context.Context, scope, domain.ImageUploadInput, time.Time) (domain.ImageRecord, error)
	Read(context.Context, scope, string) (domain.ImageRecord, error)
	Confirm(context.Context, scope, string, int64) (domain.ImageRecord, error)
	Attach(context.Context, scope, string, string) error
	Detach(context.Context, scope, string) error
	ForProducts(context.Context, scope, []string) (map[string]domain.ImageRecord, error)
}
type ImageUploadStorage interface {
	Authorize(context.Context, domain.ImageRecord) (domain.ImageUploadAuthorization, error)
	Inspect(context.Context, string) (ImageObject, error)
	ReadURL(context.Context, string, int64) (string, time.Time, error)
}
type ProductImages struct {
	repo    ImageRepository
	storage ImageUploadStorage
}

func NewProductImages(repo ImageRepository, storage ImageUploadStorage) *ProductImages {
	return &ProductImages{repo, storage}
}

func (a *Catalog) AuthorizeImage(ctx context.Context, s scope, in domain.ImageUploadInput) (domain.ImageUploadAuthorization, error) {
	if invalid := in.Validate(); invalid != "" {
		return domain.ImageUploadAuthorization{}, menuFailure(apperror.Invalid, "invalid_image", invalid)
	}
	if a.direct == nil {
		return domain.ImageUploadAuthorization{}, menuFailure(apperror.Unavailable, "direct_upload_unavailable", "La carga directa de imágenes aún no está configurada.")
	}
	if in.ProductID != nil {
		exists, err := a.repo.ProductExists(ctx, s, *in.ProductID)
		if err != nil {
			return domain.ImageUploadAuthorization{}, menuFailure(apperror.Unavailable, "image_unavailable", "No pudimos validar el producto.")
		}
		if !exists {
			return domain.ImageUploadAuthorization{}, menuFailure(apperror.NotFound, "product_not_found", "El producto no existe.")
		}
	}
	record, err := a.direct.repo.Reserve(ctx, s, in, time.Now().UTC().Add(10*time.Minute))
	if errors.Is(err, ErrNotFound) {
		return domain.ImageUploadAuthorization{}, menuFailure(apperror.Conflict, "image_upload_limit", "Espera unos minutos antes de autorizar otra imagen.")
	}
	if err != nil {
		return domain.ImageUploadAuthorization{}, menuFailure(apperror.Unavailable, "image_unavailable", "No pudimos autorizar la imagen.")
	}
	out, err := a.direct.storage.Authorize(ctx, record)
	if err != nil {
		return out, menuFailure(apperror.Unavailable, "image_unavailable", "No pudimos autorizar la subida a Storage.")
	}
	return out, nil
}
func (a *Catalog) CompleteImage(ctx context.Context, s scope, id string) (domain.ImageResult, error) {
	if a.direct == nil {
		return domain.ImageResult{}, menuFailure(apperror.Unavailable, "direct_upload_unavailable", "La carga directa de imágenes aún no está configurada.")
	}
	record, err := a.direct.repo.Read(ctx, s, id)
	if err != nil || record.UploadedBy != s.UserID || record.LocationID != s.LocationID {
		return domain.ImageResult{}, menuFailure(apperror.NotFound, "image_not_found", "La autorización de imagen no existe.")
	}
	if record.ConfirmedAt == nil {
		object, err := a.direct.storage.Inspect(ctx, record.OriginalKey)
		if err != nil {
			return domain.ImageResult{}, menuFailure(apperror.Conflict, "image_upload_incomplete", "La subida no se ha confirmado. Intenta nuevamente.")
		}
		if object.Size != record.SizeBytes || object.ContentType != record.ContentType || object.Created.After(record.ExpiresAt) || object.Generation <= 0 {
			return domain.ImageResult{}, menuFailure(apperror.Invalid, "image_metadata_mismatch", "El archivo subido no coincide con la imagen autorizada.")
		}
		record, err = a.direct.repo.Confirm(ctx, s, id, object.Generation)
		if err != nil {
			return domain.ImageResult{}, menuFailure(apperror.Conflict, "image_upload_conflict", "No pudimos confirmar esta versión de la imagen.")
		}
	}
	return a.direct.view(ctx, record, "medium", false)
}
func (a *Catalog) GetImage(ctx context.Context, s scope, id string) (domain.ImageResult, error) {
	if a.direct == nil {
		return domain.ImageResult{}, menuFailure(apperror.NotFound, "image_not_found", "La imagen no existe.")
	}
	record, err := a.direct.repo.Read(ctx, s, id)
	if err != nil || (record.ConfirmedAt == nil && record.UploadedBy != s.UserID) {
		return domain.ImageResult{}, menuFailure(apperror.NotFound, "image_not_found", "La imagen no existe.")
	}
	return a.direct.view(ctx, record, "medium", true)
}
func (a *Catalog) AttachImage(ctx context.Context, s scope, productID, imageID string) (domain.Product, error) {
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return domain.Product{}, menuFailure(apperror.Unavailable, "image_unavailable", "No pudimos vincular la imagen.")
	}
	defer tx.Rollback(ctx)
	if _, _, _, _, err = a.repo.LockProduct(tx.Context(), s, productID); err != nil {
		return domain.Product{}, menuFailure(apperror.NotFound, "product_not_found", "El producto no existe.")
	}
	if err = a.attachProductImage(tx.Context(), s, productID, domain.ProductInput{ImageID: &imageID}); err != nil {
		return domain.Product{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Product{}, menuFailure(apperror.Unavailable, "image_unavailable", "No pudimos confirmar la imagen.")
	}
	// Do not report a committed write as failed if subsequent read URL signing is unavailable.
	product, err := a.Product(ctx, s, productID)
	if err != nil {
		return domain.Product{ID: productID}, nil
	}
	return product, nil
}
func (p *ProductImages) view(ctx context.Context, record domain.ImageRecord, variant string, detail bool) (domain.ImageResult, error) {
	key := record.OriginalKey
	generation := int64(0)
	if record.Generation != nil {
		generation = *record.Generation
	}
	if record.Status == "READY" {
		key = domain.ImagePrefix(key) + variant + ".webp"
		generation = 0
	}
	result := domain.ImageResult{ImageView: domain.ImageView{ImageID: &record.ID, ImageKey: &key, ImageStatus: &record.Status}}
	// Reservations without an uploaded object are not advertised as usable images.
	if generation == 0 && record.Status != "READY" {
		return result, nil
	}
	url, expiry, err := p.storage.ReadURL(ctx, key, generation)
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "image_url_unavailable", "No pudimos cargar la imagen. Intenta nuevamente.")
	}
	result.ImageURL = &url
	result.ImageExpiresAt = &expiry
	if detail && record.Status == "READY" {
		result.ImageVariants = map[string]string{}
		for _, name := range []string{"thumbnail", "medium"} {
			signed, expires, err := p.storage.ReadURL(ctx, domain.ImagePrefix(record.OriginalKey)+name+".webp", 0)
			if err != nil {
				return result, menuFailure(apperror.Unavailable, "image_url_unavailable", "No pudimos cargar las versiones de la imagen.")
			}
			result.ImageVariants[name] = signed
			if expires.Before(*result.ImageExpiresAt) {
				result.ImageExpiresAt = &expires
			}
		}
	}
	return result, nil
}
func (a *Catalog) attachProductImage(ctx context.Context, s scope, id string, in domain.ProductInput) error {
	if in.RemoveImage && in.ImageID != nil {
		return menuFailure(apperror.Invalid, "invalid_image", "Selecciona una imagen o quítala, no ambas opciones.")
	}
	if a.direct == nil {
		if in.ImageID != nil {
			return menuFailure(apperror.Unavailable, "direct_upload_unavailable", "La carga directa aún no está configurada.")
		}
		return nil
	}
	if in.RemoveImage {
		return a.direct.repo.Detach(ctx, s, id)
	}
	if in.ImageID == nil {
		return nil
	}
	if err := a.direct.repo.Attach(ctx, s, id, *in.ImageID); err != nil {
		return menuFailure(apperror.Conflict, "image_not_confirmed", "La imagen debe estar subida y pertenecer a este producto y restaurante.")
	}
	return nil
}
func (a *Catalog) productImageViews(ctx context.Context, s scope, ids []string, variant string, detail bool) (map[string]domain.ImageResult, error) {
	out := map[string]domain.ImageResult{}
	if a.direct == nil || len(ids) == 0 {
		return out, nil
	}
	records, err := a.direct.repo.ForProducts(ctx, s, ids)
	if err != nil {
		return nil, menuFailure(apperror.Unavailable, "images_unavailable", "No pudimos cargar las imágenes del catálogo.")
	}
	// One batched SQL lookup, bounded concurrent signing; not one browser request per photo.
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(8)
	var mu sync.Mutex
	for id, record := range records {
		group.Go(func() error {
			view, err := a.direct.view(groupCtx, record, variant, detail)
			if err != nil {
				if detail { return err } // Missing optional photos must not prevent ordering from the catalogue.
			}
			mu.Lock()
			out[id] = view
			mu.Unlock()
			return nil
		})
	}
	return out, group.Wait()
}
func (a *Catalog) hydratedWrite(ctx context.Context, s scope, p domain.Product) (domain.Product, error) {
	hydrated, err := a.hydrateProduct(ctx, s, p, false)
	if err != nil {
		return p, nil
	} // The write already committed; a photo read outage must not invite duplicate creates.
	return hydrated, nil
}
func (a *Catalog) hydrateProduct(ctx context.Context, s scope, p domain.Product, detail bool) (domain.Product, error) {
	views, err := a.productImageViews(ctx, s, []string{p.ID}, "medium", detail)
	if v, ok := views[p.ID]; ok {
		p.ImageView = v.ImageView
		p.ImageURL = v.ImageURL
	}
	return p, err
}
