// Package contracts is the published integration API of Menu.
// Consumers must not depend on Menu's domain or application package directly.
package contracts

import "github.com/foods-platform/foods-backend/internal/modules/menu/domain"

type ProductInput = domain.ProductInput
type PublicMenuItem = domain.PublicMenuItem
type PublicComboDetail = domain.PublicComboDetail
type ModifierConfigView = domain.ModifierConfigView

var NormalizeProduct = domain.NormalizeProduct
var ServiceDestinations = domain.ServiceDestinations
