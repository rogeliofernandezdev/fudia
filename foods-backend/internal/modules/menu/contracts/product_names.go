package contracts

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

// ProductNames participates in the inventory editor's transaction. Menu keeps
// ownership of the commercial product; inventory authorizes the editing workflow.
type ProductNames interface {
	RenameInventoryProduct(context.Context, requestscope.Scope, string, string) error
}
