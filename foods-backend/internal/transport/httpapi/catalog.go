package httpapi

import (
	"net/http"
)

func (a *API) listCategories(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.ListCategories(w, r)
}
func (a *API) listProducts(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.ListProducts(w, r)
}
func (a *API) getProduct(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.GetProduct(w, r)
}
func (a *API) listAllergens(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.ListAllergens(w, r)
}
func (a *API) createAllergen(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.CreateAllergen(w, r)
}
func (a *API) uploadProductImage(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.UploadProductImage(w, r)
}
func (a *API) createCategory(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.CreateCategory(w, r)
}
func (a *API) updateCategory(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.UpdateCategory(w, r)
}
func (a *API) deactivateCategory(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.DeactivateCategory(w, r)
}
func (a *API) updateCategoryStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.UpdateCategoryStatus(w, r)
}
func (a *API) createProduct(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.CreateProduct(w, r)
}
func (a *API) updateProduct(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.UpdateProduct(w, r)
}
func (a *API) deactivateProduct(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.DeactivateProduct(w, r)
}
func (a *API) updateProductStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.UpdateProductStatus(w, r)
}

type MenuCatalogHandler interface {
	AuthorizeProductImage(http.ResponseWriter, *http.Request)
	CompleteProductImage(http.ResponseWriter, *http.Request)
	GetProductImage(http.ResponseWriter, *http.Request)
	AttachProductImage(http.ResponseWriter, *http.Request)
	ListRecipes(http.ResponseWriter, *http.Request)
	GetRecipe(http.ResponseWriter, *http.Request)
	SaveRecipe(http.ResponseWriter, *http.Request)
	ListProductAvailability(http.ResponseWriter, *http.Request)
	UpdateProductAvailability(http.ResponseWriter, *http.Request)
	ListProductAvailabilityHistory(http.ResponseWriter, *http.Request)
	GetProductModifiers(http.ResponseWriter, *http.Request)
	SaveProductModifiers(http.ResponseWriter, *http.Request)
	ListCombos(http.ResponseWriter, *http.Request)
	ListOrderCombos(http.ResponseWriter, *http.Request)
	GetCombo(http.ResponseWriter, *http.Request)
	GetOrderCombo(http.ResponseWriter, *http.Request)
	SaveCombo(http.ResponseWriter, *http.Request)
	UpdateComboStatus(http.ResponseWriter, *http.Request)
	ListCategories(http.ResponseWriter, *http.Request)
	ListProducts(http.ResponseWriter, *http.Request)
	GetProduct(http.ResponseWriter, *http.Request)
	ListAllergens(http.ResponseWriter, *http.Request)
	CreateAllergen(http.ResponseWriter, *http.Request)
	UploadProductImage(http.ResponseWriter, *http.Request)
	CreateCategory(http.ResponseWriter, *http.Request)
	UpdateCategory(http.ResponseWriter, *http.Request)
	DeactivateCategory(http.ResponseWriter, *http.Request)
	UpdateCategoryStatus(http.ResponseWriter, *http.Request)
	CreateProduct(http.ResponseWriter, *http.Request)
	UpdateProduct(http.ResponseWriter, *http.Request)
	DeactivateProduct(http.ResponseWriter, *http.Request)
	UpdateProductStatus(http.ResponseWriter, *http.Request)
}
