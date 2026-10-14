package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/menu/application"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"net/http"
)

type Catalog struct{ service *application.Catalog }

func NewCatalog(service *application.Catalog) *Catalog { return &Catalog{service} }
func menuScope(r *http.Request) requestscope.Scope     { s, _ := requestscope.From(r.Context()); return s }
func respond(w http.ResponseWriter, status int, out any, err error) {
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	if status == 204 {
		w.WriteHeader(status)
		return
	}
	shared.WriteJSON(w, status, out)
}
func catalogFilter(r *http.Request) application.CatalogFilter {
	page, size := shared.PageParams(r)
	q := r.URL.Query()
	return application.CatalogFilter{Page: page, Size: size, Query: q.Get("q"), Status: q.Get("status"), CategoryID: q.Get("categoryId"), ProductType: q.Get("productType"), IncludeInactive: q.Get("includeInactive") == "true"}
}
func (h *Catalog) ListCategories(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Categories(r.Context(), menuScope(r), catalogFilter(r))
	respond(w, 200, out, err)
}
func (h *Catalog) ListProducts(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Products(r.Context(), menuScope(r), catalogFilter(r))
	respond(w, 200, out, err)
}
func (h *Catalog) GetProduct(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Product(r.Context(), menuScope(r), r.PathValue("id"))
	respond(w, 200, out, err)
}
func (h *Catalog) ListAllergens(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Allergens(r.Context(), menuScope(r), r.URL.Query().Get("q"))
	respond(w, 200, map[string]any{"items": out}, err)
}
func (h *Catalog) CreateAllergen(w http.ResponseWriter, r *http.Request) {
	var in domain.AllergenInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_allergen", "El nombre del alérgeno debe tener entre 2 y 60 caracteres.")
		return
	}
	out, err := h.service.CreateAllergen(r.Context(), menuScope(r), in)
	respond(w, 201, out, err)
}
func (h *Catalog) UploadProductImage(w http.ResponseWriter, r *http.Request) {
	if h.service.DirectImagesEnabled() {
		shared.Fail(w, 400, "direct_upload_required", "La imagen debe subirse directamente a Storage.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		shared.Fail(w, 400, "image_too_large", "La imagen no puede pesar más de 5 MB.")
		return
	}
	defer r.MultipartForm.RemoveAll()
	file, header, err := r.FormFile("file")
	if err != nil {
		shared.Fail(w, 400, "image_required", "Selecciona una imagen para subir.")
		return
	}
	defer file.Close()
	out, err := h.service.UploadImage(r.Context(), menuScope(r), r.PathValue("id"), header.Header.Get("Content-Type"), file)
	respond(w, 200, out, err)
}
func (h *Catalog) AuthorizeProductImage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	var in domain.ImageUploadInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil {
		shared.Fail(w, 400, "invalid_image", "Los datos de la imagen no son válidos.")
		return
	}
	out, err := h.service.AuthorizeImage(r.Context(), menuScope(r), in)
	respond(w, 201, out, err)
}
func (h *Catalog) CompleteProductImage(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.CompleteImage(r.Context(), menuScope(r), r.PathValue("imageId"))
	respond(w, 200, out, err)
}
func (h *Catalog) GetProductImage(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.GetImage(r.Context(), menuScope(r), r.PathValue("imageId"))
	respond(w, 200, out, err)
}
func (h *Catalog) AttachProductImage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var in struct {
		ImageID string `json:"imageId"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.ImageID == "" {
		shared.Fail(w, 400, "invalid_image", "Selecciona una imagen subida.")
		return
	}
	out, err := h.service.AttachImage(r.Context(), menuScope(r), r.PathValue("id"), in.ImageID)
	respond(w, 200, out, err)
}
func (h *Catalog) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var in domain.CategoryInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_category", "El nombre de la categoría es obligatorio.")
		return
	}
	out, err := h.service.CreateCategory(r.Context(), menuScope(r), in)
	respond(w, 201, out, err)
}
func (h *Catalog) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	var in domain.CategoryInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_category", "El nombre de la categoría es obligatorio.")
		return
	}
	out, err := h.service.UpdateCategory(r.Context(), menuScope(r), r.PathValue("id"), in)
	respond(w, 200, out, err)
}
func (h *Catalog) DeactivateCategory(w http.ResponseWriter, r *http.Request) {
	err := h.service.DeactivateCategory(r.Context(), menuScope(r), r.PathValue("id"))
	respond(w, 204, nil, err)
}
func (h *Catalog) UpdateCategoryStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Active *bool `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.Active == nil {
		shared.Fail(w, 400, "invalid_status", "Indica si la categoría debe estar activa o inactiva.")
		return
	}
	err := h.service.SetCategoryActive(r.Context(), menuScope(r), r.PathValue("id"), *in.Active)
	respond(w, 204, nil, err)
}
func (h *Catalog) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var in domain.ProductInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_product", "Nombre y precio son obligatorios.")
		return
	}
	out, err := h.service.CreateProduct(r.Context(), menuScope(r), in)
	respond(w, 201, out, err)
}
func (h *Catalog) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	var in domain.ProductInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_product", "Nombre y precio son obligatorios.")
		return
	}
	out, err := h.service.UpdateProduct(r.Context(), menuScope(r), r.PathValue("id"), in)
	respond(w, 200, out, err)
}
func (h *Catalog) DeactivateProduct(w http.ResponseWriter, r *http.Request) {
	err := h.service.DeactivateProduct(r.Context(), menuScope(r), r.PathValue("id"))
	respond(w, 204, nil, err)
}
func (h *Catalog) UpdateProductStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Active *bool `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.Active == nil {
		shared.Fail(w, 400, "invalid_status", "Indica si el producto debe estar activo o inactivo.")
		return
	}
	err := h.service.SetProductActive(r.Context(), menuScope(r), r.PathValue("id"), *in.Active)
	respond(w, 204, nil, err)
}
