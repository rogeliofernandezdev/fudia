package httpapi

import (
	menudomain "github.com/foods-platform/foods-backend/internal/modules/menu/domain"
)

type category = menudomain.Category
type product = menudomain.Product
type productInput = menudomain.ProductInput
type productListItem = menudomain.ProductListItem

var normalizeCategoryProductScope = menudomain.NormalizeCategoryProductScope
var normalizeProduct = menudomain.NormalizeProduct
