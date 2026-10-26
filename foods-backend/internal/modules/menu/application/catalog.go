package application

import (
	"context"
	"errors"
	"fmt"
	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrNotFound = errors.New("menu record not found")
var ErrAllergenConflict = errors.New("allergen already exists")

type scope = requestscope.Scope
type category = domain.Category
type product = domain.Product

var normalizeCategoryProductScope = domain.NormalizeCategoryProductScope
var normalizeProduct = domain.NormalizeProduct
var normalizeProductCategory = domain.NormalizeProductCategory
var validateInitialPortions = domain.ValidateInitialPortions

type CatalogFilter struct {
	Query, Status, CategoryID, ProductType string
	IncludeInactive                        bool
	Page, Size                             int
}
type CategoryPage struct {
	Items    []domain.Category `json:"items"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}
type ProductPage struct {
	Items                     []domain.ProductListItem `json:"items"`
	Total                     int                      `json:"total"`
	Page                      int                      `json:"page"`
	PageSize                  int                      `json:"pageSize"`
	ServiceDestinationOptions []map[string]string      `json:"serviceDestinationOptions"`
}
type ImageStore interface {
	Save(context.Context, string, string, io.Reader) (string, error)
}
type Catalog struct {
	repo   CatalogRepository
	work   unitofwork.Manager
	audit  audit.Repository
	images ImageStore
	direct *ProductImages
}

func NewCatalog(repo CatalogRepository, work unitofwork.Manager, audit audit.Repository, images ImageStore) *Catalog {
	return &Catalog{repo: repo, work: work, audit: audit, images: images}
}
func (a *Catalog) WithProductImages(images *ProductImages) *Catalog { a.direct = images; return a }
func (a *Catalog) DirectImagesEnabled() bool                        { return a.direct != nil }
func menuFailure(kind apperror.Kind, code, message string) error {
	return apperror.New(kind, code, message)
}

type CatalogRepository interface {
	PublicMenu(context.Context, scope, []string, string, time.Time) ([]domain.PublicMenuSnapshot, error)
	PublicCombo(ctx context.Context, s scope, id string, day time.Time) (domain.PublicComboDetail, bool, error)
	PublicComboGroups(context.Context, scope, string, time.Time) ([]domain.PublicComboGroup, error)
	PublicModifierProduct(ctx context.Context, s scope, productID string) (bool, bool, error)
	RecipeModules(context.Context, scope) (bool, bool, error)
	Recipes(context.Context, scope, CatalogFilter) (RecipePage, error)
	Recipe(context.Context, scope, string) (domain.RecipeView, error)
	LockRecipeProduct(ctx context.Context, s scope, productID string) (string, string, error)
	RecipeIngredientActive(ctx context.Context, s scope, x domain.RecipeIngredientInput) (bool, error)
	SaveRecipe(ctx context.Context, s scope, productID string, in domain.RecipeInput) (string, error)
	DeleteRecipeIngredients(ctx context.Context, s scope, recipeID string) (int64, error)
	CreateRecipeIngredient(ctx context.Context, s scope, recipeID string, x domain.RecipeIngredientInput) (int64, error)
	Availability(context.Context, scope, CatalogFilter, time.Time, bool) ([]domain.AvailabilitySnapshot, int, error)
	ComboAvailable(context.Context, scope, string, time.Time) (bool, error)
	AvailabilityHistory(context.Context, scope, string, CatalogFilter) (AvailabilityHistoryPage, error)
	ProductExists(context.Context, scope, string) (bool, error)
	LockAvailabilityProduct(ctx context.Context, s scope, productID string) (string, error)
	LockDailyAvailability(ctx context.Context, s scope, productID string, day time.Time) (*int, int, string, string, error)
	UpdateAvailability(ctx context.Context, s scope, productID string, day time.Time, in domain.AvailabilityInput) (int64, error)
	AuditAvailability(ctx context.Context, s scope, productID string, day time.Time, in domain.AvailabilityInput, currentPortionQuantity *int, soldQuantity int, currentStatus string) (int64, error)
	SupportsModifiers(context.Context, scope, string) (bool, error)
	Modifiers(context.Context, scope, string) (domain.ModifierConfigView, error)
	DeleteModifiers(ctx context.Context, s scope, productID string) (int64, error)
	CreateModifierGroup(ctx context.Context, s scope, productID string, group domain.ModifierGroupInput, groupIndex int) (string, error)
	CreateModifierOption(ctx context.Context, s scope, groupID string, option domain.ModifierOptionInput, surcharge string, optionIndex int) (int64, error)
	SetComboActive(ctx context.Context, s scope, id string, in struct{ Active *bool }) (int64, error)
	CreateComboProduct(ctx context.Context, s scope, in domain.ComboInput, sku string) (string, error)
	CreateCombo(ctx context.Context, s scope, id string) (int64, error)
	UpdateCombo(ctx context.Context, s scope, id string, in domain.ComboInput) (int64, error)
	DeleteComboGroups(ctx context.Context, s scope, id string) (int64, error)
	CreateComboGroup(ctx context.Context, s scope, id string, group domain.ComboGroupInput, groupIndex int) (string, error)
	CreateComboOption(ctx context.Context, s scope, id string, groupID string, option domain.ComboOptionInput, optionIndex int) (int64, error)
	Combos(context.Context, scope, CatalogFilter) (ComboPage, error)
	OrderCombos(context.Context, scope, CatalogFilter, time.Time) (OrderComboPage, error)
	Combo(context.Context, scope, string) (domain.Combo, error)
	OrderCombo(context.Context, scope, string, time.Time) (domain.OrderCombo, bool, error)
	BusinessDate(context.Context, scope) (time.Time, error)
	CreateCategory(ctx context.Context, s scope, in domain.CategoryInput, productScope string) (domain.Category, error)
	IncompatibleCategoryProducts(ctx context.Context, s scope, id string, incompatibleType string) (int, error)
	UpdateCategory(ctx context.Context, s scope, id string, in domain.CategoryInput, active bool, productScope string) (domain.Category, error)
	ActiveCategoryProducts(ctx context.Context, s scope, id string) (int, error)
	DeactivateCategory(ctx context.Context, s scope, id string) (int64, error)
	ActivateCategory(ctx context.Context, s scope, id string) (int64, error)
	Product(ctx context.Context, s scope, id string) (domain.Product, error)
	CreateProduct(ctx context.Context, s scope, in domain.ProductInput, createFeatured bool) (domain.Product, error)
	InitialPortions(ctx context.Context, s scope, in domain.ProductInput, p domain.Product) (string, error)
	ProductCreatedAudit(ctx context.Context, s scope, p domain.Product, in domain.ProductInput) (int64, error)
	LockProduct(ctx context.Context, s scope, id string) (string, string, string, string, error)
	ProductCategoryAvailable(ctx context.Context, s scope, categoryID string, productID *string) (bool, error)
	HasStockMovements(ctx context.Context, s scope, id string) (bool, error)
	HasPortionSales(ctx context.Context, s scope, id string) (bool, error)
	UpdateProduct(ctx context.Context, s scope, id string, in domain.ProductInput, skuValue string, active bool, featured bool) (domain.Product, error)
	SetProductActive(ctx context.Context, s scope, id string, in struct{ Active *bool }) (int64, error)
	DeactivateProduct(ctx context.Context, s scope, id string) (int64, error)
	Categories(context.Context, scope, CatalogFilter) (CategoryPage, error)
	Products(context.Context, scope, CatalogFilter) (ProductPage, error)
	ProductCount(context.Context, scope) (int, error)
	HasCancellableOrderUsage(context.Context, scope, string) (bool, error)
	Allergens(context.Context, scope, string) ([]string, error)
	CreateAllergen(context.Context, scope, string) (domain.Allergen, error)
	SetImage(context.Context, scope, string, string) error
}

func (a *Catalog) CreateCategory(ctx context.Context, s scope, in domain.CategoryInput) (domain.Category, error) {
	var result domain.Category
	if strings.TrimSpace(in.Name) == "" {
		return result, menuFailure(apperror.Invalid, "invalid_category", "El nombre de la categoría es obligatorio.")
	}
	productScope, invalid := normalizeCategoryProductScope(in.ProductScope, "prepared")
	if invalid != "" {
		return result, menuFailure(apperror.Invalid, "invalid_category", invalid)
	}
	var c category
	err := func() error { var err error; c, err = a.repo.CreateCategory(ctx, s, in, productScope); return err }()
	if err != nil {
		return result, menuFailure(apperror.Conflict, "category_conflict", "Ya existe una categoría con ese nombre.")
	}
	_ = a.audit.Record(ctx, s, "category.created", "menu_category", c.ID)
	return c, nil

}
func (a *Catalog) UpdateCategory(ctx context.Context, s scope, id string, in domain.CategoryInput) (domain.Category, error) {
	var result domain.Category
	if strings.TrimSpace(in.Name) == "" {
		return result, menuFailure(apperror.Invalid, "invalid_category", "El nombre de la categoría es obligatorio.")
	}
	productScope, invalid := normalizeCategoryProductScope(in.ProductScope, "")
	if invalid != "" {
		return result, menuFailure(apperror.Invalid, "invalid_category", invalid)
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	if productScope == "prepared" || productScope == "retail" {
		incompatibleType := "retail"
		if productScope == "retail" {
			incompatibleType = "prepared"
		}
		var incompatibleProducts int
		if err := func() error {
			var err error
			incompatibleProducts, err = a.repo.IncompatibleCategoryProducts(ctx, s, id, incompatibleType)
			return err
		}(); err != nil {
			return result, menuFailure(apperror.Unavailable, "category_unavailable", "No pudimos validar los productos de la categoría.")
		}
		if incompatibleProducts > 0 {
			return result, menuFailure(apperror.Conflict, "category_scope_in_use", "La categoría contiene productos de otro tipo. Usa «Ambos» o mueve esos productos antes de cambiar su uso.")
		}
	}
	var c category
	err := func() error {
		var err error
		c, err = a.repo.UpdateCategory(ctx, s, id, in, active, productScope)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		return result, menuFailure(apperror.NotFound, "category_not_found", "La categoría no existe.")
	}
	if err != nil {
		return result, menuFailure(apperror.Conflict, "category_conflict", "No se pudo actualizar la categoría.")
	}
	_ = a.audit.Record(ctx, s, "category.updated", "menu_category", c.ID)
	return c, nil

}
func (a *Catalog) DeactivateCategory(ctx context.Context, s scope, id string) error {
	var products int
	if err := func() error { var err error; products, err = a.repo.ActiveCategoryProducts(ctx, s, id); return err }(); err != nil {
		return menuFailure(apperror.Unavailable, "category_unavailable", "No pudimos validar la categoría.")
	}
	if products > 0 {
		return menuFailure(apperror.Conflict, "category_in_use", "La categoría se encuentra en uso porque tiene productos activos asociados. Desactiva esos productos o muévelos a otra categoría antes de desactivarla.")
	}
	tag, err := a.repo.DeactivateCategory(ctx, s, id)
	if err != nil || tag == 0 {
		return menuFailure(apperror.NotFound, "category_not_found", "La categoría no existe o ya está inactiva.")
	}
	_ = a.audit.Record(ctx, s, "category.deactivated", "menu_category", id)
	return nil

}
func (a *Catalog) SetCategoryActive(ctx context.Context, s scope, id string, active bool) error {
	if !active {
		return a.DeactivateCategory(ctx, s, id)
	}
	tag, err := a.repo.ActivateCategory(ctx, s, id)
	if err != nil {
		return menuFailure(apperror.Unavailable, "category_unavailable", "No pudimos actualizar el estado de la categoría.")
	}
	if tag == 0 {
		return menuFailure(apperror.NotFound, "category_not_found", "La categoría no existe.")
	}
	_ = a.audit.Record(ctx, s, "category.activated", "menu_category", id)
	return nil

}
func (a *Catalog) Product(ctx context.Context, s scope, id string) (domain.Product, error) {
	var result domain.Product
	var p product
	err := func() error { var err error; p, err = a.repo.Product(ctx, s, id); return err }()
	if errors.Is(err, ErrNotFound) {
		return result, menuFailure(apperror.NotFound, "product_not_found", "El producto no existe.")
	}
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "product_unavailable", "No pudimos cargar el producto.")
	}
	return a.hydrateProduct(ctx, s, p, true)

}
func (a *Catalog) CreateProduct(ctx context.Context, s scope, in domain.ProductInput) (domain.Product, error) {
	var result domain.Product
	if in.ImageURL != nil && strings.Contains(strings.ToLower(*in.ImageURL), "x-goog-signature=") {
		return result, menuFailure(apperror.Invalid, "invalid_image_reference", "Guarda el identificador de imagen, no una URL temporal.")
	}
	if in.ImageID != nil || in.RemoveImage {
		in.ImageURL = nil
	}
	in, invalid := normalizeProduct(in)
	if invalid == "" {
		invalid = validateInitialPortions(in)
	}
	if invalid != "" {
		return result, menuFailure(apperror.Invalid, "invalid_product", invalid)
	}
	if in, invalid = normalizeProductCategory(in); invalid != "" {
		return result, menuFailure(apperror.Invalid, "category_required", invalid)
	}
	if strings.TrimSpace(in.SKU) == "" {
		generated, err := a.nextSKU(ctx, s)
		if err != nil {
			return result, menuFailure(apperror.Unavailable, "sku_unavailable", "No pudimos generar un código de producto.")
		}
		in.SKU = generated
	}
	createFeatured := false
	if in.Featured != nil {
		createFeatured = *in.Featured
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "product_unavailable", "No pudimos iniciar el registro del producto.")
	}
	defer tx.Rollback(ctx)
	if err = a.requireProductCategory(tx.Context(), s, *in.CategoryID, nil); err != nil {
		return result, err
	}
	var p product
	err = func() error {
		var err error
		p, err = a.repo.CreateProduct(tx.Context(), s, in, createFeatured)
		return err
	}()
	if err != nil {
		return result, menuFailure(apperror.Conflict, "product_conflict", "Revisa la categoría y el precio.")
	}
	if in.InitialPortionQuantity != nil {

		err = func() error {
			var err error
			_, err = a.repo.InitialPortions(tx.Context(), s, in, p)
			return err
		}()
		if err != nil {
			return result, menuFailure(apperror.Unavailable, "product_unavailable", "No pudimos registrar las porciones de hoy. El producto no se guardó.")
		}
	}
	_, err = a.repo.ProductCreatedAudit(tx.Context(), s, p, in)
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "product_unavailable", "No pudimos confirmar el registro del producto.")
	}
	if err = a.attachProductImage(tx.Context(), s, p.ID, in); err != nil {
		return result, err
	}
	if err = tx.Commit(ctx); err != nil {
		return result, menuFailure(apperror.Unavailable, "product_unavailable", "No pudimos confirmar el registro del producto.")
	}
	return a.hydratedWrite(ctx, s, p)

}
func (a *Catalog) UpdateProduct(ctx context.Context, s scope, id string, in domain.ProductInput) (domain.Product, error) {
	var result domain.Product
	if in.ImageURL != nil && strings.Contains(strings.ToLower(*in.ImageURL), "x-goog-signature=") {
		return result, menuFailure(apperror.Invalid, "invalid_image_reference", "Guarda el identificador de imagen, no una URL temporal.")
	}
	if in.ImageID != nil || in.RemoveImage {
		in.ImageURL = nil
	}
	destinationProvided := in.ServiceDestination != ""
	if in.InitialPortionQuantity != nil {
		return result, menuFailure(apperror.Invalid, "invalid_product", "Actualiza la cantidad de un producto existente desde Disponibilidad.")
	}
	productTypeProvided := strings.TrimSpace(in.ProductType) != ""
	in, invalid := normalizeProduct(in)
	if invalid != "" {
		return result, menuFailure(apperror.Invalid, "invalid_product", invalid)
	}
	if in, invalid = normalizeProductCategory(in); invalid != "" {
		return result, menuFailure(apperror.Invalid, "category_required", invalid)
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	featured := false
	if in.Featured != nil {
		featured = *in.Featured
	}
	skuValue := strings.TrimSpace(in.SKU)

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "product_unavailable", "No pudimos actualizar el producto.")
	}
	defer tx.Rollback(ctx)

	var currentSKU, currentControl, currentProductType, currentDestination string
	err = func() error {
		var err error
		currentSKU, currentControl, currentProductType, currentDestination, err = a.repo.LockProduct(tx.Context(), s, id)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		return result, menuFailure(apperror.NotFound, "product_not_found", "El producto no existe.")
	}
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "product_unavailable", "No pudimos validar el producto.")
	}
	if err = a.requireProductCategory(tx.Context(), s, *in.CategoryID, &id); err != nil {
		return result, err
	}
	if skuValue == "" {
		skuValue = currentSKU
	}
	if !destinationProvided {
		in.ServiceDestination = currentDestination
	}
	if !productTypeProvided {
		in.ProductType = currentProductType
	}
	if currentControl != in.QuantityControl {
		if currentControl == "none" && in.QuantityControl != "none" {
			usedByOpenOrder, usageErr := a.repo.HasCancellableOrderUsage(tx.Context(), s, id)
			if usageErr != nil {
				return result, menuFailure(apperror.Unavailable, "product_unavailable", "No pudimos validar los pedidos abiertos del producto.")
			}
			if usedByOpenOrder {
				return result, menuFailure(apperror.Conflict, "quantity_control_open_orders", "Cierra o cancela los pedidos abiertos de este producto antes de activar un control de cantidad.")
			}
		}
		var used bool
		switch currentControl {
		case "inventory":
			err = func() error { var err error; used, err = a.repo.HasStockMovements(tx.Context(), s, id); return err }()
		case "portions":
			err = func() error { var err error; used, err = a.repo.HasPortionSales(tx.Context(), s, id); return err }()
		}
		if err != nil {
			return result, menuFailure(apperror.Unavailable, "product_unavailable", "No pudimos validar el historial de cantidades.")
		}
		if used {
			return result, menuFailure(apperror.Conflict, "quantity_control_in_use", "No puedes cambiar el control de cantidad porque el producto ya tiene movimientos o ventas asociados.")
		}
	}

	var p product
	err = func() error {
		var err error
		p, err = a.repo.UpdateProduct(tx.Context(), s, id, in, skuValue, active, featured)
		return err
	}()
	if err != nil {
		return result, menuFailure(apperror.Conflict, "product_conflict", "No se pudo actualizar el producto.")
	}
	if err = a.attachProductImage(tx.Context(), s, p.ID, in); err != nil {
		return result, err
	}
	if err = tx.Commit(ctx); err != nil {
		return result, menuFailure(apperror.Unavailable, "product_unavailable", "No pudimos actualizar el producto.")
	}
	_ = a.audit.Record(ctx, s, "product.updated", "product", p.ID)
	return a.hydratedWrite(ctx, s, p)

}
// requireProductCategory acepta una categoría activa de la empresa o, al editar,
// la categoría que el producto ya tiene aunque se haya desactivado.
func (a *Catalog) requireProductCategory(ctx context.Context, s scope, categoryID string, productID *string) error {
	available, err := a.repo.ProductCategoryAvailable(ctx, s, categoryID, productID)
	if err != nil {
		return menuFailure(apperror.Unavailable, "category_unavailable", "No pudimos validar la categoría.")
	}
	if !available {
		return menuFailure(apperror.Invalid, "invalid_category", "La categoría seleccionada no existe o está inactiva. Elige otra categoría.")
	}
	return nil
}
func (a *Catalog) SetProductActive(ctx context.Context, s scope, id string, active bool) error {
	tag, err := a.repo.SetProductActive(ctx, s, id, struct{ Active *bool }{&active})
	if err != nil {
		return menuFailure(apperror.Unavailable, "product_unavailable", "No pudimos actualizar el estado del producto.")
	}
	if tag == 0 {
		return menuFailure(apperror.NotFound, "product_not_found", "El producto no existe.")
	}
	action := "product.deactivated"
	if active {
		action = "product.activated"
	}
	_ = a.audit.Record(ctx, s, action, "product", id)
	return nil

}
func (a *Catalog) DeactivateProduct(ctx context.Context, s scope, id string) error {
	tag, err := a.repo.DeactivateProduct(ctx, s, id)
	if err != nil || tag == 0 {
		return menuFailure(apperror.NotFound, "product_not_found", "El producto no existe o ya está inactivo.")
	}
	_ = a.audit.Record(ctx, s, "product.deactivated", "product", id)
	return nil

}
func (a *Catalog) Categories(ctx context.Context, s scope, f CatalogFilter) (CategoryPage, error) {
	f.ProductType = strings.ToLower(strings.TrimSpace(f.ProductType))
	if f.ProductType != "" && f.ProductType != "prepared" && f.ProductType != "retail" {
		return CategoryPage{}, menuFailure(apperror.Invalid, "invalid_category_filter", "El tipo de producto para filtrar categorías no es válido.")
	}
	out, err := a.repo.Categories(ctx, s, f)
	if err != nil {
		return out, menuFailure(apperror.Unavailable, "categories_unavailable", "No pudimos cargar las categorías.")
	}
	return out, nil
}
func (a *Catalog) Products(ctx context.Context, s scope, f CatalogFilter) (ProductPage, error) {
	out, err := a.repo.Products(ctx, s, f)
	if err != nil {
		return out, menuFailure(apperror.Unavailable, "products_unavailable", "No pudimos cargar los productos.")
	}
	out.ServiceDestinationOptions = domain.ServiceDestinations
	ids := make([]string, 0, len(out.Items))
	for _, p := range out.Items {
		ids = append(ids, p.ID)
	}
	views, err := a.productImageViews(ctx, s, ids, "thumbnail", false)
	if err != nil {
		return out, err
	}
	for i := range out.Items {
		if v, ok := views[out.Items[i].ID]; ok {
			out.Items[i].ImageView = v.ImageView
			out.Items[i].ImageURL = v.ImageURL
		}
	}
	return out, nil
}
func (a *Catalog) nextSKU(ctx context.Context, s scope) (string, error) {
	count, err := a.repo.ProductCount(ctx, s)
	return fmt.Sprintf("PROD-%04d", count+1), err
}
func (a *Catalog) Allergens(ctx context.Context, s scope, query string) ([]string, error) {
	out, err := a.repo.Allergens(ctx, s, query)
	if err != nil {
		return nil, menuFailure(apperror.Unavailable, "allergens_unavailable", "No pudimos cargar los alérgenos.")
	}
	return out, nil
}
func (a *Catalog) CreateAllergen(ctx context.Context, s scope, in domain.AllergenInput) (domain.Allergen, error) {
	name := strings.Join(strings.Fields(in.Name), " ")
	if length := utf8.RuneCountInString(name); length < 2 || length > 60 {
		return domain.Allergen{}, menuFailure(apperror.Invalid, "invalid_allergen", "El nombre del alérgeno debe tener entre 2 y 60 caracteres.")
	}
	out, err := a.repo.CreateAllergen(ctx, s, name)
	if errors.Is(err, ErrAllergenConflict) {
		return out, menuFailure(apperror.Conflict, "allergen_conflict", "Ya existe un alérgeno con ese nombre.")
	}
	if err != nil {
		return out, menuFailure(apperror.Unavailable, "allergen_unavailable", "No pudimos registrar el alérgeno.")
	}
	_ = a.audit.Record(ctx, s, "allergen.created", "allergen", out.ID)
	return out, nil
}
func (a *Catalog) UploadImage(ctx context.Context, s scope, id, contentType string, content io.Reader) (map[string]string, error) {
	if a.direct != nil {
		return nil, menuFailure(apperror.Invalid, "direct_upload_required", "Sube la imagen directamente a Storage con una autorización temporal.")
	}
	exists, err := a.repo.ProductExists(ctx, s, id)
	if err != nil || !exists {
		return nil, menuFailure(apperror.NotFound, "product_not_found", "El producto no existe.")
	}
	ext := ""
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	case "image/jpeg", "image/jpg":
		ext = ".jpg"
	default:
		return nil, menuFailure(apperror.Invalid, "image_format", "Solo se aceptan imágenes PNG, JPEG o WebP.")
	}
	url, err := a.images.Save(ctx, id, ext, content)
	if err != nil {
		return nil, menuFailure(apperror.Unavailable, "image_storage", "No pudimos guardar la imagen.")
	}
	if err = a.repo.SetImage(ctx, s, id, url); err != nil {
		return nil, menuFailure(apperror.Unavailable, "image_storage", "No pudimos vincular la imagen al producto.")
	}
	_ = a.audit.Record(ctx, s, "product.image_uploaded", "product", id)
	return map[string]string{"imageUrl": url}, nil
}
