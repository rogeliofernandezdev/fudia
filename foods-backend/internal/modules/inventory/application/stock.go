package application

import (
	"context"
	"errors"
	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	inventorycontracts "github.com/foods-platform/foods-backend/internal/modules/inventory/contracts"
	menucontracts "github.com/foods-platform/foods-backend/internal/modules/menu/contracts"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"math"
	"strings"
	"time"
)

type scope = requestscope.Scope

var ErrNotFound = errors.New("inventory record not found")

type StockFilter struct {
	Query, InventoryItemID, ProductID, MovementType, SourceType, From, To string
	Page, Size                                                            int
}
type Stock struct {
	repo  StockRepository
	work  unitofwork.Manager
	audit audit.Repository
}

func NewStock(repo StockRepository, work unitofwork.Manager, audit audit.Repository) *Stock {
	return &Stock{repo, work, audit}
}
func stockFailure(kind apperror.Kind, code, message string) error {
	return apperror.New(kind, code, message)
}

type StockRepository interface {
	OrderBusinessDate(ctx context.Context, s scope, orderID string) (string, error)
	LockQuantityProduct(ctx context.Context, s scope, productID string) (string, string, error)
	LockPortions(ctx context.Context, s scope, productID string, businessDate string) (*int, int, string, error)
	UpdateSoldPortions(ctx context.Context, s scope, productID string, businessDate string, nextSold int) (int64, error)
	LockProductInventory(ctx context.Context, s scope, productID string) (string, error)
	RecipeModules(context.Context, scope) (bool, bool, error)
	ActiveRecipe(context.Context, scope, string) (string, float64, error)
	RecipeIngredients(context.Context, scope, string) ([]domain.RecipeIngredient, error)
	NetRecipeUsage(context.Context, scope, string) (map[string]float64, error)
	SaveBalance(context.Context, scope, string, float64, float64) (int64, error)
	TransferDestination(ctx context.Context, s scope, in domain.TransferInput) (string, error)
	ExistingTransfer(ctx context.Context, s scope, in domain.TransferInput) (string, string, string, error)
	CreateTransfer(ctx context.Context, s scope, in domain.TransferInput) (string, string, error)
	TransferItem(ctx context.Context, s scope, line domain.TransferLine) (*string, string, error)
	EnsureTransferBalances(ctx context.Context, s scope, line domain.TransferLine, in domain.TransferInput) (int64, error)
	TransferSourceBalance(ctx context.Context, s scope, line domain.TransferLine, srcAfter float64, srcAvg float64) (int64, error)
	TransferDestinationBalance(ctx context.Context, s scope, line domain.TransferLine, in domain.TransferInput, dstAfter float64, dstAvg float64) (int64, error)
	CreateTransferItem(ctx context.Context, s scope, transferID string, line domain.TransferLine, src domain.LocationBalance) (int64, error)
	TransferIncomingMovement(ctx context.Context, s scope, in domain.TransferInput, productID *string, line domain.TransferLine, dstAfter float64, transferID string, code string, src domain.LocationBalance, valueDelta float64, balanceValue float64) (int64, error)
	LockTransferBalances(context.Context, scope, string, string) (map[string]domain.LocationBalance, error)
	Transfers(context.Context, scope, StockFilter) (TransfersPage, error)
	Presentations(context.Context, scope, string) ([]domain.InventoryPresentationOption, error)
	LockUnit(ctx context.Context, s scope, in domain.InventoryCombinationInput) (string, error)
	EnsurePresentationType(ctx context.Context, s scope, in domain.InventoryCombinationInput) (int64, error)
	PresentationTypeName(ctx context.Context, s scope, in domain.InventoryCombinationInput) (string, error)
	ClearDefaultCombination(ctx context.Context, s scope, in domain.InventoryCombinationInput) (int64, error)
	SaveCombination(ctx context.Context, s scope, in domain.InventoryCombinationInput) (string, error)
	AuditCombination(ctx context.Context, s scope, id string, in domain.InventoryCombinationInput) (int64, error)
	LockPresentationItem(ctx context.Context, s scope, id string) (string, error)
	AuditPresentation(ctx context.Context, s scope, id string, itemID string, in domain.InventoryPresentationInput) (int64, error)
	Presentation(ctx context.Context, s scope, id string) (domain.InventoryPresentationOption, error)
	Combinations(context.Context, scope, string) ([]domain.InventoryCombinationView, error)
	ItemExists(ctx context.Context, s scope, id string) (bool, error)
	LocationSettings(ctx context.Context, s scope, id string, in domain.LocationSettings) (int64, error)
	UnitExists(context.Context, scope, string) (bool, error)
	CombinationAllowed(context.Context, scope, string, string) (bool, error)
	EnsurePresentation(context.Context, scope, string, string, float64) (string, error)
	DefaultPresentation(context.Context, scope, string, string) error
	SaveMovement(context.Context, scope, Movement) error
	LockBalance(context.Context, scope, string) (domain.InventoryBalanceLock, error)
	RetailCategoryAllowed(ctx context.Context, s scope, organizationID string, in domain.InventoryEntryInput) (bool, error)
	CreateRetailProduct(ctx context.Context, s scope, organizationID string, in domain.InventoryEntryInput) (string, string, string, error)
	LinkRetailProduct(ctx context.Context, s scope, organizationID string, productID string, in domain.InventoryEntryInput, result domain.InventoryCatalogCreateResult) (string, string, error)
	CreateIngredient(ctx context.Context, s scope, organizationID string, in domain.InventoryEntryInput, result domain.InventoryCatalogCreateResult) (string, string, string, error)
	LockEntryItem(ctx context.Context, s scope, in domain.InventoryEntryInput) (string, string, string, string, *string, *string, error)
	LockEntryProduct(ctx context.Context, s scope, in domain.InventoryEntryInput) (string, string, string, string, error)
	LinkEntryProduct(ctx context.Context, s scope, in domain.InventoryEntryInput, existingProductID string, itemName string) (string, string, error)
	EntryLocationSettings(ctx context.Context, s scope, inventoryItemID string, in domain.InventoryEntryInput) (int64, error)
	EnsureEntryBalance(ctx context.Context, s scope, inventoryItemID string) (int64, error)
	LockEntryBalance(ctx context.Context, s scope, inventoryItemID string) (float64, float64, error)
	UpdateEntryBalance(ctx context.Context, s scope, inventoryItemID string, balanceAfter float64) (int64, error)
	CreateEntry(ctx context.Context, s scope, productID *string, inventoryItemID string, in domain.InventoryEntryInput, presentationID string) (string, error)
	LockAdjustmentItem(ctx context.Context, s scope, in domain.InventoryAdjustmentInput) (*string, string, string, error)
	EnsureAdjustmentBalance(ctx context.Context, s scope, in domain.InventoryAdjustmentInput) (int64, error)
	LockAdjustmentBalance(ctx context.Context, s scope, in domain.InventoryAdjustmentInput) (float64, float64, error)
	UpdateAdjustmentBalance(ctx context.Context, s scope, in domain.InventoryAdjustmentInput, stockAfter float64, newAverage float64) (int64, error)
	CreateAdjustment(ctx context.Context, s scope, in domain.InventoryAdjustmentInput, stockBefore float64, stockAfter float64) (string, time.Time, error)
	Items(context.Context, scope, StockFilter) (ItemsPage, error)
	Products(context.Context, scope, StockFilter) (ProductsPage, error)
	Movements(context.Context, scope, StockFilter) (MovementsPage, error)
	LinkableProducts(ctx context.Context, s scope, query string, limit int) ([]domain.LinkableProduct, error)
	LockLinkableProduct(ctx context.Context, s scope, productID string) (domain.LinkProductCandidate, error)
	ProductInOpenOrders(ctx context.Context, s scope, productID string) (bool, error)
	MarkProductInventoryControlled(ctx context.Context, s scope, product domain.LinkProductCandidate) error
}

func (a *Stock) validateUnit(ctx context.Context, s scope, code string) error {
	valid, err := a.repo.UnitExists(ctx, s, code)
	if err != nil {
		return stockFailure(apperror.Unavailable, "inventory_units_unavailable", "No pudimos validar la unidad base.")
	}
	if !valid {
		return stockFailure(apperror.Invalid, "invalid_inventory_unit", "Selecciona una unidad base registrada en tu empresa.")
	}
	return nil
}
func (a *Stock) ensurePresentation(ctx context.Context, s scope, item, kind string, factor float64) (string, error) {
	allowed, err := a.repo.CombinationAllowed(ctx, s, item, kind)
	if err != nil {
		return "", err
	}
	if !allowed {
		return "", &inventoryCombinationError{}
	}
	return a.repo.EnsurePresentation(ctx, s, item, kind, factor)
}

type inventoryCombinationError struct{}

func (*inventoryCombinationError) Error() string {
	return "La presentación no está configurada para la unidad base de este artículo."
}

// Movement remains an alias for existing callers; published ledger DTO lives in contracts.
type Movement = inventorycontracts.Movement

func (a *Stock) movement(ctx context.Context, s scope, productID *string, item, kind string, delta, balance, cost float64, source, sourceID, note string) error {
	return a.repo.SaveMovement(ctx, s, Movement{ProductID: productID, ItemID: item, Type: kind, Delta: delta, Balance: balance, UnitCost: cost, ValueDelta: math.Round(delta*cost*10000) / 10000, BalanceValue: math.Round(balance*cost*10000) / 10000, SourceType: source, SourceID: sourceID, Note: note})
}
func (a *Stock) CreateCatalogItem(ctx context.Context, s scope, in domain.InventoryEntryInput) (domain.InventoryCatalogCreateResult, error) {
	organizationID := s.OrganizationID
	result := domain.InventoryCatalogCreateResult{Unit: in.Unit}
	if invalid := a.validateUnit(ctx, s, in.Unit); invalid != nil {
		return result, invalid
	}
	switch {
	case in.NewProduct != nil:
		productIn := menucontracts.ProductInput{
			SKU:             in.NewProduct.SKU,
			Name:            in.NewProduct.Name,
			Description:     in.NewProduct.Description,
			CategoryID:      in.NewProduct.CategoryID,
			Price:           in.NewProduct.Price,
			ProductType:     "retail",
			QuantityControl: "inventory",
		}
		if _, invalidProduct := menucontracts.NormalizeProduct(productIn); invalidProduct != "" {
			return result, stockFailure(apperror.Invalid, "invalid_product", invalidProduct)
		}
		var categoryAllowed bool
		if err := func() error {
			var err error
			categoryAllowed, err = a.repo.RetailCategoryAllowed(ctx, s, organizationID, in)
			return err
		}(); err != nil {
			return result, stockFailure(apperror.Unavailable, "categories_unavailable", "No pudimos validar la categoría.")
		}
		if !categoryAllowed {
			return result, stockFailure(apperror.Conflict, "category_unavailable", "La categoría seleccionada no existe o está inactiva.")
		}
		var productID string
		if err := func() error {
			var err error
			productID, result.SKU, result.Name, err = a.repo.CreateRetailProduct(ctx, s, organizationID, in)
			return err
		}(); err != nil {
			return result, stockFailure(apperror.Conflict, "product_conflict", "No pudimos crear el producto. Revisa nombre, categoría y precio.")
		}
		result.ProductID = &productID
		result.Kind = "product"
		if err := func() error {
			var err error
			result.InventoryItemID, result.Unit, err = a.repo.LinkRetailProduct(ctx, s, organizationID, productID, in, result)
			return err
		}(); err != nil {
			return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos vincular el producto al inventario.")
		}
	case in.NewIngredient != nil:
		result.Name = in.NewIngredient.Name
		result.Kind = "ingredient"
		if err := func() error {
			var err error
			result.InventoryItemID, result.SKU, result.Unit, err = a.repo.CreateIngredient(ctx, s, organizationID, in, result)
			return err
		}(); err != nil {
			return result, stockFailure(apperror.Conflict, "ingredient_conflict", "No pudimos crear el insumo.")
		}
	default:
		return result, stockFailure(apperror.Invalid, "invalid_inventory_item", "Selecciona el tipo de artículo a crear.")
	}
	return result, nil
}
func (a *Stock) CreateEntry(ctx context.Context, s scope, in domain.InventoryEntryInput) (map[string]any, error) {
	var result map[string]any
	var invalid string
	in, invalid = domain.NormalizeInventoryEntry(in)
	if invalid != "" {
		return result, stockFailure(apperror.Invalid, "invalid_inventory_entry", invalid)
	}

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos registrar la entrada.")
	}
	defer tx.Rollback(ctx)

	var productID *string
	var itemSKU, itemName, inventoryItemID, inventoryUnit string
	createdProduct := false
	createdInventoryItem := false

	switch {
	case in.NewProduct != nil, in.NewIngredient != nil:
		created, createErr := a.CreateCatalogItem(tx.Context(), s, in)
		if createErr != nil {
			return result, createErr
		}
		inventoryItemID = created.InventoryItemID
		productID = created.ProductID
		itemSKU = created.SKU
		itemName = created.Name
		inventoryUnit = created.Unit
		createdProduct = created.Kind == "product"
		createdInventoryItem = true

	case in.InventoryItemID != "":
		var linkedProductID *string
		var quantityControl *string
		err = func() error {
			var err error
			inventoryItemID, itemSKU, itemName, inventoryUnit, linkedProductID, quantityControl, err = a.repo.LockEntryItem(tx.Context(), s, in)
			return err
		}()
		if errors.Is(err, ErrNotFound) {
			return result, stockFailure(apperror.NotFound, "inventory_item_not_found", "El artículo de inventario no existe o está inactivo.")
		}
		if err != nil {
			return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos validar el artículo de inventario.")
		}
		if linkedProductID != nil {
			if quantityControl == nil || *quantityControl != "inventory" {
				return result, stockFailure(apperror.Conflict, "quantity_control_conflict", "El producto ya no está configurado como Inventario físico.")
			}
			productID = linkedProductID
		}
	case in.ProductID != "":
		if invalid := a.validateUnit(tx.Context(), s, in.Unit); invalid != nil {
			return result, invalid
		}
		var quantityControl string
		var existingProductID string
		err = func() error {
			var err error
			existingProductID, itemSKU, itemName, quantityControl, err = a.repo.LockEntryProduct(tx.Context(), s, in)
			return err
		}()
		if errors.Is(err, ErrNotFound) {
			return result, stockFailure(apperror.NotFound, "product_not_found", "El producto no existe o está inactivo.")
		}
		if err != nil {
			return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos validar el producto.")
		}
		if quantityControl != "inventory" {
			return result, stockFailure(apperror.Conflict, "quantity_control_conflict", "Solo los productos de Inventario físico pueden recibir entradas.")
		}
		productID = &existingProductID
		err = func() error {
			var err error
			inventoryItemID, inventoryUnit, err = a.repo.LinkEntryProduct(tx.Context(), s, in, existingProductID, itemName)
			return err
		}()
		if err != nil {
			return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos vincular el producto al inventario.")
		}
	}

	if inventoryUnit != in.Unit {
		return result, stockFailure(apperror.Conflict, "inventory_unit_conflict", "La unidad base de este artículo es "+inventoryUnit+". Registra la entrada usando esa misma unidad base.")
	}
	if _, err = a.repo.EntryLocationSettings(tx.Context(), s, inventoryItemID, in); err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos actualizar el stock mínimo del local.")
	}

	presentationID, err := a.ensurePresentation(tx.Context(), s, inventoryItemID, in.PresentationType, in.UnitsPerPresentation)
	var invalidCombination *inventoryCombinationError
	if errors.As(err, &invalidCombination) {
		return result, stockFailure(apperror.Invalid, "invalid_inventory_combination", invalidCombination.Error())
	}
	if err == nil && createdInventoryItem {
		err = a.repo.DefaultPresentation(tx.Context(), s, inventoryItemID, presentationID)
	}
	if err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos guardar la presentación de entrada.")
	}

	stockQuantity := math.Round(in.Quantity*in.UnitsPerPresentation*1000) / 1000
	if stockQuantity <= 0 {
		return result, stockFailure(apperror.Invalid, "invalid_inventory_entry", "La equivalencia en unidades base debe ser mayor que cero.")
	}

	if _, err = a.repo.EnsureEntryBalance(tx.Context(), s, inventoryItemID); err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos preparar el saldo de inventario.")
	}

	var current, averageCost float64
	if err = func() error {
		var err error
		current, averageCost, err = a.repo.LockEntryBalance(tx.Context(), s, inventoryItemID)
		return err
	}(); err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos bloquear el saldo de inventario.")
	}
	balanceAfter := math.Round((current+stockQuantity)*1000) / 1000
	if _, err = a.repo.UpdateEntryBalance(tx.Context(), s, inventoryItemID, balanceAfter); err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos actualizar el saldo.")
	}

	var entryID string
	if err = func() error {
		var err error
		entryID, err = a.repo.CreateEntry(tx.Context(), s, productID, inventoryItemID, in, presentationID)
		return err
	}(); err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos registrar el documento de entrada.")
	}

	if err = a.movement(tx.Context(), s, productID, inventoryItemID, "entry", stockQuantity, balanceAfter, averageCost, "inventory_entry", entryID, in.Note); err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos registrar el movimiento de Kárdex.")
	}

	if err = tx.Commit(ctx); err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos registrar la entrada.")
	}

	if createdProduct && productID != nil {
		_ = a.audit.Record(ctx, s, "product.created_from_inventory", "product", *productID)
	}
	if createdInventoryItem && productID == nil {
		_ = a.audit.Record(ctx, s, "inventory.ingredient_created", "inventory_item", inventoryItemID)
	}
	_ = a.audit.Record(ctx, s, "inventory.entry_created", "inventory_item", inventoryItemID)

	kind := "ingredient"
	if productID != nil {
		kind = "product"
	}
	return map[string]any{
		"id":                   entryID,
		"inventoryItemId":      inventoryItemID,
		"productId":            productID,
		"sku":                  itemSKU,
		"name":                 itemName,
		"kind":                 kind,
		"quantity":             in.Quantity,
		"presentationId":       presentationID,
		"presentationType":     in.PresentationType,
		"unitsPerPresentation": in.UnitsPerPresentation,
		"stockQuantity":        stockQuantity,
		"unit":                 in.Unit,
		"balance":              balanceAfter,
		"createdProduct":       createdProduct,
		"createdInventoryItem": createdInventoryItem,
	}, nil

}
func (a *Stock) CreateAdjustment(ctx context.Context, s scope, in domain.InventoryAdjustmentInput) (map[string]any, error) {
	var result map[string]any
	var invalid string
	in, invalid = domain.NormalizeInventoryAdjustment(in)
	if invalid != "" {
		return result, stockFailure(apperror.Invalid, "invalid_inventory_adjustment", invalid)
	}

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos registrar el ajuste.")
	}
	defer tx.Rollback(ctx)

	var productID *string
	var itemName, unit string
	err = func() error {
		var err error
		productID, itemName, unit, err = a.repo.LockAdjustmentItem(tx.Context(), s, in)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		return result, stockFailure(apperror.NotFound, "inventory_item_not_found", "El artículo no existe o está inactivo.")
	}
	if err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos validar el artículo.")
	}

	if _, err = a.repo.EnsureAdjustmentBalance(tx.Context(), s, in); err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos preparar el saldo del artículo.")
	}

	var stockBefore, averageCost float64
	if err = func() error {
		var err error
		stockBefore, averageCost, err = a.repo.LockAdjustmentBalance(tx.Context(), s, in)
		return err
	}(); err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos bloquear el saldo del artículo.")
	}

	delta := in.Quantity
	if in.MovementType == "exit" {
		delta = -in.Quantity
	}
	stockAfter := math.Round((stockBefore+delta)*1000) / 1000
	if stockAfter < -0.000001 {
		return result, stockFailure(apperror.Conflict, "insufficient_stock", "La salida supera el stock actual del artículo.")
	}
	if stockAfter < 0 {
		stockAfter = 0
	}

	newAverage := averageCost
	if _, err = a.repo.UpdateAdjustmentBalance(tx.Context(), s, in, stockAfter, newAverage); err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos actualizar el stock.")
	}

	var adjustmentID string
	var createdAt time.Time
	if err = func() error {
		var err error
		adjustmentID, createdAt, err = a.repo.CreateAdjustment(tx.Context(), s, in, stockBefore, stockAfter)
		return err
	}(); err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos registrar la trazabilidad del ajuste.")
	}

	if err = a.movement(tx.Context(), s, productID, in.InventoryItemID, "inventory_adjustment", delta, stockAfter, averageCost, "inventory_adjustment", adjustmentID, in.Observation); err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos registrar el movimiento de Kárdex.")
	}

	if err = tx.Commit(ctx); err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos registrar el ajuste.")
	}

	_ = a.audit.Record(ctx, s, "inventory.adjusted", "inventory_item", in.InventoryItemID)
	return map[string]any{
		"id":              adjustmentID,
		"inventoryItemId": in.InventoryItemID,
		"name":            itemName,
		"unit":            unit,
		"movementType":    in.MovementType,
		"reason":          in.Reason,
		"quantity":        in.Quantity,
		"stockBefore":     stockBefore,
		"stockAfter":      stockAfter,
		"observation":     in.Observation,
		"createdAt":       createdAt,
		"createdByName":   s.Name,
	}, nil

}

type ItemsPage struct {
	Items    []domain.InventoryItemView `json:"items"`
	Total    int                        `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"pageSize"`
}

func (a *Stock) Items(ctx context.Context, s scope, f StockFilter) (ItemsPage, error) {
	out, err := a.repo.Items(ctx, s, f)
	if err != nil {
		return out, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos cargar el inventario.")
	}
	return out, nil
}

type ProductsPage struct {
	Items    []domain.InventoryProductOption `json:"items"`
	Total    int                             `json:"total"`
	Page     int                             `json:"page"`
	PageSize int                             `json:"pageSize"`
}

func (a *Stock) Products(ctx context.Context, s scope, f StockFilter) (ProductsPage, error) {
	out, err := a.repo.Products(ctx, s, f)
	if err != nil {
		return out, stockFailure(apperror.Unavailable, "inventory_products_unavailable", "No pudimos cargar los artículos de inventario.")
	}
	return out, nil
}

type MovementsPage struct {
	Items    []domain.InventoryMovementView `json:"items"`
	Total    int                            `json:"total"`
	Page     int                            `json:"page"`
	PageSize int                            `json:"pageSize"`
}

func (a *Stock) Movements(ctx context.Context, s scope, f StockFilter) (MovementsPage, error) {
	f.InventoryItemID = strings.TrimSpace(f.InventoryItemID)
	f.ProductID = strings.TrimSpace(f.ProductID)
	f.MovementType = strings.TrimSpace(f.MovementType)
	f.SourceType = strings.TrimSpace(f.SourceType)
	f.From = strings.TrimSpace(f.From)
	f.To = strings.TrimSpace(f.To)
	if !domain.ValidDateFilters(f.From, f.To) {
		return MovementsPage{}, stockFailure(apperror.Invalid, "invalid_kardex_filter", "Revisa el rango de fechas.")
	}
	out, err := a.repo.Movements(ctx, s, f)
	if err != nil {
		return out, stockFailure(apperror.Unavailable, "kardex_unavailable", "No pudimos cargar los movimientos del Kárdex.")
	}
	return out, nil
}
